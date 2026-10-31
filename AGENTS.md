# Agent 工程指南（homebrew-api-starter）

本文件适用于所有在 `homebrew-api-starter` 目录内工作的 AI Agent、自动化脚本维护者和代码审查者。它是协作契约，不是建议清单。

## 1. 指令优先级与上下文路由

1. 用户当前的明确指令。
2. 本文件。
3. [架构说明](./docs/architecture.md) 与 [工程说明](./docs/engineering.md) 中与当前任务相关的部分。
4. [变更记录](./docs/changelog.md)、[功能规格](./specs/README.md) 与历史实现。

### 1.1 始终遵守

- 本文件的规则与边界。
- 当前工作区未被覆盖的现状；不覆盖用户未提交的改动。
- 用户明确限制的范围和交付方式。
- 不提交凭据；不执行破坏性数据库操作；不绕过路由契约。

### 1.2 按任务读取

| 任务类型 | 读取范围 |
| --- | --- |
| Agent 规则 / 文档治理 | 本文件 + [docs/README](./docs/README.md) + [changelog](./docs/changelog.md) |
| 新增 / 修改接口 | [architecture](./docs/architecture.md) + [engineering](./docs/engineering.md) + 目标 spec + `internal/router`、`internal/api` |
| 鉴权 / 权限 / 审计 | 同上 + [设计文档](docs/design.md) + 相关 spec |
| 数据库 / 迁移 / 查询 | [engineering](./docs/engineering.md) + `migrations/` + `internal/store/` |
| 依赖 / CI / 部署 | [engineering](./docs/engineering.md) + `compose.yaml`、`Dockerfile`、`.github/` |
| 不确定需要哪些上下文 | 先看 [docs/README](./docs/README.md) 的索引，再读目标章节 |

多领域任务取并集，不要退化为“全部读一遍”。文档与代码不一致时，以当前代码为事实，并在同一次变更中修正文档或记录为已知差异。

## 2. 项目定位

这是一个 **homebrew 管理后台 API 的起始模板**，不是完整产品。

- 当前已实现 P1–P3：登录/session/RBAC/账号、文章 CRUD/发布/版本、审计落库/查询、Prometheus metrics；MFA、SSO、refresh/Cookie、OTel 尚未实现。
- 不要擅自引入 proto3/gRPC、ORM、Redis、微服务或前端代码。确有需要时，先写 spec 或 ADR，并得到用户明确要求。
- 模板优先沉淀可复用的边界与约束，而不是针对某个页面或某张表写一次性补丁。
- 非平凡业务功能按 [Spec Kit Lite](./specs/README.md) 执行；纯修复、格式化、依赖补丁不强制建 spec。

## 3. 最高优先级：框架变更必须更新文档

**任何非业务层的框架功能变更，都必须在同一次变更中更新文档。没有更新文档的框架变更是未完成变更，不得声称已完成。**

框架变更包括：依赖/工具链、`router`/`api`/`middleware`/`platform` 结构、配置、迁移、路由契约、鉴权/审计机制、CI、Docker、Compose、发布流程、Agent 规则。

### 3.1 强制文档更新矩阵

| 变更内容 | 必须更新 |
| --- | --- |
| 分层、目录职责、请求链、路由契约 | [architecture](./docs/architecture.md) + [changelog](./docs/changelog.md) |
| 测试边界、命名、确定性、覆盖率 | [testing](./docs/testing.md) + [changelog](./docs/changelog.md) |
| 依赖、命令、CI、部署、环境变量 | [engineering](./docs/engineering.md) + [README](./README.md) + [changelog](./docs/changelog.md) |
| Agent 规则、完成定义、文档治理 | 本文件 + [engineering](./docs/engineering.md) + [changelog](./docs/changelog.md) |
| spec 流程、模板、状态机 | [specs/README](./specs/README.md) + [constitution](./specs/constitution.md) + [changelog](./docs/changelog.md) |
| 业务功能 | 目标 `specs/NNN-*/` + `verification.md`；用户可见时更新 [README](./README.md) |
| 文档索引或链接调整 | [docs/README](./docs/README.md) + [changelog](./docs/changelog.md) |

### 3.2 变更分类流程

编辑前先写出判断：

- 这是业务变更、框架变更，还是混合变更？混合按框架变更处理。
- 哪些文档会因为行为、命令、目录或约束变化而失效？
- 验证命令是否变化？如果变化，必须同步 [engineering](./docs/engineering.md) 和 [README](./README.md)。

