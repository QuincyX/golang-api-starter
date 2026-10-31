# Feature Specification: 管理员账号与鉴权

- ID: `001`
- Status: `Implementing`
- Created: `2026-10-31`
- Supersedes: `none`

## 问题与目标

P0 只有 JWT 验签和静态开发权限：没有登录、登出落库，也没有 session/账号状态校验。任何人都可以用 `cmd/token` 的行为方式理解鉴权边界，这不适合真实管理后台。

目标：为 homebrew 管理后台提供一套可上线的最小账号体系——邮箱密码登录、JWT Bearer 会话、即时吊销、RBAC 默认拒绝、账号/session 管理、登录与权限变更审计。

## 已澄清决策（2026-10-31）

- **MFA/TOTP**：暂不做。P1 不引入 TOTP、恢复码和绑定流程；后续需要时另立 spec。
- **会话方式**：JWT Bearer，**不使用 Cookie**。P1 不引入 refresh token；access 默认 8 小时（30 分钟–12 小时可配置），过期后重新登录。
- **邀请/重置密码**：超管生成一次性链接，不依赖邮件服务。链接单次使用、30 分钟过期、改密后吊销旧 session。
- **SSO**：暂不接。飞书/OIDC 以后另立 spec，不在 001 范围内。

## 非目标

- 公开注册、邀请邮件、找回密码邮件
- 飞书/企业微信/OIDC SSO
- TOTP/MFA
- refresh token / Cookie / CSRF 流程
- 文章 CRUD、审计查询 UI、多租户、proto3/gRPC

## 用户故事

### User Story 1 - 邮箱密码登录与吊销 (Priority: P1)

管理员用邮箱和密码登录，获得 JWT access token；登出、改密、禁用账号后，旧 token 立即失效；access 过期后重新登录。

**为什么是 P1**：没有登录就没有真实账号边界，后续 RBAC、审计都无法落地。

**独立验证方式**：集成测试覆盖登录成功/失败、登出/禁用后 401、过期 401、限流。

**验收场景**：

1. **Given** 一个 active 管理员，**When** 提交正确邮箱密码，**Then** 返回 accessToken/expiresIn/user/permissions，不设置任何 Cookie。
2. **Given** 已登出或已禁用的账号，**When** 携带旧 access 请求受保护接口，**Then** 返回 401。
3. **Given** access 已过期，**When** 请求受保护接口，**Then** 返回 401；前端清空会话并要求重新登录。
4. **Given** 连续失败登录，**When** 超过阈值，**Then** 账号/IP 被限流或临时锁定。
5. **Given** 登录成功，**When** 检查响应，**Then** 响应头没有 `Set-Cookie`，token 只在 JSON body 中。

### User Story 2 - RBAC 默认拒绝 (Priority: P1)

超管给账号分配角色；editor 只能读写/发布文章，auditor 只能读文章和审计；服务端逐请求校验权限。

**为什么是 P1**：前端隐藏菜单不构成权限；必须有服务端默认拒绝。

**独立验证方式**：用 editor token 遍历管理接口，写操作断言 403；super_admin 可改角色。

**验收场景**：

1. **Given** editor，**When** 调用 `POST /articles/{id}/publish`，**Then** 允许（有 `article:publish`）。
2. **Given** editor，**When** 调用角色管理接口，**Then** 403。
3. **Given** 最后一个 super_admin，**When** 尝试禁用/降级，**Then** 拒绝。
4. **Given** 角色权限变更，**When** 下一次请求，**Then** 新权限立即生效（不依赖旧 JWT）。

### User Story 3 - 账号与会话管理 (Priority: P2)

超管创建/邀请账号、禁用账号、分配角色、查看并吊销某个账号的活跃会话；邀请使用一次性链接。

**独立验证方式**：账号列表/创建/禁用/角色分配/session 吊销各有 handler 测试，关键路径有集成测试。

### User Story 4 - 登录与权限变更审计 (Priority: P2)

登录成功/失败、登出、锁定、改密、角色变更写入 `audit_logs`；关键写操作与业务同事务。

**独立验证方式**：审计行包含 actor、action、result、IP、request_id；业务回滚不残留审计。

## 功能需求

