# Agent 工程指南（golang-api-starter）

本文件是 Agent 的入口地图，不是完整手册。细节在 `docs/`，按任务读取；文档与代码冲突时以代码为准，并在同一次变更中修正文档。

## 1. 优先级与读取路由

1. 用户当前的明确指令。
2. 本文件。
3. 目标文档、现有代码与测试。

| 任务 | 先读 |
| --- | --- |
| 新增 / 修改接口 | [architecture](docs/architecture.md)、[engineering](docs/engineering.md)、`internal/router`、`internal/api` |
| 鉴权 / 权限 / 审计 | [design](docs/design.md)、相关 spec、`internal/{auth,audit,middleware}` |
| 数据库 / 迁移 | [engineering](docs/engineering.md)、`migrations/`、`internal/store/` |
| 依赖 / 部署 / 门禁 | [engineering](docs/engineering.md)、`Taskfile.yml`、`.golangci.yml` |
| 规格 / 验收 | [specs](specs/README.md)、[constitution](specs/constitution.md) |
| 不确定需要哪些上下文 | [docs/README](docs/README.md) 的索引 |

## 2. 项目定位

- Golang 管理后台 API 起始模板；当前已实现 P1–P3：登录/RBAC/账号、文章、审计、Prometheus metrics。
- 不引入 proto3/gRPC、ORM、Redis 连接、消息队列、微服务或前端代码；确有需要时先写 spec/ADR，并得到用户明确要求。
- 模板优先沉淀可复用边界，不写针对单个页面或单张表的一次性补丁。
- 非平凡业务功能按 Spec Kit Lite（[specs](specs/README.md)）执行：spec → plan → tasks → verification。

## 3. 完成定义

框架变更必须全部满足：

- `task verify` 通过（gofmt 检查、`go vet`、`go test`、`go build`、lint、vuln）。
- `task lint`、`task vuln` 无新增问题；工具版本固定在 `Taskfile.yml`，不要依赖全局安装。
- 同一次变更更新 [changelog](docs/changelog.md) 和受影响文档。
- 新增/修改路由有覆盖测试；鉴权路径覆盖 401/403；富文本覆盖 XSS。
- 未执行的检查在交付说明中列出原因和风险。

完整门禁与命令见 [engineering](docs/engineering.md)。

## 4. 架构红线

1. 路由只经 `router.Register`；非 GET 必须有 `AuditAction`，`Public` 不得带 `Permission`。禁止直接调用 `huma.Register`。
2. 分层单向：api → service → store。api 不查库、不开事务；service 不 import `net/http`/api/router；store 不做业务判断。
3. SQL 只放在 `internal/store` 且参数化；迁移只追加，已应用文件不可改（`TestMigrationChecksums` 强制）。
4. JWT 只放身份；权限每请求从 RBAC 读；日志不打印 Authorization/token/密码/一次性链接/正文。
5. 审计与业务同事务，字段白名单；401/403 严格区分；5xx 不回显数据库细节。
6. 时间 UTC、API RFC 3339、UUIDv7、`pageSize` ≤ 100；错误用 problem+json。

`internal/arch` 会机械检查分层、路由入口、环境变量读取、迁移校验和与 OpenAPI 快照，不要用注释或局部 ignore 绕过。

## 5. 框架变更与文档

框架变更包括：依赖/工具链、`router`/`api`/`middleware`/`platform` 结构、配置、迁移、路由契约、鉴权/审计机制、部署方式、Agent 规则。

- 必须同次更新 [changelog](docs/changelog.md)，并按 [engineering 的变更矩阵](docs/engineering.md) 更新 architecture / engineering / README / testing / AGENTS 中的受影响项。
- 全局规则只写“触发条件 → 边界 → 验证方式”，禁止空泛要求；规则准入与季度审计见 [engineering](docs/engineering.md)。
- 新增依赖必须说明用途、维护状态、替代方案，并写入 changelog。

## 6. 安全边界

- 密钥只从环境/Secret 注入；`JWT_SECRET` ≥32 字节；不提交 `.env`、证书、token。
- 不开放公开注册；账号通过 seed/邀请创建，首次登录强制改密。
- 不执行破坏性数据库操作；生产变更、凭据、跨项目改动先停下询问。
- 不为了通过检查删除测试、放宽门禁或加入无依据的 ignore；依赖漏洞用 `task vuln`。

## 7. 工作流程

- 非平凡任务先写契约：目标、不可变约束、可自主执行范围、验收证据、停止条件。
- 小改动（拼写、单行、明确文案）不需要完整契约。
- 读现有源码/测试/文档，最小实现，不做顺手重构；不覆盖用户未提交改动，不修改无关目录。
- 未获用户明确要求，不执行 `git commit`、`git push` 或改写历史。
- 完成后报告：改了什么、文档更新、执行的命令与结果、未验证项。

## 8. 文档入口

- [architecture](docs/architecture.md)：分层、请求链、扩展位置、已知约束
- [engineering](docs/engineering.md)：命令、质量门禁、编码约束、部署
- [testing](docs/testing.md)：测试边界、快照、集成与验收
- [design](docs/design.md)：技术选型、数据与安全设计
- [specs](specs/README.md)：spec / plan / tasks / verification
- [changelog](docs/changelog.md)：框架变更历史
