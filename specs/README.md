# 功能规格索引

本目录采用轻量 Spec-Driven Development 流程，参考 Spec Kit 的工作方式，但不依赖 Spec Kit CLI 或自动生成 skill。

## 使用方式

- 流程规则：[constitution.md](./constitution.md)
- 模板目录：[_templates/](./_templates/)
- 功能目录：`specs/NNN-feature-slug/`
- 一个功能目录至少包含：`spec.md`、`plan.md`、`tasks.md`、`verification.md`
- 涉及外部接口时可增加 `contracts/api.md`；涉及复杂模型时可增加 `data-model.md`

## 状态

`Draft → Clarified → Planned → Implementing → Verified → Superseded`

已验收目录冻结为历史记录。新需求创建新编号，并在 spec 中写 `Supersedes`。

## 功能索引

> 001–003 是模板自带的示例功能，用来演示 spec/plan/tasks/verification 的写法；状态 `Implementing`/`Partial` 表示核心代码完成、自动化集成测试仍可加强。复制模板后按业务需要保留或删除。

| 编号 | 功能 | 状态 | 说明 |
| --- | --- | --- | --- |
| 001 | [管理员账号与鉴权](./001-account-auth/spec.md) | Implementing | JWT Bearer（无 Cookie/refresh）、RBAC、账号/session、一次性链接、登录审计；代码已落地，PG 集成待补 |
| 002 | [文章管理](./002-article-management/spec.md) | Implementing | 文章 CRUD、发布/下线、版本、Markdown 清洗；代码已落地，PG 集成待补 |
| 003 | [审计与可观测性](./003-audit-observability/spec.md) | Implementing | 同事务审计、keyset 查询、Prometheus metrics；代码已落地，PG/抓取待补 |

## 只在这些场景使用

- 新业务模块（例如文章 CRUD、审计查询）
- API/数据模型/权限变更
- 多角色或多步骤流程
- 高不确定性需求

纯修复、格式化、单文件重构、依赖补丁和工具链升级继续使用任务契约或工程文档，不创建 spec。

## 闭环

1. `spec.md` 评审通过（目标、非目标、P1 故事、验收场景）
2. `plan.md` 评审通过（文件影响、接口、测试、文档义务）
3. `tasks.md` 按 P1 切片实现
4. `verification.md` 记录验收证据
5. 有缺口则回到 tasks，直到 `Verified`
6. 关闭并冻结目录

## 约束

- 上位规则：[AGENTS.md](../AGENTS.md)、[架构说明](../docs/architecture.md)、[工程说明](../docs/engineering.md)
- `spec.md` 只写 What / Why；技术方案写在 `plan.md`。
- 每个 P1 故事必须能独立实现、独立验证、独立演示。
- 未解决的产品/权限/数据问题标 `[NEEDS CLARIFICATION]`，进入实现前必须解决。
- 不把 spec 当作第二份工程规范；架构与安全边界以 `AGENTS.md` 和 `docs/` 为准。
