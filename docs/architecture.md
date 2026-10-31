# 架构说明

本文描述 `homebrew-api-starter` 的运行时结构、模块职责、请求链和扩展位置。它回答“代码为什么这样分层”和“新功能应该放在哪里”。

> 修改分层、模块边界、路由契约、鉴权或审计机制时，必须同步更新本文和 [变更记录](./changelog.md)。

## 1. 定位与边界

这是一个 **homebrew 管理后台接口服务的起始模板**。当前已实现 P1–P3：

- P1：邮箱密码登录、Argon2id、8h access JWT、session/`token_version` 校验、RBAC、账号管理、一次性设密链接
- P2：文章 CRUD、发布/下线、版本记录、Markdown 渲染与 HTML 清洗
- P3：操作审计（同事务写入 + keyset 查询）、Prometheus metrics

**不包含**：refresh/Cookie、MFA、SSO/OIDC、公开注册、邮件找回密码、多租户、proto3/gRPC、Redis 实际连接、审计导出/保留策略、OTel。需要时另立 spec/ADR。

> 完整设计见 [设计文档](./design.md)；功能规格见 [specs/](../specs/README.md)。

## 2. 运行时的分层

```text
Caddy（同机 TLS）
  │
  ▼
chi 中间件链
  RequestID → RequestMeta → BodyLimit → RequestLogger → Recover → Metrics → Authenticate
  │
  ▼
huma（OpenAPI 3.1 / 校验 / problem+json）
  │
  ▼
router.Register 包装层（permission 校验 + 路由契约）
  │
  ▼
internal/api（handler / DTO / 错误映射）
  │
  ▼
internal/service（业务规则、事务、审计）
  │
  ▼
internal/store（pgx 参数化查询）
  │
  ▼
PostgreSQL 18
```

横向支撑：

- `internal/platform/`：pgxpool、JWT、slog、request id、request meta、Prometheus
- `internal/audit/`：同事务审计写入
- `internal/auth/`：Principal、权限集合、`SessionValidator`
- `internal/model/`：领域类型
- `internal/middleware/`：request-id、body limit、访问日志、recover、JWT 认证
- `internal/router/`：路由注册契约、启动覆盖检查
- `migrations/`：认证/文章/审计/一次性链接/种子

**依赖方向**：`api` → `service` → `store`；`middleware` → `router`/`auth`/`platform`；`cmd/api` 负责组装。`api` 不得直接 import `store` 或开事务；`service` 不得 import `net/http`、`api`、`router`；`store` 不做业务判断。

## 3. 目录职责

| 路径 | 职责 | 不应包含 |
| --- | --- | --- |
| [cmd/api](../cmd/api/) | 组装、HTTP server、metrics server、优雅退出 | 业务规则、SQL |
| [cmd/migrate](../cmd/migrate/) | goose 迁移 | 业务逻辑 |
| [cmd/seed](../cmd/seed/) | 表为空时创建首个超管 | 常规业务 |
| [cmd/token](../cmd/token/) | 创建真实 session 并签发开发 token | 生产登录接口 |
| [internal/router](../internal/router/) | `Route` 契约、`Register` 包装、启动校验 | handler、SQL、业务判断 |
| [internal/api](../internal/api/) | handler、请求/响应 DTO、参数校验、错误映射 | SQL、事务、store 类型 |
| [internal/api/shared](../internal/api/shared/) | problem+json 写入 | 业务错误定义 |
| [internal/service](../internal/service/) | 业务规则、事务边界、审计写入 | HTTP、SQL 细节 |
| [internal/store](../internal/store/) | pgx 参数化查询、事务、行↔model 映射 | 业务规则 |
| [internal/audit](../internal/audit/) | 审计事件写入（事务内/失败兜底） | 业务判断 |
| [internal/model](../internal/model/) | User、Session、Article、AuditLog 等领域类型 | 依赖上层 |
| [internal/middleware](../internal/middleware/) | request-id/meta、body limit、日志、recover、认证 | 业务权限判断 |
| [internal/auth](../internal/auth/) | Principal、权限集合、session 校验接口 | HTTP、数据库实现 |
| [internal/platform](../internal/platform/) | db、jwt、logx、reqid、requestmeta、metrics | 业务规则 |
| [migrations](../migrations/) | 版本化 SQL；已应用文件不可改 | 业务代码 |
| [rest](../rest/) | `.rest` 手工冒烟脚本 | 正式契约（以 OpenAPI 为准） |
| [specs](../specs/README.md) | 功能规格、计划、任务、验收 | 运行时代码 |

