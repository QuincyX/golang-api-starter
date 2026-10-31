# Implementation Plan: 管理员账号与鉴权

- Spec: `./spec.md`
- Status: `Implementing`
- Date: `2026-10-31`

## 技术方案

在现有基座上补一条 JWT Bearer 认证链路，不引入 Cookie/refresh：

```text
POST /auth/login
  → auth service：限流 → 查用户 → Argon2id 校验 → 建 session → 签 access
  → 审计 auth.login → 返回 JSON（无 Set-Cookie）

请求受保护接口
  → JWT 验签 → 查 session + user（状态/过期/token_version）
  → 查权限 → Principal → router.Register 权限包装 → handler

POST /auth/logout
  → 吊销当前 session → 审计
```

关键取舍：

- **JWT Bearer，无 Cookie**：不做 CSRF 流程；access 放前端 `sessionStorage`，过期重新登录。
- **不做 refresh**：P1 只发 access；`refresh_tokens` 表保留给未来 spec。8 小时 TTL + 每请求 session 校验，兼顾管理后台工作日和吊销能力。
- **access 不放权限**：权限每请求查 RBAC；角色变更立即生效。
- **审计同事务**：登录失败无业务事务，单独同步写；改密/角色变更/禁用与审计同事务。
- **最后超管保护**：事务内 `SELECT ... FOR UPDATE` 统计 active super_admin，不能先查后写。
- **限流进程内**：单机管理后台用内存令牌桶；多实例再换 Redis。

## 受影响区域

| 区域 | 文件/模块 | 变更 |
| --- | --- | --- |
| 路由/契约 | `internal/router/register.go`、`cmd/api/app.go` | 注册 auth/account/audit 路由；保持 permission/audit 元数据 |
| API | `internal/api/auth/`、`internal/api/account/`、`internal/api/audit/` | DTO、token 响应、session/账号/审计错误映射 |
| Service | `internal/service/auth/`、`internal/service/account/`、`internal/service/auditlog/`、`internal/audit/` | 事务边界、密码、session、权限、审计 |
| Store/SQL | `internal/store/auth.go`、`account.go` 相关方法 | 用户/角色/权限/session/一次性链接参数化查询 |
| 迁移 | `migrations/` | 复用现有表；新增 `00005_password_tokens.sql`；新增 `cmd/seed` |
| 中间件/平台 | `internal/middleware/auth.go`、`internal/platform/jwt/` | 每请求 session/版本校验；登录限流在 service |
| 配置 | `internal/config/config.go`、`.env.example` | `ACCESS_TTL=8h`、`ADMIN_WEB_URL`、`BOOTSTRAP_*`、`METRICS_ADDR` |
| 测试 | `cmd/api/app_test.go`、`internal/**/*_test.go` | 登录/吊销/过期/越权/路由覆盖 |
| 文档 | `docs/architecture.md`、`docs/engineering.md`、`docs/changelog.md`、README、本 spec | 同步 JWT-only 实现与边界 |

## 数据与接口

