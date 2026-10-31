# Tasks: [FEATURE NAME]

- Spec: `[../NNN-slug/spec.md]`
- Plan: `[../NNN-slug/plan.md]`
- Status: `Pending`

## Phase 1: Setup

- [ ] T001 [US?] [精确文件路径] 任务描述

## Phase 2: Foundation

- [ ] T002 [P] [US?] [精确文件路径] 任务描述
- [ ] T003 [US?] [精确文件路径] 任务描述

## Phase 3: User Story 1（P1）

- [ ] T004 [US1] [精确文件路径] 任务描述
- [ ] T005 [P] [US1] [精确文件路径] 任务描述

**Checkpoint**：US1 独立验收

## Phase 4: User Story 2（P2）

- [ ] T006 [US2] [精确文件路径] 任务描述

## Phase 5: Verification

- [ ] T007 运行 `gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...`
- [ ] T008 涉及数据库时运行迁移与集成测试
- [ ] T009 涉及路由时确认 `verifyRouteCoverage` 通过
- [ ] T010 更新 `verification.md` 和必要文档

## 规则

- 任务必须可追溯到 spec 或 plan。
- `[P]` 只用于不同文件且无依赖的任务。
- 不使用 tasks.md 记录纯个人提醒。
- tasks.md 是跨会话的唯一任务源；当前会话 todo 只是执行镜像。
