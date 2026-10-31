# Feature Specification: [FEATURE NAME]

- ID: `[NNN]`
- Status: `Draft`
- Created: `[YYYY-MM-DD]`
- Supersedes: `[none | NNN-slug]`

## 问题与目标

[描述当前问题、目标用户和期望结果。只写 What / Why。]

## 非目标

- [明确不做什么，避免范围膨胀]

## 用户故事

### User Story 1 - [标题] (Priority: P1)

[用用户/调用方语言描述一个可独立交付的场景。]

**为什么是 P1**：[用户或系统价值]

**独立验证方式**：[如何只验证这个切片，例如一个接口 + 一个测试]

**验收场景**：

1. **Given** [初始状态]，**When** [操作]，**Then** [结果]
2. **Given** [初始状态]，**When** [操作]，**Then** [结果]

### User Story 2 - [标题] (Priority: P2)

[同上]

## 功能需求

- **FR-001**：[可验证的行为]
- **FR-002**：[可验证的行为]
- **FR-003**：[可验证的行为]

## 边界与异常

- [空状态、失败、权限、超时、冲突、重复请求、并发]
- [数据库不可用、迁移失败、限流、会话过期]

## 安全与审计影响

- 权限码：[新增/复用]
- 审计动作：[domain.verb；是否与业务同事务]
- 敏感数据：[密码/token/PII 是否进入日志或审计]

## 约束与依赖

- 上位约束：[AGENTS.md](../../AGENTS.md)
- 架构：[docs/architecture.md](../../docs/architecture.md)
- 工程：[docs/engineering.md](../../docs/engineering.md)
- 设计文档：[docs/design.md](../../docs/design.md)
- 业务/外部依赖：[...]

## 待澄清问题

- [ ] [NEEDS CLARIFICATION: ...]

## 成功指标

- [ ] [可度量结果，例如“某接口 401/403/200 三条路径有集成测试”]

## 框架/文档影响

- [ ] 无框架变更
- [ ] 需要更新 `docs/changelog.md`
- [ ] 需要更新 README / 架构 / 工程说明
- [ ] 需要新增或修改迁移
