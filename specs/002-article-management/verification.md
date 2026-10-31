# Verification: 文章管理

- Spec: `./spec.md`
- Plan: `./plan.md`
- Status: `Partial`
- Date: `2026-10-31`

## 验收结果

| 验收场景 | 结果 | 证据 |
| --- | --- | --- |
| [US1-1] 创建草稿 + HTML 清洗 | ✅ | WSL PG 创建 200；`<script>` 被移除 |
| [US1-2] 错误 version 返回 409 | ✅ | 1→409；version=2→200/version=3 |
| [US1-3] 无权限 403 | ✅ 路由 | `article:write` 契约 |
| [US2-1] 发布状态与审计 | ✅ | 发布 200，审计 `article.publish` |
| [US2-2] 重复发布 409 | ✅ 代码 | 条件 UPDATE |
| [US2-3] revisions 倒序 | ✅ 代码 | `ListArticleRevisions` |

## 命令与结果

```text
go test ./internal/service/article  PASS
go test ./...                       PASS（路由覆盖/OpenAPI）
go build ./...                      PASS
```

## 数据库/迁移验证

- [x] 创建文章
- [x] 发布（下线代码路径已实现）
- [x] revisions 数量与内容
- [x] 审计行与业务写入

> 已在 WSL PostgreSQL 18.6 验证创建/发布/list/revisions、XSS 清洗和审计。

## 未覆盖范围

- 分类/标签 CRUD
- 定时发布
- 图片上传
- 公开读取 API

## 结论

- [ ] Verified
- [x] Partial
- [ ] Failed

## 收敛记录

- 2026-10-31：P2 代码与单元/路由测试完成，PG 集成待补。
