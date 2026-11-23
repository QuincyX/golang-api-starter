# 工程说明与约束

本文是开发、评审和自动化执行的操作手册。Agent 的协作规则见 [AGENTS.md](../AGENTS.md)，系统结构见 [架构说明](./architecture.md)。

## 1. 环境要求

| 项 | 要求 |
| --- | --- |
| Go | 1.27.x；`go.mod` 固定 `toolchain go1.27.4`（含 `net/http` 安全修复） |
| PostgreSQL | 18.x（`uuidv7()` 依赖 18） |
| Redis | 可选；当前只做配置校验，未建立连接 |
| 质量工具 | golangci-lint、govulncheck、oasdiff；版本固定在 `Taskfile.yml` 的 vars，无需全局安装 |
| 系统 | Windows / Linux / macOS 均可；命令以跨平台 `go` 为准 |

本机 Go 低于 `go.mod` 要求时，`GOTOOLCHAIN=auto` 会自动下载；网络受限时先升级本地 Go。首次执行 `task lint` / `task vuln` 会构建 pinned 工具，耗时明显高于后续运行。

## 2. 安装与启动

### 2.1 本地启动

前置：本地 PostgreSQL 18 已启动，并按 `.env.example` 创建数据库（默认 `admin`）。

```bash
cp .env.example .env
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api
```

`cmd/seed` 只在 `admin_users` 为空时创建超管，账号来自 `BOOTSTRAP_ADMIN_*`。API 默认监听 `:8080`；健康检查：

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
```

### 2.2 PostgreSQL 与 Redis 配置

PostgreSQL 使用分项环境变量，由 `config.Postgres.DSN()` 拼成 DSN：

```text
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=admin
POSTGRES_USER=admin
POSTGRES_PASSWORD=admin
POSTGRES_SSLMODE=disable
```

`DATABASE_URL` 是可选覆盖；一旦设置，优先于上面的分项，适合托管数据库或已有 DSN 的环境。

Redis 配置同样使用分项：

```text
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
REDIS_USERNAME=
REDIS_PASSWORD=
```

当前只校验 Redis 配置格式，**不建立连接**；后续做缓存/分布式限流时再启用，本地需要时自行启动 Redis。

其他运维配置：

- `ADMIN_WEB_URL`：一次性设密链接的前端基地址，默认 `http://localhost:5173`
- `METRICS_ADDR`：Prometheus 监听地址，默认 `127.0.0.1:9090`
- `BOOTSTRAP_ADMIN_EMAIL/PASSWORD/NAME`：仅 `cmd/seed` 使用，表为空时创建首个超管

### 2.3 登录与开发 token

首选登录接口：

```bash
curl -s http://127.0.0.1:8080/api/admin/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"ChangeMe123!"}'
```

脚本场景可以用 CLI，它会创建真实 session：

```bash
go run ./cmd/token -email admin@example.com
```

输出填入 `rest/auth.rest` 的 `@token`。没有静态开发权限开关；权限来自 RBAC 表。

## 3. 命令

| 命令 | 用途 | 是否必须 |
| --- | --- | --- |
| `task verify` | 聚合门禁：vet、test、build、lint、vuln | 是 |
| `task lint` | `go run ...golangci-lint@<pinned> run ./...` | 是 |
| `task vuln` | `go run ...govulncheck@<pinned> ./...` | 是 |
| `task test` | 单元/契约/文档测试（含 gofmt 检查） | 是 |
| `task test:race` | 竞态检测（需 CGO/Linux） | 是（Linux） |
| `task test:integration` | 真实 PostgreSQL 集成测试（需 `TEST_DATABASE_URL`） | 涉及 SQL/事务时 |
| `task api:update` | 重新生成 OpenAPI 快照 | 契约有意变更时 |
| `task api:breaking` | 对 base 分支做 OpenAPI 破坏性变更检测 | 改 API 时 |
| `task migrations:checksum` | 刷新迁移校验和清单 | 新增迁移时 |
| `go run ./cmd/api` | 启动 API | 否 |
| `go run ./cmd/migrate` | 执行 goose 迁移 | 否（部署时先执行） |
| `go run ./cmd/seed` | 初始化首个超管（表为空时） | 否 |
| `go run ./cmd/token -email ...` | 签发带 session 的开发 access token | 否 |
| `go build ./...` | 编译全部命令 | 是 |
| `go vet ./...` | 静态检查 | 是 |
| `gofmt -l .` | 格式检查；输出必须为空（`task test` 已包含） | 是 |
| `go mod verify` | 依赖完整性 | 否 |
| `go mod tidy` | 整理依赖 | 否 |

