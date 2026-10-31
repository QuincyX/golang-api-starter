# Homebrew Admin API Starter

`homebrew` 管理后台接口服务的起始模板（P0 骨架）：账号/鉴权、文章、审计的可演进底座。

## 定位

- 复制本目录即可开始一个新的 homebrew 管理后台 API 项目。
- 保留路由契约、JWT 中间件、迁移、Compose 与测试骨架，按 P1/P2 填充业务。
- 模块路径 `github.com/example/homebrew-api-starter` 是占位，接入时替换成团队仓库地址。

当前包含：

- Go 1.27 + chi + huma（OpenAPI 3.1 / RFC 9457）
- PostgreSQL 18 + pgxpool + goose 迁移
- JWT access token 中间件（P0 只验签；P1 接 session / `token_version`）
- **路由契约**：每条路由必须声明 `Permission`，所有非 GET 路由必须声明 `AuditAction`
- 启动自检 + 测试：漏声明会直接失败
- 单机 Compose：Caddy + API + PostgreSQL + 一次性迁移
- `rest/*.rest` 手工冒烟脚本

## P0 边界

这是可运行的骨架，不是可直接上线的鉴权系统：

- 没有登录/刷新接口。本地用 `cmd/token` 签发 token；P1 接 Argon2id + refresh 轮换。
- JWT 中间件只验签，不查 session、`token_version`、账号状态；P1 补每请求校验。
- 审计只做路由契约声明，还不写 `audit_logs`；P1 在业务事务内落库。
- 文章列表是占位实现，P2 接 sqlc + 数据库。

## 目录

```
cmd/api        # 组装、HTTP server、优雅退出
cmd/migrate    # goose 迁移（部署时一次性执行）
cmd/token      # 本地签发 access token
internal/
  router/      # 路由注册契约 + 启动校验
  api/         # HTTP 接口层：handler / DTO
  middleware/  # request-id、recover、日志、JWT 认证
  auth/        # Principal、权限集合
  platform/    # db、jwt、log、reqid
migrations/    # 认证 / 文章 / 审计 / 种子数据
rest/          # .rest 冒烟脚本
```

依赖方向：`api` 依赖 `router` 的注册契约；`middleware` 依赖 `router` / `auth` / `platform`；`cmd/api` 负责组装。`api` 不直接碰 SQL；P1/P2 再引入 `service` 与 `store`。

## 快速开始

### 方式一：Compose

```bash
cp .env.example .env
docker compose up -d --build
curl http://127.0.0.1:8080/healthz
```

Compose 会先跑 `migrate`，再启动 `api`，最后起 Caddy。

### 方式二：本地进程

```bash
cp .env.example .env
docker compose up -d postgres
go run ./cmd/migrate
go run ./cmd/api
```

> 目标工具链 Go 1.27.3（`go.mod` 已固定）；本机若只有旧版 Go，`GOTOOLCHAIN=auto` 会自动下载。

## 生成开发 token

```bash
go run ./cmd/token
# 或指定主体
go run ./cmd/token -sub 11111111-1111-1111-1111-111111111111
```

把输出填进 `rest/auth.rest` 的 `@token`。开发权限来自 `.env` 的 `DEV_PERMISSIONS`；P1 接 RBAC 后会从数据库读取。

## 接口

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/healthz` | 公开 | 存活 |
| GET | `/readyz` | 公开 | 数据库就绪 |
| GET | `/api/admin/v1/auth/me` | 已登录 | 当前主体与权限 |
| POST | `/api/admin/v1/auth/logout` | 已登录 | 审计动作声明 `auth.logout`（P1 落库） |
| GET | `/api/admin/v1/articles` | `article:read` | P0 返回空列表，P2 接数据库 |
| GET | `/docs` | 公开 | Huma 默认 Stoplight Elements 文档页 |
| GET | `/openapi.json` | 公开 | OpenAPI 3.1 |

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

`router.Registry.Add` 会拒绝：

- 非 GET/HEAD/OPTIONS 但没有 `AuditAction`
- `Public` 却声明了 `Permission`
- 重复的 `Method + Path` 或 `OperationID`
- 非法的权限码/审计动作格式
- 空路径、非法方法

`newApp` 在监听前调用 `reg.Validate()` 和 `verifyRouteCoverage()`：后者用 `chi.Walk` 做双向比对，任何绕过 `router.Register` 的路由都会导致启动失败。`TestRouteContractAndAuth` 会遍历所有路由再检查一遍。

## 配置

见 `.env.example`。生产必须设置 `DATABASE_URL`、至少 32 字节的 `JWT_SECRET`，且不能设置 `DEV_PERMISSIONS`。

## 迁移

```bash
go run ./cmd/migrate
```

迁移文件内嵌在 `cmd/migrate` 二进制里，部署时先跑迁移再滚动 API。

## 下一步

- P1：登录/刷新/登出、Argon2id、session + `token_version`、RBAC 查询、同事务审计
- P2：文章 CRUD、状态机、版本、分类标签、Markdown 清洗
- P3：审计查询/导出、OTel/Prometheus、前端联调

完整设计见 [../docs/homebrew-api-design.md](../docs/homebrew-api-design.md)。
