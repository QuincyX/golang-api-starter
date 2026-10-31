# 技术方案

> 项目：`homebrew-api-starter`（homebrew 管理后台 API 起始模板）  
> 适用前提：PostgreSQL 18、同源 Vue 管理后台、单机部署、1–3 人团队、管理员 < 200、日请求 < 百万  
> 鉴权：JWT Bearer access（无 Cookie/refresh，默认 8h + 每请求 session 校验）  
> 部署：单机 Docker Compose + Caddy

本文说明“为什么这样选”和“数据/安全边界”。目录职责、请求链和扩展位置见 [architecture.md](./architecture.md)；命令与门禁见 [engineering.md](./engineering.md)；测试规范见 [testing.md](./testing.md)。

## 1. 目标与非目标

**目标**

- 管理后台 API 的可复用底座：账号、登录、RBAC、文章、审计、健康检查、指标。
- 单机可部署，配置全部来自环境变量，迁移可独立执行。
- 路由级权限/审计元数据，启动自检，错误语义统一。
- 新项目复制后能删掉示例业务，保留边界与约束。

**非目标**

- 公开注册、公开文章站点、多租户、计费。
- refresh token、Cookie session、MFA、SSO/OIDC、邮件服务。
- proto3/gRPC、ORM、消息队列、微服务拆分、Kubernetes。
- 审计导出/归档、OTel trace、前端代码。

这些需要新 spec 或 ADR，不在模板默认范围。

## 2. 技术选型

| 层次 | 选型 | 理由 |
| --- | --- | --- |
| 语言 | Go 1.27.3 | 标准库 HTTP、部署简单、静态二进制 |
| 路由 | `go-chi/chi/v5` | 完全兼容 `net/http`，中间件组合清晰 |
| API 契约 | `danielgtaylor/huma/v2` | 结构体生成 OpenAPI 3.1，请求校验与 problem+json |
| 数据库 | PostgreSQL 18 | `uuidv7()`、JSONB、部分索引、`timestamptz` |
| 驱动 | `jackc/pgx/v5` + pgxpool | 原生协议、事务与类型支持好 |
| 查询 | 显式参数化 SQL | 当前规模不需要 sqlc 生成层；SQL 集中在 `internal/store` |
| 迁移 | `pressly/goose/v3` + `embed` | SQL 迁移，独立 Job 执行 |
| 密码 | Argon2id | 内存硬、抗 GPU；参数可升级并透明重哈希 |
| 令牌 | `golang-jwt/jwt/v5` | HS256 access token；无 refresh/Cookie |
| 配置 | `caarlos0/env/v11` + `.env` | 12-factor，启动即校验 |
| 日志 | `log/slog` | JSON、请求级字段、零依赖 |
| Markdown | `goldmark` + `bluemonday` | 渲染与 HTML 清洗分离 |
| 指标 | `prometheus/client_golang` | 请求计数/延迟，独立内网端口 |
| 测试 | 标准库 `testing` | 单元/契约测试；集成测试按需启用 |

版本以 `go.mod` 为准；不追求文档里锁死补丁号。新增直接依赖必须说明用途与替代方案，并写入 [changelog](./changelog.md)。

## 3. 架构

```text
Caddy → chi middleware → huma → router.Register → api → service → store → PostgreSQL
                                      │
                                      ├→ auth.Principal / PermissionStore
                                      ├→ audit.Recorder（同事务）
                                      └→ metrics（独立 :9090）
```

- `api`：HTTP/DTO/错误映射，不查库。
- `service`：业务规则、事务、审计，不 import `net/http`。
- `store`：参数化 SQL、行↔model，不做业务判断。
- `router`：唯一注册入口、权限/审计元数据、启动覆盖检查。
- `middleware`：request-id/meta、body limit、日志、recover、JWT 认证。
- `platform`：db、jwt、logx、reqid、requestmeta、metrics。

依赖方向由 `internal/arch/layers_test.go` 静态检查。

## 4. 数据模型