### 3.3 全局规则准入

新增全局规则前，至少满足一项：

1. 模型无法凭空推断的**项目事实**（命令、目录、技术栈约束）。
2. 必须确定执行的**风险边界**（安全、数据、权限、生产操作）。
3. 有真实失败样本、回归记录或明确审计结论作为证据。

同时必须：

- 写成“触发条件 → 动作/边界 → 验证方式”，不使用无条件的 `ALWAYS` / `NEVER`；确定性的安全/数据边界除外。
- 范围具体，不使用“处理数据库时”这类过宽领域词。
- 不与 DSH 已覆盖的能力重复。
- 能通过命令、测试或 Review 验证。

**禁止写入**：“认真思考”“不要犯错”“始终自检”等泛化要求；仪式化流程；仅弥补旧模型缺陷但没有当前失败证据的防御性规则。

### 3.4 规则生命周期

全局规则会持续占用上下文，必须像依赖一样维护：

- Go、DSH、主要依赖升级后审计一次。
- 至少每季度审计一次。
- 删除无证据、重复、冲突、过宽或已被覆盖的规则。
- 规则变更写入 [changelog](./docs/changelog.md)。

## 4. 架构红线

1. **路由只能通过 [`router.Register`](./internal/router/register.go) 注册。** 不允许直接调用 `huma.Register`；非 GET/HEAD/OPTIONS 路由必须声明 `AuditAction`；`Public` 路由不得声明 `Permission`。
2. **分层单向依赖。** `router` 不写业务；`api` 不直接查库、不 import `store`、不开事务；`service` 不 import `net/http`、`api`、`router`；`store` 不做业务判断、不反向依赖上层。
3. **DTO 不泄漏。** `store` 返回 `model` 类型；`api` 使用请求/响应 DTO，service 使用领域类型，必要时显式映射。
4. **SQL 只走显式参数化查询（或未来引入的 sqlc）。** 禁止拼接用户输入；禁止在 `api` 里写 SQL；迁移只追加，不改已应用文件。
5. **JWT 只放身份。** claims 只放 `sub/sid/ver/iat/exp`，权限每次从 `PermissionStore` 取；禁止在日志打印 Authorization/token。
6. **审计与业务同事务。** 关键写操作不得用 goroutine/channel 异步落库；字段白名单，禁止记录密码、token、一次性链接、完整正文。
7. **错误语义化。** HTTP 状态码必须正确；错误体用 RFC 9457 `application/problem+json`；401 与 403 严格区分；5xx 不回显数据库细节。
8. **时间/ID/分页统一。** DB 存 UTC `timestamptz`；API 输出 RFC 3339；主键 UUIDv7；`pageSize` 上限 100，审计用 keyset。
9. **路由契约有测试。** 新增路由必须让 `go test ./...` 通过；`verifyRouteCoverage` 不允许被绕过。
10. **测试先于“完成”。** 新增/修改业务代码必须补对应单元或契约测试；鉴权改动覆盖 401/403，富文本改动覆盖 XSS。测试规范见 [docs/testing.md](./docs/testing.md)。
11. **不引入未决策的重型栈。** proto3/gRPC、GORM/Ent、Redis、消息队列、微服务拆分都必须先有明确需求和 spec/ADR；当前模板默认不包含。

## 5. 工具链约束

- Go 版本以 `go.mod` 的 `go`/`toolchain` 为准（当前 1.27.3）；提交前 `gofmt`。
- 优先标准库；已选型：chi、huma、pgx、goose、`golang-jwt/jwt/v5`、`caarlos0/env`、`google/uuid`、`log/slog`、goldmark、bluemonday、prometheus/client_golang。SQL 目前使用 pgx 参数化查询；引入 sqlc 时生成目录禁止手改。
- 新增直接依赖必须有明确用途、维护状态和替代方案说明，并写入 [changelog](./docs/changelog.md)。
- `internal/store` 的 SQL 必须参数化；`migrations/` 已应用文件禁止修改，只能新增序号。
- Docker/Compose 只通过环境变量注入配置；不把 `.env`、密钥、证书提交仓库。
- 命令、参数、CI 和部署细节只维护在 [engineering](./docs/engineering.md)，本文件不复制完整命令表。

## 6. 安全与合规

