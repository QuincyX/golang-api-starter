# Implementation Plan: 文章管理

- Spec: `./spec.md`
- Status: `Implementing`
- Date: `2026-10-31`

## 技术方案

```text
api/article → service/article → store/article.go → PostgreSQL
                         └──────→ audit.Record（同事务）
```

- 写操作在 service 事务内完成：文章行 + revision + audit。
- `version` 乐观锁：`UPDATE ... WHERE id = $1 AND version = $2`。
- 发布/下线使用条件 UPDATE，非法状态转换返回 409。
- Markdown 用 goldmark 渲染，bluemonday UGCPolicy 清洗；HTML 存库，读时不再渲染。
- 审计 diff 只记录 title/slug/status 与正文哈希。

## 受影响区域

| 区域 | 文件/模块 | 变更 |
| --- | --- | --- |
| 路由 | `internal/api/article/handler.go` | 8 条文章路由 |
| Service | `internal/service/article/service.go` | CRUD、状态机、版本、渲染、审计 |
| Store | `internal/store/article.go` | 列表/详情/创建/更新/软删/发布/版本 |
| 迁移 | `00002_article.sql`、`00003_audit.sql` | 复用，无新增 |
| 测试 | `internal/service/article/service_test.go` | XSS、slug、diff |
| 文档 | `docs/architecture.md`、README、spec | 同步 |

## 数据与接口

| 方法 | 路径 | 权限 |
| --- | --- | --- |
| GET | `/articles` | `article:read` |
| GET | `/articles/{id}` | `article:read` |
| POST | `/articles` | `article:write` |
| PUT | `/articles/{id}` | `article:write` |
| DELETE | `/articles/{id}` | `article:delete` |
| POST | `/articles/{id}/publish` | `article:publish` |
| POST | `/articles/{id}/unpublish` | `article:publish` |
| GET | `/articles/{id}/revisions` | `article:read` |

## 事务、鉴权与审计

- 创建/更新/发布/下线/删除：`store.WithTx` + `audit.Record`。
- 审计动作与 route 的 `AuditAction` 一致。
- 权限由 `router.Register` 的包装层统一判断。

## 测试策略

- 单元：Markdown 清洗、slug、diff。
- Handler：路由契约 + 401/403（通过 app 测试的 fake）。
- 集成：待 PostgreSQL；创建→发布→下线→revisions。

## 风险与回滚

| 风险 | 缓解 |
| --- | --- |
| 正文 XSS | goldmark + bluemonday；单测覆盖 |
| 并发覆盖 | `version` 乐观锁 + 409 |
| 发布重复 | 条件 UPDATE，非法状态 409 |
| 审计丢失 | 同事务写入 |

## 文档影响

`docs/architecture.md`、`README.md`、`docs/changelog.md`、本 spec/verification。
