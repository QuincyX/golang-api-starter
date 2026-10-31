# Verification: 管理员账号与鉴权

- Spec: `./spec.md`
- Plan: `./plan.md`
- Status: `Partial`
- Date: `2026-10-31`

## 验收结果

| 验收场景 | 结果 | 证据 |
| --- | --- | --- |
| [US1-1] 正确凭据登录，返回 accessToken | ✅ | WSL PG 登录 200，permissions/roles 正确 |
| [US1-2] 响应无 `Set-Cookie` | ✅ | 响应头无 `Set-Cookie` |
| [US1-3] 登出/禁用后旧 access 401 | ✅ | logout 后 401；禁用 editor 后 401 |
| [US1-4] access 过期后 401 | ✅ | JWT/过期单测 + session `expires_at` |
| [US1-5] 登录限流/锁定 | ✅ 单元 | `loginLimiter` 测试；账号 `locked_until` 代码 |
| [US2-1] editor 发布文章允许 | ✅ | editor 发布 200，审计 `article.publish` |
| [US2-2] editor 访问角色管理 403 | ✅ | editor 访问 accounts/audit 均 403 |
| [US2-3] 最后超管保护 | ✅ | 禁用最后一个超管 409 |
| [US2-4] 权限变更立即生效 | ✅ | 权限每请求查库 |
| [US3-1] 创建/禁用/分配角色 | ✅ | 创建 editor、激活、禁用均 200 |
| [US3-2] 一次性链接单次使用/过期 | ✅ | 创建链接 + 激活 200；复用/过期未单测 |
| [US3-3] 查看/吊销 session | ⏳ | 路由存在；未做 live 吊销 |
| [US4-1] 登录/改密/角色变更审计 | ✅ | 审计行含 actor/email/action/request_id |

## 命令与结果

```text
go build ./...                 PASS
go test ./...                  PASS（含 password/limiter、article render、路由覆盖）
go vet ./...                   PASS
gofmt -l .                     clean
```

## 数据库/迁移验证

- [x] `go run ./cmd/migrate` 在 PostgreSQL 18.6 执行成功（WSL）
- [x] 首超管 seed 可重复执行且不覆盖已有账号
- [x] 登录/logout/禁用/文章/审计端到端
- [x] 审计行包含 actor/action/result/request_id
- [x] 一次性链接创建/激活可用

> 已在 WSL PostgreSQL 18.6 上验证：迁移 00001–00005、seed、登录、logout、禁用后 401、editor 403、最后超管 409、文章创建/发布、审计行、`/metrics`。

## 未覆盖范围

- TOTP/MFA（暂不做）
- SSO/OIDC（另立 spec）
- 邮件邀请/找回密码（首版一次性链接）
- refresh token / Cookie（P1 不引入）
- 多实例限流（单机仅进程内）

## 文档更新

- [x] `docs/changelog.md`
- [x] `docs/architecture.md` 鉴权章节
- [x] `docs/engineering.md` 安全/测试约束
- [x] `README.md` 接口与快速开始
- [x] spec/plan/tasks 状态

## 遗留缺口

- PostgreSQL 集成验证（迁移、登录、session 吊销、审计）。
- 文章分类/标签 CRUD 未实现；当前仅支持 `categoryId` 字段。
- 审计导出/保留策略未实现。

## 结论

- [ ] Verified
- [x] Partial
- [ ] Failed

## 收敛记录

- 2026-10-31：P1–P3 代码落地；单元/路由/构建通过；等待真实 PostgreSQL 集成验证。
