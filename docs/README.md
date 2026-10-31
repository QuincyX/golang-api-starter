# 工程文档

本目录是 `homebrew-api-starter` 的工程文档入口。文档以**当前代码**为准，任何非业务层的框架功能变更都必须同步更新相应文档。

| 文档 | 内容 | 主要读者 |
| --- | --- | --- |
| [AGENTS.md](../AGENTS.md) | Agent/自动化协作规则、变更分类、完成定义 | AI Agent、维护者 |
| [功能规格](../specs/README.md) | 非平凡功能的 spec/plan/tasks/verification 流程 | 产品、后端、Agent |
| [架构说明](./architecture.md) | 分层、模块边界、请求链、扩展位置、已知约束 | 后端开发者 |
| [工程说明与约束](./engineering.md) | 环境、命令、质量门禁、依赖策略、测试与发布约束 | 开发者、Reviewer |
| [测试规范](./testing.md) | 单元/集成测试边界、命名、确定性、覆盖率与用例清单 | 开发者、Reviewer |
| [变更记录](./changelog.md) | 非业务层框架变更的追加式记录 | 全体维护者 |
| [README](../README.md) | 项目介绍、快速开始、接口清单 | 新成员、使用者 |
| [设计文档](./design.md) | 完整技术方案、数据模型、审计与部署设计 | 架构评审、维护者 |
| [模板使用指南](./template-guide.md) | 复制、重命名、首次启动、清理示例业务 | 新项目初始化者 |

## 事实归属

为减少重复和文档漂移，每一类事实只有一个权威来源：

| 事实类型 | 唯一来源 | 其他文档的责任 |
| --- | --- | --- |
| 命令、参数、CI、环境变量、发布流程 | [engineering.md](./engineering.md) | 只链接或给最小入口 |
| 分层、目录职责、请求链、架构边界 | [architecture.md](./architecture.md) | 只链接或摘要 |
| Agent 行为规则、准入、完成定义 | [AGENTS.md](../AGENTS.md) | 只链接，不复制完整规则 |
| 用户使用方式、接口、部署入口 | [README.md](../README.md) | 保持面向用户的最新说明 |
| 功能规格、技术计划、任务与验收 | `specs/NNN-slug/` | 用 [specs/README.md](../specs/README.md) 路由；完成后冻结 |
| 测试边界、命名、确定性、覆盖率 | [testing.md](./testing.md) | 只链接，不复制完整规范 |
| 复制模板、重命名、首次启动、删除示例业务 | [template-guide.md](./template-guide.md) | 只链接，不复制步骤 |
| 框架变更历史 | [changelog.md](./changelog.md) | 追加记录，不重写历史 |
| 文档索引与治理规则 | 本文件 | 维护路由和归属 |
| 代码契约、DTO、路由声明 | `internal/router/`、`internal/api/` 与源码 | 文档解释设计，不复制实现 |

允许为了可读性保留最小摘要和入口链接，但不要复制完整流程、命令表或规则正文。发现重复内容时，以本表指定的来源为准，并在同一次变更中合并或删除重复部分。

例外：README 可以保留最小启动和接口入口；AGENTS 的完成定义可以列出必须通过的检查名；这些是入口/验收，不是完整命令文档。

## 文档规则

1. **业务功能变更**：至少保证现有文档不失效；影响用户使用方式时更新 `README.md`。
2. **非业务层框架变更**：必须在同一次变更中更新 [变更记录](./changelog.md)，以及至少一份受影响的说明文档（architecture / engineering / README / AGENTS）。
3. **规则本身变更**：同步更新 `AGENTS.md` 与 [工程说明的规则生命周期](./engineering.md#agent-规则生命周期与审计)，并写入变更记录。
4. **非平凡功能**：使用 [Spec Kit Lite 流程](../specs/README.md)，完成后将 `verification.md` 置为 `Verified` 并冻结目录；新需求新建编号。
5. 文档中的相对链接由 `go test ./internal/arch` 校验；失效链接会阻断测试。
6. 历史记录只追加、不重写；已发生的框架变更不得从记录中删除。
