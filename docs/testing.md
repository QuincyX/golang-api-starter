# 测试规范

本文定义 `homebrew-api-starter` 的单元测试、契约测试和集成测试边界。测试是完成定义的一部分，不是“有空再补”的附加项。

## 1. 分层与边界

| 层 | 依赖 | 允许的测试 | 不允许 |
| --- | --- | --- | --- |
| 纯单元 | 标准库 | 密码、JWT、slug、diff、分页、限流、状态机 | DB、网络、真实时间 sleep |
| Handler | `httptest` + 真实 chi/huma 路由 | 状态码、problem+json、鉴权中间件、DTO 映射 | 真实 DB、外部服务 |
| Service | fake 依赖或纯函数 | 业务规则、错误映射、审计字段 | `net/http`、真实 DB |
| 集成 | testcontainers/真实 PG | SQL、事务、约束、乐观锁、审计同事务 | 用 mock 替代 SQL 行为 |
| 架构/文档 | `go/parser`、文件系统 | 分层依赖、必需文档、相对链接 | 运行时依赖 |

当前默认单元测试**不依赖** `.env`、本地 PostgreSQL、Redis、Docker 或网络。需要真实 PostgreSQL 的测试必须放到集成测试并在 CI/本地显式开启。

## 2. 命名与结构

- 测试函数：`TestSubject_Behavior`，例如 `TestHashAndVerifyPassword`、`TestAuthenticatePublicAndPrivate`。
- 子测试：`t.Run("case name", ...)`；表驱动优先。
- Helper 必须调用 `t.Helper()`，避免失败行号指向 helper。
- 断言失败用 `t.Fatalf`（当前用例无法继续）或 `t.Errorf`（继续收集）；错误信息包含实际值与期望值。
- 只比较行为，不比较完整错误字符串；领域错误用 `errors.Is`。
- 测试文件与被测包同目录，包名一致；只测导出行为时可用 `_test` 包，当前项目不强制。

## 3. 确定性

- 禁止 `time.Sleep` 等待业务状态；需要时间控制时注入 clock，或用 `testing/synctest`（Go 1.25+）。
- 随机值必须可验证：长度/唯一性/哈希，而不是期待固定字符串。
- 不依赖当前时区、系统语言、机器路径、真实 IP。
- 文件写入用 `t.TempDir()`；不污染仓库。
- 环境变量用 `t.Setenv`；测试结束自动恢复。
- 并发测试必须 `t.Parallel` 仅在无共享可写状态时使用；`t.Setenv` 测试不要并行。

## 4. 单元测试约束

- 优先标准库 `testing`；当前不引入 testify 等断言框架。
- 纯函数优先：密码 PHC、JWT claims、slug、diff、分页、限流窗口。
- 服务测试用 fake store/service 接口，不用 pgx mock 伪造 SQL 行为。
- Handler 测试使用真实 chi/huma 路由，确保 OpenAPI schema 与中间件顺序也被覆盖；至少有一个 `/openapi.json` 生成测试。
- 每个鉴权相关改动至少覆盖：无 token 401、坏 token 401、无权限 403、正常 200。
- 每个 Markdown/富文本入口至少覆盖一个 XSS 用例（`<script>` 必须被清洗）。
- 路由新增/修改必须让 `verifyRouteCoverage` 测试通过；不能只测 handler。

## 5. 集成测试约束

- 使用 `//go:build integration` 或 `TESTCONTAINERS=1` 显式启用，默认 `go test ./...` 不启动容器。
- 容器内 PostgreSQL 版本与生产一致（18.x）；迁移使用 `cmd/migrate` 或 goose embed。
- 每个集成测试自己创建/清理数据；不依赖执行顺序。
- 断言真实约束：唯一索引、外键、乐观锁、`token_version`、审计同事务。
- 集成测试失败必须保留足够日志，但不得打印密码/token。

## 6. 当前测试清单

| 文件 | 覆盖 |
| --- | --- |
| `internal/config/config_test.go` | DSN 拼接/转义、URL 覆盖、Redis、嵌套 env |
| `internal/platform/jwt/jwt_test.go` | 签发/解析、篡改、错误 audience |
| `internal/service/auth/password_test.go` | Argon2id、重哈希、登录限流 |
| `internal/api/auth/handler_test.go` | login 成功映射、401/429/500 |
| `internal/service/account/service_test.go` | 随机 token、hash、激活链接、分页 |
| `internal/service/article/service_test.go` | Markdown XSS、slug、diff |
| `internal/middleware/auth_test.go` | Bearer 解析、公开/受保护路由 401 |
| `internal/platform/metrics/metrics_test.go` | 请求计数/延迟、metrics handler |
| `internal/router/registry_test.go` | 路由契约拒绝规则、路径参数匹配 |
| `cmd/api/app_test.go` | 路由覆盖、OpenAPI、login 401、无 Cookie、权限 403 |
| `internal/arch/docs_test.go` | 必需文档、相对链接 |
| `internal/arch/layers_test.go` | api/service/store 依赖方向 |

## 7. 命令

```bash
gofmt -l .                  # 必须为空
go vet ./...
go test ./...               # 默认单元/契约/文档
go test -race ./...         # 需要 CGO；CI Linux 执行
go test ./internal/arch     # 只跑文档/分层检查
```

涉及数据库的测试：

```bash
TESTCONTAINERS=1 go test -tags=integration ./...
```

## 8. 覆盖率与质量门槛

- 新增业务代码目标 70%（排除生成代码、迁移、纯 DTO）；当前基座因大量 store 代码尚无自动化集成，整体覆盖率未达该目标。
- 新增业务代码至少覆盖成功路径 + 一个失败/权限路径。
- 不为了让覆盖率变高写无断言测试。
- 不删除测试、不放宽断言、不加无依据的 `t.Skip` 来通过 CI。
- 修复 bug 时先补一个能复现的测试，再修实现。

## 9. 完成定义

框架/功能变更至少满足：

- [ ] `go test ./...` 通过
- [ ] 新增/修改路由有覆盖测试
- [ ] 新增鉴权路径有 401/403 用例
- [ ] 新增富文本路径有 XSS 用例
- [ ] 新增迁移有集成验证或明确未验证说明
- [ ] `go vet`、`gofmt` 通过
- [ ] 未执行的测试在交付说明中列出原因与风险