## 4. 请求生命周期

1. Caddy 终止 TLS，转发到 `api:8080`。
2. `RequestID` 接受合法 `X-Request-Id`，否则生成 UUIDv7。
3. `RequestMeta` 写入 IP/UA/request id，供审计使用。
4. `BodyLimit` 将请求体限制为 2 MiB。
5. `RequestLogger` 记录方法、路径、状态码、耗时、request id；不记录 query/body/token。
6. `Recover` 捕获 panic，记录堆栈，返回 500 problem+json。
7. `Metrics` 记录 `http_requests_total` 与耗时直方图。
8. `Authenticate` 对非公开路由校验 Bearer JWT，并查 session/user/`token_version` + 权限，写入 `Principal`。
9. Huma 完成 JSON 解码、字段校验、OpenAPI 路由分发。
10. `router.Register` 的包装层做权限判断；无权限返回 403。
11. handler 调 service；service 在事务内写业务数据和审计。
12. 错误统一映射为 problem+json；5xx 不泄露数据库细节。

公开路由由注册表决定，而不是在中间件里维护路径白名单。新增公开路由必须在 `Route{Public: true}` 中声明。

## 5. 路由契约与覆盖检查

唯一注册入口是 [router.Register](../internal/router/register.go)：

```go
router.Register(reg, api, router.Route{
    Method:      http.MethodPost,
    Path:        "/api/admin/v1/articles",
    OperationID: "article.create",
    Permission:  "article:write",
    AuditAction: "article.create",
}, h.create)
```

[registry.go](../internal/router/registry.go) 在 `Add` 时拒绝：非 GET 路由缺 `AuditAction`、`Public` 带 `Permission`、重复 `Method + Path`/`OperationID`、非法权限码/审计动作、空路径。

[app.go](../cmd/api/app.go) 的 `verifyRouteCoverage` 用 `chi.Walk` 做双向比对：chi 有但 registry 没有 → 绕过 `Register`；registry 有但 chi 未挂载 → 声明漂移。两类问题都会让进程在监听前失败。

## 6. 鉴权与会话

- access JWT：HS256，默认 8h（`ACCESS_TTL` 可配置 30m–12h）；claims 只含 `iss/aud/sub/sid/ver/iat/exp/jti`。
- 登录：Argon2id 校验 → 建 `admin_sessions` → 签 access；响应只含 JSON，不设置 Cookie。
- 每请求：验签 → 查 `admin_sessions` + `admin_users`，校验 session 未吊销/未过期、账号 active、`claims.ver == token_version` → 查 RBAC 权限。
- 登出：吊销当前 session；禁用/改密：`token_version++` + 吊销 session；角色变更因权限每请求查库而立即生效。
- 前端 token 存 `sessionStorage`；无 Cookie 即无 CSRF，主要风险是 XSS，需要 CSP、输入清洗、禁用 `v-html`。
- 一次性设密链接：32B 随机 token，DB 存 SHA-256，30 分钟过期、单次使用；`cmd/seed` 只负责首个超管。
- 权限来源：`admin_user_roles → role_permissions → permissions`；`super_admin` 在 seed 中拥有全部权限；默认拒绝。

## 7. 审计

审计分三层：

| 层 | 内容 | 存储 | 保留 |
| --- | --- | --- | --- |
| 访问日志 | 所有请求的 route/status/latency/request_id | 日志管道 | 30 天 |
| 业务审计 | 关键写操作的 actor/before/after/result | `audit_logs` | 1 年 |
| 安全事件 | 登录失败、权限变更、会话吊销、一次性链接异常 | `audit_logs` + 告警 | 2 年 |

当前实现：

