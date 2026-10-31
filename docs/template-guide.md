# 作为模板使用

本目录是 `homebrew` 管理后台 API 的起始模板。复制后按下面的清单替换项目身份，然后开始写业务。

## 1. 复制与初始化

```bash
cp -r homebrew-api-starter my-admin-api
cd my-admin-api
cp .env.example .env
```

不要把原仓库的 `.git` 复制到新项目；在新目录重新 `git init`。

## 2. 替换项目身份

| 位置 | 需要替换 |
| --- | --- |
| `go.mod` | `module github.com/example/homebrew-api-starter` → 团队仓库地址；全仓库 import 同步 |
| `JWT_ISSUER` / `JWT_AUDIENCE` | `homebrew-api` / `admin-web` → 新服务标识 |
| `internal/api/app.go` | OpenAPI 标题 `Homebrew Admin API` |
| `Caddyfile` / `ADMIN_DOMAIN` | 新域名 |
| `.env.example` / `compose.yaml` | 数据库名、账号、密码、端口 |
| `README.md` | 项目名、快速开始、接口说明 |
| `docs/design.md` | 保留结构，替换业务边界和扩展点 |

## 3. 首次启动

```bash
docker compose up -d postgres
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api
```

`.env.example` 的默认超管是 `admin@example.com` / `ChangeMe123!`，只用于本地。生产必须替换密码并删除 `BOOTSTRAP_ADMIN_*` 或改用安全注入。

## 4. 保留什么

- `internal/router` 的路由契约与启动覆盖检查
- `internal/middleware` 的 request-id、body limit、认证、日志、recover
- `internal/auth` / `internal/audit` 的 Principal、权限、同事务审计
- `internal/store` 的事务封装与参数化查询模式
- `migrations/` 的认证/审计/文章基础表
- `specs/` 的 Spec Kit Lite 流程与模板
- `docs/testing.md` 的测试边界与约束

## 5. 清理示例业务

如果新项目不需要文章：

1. 删除 `internal/api/article`、`internal/service/article`、`internal/store/article.go`。
2. 删除 `cmd/api/app.go` 中的 article 注册。
3. 删除 `migrations/00002_article.sql`（若已应用，不要改历史迁移，新增迁移撤销）。
4. 删除 `rest/articles.rest` 和对应测试。
5. 更新 `docs/architecture.md`、`README.md`、`docs/changelog.md`。

不要保留无路由引用的 handler/service 目录；模板应保持“删得干净”。

## 6. 必须同步的约束

- 路由只能通过 `router.Register` 注册；非 GET 必须有 `AuditAction`。
- 新增业务代码必须有单元/契约测试；鉴权改动覆盖 401/403，富文本覆盖 XSS。
- 迁移只追加；不要修改已应用文件。
- 框架变更同一次更新 `docs/changelog.md` 和受影响文档。
- 不引入 proto3、ORM、Redis、消息队列或微服务，除非先写 spec/ADR。

## 7. 交付前检查

```bash
gofmt -l .
go vet ./...
go test ./...
go build ./...
docker compose config --quiet
```

再确认：没有 `.env`、密钥、本地 DB 数据、测试 token、生成的 Linux 二进制。
