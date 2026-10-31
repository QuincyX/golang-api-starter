# Tasks: 管理员账号与鉴权

- Spec: `./spec.md`
- Plan: `./plan.md`
- Status: `Implementing`（核心实现完成，自动化集成测试待补）

## Phase 1: Setup

- [x] T001 确认四个决策：MFA 暂不做；JWT Bearer 无 Cookie/refresh；一次性链接；SSO 以后
- [x] T002 `internal/config` 增加 `ACCESS_TTL=8h`、`ADMIN_WEB_URL`、`BOOTSTRAP_*`、`METRICS_ADDR` 并校验
- [x] T003 `internal/store/auth.go` 用户/角色/权限/session/一次性链接参数化查询
- [x] T004 `internal/audit/recorder.go` 同事务审计与字段白名单

## Phase 2: 登录与吊销

- [x] T005 `internal/service/auth/password.go` Argon2id 哈希/校验/重哈希
- [x] T006 JWT 签发/解析（8h，claims 不含权限）
- [x] T007 `auth.Service` Login/Logout/ChangePassword/ValidateSession
- [x] T008 `internal/api/auth` login/logout/me/password/activate
- [x] T009 `internal/middleware/auth.go` 每请求 session/账号状态/`token_version` 校验
- [x] T010 登录 IP + email 限流；失败计数与 15 分钟锁定
- [x] T011 `cmd/api/app.go` 注册公开 login 与受保护路由
- [x] T012 测试：密码/限流、login handler 401/429/500、middleware 401/403
- [x] T013 测试：app 覆盖登录 401、无 `Set-Cookie`、logout 401

## Phase 3: RBAC

- [x] T014 从 RBAC 表加载权限，替换静态开发权限
- [x] T015 角色分配、最后超管保护（409）
- [x] T016 account 路由声明 `account:*`、`role:*` 权限
- [x] T017 手工验证：editor 访问 accounts/audit 403；最后超管禁用 409

## Phase 4: 账号、一次性链接、会话

- [x] T018 `internal/api/account` 列表/创建/禁用/角色/重置密码
- [x] T019 活跃会话列表/吊销
- [x] T020 一次性设密：32B token、SHA-256、30 分钟、单次使用
- [x] T021 `cmd/seed` 表为空时创建首超管
- [x] T022 `cmd/api/app.go` 注册账号/会话/activate 路由
- [x] T023 手工验证：创建 editor → 激活 → 登录；seed 重复执行跳过

## Phase 5: 审计

- [x] T024 登录/登出/改密/账号/文章写操作接入审计
- [x] T025 关键写操作与业务同事务
- [x] T026 审计 actor_email/request_id/IP/UA 快照
- [x] T027 手工验证：登录/文章发布审计行可查

## Phase 6: Verification

- [x] T028 `gofmt`、`go vet`、`go test`、`go build`
- [x] T029 PostgreSQL 18.6 执行 00001–00005 迁移
- [x] T030 `docker compose config --quiet`
- [x] T031 更新 architecture/engineering/testing/README/changelog
- [x] T032 `verification.md` 记录手工证据

## 待补

- [ ] T033 自动化集成测试（testcontainers）：登录、禁用、session 吊销、链接复用
- [ ] T034 短 TTL access 过期测试
- [ ] T035 审计回滚/并发最后超管测试