Windows 下 `go test -race` 需要可用的 CGO/gcc；没有时在 Linux 环境执行一次，不要为绕过检查而删除 race 步骤。

## 4. 完整质量门禁

框架变更在提交前执行：

```bash
task verify   # gofmt 检查 + go vet + go test + go build + lint + vuln
```

`task test` 内的 `internal/arch` 会执行 `gofmt -l .`，格式不干净会直接失败。涉及环境变量或迁移时，再执行：

```bash
go run ./cmd/migrate   # 需要可用 PostgreSQL
```

涉及 SQL、事务或查询时，用已迁移的测试库跑一次：

```bash
TEST_DATABASE_URL=postgres://... task test:integration
```

涉及并发、认证、事务或审计时，必须在 Linux 环境跑一次：

```bash
task test:race
```

新增迁移后刷新校验和清单并 Review 差异：

```bash
task migrations:checksum
```

修改 API 契约后更新快照，并用 base 分支检查破坏性变更：

```bash
task api:update
BASE=main task api:breaking
```

没有执行的检查必须在最终说明中列出原因和风险，不能默认为通过。

## 5. 变更分类与文档义务

这是硬性要求。`AGENTS.md` 只保留入口地图，矩阵以本节为准：

| 变更内容 | 必须更新 |
| --- | --- |
| 分层、目录职责、请求链、路由契约 | [architecture](./architecture.md) + [changelog](./changelog.md) |
| 测试边界、命名、确定性、快照/集成门禁 | [testing](./testing.md) + [changelog](./changelog.md) |
| 依赖、命令、部署、环境变量、质量工具 | 本文 + [README](../README.md) + [changelog](./changelog.md) |
| Agent 规则、完成定义、文档治理 | [AGENTS](../AGENTS.md) + 本文 + [changelog](./changelog.md) |
| spec 流程、模板、状态机 | [specs/README](../specs/README.md) + [constitution](../specs/constitution.md) + [changelog](./changelog.md) |
| 业务功能 | 目标 `specs/NNN-*/` + `verification.md`；用户可见时更新 [README](../README.md) |
| 文档索引或链接调整 | [docs/README](./README.md) + [changelog](./changelog.md) |

### 业务变更

只修改占位业务（例如把文章列表接到真实查询）时：

- 至少运行 `task verify`；
- 影响用户可见接口时，检查 README 与 `rest/` 是否仍准确，并更新 OpenAPI 快照；
- 不强制写框架变更记录。

### 框架变更

涉及依赖、Go 版本、路由契约、`api`/`middleware`/`platform` 结构、配置、迁移、鉴权/审计机制、部署方式、质量门禁或 Agent 规则时：

1. 更新 [变更记录](./changelog.md)。
2. 按上面的矩阵更新受影响的文档。
3. 执行完整质量门禁（`task verify`）。
4. 在最终说明中列出更新过的文档。

**“代码先合并，文档以后补”不被接受。**

### Agent 规则生命周期与审计

`AGENTS.md` 中的全局规则会进入所有任务上下文，必须像依赖一样审计：

- Go、DSH、chi/huma/pgx/goose 等主要依赖升级后审计一次；
- 至少每季度一次；
- 出现规则冲突、重复读取、上下文异常增长或过早停止时审计。

审计方法：

1. 列出 `AGENTS.md`、`docs/` 和模板中的强制规则。
2. 标记重复、冲突、过宽、已被 DSH 覆盖、没有证据的规则。
3. 用三类真实任务对照：单文件修复、常规接口、跨层/架构改动。
4. 只保留项目事实、风险边界和有证据的失败检查。
5. 规则变更写入 [变更记录](./changelog.md)。

