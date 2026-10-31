# Feature Specification: 文章管理

- ID: `002`
- Status: `Implementing`
- Created: `2026-10-31`
- Supersedes: `none`

## 问题与目标

P1 已经具备管理后台的账号与权限，但没有内容管理能力。需要一个能支撑编辑、发布、下线、版本回溯的最小文章模块，并且所有写操作受 RBAC 与审计约束。

## 非目标

- 公开文章读取 API、CDN 缓存、站点渲染
- 定时发布、审核流、多人协同编辑
- 分类/标签 CRUD（表已建，当前只支持 `categoryId` 字段）
- 图片上传、对象存储直传
- 全文搜索/外部搜索引擎

## 用户故事

### User Story 1 - 文章 CRUD (Priority: P1)

editor 可以创建草稿、编辑标题/正文/摘要/封面/分类，按分页和状态筛选列表；删除是软删。

**验收场景**：

1. **Given** editor，**When** 创建草稿，**Then** 返回 `draft` 文章，正文 HTML 已清洗。
2. **Given** 已有文章，**When** 用错误 `version` 更新，**Then** 返回 409，不覆盖他人修改。
3. **Given** 没有 `article:write`，**When** 创建/更新，**Then** 403。

### User Story 2 - 发布、下线与版本 (Priority: P1)

editor 可以把草稿发布为 `published`，把已发布文章下线回 `draft`；每次创建/更新/发布/下线生成版本记录，可查询历史。

**验收场景**：

1. **Given** 草稿，**When** 发布，**Then** `status=published`、`publishedAt` 有值、审计 `article.publish`。
2. **Given** 已发布，**When** 再次发布，**Then** 409，不重复发布。
3. **Given** 文章，**When** 查询 revisions，**Then** 按版本倒序返回。

## 功能需求

- **FR-001**：`GET /articles` 支持 `page/pageSize/status/q`，默认按创建时间倒序，`pageSize<=100`。
- **FR-002**：`POST /articles` 创建草稿；slug 为空时从标题生成，重复 slug 返回 409。
- **FR-003**：`PUT /articles/{id}` 使用 `version` 乐观锁，成功则 `version+1` 并生成 revision。
- **FR-004**：`DELETE /articles/{id}` 软删，保留历史。
- **FR-005**：`POST /articles/{id}/publish`、`unpublish` 使用状态机，只允许合法转换。
- **FR-006**：Markdown 渲染 + HTML 清洗，禁止原始 `<script>` 等危险标签。
- **FR-007**：创建/更新/发布/下线/删除写审计，记录 actor、资源、结果和字段级变化。
- **FR-008**：权限码 `article:read/write/publish/delete`，由路由契约逐请求校验。

## 边界与异常

- 文章不存在：404。
- slug 冲突或版本不匹配：409。
- 未登录：401；无权限：403。
- 正文为空：422。
- 数据库不可用：5xx problem+json，不泄露 SQL。

## 安全与审计影响

- 审计动作：`article.create`、`article.update`、`article.publish`、`article.unpublish`、`article.delete`。
- `changes` 中正文只存哈希，不复制全文。
- 渲染 HTML 与审计同事务；发布失败不留下半成品。

## 约束与依赖

- 上位约束：[AGENTS.md](../../AGENTS.md)
- 架构：[docs/architecture.md](../../docs/architecture.md)
- 工程：[docs/engineering.md](../../docs/engineering.md)
- 设计文档：[docs/design.md](../../docs/design.md)
- 表结构：`00002_article.sql`；审计：`00003_audit.sql`。

## 成功指标

- [x] 创建/更新/发布/下线/版本查询代码路径存在。
- [x] Markdown XSS 清洗单测通过。
- [x] 路由权限/审计声明通过 `verifyRouteCoverage`。
- [ ] PostgreSQL 端到端：创建 → 发布 → 下线 → revisions。

## 框架/文档影响

- [x] `internal/service/article`、`internal/store/article.go`、`internal/api/article`
- [x] `docs/architecture.md`、`README.md`、`docs/changelog.md`
- [ ] 真实 PG 集成验证
