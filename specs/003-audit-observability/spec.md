# Feature Specification: 审计与可观测性

- ID: `003`
- Status: `Implementing`
- Created: `2026-10-31`
- Supersedes: `none`

## 问题与目标

管理后台需要回答“谁在什么时候改了什么”，并且运维需要看到请求量、错误和延迟。P1/P2 已完成业务写操作，但缺少统一的审计查询面和运行时指标。

## 非目标

- 审计导出、按月分区、冷存储归档、哈希链/WORM
- 全量 OTel trace、日志平台、告警规则落地
- 前端审计页面

## 用户故事

### User Story 1 - 审计查询 (Priority: P1)

auditor 或超管可以按 action/actor 查询审计，keyset 分页；每条记录包含 actor、action、resource、result、IP、request_id、changes。

**验收场景**：

1. **Given** 一次登录成功，**When** 查询 audit-logs，**Then** 有 `auth.login` 行。
2. **Given** 一次文章发布，**When** 查询，**Then** 有 `article.publish` 和字段变化。
3. **Given** 无 `audit:read`，**When** 查询，**Then** 403。

### User Story 2 - Prometheus 指标 (Priority: P2)

运维可以从内网 `:9090/metrics` 抓取请求量、状态码和延迟。

**验收场景**：

1. **Given** API 启动，**When** 请求 `/metrics`，**Then** 返回 Prometheus 文本。
2. **Given** 若干 API 请求，**When** 查看指标，**Then** `http_requests_total` 增长。

## 功能需求

- **FR-001**：关键写操作与业务同事务写 `audit_logs`；登录失败使用 best-effort。
- **FR-002**：审计字段白名单；密码/token/一次性链接/正文不落审计。
- **FR-003**：`GET /audit-logs` 支持 `limit/cursor/action/actorId`，按 id 倒序 keyset。
- **FR-004**：`audit:read` 权限；没有权限 403。
- **FR-005**：Prometheus 指标包含 method/route/status 维度的请求计数与延迟。
- **FR-006**：metrics 只监听内网/loopback，不经过 Caddy 公开暴露。

## 边界与异常

- 审计写失败：业务事务回滚；登录失败审计失败只记错误日志。
- 审计查询无数据：空列表。
- metrics 端口被占用：记录错误，不阻断 API。
- 数据库不可用：审计查询 5xx，不泄露 SQL。

## 安全与审计影响

- `audit_logs` 应用账号应只授予 `INSERT/SELECT`（部署时配置）。
- `changes` 存字段级 diff；正文存哈希。
- 审计查询/导出本身也应被审计（当前未实现导出）。

## 约束与依赖

- [AGENTS.md](../../AGENTS.md)、[docs/architecture.md](../../docs/architecture.md)、[docs/engineering.md](../../docs/engineering.md)
- Prometheus 通过 `internal/platform/metrics` 暴露；无 OTel。

## 成功指标

- [x] 写操作审计代码路径存在。
- [x] `GET /audit-logs` 路由、权限、keyset 查询存在。
- [x] `/metrics` 服务启动。
- [ ] PostgreSQL 审计行断言。
- [ ] Prometheus 实际抓取验证。

## 框架/文档影响

- [x] `internal/audit`、`internal/api/audit`、`internal/service/auditlog`
- [x] `internal/platform/metrics`、`cmd/api/main.go`
- [x] 文档同步
- [ ] 审计导出/保留/Otel 另立 spec
