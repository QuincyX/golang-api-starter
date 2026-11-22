# Golang Admin API Starter

Golang 管理后台接口服务的起始模板：账号/鉴权、文章、审计的可演进底座。

## 定位

- 复制本目录即可开始一个新的管理后台 API 项目。
- 保留路由契约、JWT 会话、RBAC、文章、审计、迁移与测试骨架。
- 模块路径 `github.com/example/golang-api-starter` 是占位，接入时替换成团队仓库地址。
- 复制/重命名/首次启动见 [docs/template-guide.md](docs/template-guide.md)。

当前包含：

- Go 1.27.4 + chi + huma（OpenAPI 3.1 / RFC 9457）；工具链版本固定在 `go.mod`
- PostgreSQL 18 + pgxpool + goose 迁移；`POSTGRES_*` 分项，`DATABASE_URL` 可选覆盖
- P1：Argon2id 登录、8h access JWT、session/`token_version` 校验、RBAC、账号管理、一次性设密链接
- P2：文章 CRUD、发布/下线、版本记录、Markdown 渲染与 HTML 清洗
- P3：操作审计（同事务落库 + 查询）、Prometheus metrics
- Redis 配置（`REDIS_*`，后续预留；当前只校验不连接）
- **路由契约**：每条路由声明 `Permission`，非 GET 路由声明 `AuditAction`，启动双向覆盖检查
- 配置全部来自环境变量；迁移作为独立命令执行，不依赖容器编排
- Agent 友好门禁：`task verify` 聚合格式、测试、构建、lint、漏洞扫描；OpenAPI 快照与迁移校验和防止静默漂移
- [AGENTS.md](AGENTS.md) 是 Agent 入口地图 + [docs/](docs/README.md) + [specs/](specs/README.md)
- 测试规范见 [docs/testing.md](docs/testing.md)，含单元/契约/架构/集成测试清单

## 明确不包含

- refresh token / Cookie / CSRF 流程
- TOTP/MFA、SSO/OIDC、公开注册、邮件找回密码
- 多租户、proto3/gRPC、Redis 实际连接、消息队列、微服务拆分

这些需要另立 spec 或 ADR，不在当前模板默认范围内。

## 目录

```
cmd/api        # 组装、HTTP server、优雅退出
cmd/migrate    # goose 迁移
cmd/seed       # 初始化首个超管
cmd/token      # 本地签发带 session 的 access token
internal/
  router/      # 路由注册契约 + 启动校验
  api/         # HTTP 接口层：handler / DTO
  service/     # auth / account / article / auditlog
  store/       # pgx 参数化查询、事务
  middleware/  # request-id、recover、日志、JWT 认证
  auth/        # Principal、权限集合
  audit/       # 同事务审计写入
  model/       # 领域类型
  platform/    # db、jwt、log、reqid、requestmeta、metrics
  arch/        # 文档链接/必需文件检查（测试）
migrations/    # 认证 / 文章 / 审计 / 一次性链接 / 种子
rest/          # .rest 冒烟脚本
docs/          # 架构、工程说明、变更记录
specs/         # spec/plan/tasks/verification
```

依赖方向：`api` → `service` → `store`；`middleware` 依赖 `router`/`auth`/`platform`；`cmd/api` 负责组装。`api` 不直接碰 SQL。

## 快速开始

前置：启动本地 PostgreSQL 18（`uuidv7()` 依赖 18）并创建数据库（默认 `admin`）；模板不内置容器编排。

```bash
cp .env.example .env
go run ./cmd/migrate   # 应用迁移
go run ./cmd/seed      # 创建首个超管（表为空时）
go run ./cmd/api       # 监听 :8080
curl http://127.0.0.1:8080/healthz
```

`.env.example` 默认超管是 `admin@example.com` / `ChangeMe123!`，仅用于本地。

## 登录

```bash
curl -s http://127.0.0.1:8080/api/admin/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"ChangeMe123!"}'
```

响应包含 `accessToken`、`expiresIn`、`user`、`permissions`、`roles`。后续请求带 `Authorization: Bearer <accessToken>`。

脚本场景可用：

```bash
go run ./cmd/token -email admin@example.com
```

它会在数据库里创建真实 session，再签发 8h access token。