### 接口

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/admin/v1/auth/login` | 公开 + 限流 | 返回 accessToken/expiresIn/user/permissions；不设 Cookie |
| POST | `/api/admin/v1/auth/logout` | 已登录 | 吊销当前 session、写审计 |
| GET | `/api/admin/v1/auth/me` | 已登录 | 用户、角色、权限 |
| PUT | `/api/admin/v1/auth/password` | 已登录 | 旧密码校验、`token_version++`、吊销全部 session，要求重登 |
| POST | `/api/admin/v1/auth/activate` | 公开 + 一次性 token | 首次设密/重置密码 |
| GET | `/api/admin/v1/accounts` | `account:read` | 分页/过滤 |
| POST | `/api/admin/v1/accounts` | `account:write` | 创建账号，返回一次性设密链接 |
| PATCH | `/api/admin/v1/accounts/{id}` | `account:write` | 资料/状态 |
| PUT | `/api/admin/v1/accounts/{id}/roles` | `role:assign` | 超管 |
| POST | `/api/admin/v1/accounts/{id}/reset-password` | `account:write` | 返回新一次性链接 |
| GET | `/api/admin/v1/accounts/{id}/sessions` | `account:read` | 活跃会话 |
| DELETE | `/api/admin/v1/accounts/{id}/sessions/{sid}` | `account:write` | 吊销 |

### 令牌

- access：HS256，默认 8h（`ACCESS_TTL` 30m–12h）；claims `iss/aud/sub/sid/ver/iat/exp/jti`。
- 响应只含 JSON body：`{accessToken, tokenType:"Bearer", expiresIn, user, permissions}`；**禁止 `Set-Cookie`**。
- 前端存 `sessionStorage`；页面刷新保留，关闭标签页失效；不在 URL、日志、localStorage 中保存。
- 401：access 过期/已吊销/`ver` 不匹配；前端清空状态跳登录，不做 refresh 重试。
- 错误：401/403/429/503，problem+json。

## 事务、鉴权与审计

- 登录：校验用户后建 session（一个事务）；更新 `last_login_at` 可异步但不得影响安全。
- 登出/禁用/改密：吊销 session + bump `token_version`；事务内写审计。改密要求重新登录。
- 一次性链接：创建时生成 32B 随机 token，存 SHA-256 + 过期时间 + 使用时间；消费时用条件 UPDATE 原子标记已用。
- 权限：`admin_user_roles → role_permissions → permissions` 查询；`super_admin` 全量权限；默认拒绝。
- 最后超管：事务内统计并保护。
- 审计：`audit.Recorder.Record(ctx, pgx.Tx, event)` 与业务同事务，字段白名单；登录失败用 best-effort。

## 迁移与回滚

- 现有 `00001_auth.sql` 已包含 users/roles/permissions/sessions/refresh_tokens；P1 只使用 `admin_sessions`，`refresh_tokens` 保留未用。
- 一次性设密链接若不复用现有字段，新增 `00005_auth_hardening.sql`（token_hash/expires_at/used_at），只做加法。
- 初始化超管用 `cmd/seed`（仅当 `admin_users` 为空），不把密码写进迁移。
- 回滚：代码回退到 P0 不影响已有表；新增列/索引保留；不做破坏性删除。

## 测试策略

- 单元：Argon2id 参数/校验、JWT 签发/过期/错误 aud、权限集合、最后超管判定、一次性链接哈希/过期。
- Handler：登录成功/失败、401/403/429、problem+json、无 `Set-Cookie`。
- 集成（testcontainers + PG18）：session 校验、登出/禁用吊销、改密 bump `token_version`、最后超管并发保护、审计同事务。
- 路由契约：新增路由全部有 permission/audit；`verifyRouteCoverage` 通过。
- 人工：`rest/auth.rest` 登录 → me → logout；禁用账号后旧 access 401；过期 token 401。

## 宪法检查

- [x] 遵守 AGENTS 架构边界（api 不查库、service 事务、store SQL）
- [x] 不重复已有工程规则
- [x] 明确非目标和范围
- [x] 有可验证的验收证据
- [x] 框架变更已列入文档任务
- [x] 迁移/鉴权/审计有回滚或恢复说明

## 风险与回滚

| 风险 | 缓解 |
| --- | --- |
| XSS 窃取 sessionStorage token | CSP、禁用 `v-html`、输入清洗、短会话；不做 Cookie 也意味着无 CSRF |
| 8h token 被盗 | 每请求 session/`ver` 校验；超管可吊销 session；异常登录审计/告警 |
| 账号锁定被用于 DoS | IP + 账号双维度、指数退避、允许超管解锁 |
| 最后超管并发 | 事务 + 行锁；数据库约束/测试覆盖 |
| 一次性链接泄漏 | 哈希存储、30 分钟过期、单次使用、改密吊销 session |
| 审计写失败 | 同事务回滚业务；登录失败单独告警 |
| 时钟漂移 | 单机 NTP；JWT leeway 60s |

## 文档影响

- `docs/architecture.md`：鉴权/中间件请求链从 P0 更新为 JWT Bearer。
- `docs/engineering.md`：`ACCESS_TTL`、无 Cookie、sessionStorage/XSS 约束、测试。
- `docs/changelog.md`：记录 001 决策与实现。
- `README.md`：接口表、快速开始、登录说明。
- `specs/001-account-auth/verification.md`：验收证据。