- 登录成功、登出、改密、账号创建/禁用/角色变更、文章创建/更新/发布/下线/删除、session 吊销写入 `audit_logs`。
- 业务写操作在 service 事务内调用 `audit.Record`，与业务同 commit；登录失败用 `RecordBestEffort`。
- `changes` 只存变化字段；文章正文存 sha256 前缀。
- `GET /api/admin/v1/audit-logs` 用 `id` keyset 分页，支持 action/actorId 过滤。
- 尚未实现：审计导出、保留/归档任务、哈希链、应用 DB 角色权限收紧。

## 8. 数据与迁移

- PostgreSQL 18；主键 UUIDv7；时间统一 `timestamptz`（UTC）。
- 连接配置：`POSTGRES_*` 分项拼 DSN，`DATABASE_URL` 可选覆盖；`REDIS_*` 只校验、不连接。
- 认证表：`admin_users`、`roles`、`permissions`、`role_permissions`、`admin_user_roles`、`admin_sessions`、`refresh_tokens`（保留未用）、`admin_password_tokens`。
- 文章表：`categories`、`tags`、`articles`、`article_tags`、`article_revisions`。
- 审计表：`audit_logs`。
- 种子：`permissions`、`roles`、`role_permissions`。

迁移只追加，不改已应用文件；大表索引使用 `CREATE INDEX CONCURRENTLY`；破坏性变更走 expand → migrate → contract。SQL 目前是 pgx 参数化查询；引入 sqlc 后生成目录禁止手改。

## 9. 错误与响应

- 成功响应裸资源；列表 `{items, page, pageSize, total}`。
- 错误使用 RFC 9457 `application/problem+json`，包含 `type/title/status/detail/code/requestId`，并保留 `message` 兼容 Vue 客户端。
- 401（未认证/过期/已吊销）与 403（已认证无权限）严格区分；403 不触发登出。
- 5xx 不回显数据库错误；日志记录 request id 与内部错误。
- Huma 默认的 `$schema` 响应链接已关闭。
- 时间输出 RFC 3339；JSON 字段 camelCase；`pageSize` 上限 100。

## 10. 扩展位置

### 新增接口

1. 在 `internal/api/<module>/` 定义 DTO 与 handler。
2. 在 `internal/service/<module>/` 写业务规则和事务。
3. 在 `internal/store/` 加参数化查询。
4. 通过 `router.Register` 声明 `Permission`、`AuditAction`、`Public`。
5. 新增权限码时同步迁移/种子（只追加）。
6. 补 handler/service 测试与路由覆盖测试。

### 新增迁移

- 新建递增序号文件，写 `-- +goose Up/Down`。
- 不改已应用文件；不在事务里执行 `CREATE INDEX CONCURRENTLY`。
- 部署时先跑 `cmd/migrate`，再滚动 API。

### 新增审计动作

- `Route.AuditAction` 使用 `domain.verb`（如 `article.publish`）。
- 在 service 内与业务同事务调用 `audit.Record`。
- 更新审计动作清单和测试。

## 11. 已知约束与技术债

- 没有 refresh/Cookie；access 过期后重新登录。没有 MFA、SSO、公开注册、邮件找回。
- Redis 仅有配置，代码未连接；限流是进程内，单机有效，多实例需 Redis。
- 文章分类/标签表已建，但分类/标签 CRUD 未做；文章只接受 `categoryId`，标签未接。
- 审计只做查询，没有导出、保留/归档任务、哈希链。
- 没有 OTel trace；只有 Prometheus metrics。
- 没有 `golangci-lint` 配置；当前门禁是 gofmt/vet/test/build。
- 迁移、seed、登录、RBAC、文章发布、审计与 metrics 已在 WSL 的 PostgreSQL 18.6 手工验证；仍缺自动化集成测试（testcontainers）。
- Docker 镜像构建与 Caddy 运行未在真实环境验证。

## 12. 相关文档

- [工程说明](./engineering.md)：命令、质量门禁、编码约束、CI/部署
- [测试规范](./testing.md)：单元/集成边界、命名、确定性、覆盖率
- [模板使用指南](./template-guide.md)：复制、重命名、清理示例业务
- [变更记录](./changelog.md)：框架变更历史
- [功能规格](../specs/README.md)：spec/plan/tasks/verification
- [设计文档](./design.md)：完整技术方案
- [README](../README.md)：快速开始与接口清单