### 功能规格流程（Spec Kit Lite）

非平凡业务功能使用 [specs/](../specs/README.md) 的轻量 SDD 流程：

| 阶段 | 产物 | 门禁 |
| --- | --- | --- |
| 规格 | `specs/NNN-slug/spec.md` | 目标、非目标、P1 故事、验收场景经人工确认 |
| 计划 | `plan.md` | 复用现有架构，列明文件、测试和文档影响 |
| 任务 | `tasks.md` | 可追溯到 spec/plan，含精确路径 |
| 实现 | 代码 + 勾选 tasks | 按 P1 切片验收 |
| 收敛 | `verification.md` | 验收场景有命令、测试或人工证据 |

不要求 Spec Kit CLI，也不要求完整 `/speckit-*` 九步流程。`tasks.md` 是跨会话任务源；当前会话 todo 只是执行镜像。

## 6. 编码约束

### 6.1 Go

- 使用 `gofmt`；`task format` 调用配置好的 gofmt/goimports，`task test` 中的 `TestGofmtClean` 会强制 `gofmt -l .` 为空。
- 工具链以 `go.mod` 的 `go`/`toolchain` 为准（当前 1.27.4）；现代化改写用 `go fix ./...` 或 `task lint` 的 `modernize`，不要手写等价改写。
- `context.Context` 作为第一个参数贯穿 handler/service/store；禁止把 `context.Background()` 带进 DB 调用。
- 错误用 `%w` 包装；领域错误保持可 `errors.Is/As`；不要用 panic 做控制流。
- 不用全局可变状态；依赖在 `cmd/api` 组装后显式传入。禁止新增 `service`/`dao` 全局单例。
- 日志用 `log/slog`，带 `request_id`/`trace_id`/`actor_id`；不打印 token、一次性链接、密码、完整请求体。
- 时间使用 UTC；API 输出 RFC 3339；数据库用 `timestamptz`。
- ID 使用 UUIDv7；不要暴露自增主键的规模信息。

### 6.2 API 与路由

- 路由只能通过 `router.Register`；非 GET 路由必须有 `AuditAction`；`Public` 路由不得有 `Permission`。
- handler 只做 HTTP 解码、DTO 校验、调用 service、错误映射；不写 SQL、不开事务。
- 请求/响应 DTO 放在 `internal/api/<module>`；`store` 行类型和 `model` 类型不得直接作为响应 schema。
- JSON 字段 camelCase；列表返回 `{items, page, pageSize, total}`；`pageSize` 上限 100。
- 错误使用 problem+json；401 与 403 区分；5xx 不回显内部错误。
- 新增/修改路由后必须让 `verifyRouteCoverage` 和 `go test` 通过。

### 6.3 数据库与迁移

- 迁移只追加，不改已应用文件；文件名 `NNNN_description.sql`。
- 破坏性变更走 expand → migrate → contract；大表索引用 `CREATE INDEX CONCURRENTLY`，不要放在事务里。
- 查询使用 `internal/store` 的显式参数化 SQL；未来引入 sqlc 时生成目录禁止手改。
- 禁止拼接用户输入到 SQL；排序/过滤字段必须来自白名单。
- 事务边界在 service；审计与业务同事务。
- 连接池默认 `MaxConns=15`、`MinConns=2`；按单机 4 GB 内存调整。

### 6.4 鉴权与安全

- JWT claims 只放身份，不放权限、昵称、邮箱；权限每请求从 `PermissionStore` 取。
- P1 access 默认 8 小时（`ACCESS_TTL` 可配置 30m–12h）；过期后重新登录，不引入 refresh/Cookie。
- `JWT_SECRET` 至少 32 字节，只从环境/Secret 注入；禁止提交 `.env`。
- 每请求校验 `admin_sessions`、`token_version` 和账号状态；登录/改密/禁用即时生效。
- 管理后台不开放公开注册；首超管通过 seed/邀请创建，首次登录强制改密。
- 登录限流、失败锁定、一次性设密链接、access 过期重登在 P1 实现；P1 不引入 refresh token。不要在日志里打印凭据。
- **JWT Bearer，不使用 Cookie**：CORS 只允许精确 origin；前端 token 存 `sessionStorage`，必须防 XSS（CSP、输入清洗、禁用 `v-html`）。
- 请求体上限 2 MiB，由 API 中间件强制；部署环境可再加一层限制。

