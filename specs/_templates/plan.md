# Implementation Plan: [FEATURE NAME]

- Spec: `[../NNN-slug/spec.md]`
- Status: `Planned`
- Date: `[YYYY-MM-DD]`

## 技术方案

[说明实现思路、复用的现有模块、关键取舍。不要复制整份架构文档。]

## 受影响区域

| 区域 | 文件/模块 | 变更 |
| --- | --- | --- |
| 路由/契约 | `internal/router/...` | ... |
| API | `internal/api/...` | ... |
| Service | `internal/service/...` | ... |
| Store/SQL | `internal/store/queries/...` | ... |
| 迁移 | `migrations/NNNN_*.sql` | ... |
| 中间件/平台 | `internal/middleware/...`、`internal/platform/...` | ... |
| 配置 | `internal/config/...` | ... |
| 测试 | `cmd/api/...`、`internal/**/*_test.go` | ... |
| 文档 | `docs/...`、`specs/...` | ... |

## 数据与接口

[接口路径、方法、请求/响应 DTO、错误码、分页、幂等、权限码、审计动作。复杂契约放 `contracts/api.md`。]

## 事务、鉴权与审计

[事务边界、锁、`SELECT ... FOR UPDATE`、权限来源、JWT claims、session/`token_version`、审计写入点。]

## 迁移与回滚

[新增表/索引/约束、是否在线、回滚方式、备份与恢复。没有迁移时写“无”。]

## 测试策略

- 单元：
- Handler：
- 集成（testcontainers/PG）：
- 路由契约：
- 人工验收：

## 宪法检查

- [ ] 遵守 AGENTS 架构边界
- [ ] 不重复已有工程规则
- [ ] 明确非目标和范围
- [ ] 有可验证的验收证据
- [ ] 框架变更已列入文档任务
- [ ] 迁移/鉴权/审计有回滚或恢复说明

## 风险与回滚

[风险、兼容性、性能、锁竞争、数据迁移、回滚方式。]

## 文档影响

[changelog、architecture、engineering、README、AGENTS、spec verification。]
