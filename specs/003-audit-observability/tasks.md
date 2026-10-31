# Tasks: 审计与可观测性

- Spec: `./spec.md`
- Plan: `./plan.md`
- Status: `Implementing`

## Phase 1: 审计

- [x] T001 `internal/audit/recorder.go` 同事务写入 + best-effort
- [x] T002 `internal/store/audit.go` keyset 查询
- [x] T003 `internal/service/auditlog` + `internal/api/audit` 路由
- [x] T004 登录/账号/文章写操作接入审计

## Phase 2: 指标

- [x] T005 `internal/platform/metrics` registry + middleware
- [x] T006 `cmd/api/main.go` 独立 metrics server
- [x] T007 Compose 只映射 loopback 9090

## Phase 3: Verification

- [ ] T008 PostgreSQL 审计行断言
- [ ] T009 事务回滚不残留审计
- [ ] T010 `curl :9090/metrics` 实际抓取
- [ ] T011 更新 `verification.md` 与文档

## 规则

- 任务必须可追溯到 spec 或 plan。
- `tasks.md` 是跨会话任务源。