### 6.5 审计

- 关键写操作在 service 事务内写 `audit_logs`；禁止 goroutine/channel 异步落库。
- 记录 actor 快照、action、resource、result、request_id、IP、UA、耗时。
- `changes` 只存变化字段；正文存 sha256 + 字节数；密码/token/一次性链接白名单排除。
- 审计表应用账号只授予 `INSERT/SELECT`（部署时）；查询用 keyset 分页。

### 6.6 测试

- 测试边界、命名、确定性、覆盖率和用例清单只维护在 [测试规范](./testing.md)。
- 新增业务代码至少覆盖成功路径 + 一个失败/权限路径；鉴权改动覆盖 401/403，富文本改动覆盖 XSS。
- `go test -race` 在 Linux 环境执行；本机无 CGO 时不要伪称已跑。

## 7. 依赖策略

- 标准库优先；新增依赖必须说明用途、维护状态、替代方案和移除成本。
- 已选型：chi、huma、pgx、goose、`golang-jwt/jwt/v5`、`caarlos0/env`、`log/slog`、goldmark、bluemonday、prometheus/client_golang；ID 使用 Go 1.27 标准库 `uuid`（UUIDv7）。SQL 当前是显式参数化查询。
- `encoding/json` 继续使用 v1 API；Go 1.27 的 v1 已由 `encoding/json/v2` 实现，行为保持兼容。需要严格拒绝重复 key/非法 UTF-8 时，再显式引入 v2 API 并补契约测试。
- 质量工具（golangci-lint、govulncheck、oasdiff）版本固定在 `Taskfile.yml` 的 vars；升级工具版本时同时更新 [changelog](./changelog.md)。
- 不引入 ORM、proto3/gRPC、Redis、消息队列或微服务，除非有 spec/ADR 和明确需求。
- 大版本升级必须更新 [变更记录](./changelog.md)、[工程说明](./engineering.md) 和 [README](../README.md)（涉及使用方式时）。
- 最新可用版本用 `go list -m -u all` 检查（只读，不改 `go.mod`）；升级后同次更新 [变更记录](./changelog.md) 并跑 `task verify`。
- 依赖漏洞用 `task vuln` 扫描；不要为了通过检查删除安全步骤。

## 8. 测试约束

完整规范见 [测试规范](./testing.md)，这里只保留工程门禁：

| 层 | 当前实现 |
| --- | --- |
| 单元 | 标准库 `testing`：密码、JWT、限流（含 `synctest` 假时钟）、slug/diff、分页、metrics、middleware |
| Handler | `httptest` + 真实 chi/huma：状态码、problem+json、401/403、OpenAPI 快照 |
| 架构/文档 | `internal/arch`：相对链接、必需文档、分层依赖、`huma.Register`/`os.Getenv` 边界、gofmt、迁移校验和 |
| 集成 | `TEST_DATABASE_URL` + `-tags=integration`；当前为只读冒烟骨架，未接入自动化流水线 |

- 默认 `go test ./...` 不依赖 `.env`、数据库、Redis 或网络。
- OpenAPI 快照在 `cmd/api/testdata/openapi.json`；契约变更用 `task api:update` 更新并 Review 差异。
- 迁移校验和在 `internal/arch/testdata/migrations.sha256`；新增迁移用 `task migrations:checksum` 更新。
- 新增业务代码目标 70%（排除生成代码）；当前基座整体覆盖率受 store/集成缺口影响，未达该目标。
- 不提交 `tmp/`、`.tmp/`、日志、覆盖率产物；`.gitignore` 已覆盖。

## 9. 部署与运维

仓库不内置容器、编排或 CI 配置；部署方式由使用方按运行环境决定。

质量门禁在本地执行（见第 4 节）。使用自建流水线时复用同一组命令：

```text
go mod download → task verify → task test:race
```

`task test` 包含路由契约、文档链接、gofmt、迁移校验和与 OpenAPI 快照检查。