| 表 | 作用 | 关键约束 |
| --- | --- | --- |
| `admin_users` | 管理员账号 | `lower(email)` 唯一；`status`；`token_version`；`failed_login_count`/`locked_until` |
| `roles` / `permissions` | RBAC 字典 | code 唯一 |
| `role_permissions` | 角色→权限 | 复合主键 |
| `admin_user_roles` | 用户→角色 | 复合主键 + 外键 |
| `admin_sessions` | 登录会话 | `user_id`、`expires_at`、`revoked_at`、`ip inet` |
| `admin_password_tokens` | 一次性设密 | `token_hash` 唯一；`expires_at`；`used_at` |
| `refresh_tokens` | 预留 | 当前不签发 |
| `categories` / `tags` | 文章分类/标签 | 当前只有表，CRUD 未接 |
| `articles` | 文章 | `slug` 唯一（未删）；`status`；`version` 乐观锁；软删 |
| `article_tags` / `article_revisions` | 标签关联/版本 | 版本按 `article_id, version` 唯一 |
| `audit_logs` | 业务审计 | `id bigserial` keyset；JSONB changes/metadata |

约定：DB 时间 `timestamptz`（UTC）；API RFC 3339；主键 UUIDv7；列表 `pageSize<=100`；审计用 `id` keyset。

迁移只追加，已应用文件不改。新增索引注意锁表；大表用 `CREATE INDEX CONCURRENTLY`（不能放在事务迁移里）。

## 5. 账号与鉴权

### 5.1 登录

1. 规范化 email，按 IP/email 限流。
2. 查用户；未知用户也执行一次 Argon2 校验，降低时序差异。
3. 检查 `locked_until`；密码错误累计失败，10 次锁 15 分钟。
4. 校验 `status=active`、Argon2id；必要时透明重哈希（不 bump `token_version`）。
5. 创建 `admin_sessions`，签 access JWT：`iss/aud/sub/sid/ver/iat/exp/jti`。
6. 同一事务写 `auth.login` 审计；失败审计 best-effort。
7. 返回 JSON：`accessToken`、`tokenType`、`expiresIn`、`user`、`permissions`、`roles`；**不设置 Cookie**。

### 5.2 每请求校验

`Authenticate` 对非公开路由：

1. 解析 Bearer JWT（HS256、issuer、audience、exp）。
2. 查 `admin_sessions` + `admin_users`：session 未吊销/未过期、账号 active、`claims.ver == token_version`。
3. 查 RBAC 权限，构造 `Principal{UserID, SessionID, TokenVersion, Email}`。
4. 路由包装层校验 `Permission`；无权限 403。

登出吊销当前 session；改密/禁用 `token_version++` 并吊销 session；权限每请求查库，角色变更立即生效。

### 5.3 一次性链接

- 创建账号/重置密码生成 32B 随机 token；DB 只存 SHA-256。
- 30 分钟过期、单次使用；激活时更新密码、`must_change_password=false`、吊销旧 session。
- `ADMIN_WEB_URL` + `/activate?token=...` 返回给调用方；不打印到日志。

### 5.4 前端配合

token 放 `sessionStorage`，请求带 `Authorization`；401 清 token 回登录，403 提示无权限。无 Cookie 即无 CSRF，主要风险是 XSS。

## 6. 文章

- 状态机：`draft ↔ published`；`publish` 只允许 draft，`unpublish` 只允许 published，非法转换 409。
- `version` 乐观锁：更新条件 `id + version`，冲突 409；成功 `version+1`。
- 每次创建/更新/发布/下线写 `article_revisions`。
- Markdown 用 goldmark 渲染，bluemonday UGCPolicy 清洗；HTML 存库，读时不再渲染。
- 删除是软删，保留历史。
- 列表支持 `page/pageSize/status/q`，按创建时间倒序。
- 权限：`article:read/write/publish/delete`。

分类/标签 CRUD、定时发布、封面上传不在当前范围。

## 7. 审计

