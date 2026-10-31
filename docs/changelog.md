# 变更记录

本文件记录**非业务层框架变更**，采用追加方式，不重写历史。业务功能的验收证据在 `specs/NNN-*/verification.md`。

记录格式：

```text
## YYYY-MM-DD - 标题
- 类型：新增 / 变更 / 修复 / 移除
- 影响：目录、依赖、命令、CI、部署、安全边界
- 验证：执行的命令与结论
- 文档：同次变更更新的文档
```

---

## 2026-10-31 - P0 骨架落地

- 类型：新增
- 影响：
  - `cmd/api`、`cmd/migrate`、`cmd/token`
  - `internal/router` 路由契约与启动双向覆盖检查
  - `internal/api` health/auth/article 占位接口
  - `internal/middleware` request-id、body limit、日志、recover、JWT 认证
  - `internal/platform` pgxpool、JWT、slog、reqid
  - `migrations/00001-00004` 认证/文章/审计/种子
  - `compose.yaml`、`Caddyfile`、`Dockerfile`、GitHub Actions
  - `rest/` 冒烟脚本
- 验证：
  - `go build ./...`
  - `go test -count=1 ./...`
  - `go vet ./...`
  - `gofmt -l .` 为空
  - `docker compose config --quiet`
  - 运行时冒烟：`/healthz` 200、`/auth/me` 200、`/openapi.json` 200、无数据库时 `/readyz` 503
- 文档：[README](../README.md)、[设计文档](./design.md)

## 2026-10-31 - 项目更名为 homebrew-api-starter

- 类型：变更
- 影响：
  - 目录 `admin-api/` → `homebrew-api-starter/`
  - Go module `github.com/example/admin-api` → `github.com/example/homebrew-api-starter`
  - JWT issuer `admin-api` → `homebrew-api`
  - OpenAPI 标题 `Admin API` → `Homebrew Admin API`
  - 设计文档 `docs/admin-api-design.md` → `docs/homebrew-api-design.md`
- 验证：`go build ./...`、`go test ./...`、`go vet ./...`、`docker compose config --quiet`
- 文档：[README](../README.md)、[设计文档](./design.md)

## 2026-10-31 - 增加 Agent 规范与 Spec Kit Lite

- 类型：新增
- 影响：
  - 新增 `AGENTS.md`：变更分类、架构红线、安全边界、完成定义
  - 新增 `docs/README.md`、`architecture.md`、`engineering.md`、本文件
  - 新增 `specs/README.md`、`constitution.md`、`_templates/`、`001-account-auth/`
  - 新增 `internal/arch` 文档链接与必需文件检查
- 验证：`go test ./...`（含 `internal/arch`）、`gofmt -l .`、`go vet ./...`
- 文档：[docs/README.md](./README.md)、[specs/README.md](../specs/README.md)

## 2026-10-31 - Caddy 域名改为环境变量

- 类型：变更
- 影响：
  - `.env.example` 新增 `ADMIN_DOMAIN=localhost`
  - `compose.yaml` 将 `ADMIN_DOMAIN` 传给 Caddy
  - `Caddyfile` 继续使用 `{$ADMIN_DOMAIN:localhost}`，本地默认不变
- 验证：`docker compose config --quiet`
- 文档：[engineering.md](./engineering.md)（部署章节）

## 2026-10-31 - 设计文档迁入项目

- 类型：变更
- 影响：
  - 设计文档从工作区根目录迁入 `homebrew-api-starter/docs/design.md`
  - 更新 AGENTS、README、architecture、changelog、specs 模板与 001 spec 的链接
  - 删除空的根 `docs/` 目录
- 验证：`go test ./internal/arch`、`go test ./...`
- 文档：[docs/README.md](./README.md)、[architecture.md](./architecture.md)、[specs/README.md](../specs/README.md)

## 2026-10-31 - 001 鉴权决策澄清

- 类型：变更
- 影响：
  - 001 状态 `Draft` → `Clarified`
  - 会话：JWT Bearer，不使用 Cookie；P1 不引入 refresh；access 默认 8h + 每请求 session/`ver` 校验
  - MFA/TOTP 暂不做；邀请/重置用一次性链接；SSO 以后另立 spec
  - 同步 spec/plan/tasks/verification、architecture、engineering、README、design
- 验证：`go test ./...`（含 `internal/arch`）、`go vet ./...`、`gofmt -l .`
- 文档：[001 spec](../specs/001-account-auth/spec.md)、[architecture.md](./architecture.md)、[engineering.md](./engineering.md)、[README](../README.md)

## 2026-10-31 - PostgreSQL/Redis 改为分项环境变量

- 类型：变更
- 影响：
  - 新增 `POSTGRES_HOST/PORT/DB/USER/PASSWORD/SSLMODE` 与 `REDIS_HOST/PORT/DB/USERNAME/PASSWORD`
  - `DATABASE_URL` 保留为可选覆盖；`config.Postgres.DSN()` 负责拼接并转义凭据
  - `cmd/migrate` 改用 `config.LoadPostgres()`，不再依赖 `JWT_SECRET`
  - `compose.yaml` 使用分项变量；新增可选 `redis` 服务（`--profile cache`）
  - 新增 `internal/config` DSN/嵌套 env 测试
- 验证：`go test ./internal/config`、`go test ./...`、`docker compose config --quiet`
- 文档：[engineering.md](./engineering.md)、[architecture.md](./architecture.md)、[README](../README.md)、[.env.example](../.env.example)

## 2026-10-31 - P1/P2/P3 代码落地

- 类型：新增
- 影响：
  - 新增 `internal/service/{auth,account,article,auditlog}`、`internal/store`、`internal/audit`、`internal/model`
  - 新增登录/登出/改密/一次性激活、账号/RBAC、文章 CRUD/发布/版本、审计查询路由
  - JWT 中间件改为每请求校验 `admin_sessions`、账号状态与 `token_version`
  - 新增 `cmd/seed`；`cmd/token` 改为创建真实 session
  - 新增 `00005_password_tokens.sql`、Prometheus `/metrics`
- 验证：`go build ./...`、`go test ./...`、`go vet ./...`；WSL PostgreSQL 18.6 上验证迁移/seed/登录/RBAC/文章发布/审计/metrics
- 文档：[architecture.md](./architecture.md)、[engineering.md](./engineering.md)、[README](../README.md)、[001 verification](../specs/001-account-auth/verification.md)

## 2026-10-31 - 测试规范与模板整理

- 类型：框架/文档
- 影响：
  - 新增 [测试规范](./testing.md) 与 [模板使用指南](./template-guide.md)
  - 新增 middleware、metrics、account、layering 测试；app 测试补 login 401/无 Cookie
  - `internal/model` 提供领域错误，`api` 不再为错误映射 import `store`
  - 重写 [设计文档](./design.md)，移除历史阶段与外部项目对比，改为当前可复用基线
  - 文档索引、AGENTS 完成定义、engineering 测试章节同步
- 验证：`go test ./...`、`go vet ./...`、`go build ./...`、`gofmt`
