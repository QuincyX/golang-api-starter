# Tasks: 文章管理

- Spec: `./spec.md`
- Plan: `./plan.md`
- Status: `Implementing`

## Phase 1: API 与 Store

- [x] T001 `internal/store/article.go` 列表/详情/创建/更新/软删/发布/版本查询
- [x] T002 `internal/service/article/service.go` CRUD、状态机、版本、渲染、审计
- [x] T003 `internal/api/article/handler.go` 8 条路由与 DTO
- [x] T004 `router.Register` 声明权限与审计动作

## Phase 2: 安全与测试

- [x] T005 goldmark + bluemonday 渲染清洗
- [x] T006 `version` 乐观锁与 409
- [x] T007 `internal/service/article/service_test.go` XSS/slug/diff
- [x] T008 路由覆盖与 OpenAPI 生成测试

## Phase 3: Verification

- [ ] T009 PostgreSQL 集成：创建 → 发布 → 下线 → revisions
- [ ] T010 审计行断言（actor/action/changes）
- [ ] T011 更新 `verification.md` 与文档

## 规则

- 任务必须可追溯到 spec 或 plan。
- `tasks.md` 是跨会话任务源。