部署约定：

- 迁移先于 API：先执行 `go run ./cmd/migrate`（或编译后的 `migrate` 二进制），再启动 API；`go run ./cmd/seed` 创建首个超管。
- 配置全部来自环境变量；`.env` 仅供本地，生产使用 Secret 注入 `JWT_SECRET` 与数据库凭据。
- API 默认监听 `:8080`，metrics 默认 `127.0.0.1:9090`；TLS、反向代理与进程守护由部署环境提供，metrics 只绑定内网/loopback。
- Redis 当前不连接；需要时由部署环境单独提供。
- 备份：每日 `pg_dump -Fc` + 异地对象存储；需要 PITR 再加 WAL 归档。
- 单机不是高可用；明确 RTO/RPO，定期恢复演练。

## 10. Pull Request 要求

- 一个 PR 只做一个可解释的变更；不混入无关重构。
- 说明变更分类（业务/框架）、影响面、验证命令和未验证项。
- 新增/修改接口附带路由契约、权限、审计动作、测试和 OpenAPI 快照 diff。
- 新增迁移附带校验和清单 diff；SQL/事务改动附带集成测试证据。
- 框架变更附带文档更新链接。
- 不提交 `.env`、密钥、token、生产配置、生成日志。
- 不删除测试、不放宽质量门禁来换取通过。

## 11. 常见问题

### `JWT_SECRET is required` / `must be at least 32 bytes`

复制 `.env.example` 为 `.env`，或直接注入至少 32 字节的 `JWT_SECRET`。生产不要使用示例值。

### `/readyz` 返回 503

`POSTGRES_*` 配置不可达，或 `DATABASE_URL` 覆盖指向了错误地址。先确认本地 PostgreSQL 已启动，再 `go run ./cmd/migrate`。

### 启动时报 `route contract mismatch`

有路由绕过 `router.Register`，或注册后未挂载。检查 `internal/api` 是否直接调用 `huma.Register`，以及 `verifyRouteCoverage` 的输出。

### `go run ./cmd/migrate` 找不到数据库

`cmd/migrate` 会读取 `.env` 的 `POSTGRES_*`（或 `DATABASE_URL`）；确认 PostgreSQL 已启动，且 `uuidv7()` 可用（PostgreSQL 18）。

### Go 工具链版本不匹配

`go.mod` 要求 1.27.0 / toolchain 1.27.4；执行 `go env GOTOOLCHAIN` 应为 `auto`，或手动升级 Go。1.27.4 修复了 `net/http` 的安全问题，不要降回 1.27.3。

### `TestGofmtClean` 失败

有 Go 文件未格式化；执行 `task format`（或 `gofmt -w .`）后重新跑 `task test`，不要删除该测试。

### OpenAPI snapshot changed

契约有意变更时，Review `cmd/api/testdata/openapi.json` 的 diff 后执行 `task api:update`；再用 `BASE=main task api:breaking` 确认没有非预期破坏性变更。

### `task api:breaking` 报 `invalid object name` / `path ... does not exist`

base 分支还没有提交 OpenAPI 快照。先把 `cmd/api/testdata/openapi.json` 合并到基线分支，或显式传 `BASE=<commit>`。该任务比较的是已提交的快照，不是工作区文件。

### `TestMigrationChecksums` 失败

已有迁移文件被改动，或新增迁移后未刷新清单。确认不是修改已应用迁移，再执行 `task migrations:checksum` 并 Review 差异。

### `task lint` / `task vuln` 首次运行很慢

首次会按 `Taskfile.yml` 固定的版本下载并构建工具，属正常现象；后续复用构建缓存。

### 文档链接检查失败

`go test ./internal/arch` 会列出缺失的必需文档或无法解析的相对链接；修复链接，不要删除检查。

## 12. 完成定义

完整规则见 [AGENTS.md](../AGENTS.md)。最低要求：

```text
task verify     PASS
task test:race  PASS（Linux/CGO 可用时）
```

涉及迁移、鉴权、审计或 API 契约时，按 [第 4 节](#4-完整质量门禁) 追加检查；未执行项必须在交付说明中列出。
