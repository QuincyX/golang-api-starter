# Implementation Plan: 审计与可观测性

- Spec: `./spec.md`
- Status: `Implementing`
- Date: `2026-10-31`

## 技术方案

```text
service 事务 ──→ audit.Record(tx) ──→ audit_logs
API 请求 ──→ metrics.Middleware ──→ prometheus registry ──→ :9090/metrics
```

- 审计写入与业务同 tx；登录失败无 tx，使用 `RecordBestEffort`。
- 审计查询用 `id < cursor ORDER BY id DESC LIMIT n`，避免 offset。
- metrics 使用独立 `http.Server`，默认 `127.0.0.1:9090`。
- 请求标签使用 `method/path/status`；管理后台规模可接受。

## 受影响区域

| 区域 | 文件/模块 | 变更 |
| --- | --- | --- |
| 审计写入 | `internal/audit/recorder.go` | `Record` / `RecordBestEffort` |
| 审计查询 | `internal/store/audit.go`、`service/auditlog`、`api/audit` | keyset 列表 |
| 指标 | `internal/platform/metrics/metrics.go`、`cmd/api/main.go` | registry + server |
| 中间件 | `internal/middleware/requestmeta.go` | IP/UA/request id |
| 文档 | `docs/architecture.md`、README、spec | 同步 |

## 数据与接口

| 方法 | 路径 | 权限 |
| --- | --- | --- |
| GET | `/api/admin/v1/audit-logs` | `audit:read` |
| GET | `:9090/metrics` | 内网 |

## 事务与失败处理

- 业务写操作：审计失败 → 整个 tx 回滚。
- 登录失败：审计 best-effort，失败只记日志。
- metrics server 启动失败：记录错误，API 继续。

## 测试策略

- 路由契约：audit 路由有 permission。
- 单元：审计字段序列化（待补）。
- 集成：PG 写入/查询、事务回滚（待补）。
- 手工：`curl :9090/metrics`。

## 风险与回滚

| 风险 | 缓解 |
| --- | --- |
| 审计表增长 | 后续分区/归档/保留策略 |
| 指标高基数 | 使用路由而非原始 URL（后续优化） |
| 审计写失败影响登录 | 登录失败 best-effort；业务写同事务 |
| 端口冲突 | 记录错误，不阻断 API |

## 文档影响

`docs/architecture.md`、`docs/engineering.md`、`README.md`、`docs/changelog.md`、本 spec/verification。