- **FR-001**：`POST /auth/login` 接受邮箱 + 密码；邮箱大小写不敏感；错误统一为“邮箱或密码错误”。
- **FR-002**：密码用 Argon2id PHC 字符串存储；参数落后时登录成功后台重哈希；未知用户也执行 dummy hash。
- **FR-003**：access JWT 默认 8 小时（`ACCESS_TTL` 可配置 30m–12h），claims 只含 `iss/aud/sub/sid/ver/iat/exp/jti`；不包含权限。
- **FR-004**：P1 不引入 refresh token 和 Cookie；access 过期后返回 401，前端清空状态并重新登录。
- **FR-005**：`POST /auth/logout` 吊销当前 session 并写审计；客户端删除本地 token。
- **FR-006**：认证中间件校验签名后，查 `admin_sessions` + `admin_users`：session 未吊销/未过期、账号 active、`claims.ver == token_version`。
- **FR-007**：权限从 `role_permissions` 读取；无权限默认 403；`super_admin` 用角色码或全量权限实现，不把通配符写进 JWT。
- **FR-008**：超管可创建/禁用账号、分配角色、查看/吊销 session；禁止公开注册。
- **FR-009**：不能禁用/降级最后一个 `super_admin`；不能给自己加角色。
- **FR-010**：登录成功/失败、登出、锁定、改密、角色变更写审计；字段白名单，不记密码/token。
- **FR-011**：登录限流：IP 与账号双维度；失败指数退避；返回 `429` + `Retry-After`。
- **FR-012**：首次创建/重置密码置 `must_change_password`；改密后 `token_version++` 并吊销全部 session，要求重新登录。
- **FR-013**：一次性设密链接：随机 32B token，DB 存哈希，30 分钟过期，单次使用；创建/重置时由超管复制链接。

## 边界与异常

- 账号不存在、密码错误、账号锁定：对外统一 401 文案，避免枚举。
- access 过期/已吊销/`ver` 不匹配：401；前端清空 token 并跳登录。
- 权限不足：403，不触发登出。
- 数据库不可用：503；不把 DB 错误返回给客户端。
- 多标签页：各自持有 token；登出只吊销当前 session；需要全端下线时由超管吊销该账号全部 session。
- 最后一名超管保护：并发下用事务/行锁，不能只靠先查后写。

## 安全与审计影响

- 权限码：`account:read/write`、`role:read/write/assign`、`article:*`、`audit:read/export`。
- 审计动作：`auth.login`、`auth.login_failed`、`auth.logout`、`auth.password_change`、`account.create`、`account.disable`、`account.role_update`、`account.reset_password`。
- 敏感数据：密码哈希、JWT、一次性链接 token 不进入日志/审计。
- 无 Cookie，因此没有 CSRF；主要风险是 XSS 窃取前端 token。前端默认存 `sessionStorage`，配合 CSP、禁止 `v-html`；不把 token 放 URL。
- 会话表与 `token_version` 已存在；`refresh_tokens` 表保留但 P1 不使用。

## 约束与依赖

- 上位约束：[AGENTS.md](../../AGENTS.md)
- 架构：[docs/architecture.md](../../docs/architecture.md)
- 工程：[docs/engineering.md](../../docs/engineering.md)
- 设计文档：[docs/design.md](../../docs/design.md)
- 技术选型：PostgreSQL 18、chi/huma、pgx、goose、`golang-jwt/jwt/v5`、Argon2id；单机，不引入 Redis。
- 现有迁移：`00001_auth.sql`、`00004_seed.sql`；如新增字段/索引，另起迁移。

## 成功指标

- [ ] 登录/登出/me 三条路径有 handler + 集成测试。
- [ ] 响应中没有 `Set-Cookie`；token 只在 JSON body。
- [ ] editor 对角色/账号管理接口全部 403。
- [ ] 禁用账号后旧 access 立即 401。
- [ ] 登录失败限流/锁定有测试。
- [x] `go test ./...`、`go vet ./...`、`go build ./...` 通过；迁移在 PG18 验证。

## 框架/文档影响

- [x] 新增 `internal/service/auth`、`internal/store/auth.go`
- [x] 修改 `internal/middleware/auth.go`、`internal/api/auth`、`internal/config`、`internal/router` 注册
- [x] 更新 `docs/architecture.md`、`docs/engineering.md`、`docs/testing.md`、`docs/changelog.md`
- [x] README 的接口表与快速开始
- [x] 新增 `00005_password_tokens.sql`