## 接口

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/admin/v1/auth/login` | 公开 + 限流 | 邮箱密码登录，返回 access token |
| POST | `/api/admin/v1/auth/logout` | 已登录 | 吊销当前 session |
| GET | `/api/admin/v1/auth/me` | 已登录 | 当前主体与权限 |
| PUT | `/api/admin/v1/auth/password` | 已登录 | 改密，吊销全部 session |
| POST | `/api/admin/v1/auth/activate` | 公开 + 一次性 token | 首次设密/重置密码 |
| GET/POST | `/api/admin/v1/accounts` | `account:read/write` | 账号列表/创建（返回一次性链接） |
| PATCH | `/api/admin/v1/accounts/{id}` | `account:write` | 启用/禁用 |
| PUT | `/api/admin/v1/accounts/{id}/roles` | `role:assign` | 分配角色 |
| POST | `/api/admin/v1/accounts/{id}/reset-password` | `account:write` | 生成新链接 |
| GET/DELETE | `/api/admin/v1/accounts/{id}/sessions...` | `account:read/write` | 查看/吊销会话 |
| GET | `/api/admin/v1/roles`、`/permissions` | `role:read` | 角色/权限清单 |
| GET/POST | `/api/admin/v1/articles` | `article:read/write` | 列表/创建草稿 |
| GET/PUT/DELETE | `/api/admin/v1/articles/{id}` | `article:read/write/delete` | 详情/更新/软删 |
| POST | `/api/admin/v1/articles/{id}/publish`、`unpublish` | `article:publish` | 状态机 |
| GET | `/api/admin/v1/articles/{id}/revisions` | `article:read` | 版本列表 |
| GET | `/api/admin/v1/audit-logs` | `audit:read` | keyset 分页 |
| GET | `/healthz`、`/readyz` | 公开 | 存活/就绪 |
| GET | `127.0.0.1:9090/metrics` | 内网 | Prometheus |
| GET | `/docs`、`/openapi.json` | 公开 | API 文档 |

## 路由契约

```go
router.Register(reg, api, router.Route{
    Method:      http.MethodPost,
    Path:        "/api/admin/v1/articles",
    OperationID: "article.create",
    Permission:  "article:write",
    AuditAction: "article.create",
}, h.create)
```

`router.Registry.Add` 会拒绝：非 GET 路由缺 `AuditAction`、`Public` 带 `Permission`、重复路由/OperationID、非法权限码。

`newApp` 在监听前调用 `reg.Validate()` 和 `verifyRouteCoverage()`：后者用 `chi.Walk` 做双向比对，任何绕过 `router.Register` 的路由都会导致启动失败。

## 配置

完整示例见 `.env.example`。

- PostgreSQL：`POSTGRES_HOST/PORT/DB/USER/PASSWORD/SSLMODE`，`DATABASE_URL` 可选覆盖
- Redis：`REDIS_HOST/PORT/DB/USERNAME/PASSWORD`，当前只校验不连接
- 鉴权：`JWT_SECRET`（≥32B）、`JWT_ISSUER`、`JWT_AUDIENCE`、`ACCESS_TTL`
- 链接：`ADMIN_WEB_URL`，用于一次性设密链接
- 指标：`METRICS_ADDR`，默认 `127.0.0.1:9090`
- 初始化：`BOOTSTRAP_ADMIN_EMAIL/PASSWORD/NAME`，仅 `cmd/seed` 使用

生产必须设置 PostgreSQL 凭据（或 `DATABASE_URL`）、至少 32 字节的 `JWT_SECRET`。

## 质量门禁

```bash
task verify   # gofmt 检查 + go vet + go test + go build + lint + vuln
```

- lint / vuln / OpenAPI 对比工具的版本固定在 `Taskfile.yml` 的 vars，无需全局安装；应用模块的 `go.mod` 保持干净。
- API 契约快照在 `cmd/api/testdata/openapi.json`：有意变更后 `task api:update`；跨分支检查 `BASE=main task api:breaking`。
- 已应用迁移有校验和清单：新增迁移后 `task migrations:checksum`。
- 集成测试：先准备已迁移的测试库，再 `TEST_DATABASE_URL=... task test:integration`。

`task test` 包含路由契约、OpenAPI 快照、401/403、middleware、metrics、文档链接、分层依赖、gofmt、迁移校验和、密码与文章渲染测试。测试边界与用例清单见 [docs/testing.md](docs/testing.md)。迁移、seed、登录、RBAC、文章发布、审计与 metrics 已在 WSL 的 PostgreSQL 18.6 上做过手工端到端验证。

## 下一步

- P4：封面直传、定时发布、限流加固、压测、恢复演练
- 另立 spec：MFA、SSO、refresh/Cookie、审计导出/保留策略

完整设计见 [docs/design.md](docs/design.md)；工程约束见 [AGENTS.md](AGENTS.md) 与 [docs/](docs/README.md)。