| 层 | 内容 | 存储 |
| --- | --- | --- |
| 访问日志 | route/status/latency/request_id | 日志管道 |
| 业务审计 | actor/action/resource/result/changes | `audit_logs` |
| 安全事件 | 登录失败、锁定、权限/会话变更 | `audit_logs` + 告警 |

- 业务写操作在 service 事务内 `audit.Record`，失败回滚业务。
- 登录失败用 `RecordBestEffort`，失败只记日志。
- `changes` 字段白名单；文章正文只存哈希；密码/token/链接不落库。
- `actor_email` 存快照；`request_id` 关联访问日志。
- 查询 `GET /audit-logs?limit&cursor&action&actorId`，按 `id` keyset。

未做：导出、保留/归档、哈希链、审计表权限收紧（部署时配置）。

## 8. API 契约

- 成功响应裸资源；列表 `{items, page, pageSize, total}`；审计 `{items, nextCursor}`。
- 错误用 RFC 9457 `application/problem+json`：`type/title/status/detail/code/requestId/message`。
- 401 未认证/过期/吊销；403 已认证无权限；404 不存在；409 冲突/版本；422 校验；429 限流；5xx 不泄露 SQL。
- 路由只能经 `router.Register`；非 GET 必须 `AuditAction`；`Public` 不得有 `Permission`。
- OpenAPI 在 `/openapi.json`，文档 `/docs`；关闭 Huma 的 `$schema` 响应链接。
- `pageSize` 上限 100。

## 9. 配置与部署

关键环境变量：

```text
APP_ENV, HTTP_ADDR, METRICS_ADDR, ADMIN_WEB_URL
POSTGRES_HOST/PORT/DB/USER/PASSWORD/SSLMODE（DATABASE_URL 可选覆盖）
REDIS_HOST/PORT/DB/USERNAME/PASSWORD（当前只校验，不连接）
JWT_SECRET/JWT_ISSUER/JWT_AUDIENCE/ACCESS_TTL
BOOTSTRAP_ADMIN_EMAIL/PASSWORD/NAME（仅 cmd/seed）
```

- 单机 Compose：`postgres` + 一次性 `migrate` + `api` + `caddy`；Redis 可选 profile。
- API 容器内监听 `:8080`，metrics 监听 `:9090`；生产只由 Caddy 暴露 443，metrics 走内网。
- 镜像：多阶段构建 + distroless 非 root。
- 备份：每日 `pg_dump -Fc` + 异地对象存储；需要 PITR 再加 WAL 归档。

完整命令见 [engineering.md](./engineering.md)。

## 10. 测试与质量

- 单元：密码、JWT、限流、slug/diff、分页、metrics、middleware。
- 契约：路由覆盖、OpenAPI 生成、401/403、problem+json。
- 架构：`internal/arch` 检查文档链接、必需文件和 api→service→store 依赖。
- 集成：真实 PostgreSQL 验证迁移、登录、RBAC、文章、审计；需要显式开启。
- 门禁：`gofmt -l`、`go vet`、`go test`、`go build`；CI 跑 `go test -race`。
- 测试规范见 [testing.md](./testing.md)。

## 11. 已知限制与演进

当前限制：

- 无 refresh/Cookie、MFA、SSO、公开注册、邮件找回。
- Redis 仅配置；限流为进程内，多实例需 Redis。
- 分类/标签 CRUD、定时发布、封面上传未做。
- 审计无导出/保留/OTel；metrics 标签暂用原始 path。
- 自动化集成测试和 Docker/Caddy 实机验证仍可加强。

演进原则：

1. 先 spec/ADR，再引依赖或改数据模型。
2. 框架变更同一次更新文档与 changelog。
3. 迁移 expand → migrate → contract；不破坏已有数据。
4. 新增业务先补测试，再扩展示例。

## 12. 参考

- [architecture.md](./architecture.md)：模块边界、请求链、扩展位置
- [engineering.md](./engineering.md)：命令、CI、部署、编码约束
- [testing.md](./testing.md)：测试边界与用例清单
- [template-guide.md](./template-guide.md)：复制与清理模板
- [specs/](../specs/README.md)：功能规格与验收