- 密钥只从环境/Secret 注入；`JWT_SECRET` 至少 32 字节；权限只从 RBAC 表读取，不使用静态开发权限。
- 管理后台**不开放公开注册**；账号通过 seed/邀请创建，首次登录强制改密。
- 日志禁止输出 Authorization、一次性链接、密码、完整请求体、文章正文。
- 审计使用字段白名单；`actor_email` 存快照；审计表应用账号只授予 `INSERT/SELECT`（部署时）。
- 请求体有上限；CORS 只允许精确 origin；当前使用 JWT Bearer（无 Cookie/refresh），必须防 XSS（CSP、输入清洗、禁用 `v-html`）。
- 依赖漏洞用 `govulncheck` 或 CI 安全扫描；不为了通过检查删除测试或放宽规则。

## 7. 工作流程

### 7.1 非平凡任务契约

以下任务开始前，先在任务描述、计划或 PR 中写清契约：

- 涉及多个文件或模块
- 框架/基础设施变更
- API、数据模型、权限、安全或性能变更
- 预计需要超过一次验证循环

```text
目标：要解决的具体问题，不写操作步骤。
不可变约束：现有接口、数据、交互、兼容性和禁止改动的范围。
可自主执行：允许读取、编辑、运行本地检查、修复本次改动引入的失败的范围。
验收证据：看到什么命令输出、测试结果或行为，才算完成。
停止条件：哪些情况必须停下来询问，例如生产变更、破坏性数据操作、凭据、跨项目改动。
```

小改动（拼写、单行修复、明确文案）不需要完整契约。

### 7.2 执行流程

1. **检查现状**：读取相关源码、测试、文档；查看工作区是否有用户未提交的改动。
2. **分类变更**：按第 3 节判断业务/框架/混合，列出必须更新的文档。
3. **最小实现**：只改与任务直接相关的文件，不做顺手重构。
4. **同步文档**：框架变更在同一变更中完成文档更新。
5. **执行验证**：至少运行与改动相关的检查；框架变更跑完成定义中的门禁。
6. **报告结果**：说明改了什么、文档更新了哪些、执行了哪些命令、还有哪些限制。

### 7.3 功能规格（Spec Kit Lite）

非平凡业务功能按 [specs/README](./specs/README.md) 和 [constitution](./specs/constitution.md) 执行：

```text
用户指令 → AGENTS + docs → specs/constitution.md → spec.md → plan.md → tasks.md → verification.md
```

- `spec.md` 只写 What / Why；技术方案写入 `plan.md`。
- `tasks.md` 是跨会话任务源；当前会话 todo 只是执行镜像。
- 已完成 feature 目录冻结；新需求新建编号并写 `Supersedes`。
- 纯文案、单行修复、依赖补丁和工具链升级不创建 spec。

### 7.4 不要做

- 不要修改与本项目无关的目录，或覆盖用户未提交的改动。
- 不要在没有用户要求时执行 `git commit`、`git push` 或改写历史。
- 不要为了让检查通过而删除测试、放宽规则或加入无依据的 ignore。
- 不要绕过 `router.Register`、直接写 SQL、改已应用的迁移或伪造审计。
- 不要提交 `.env`、密钥、token、生产配置。
- 不要在没有 spec/ADR 的情况下引入 proto3、ORM、Redis、消息队列或微服务。

## 8. 完成定义

框架变更只有在以下全部满足时才算完成：

- [ ] 功能实现符合架构边界与工程约束
- [ ] 同一次变更中更新了 [changelog](./docs/changelog.md)
- [ ] 按矩阵更新了受影响的 architecture / engineering / README / AGENTS
- [ ] `gofmt -l .` 为空
- [ ] `go vet ./...` 通过
- [ ] `go test ./...` 通过
- [ ] 新增/修改路由有覆盖测试
- [ ] 新增鉴权路径有 401/403 用例
- [ ] 新增富文本路径有 XSS 用例
- [ ] 未执行的集成测试已说明原因与风险
- [ ] `go build ./...` 通过
- [ ] 路由契约与覆盖测试通过（新增/修改路由时）
- [ ] 涉及 Compose/Docker 时 `docker compose config --quiet` 通过
- [ ] 非平凡功能有 spec/plan/tasks，且 `verification.md` 为 `Verified`
- [ ] 未执行的检查、剩余风险和未覆盖范围已说明
- [ ] 没有未说明的兼容性、迁移或回滚风险

业务变更至少通过 `gofmt`、`go vet`、相关测试和 `go build`；影响用户操作时补充集成测试。

## 9. 命令来源

完整命令、参数、CI 和故障处理只维护在 [docs/engineering.md](./docs/engineering.md#3-命令)。本文只保留完成定义所需的检查项。
