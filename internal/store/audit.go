package store

import (
	"context"
	"encoding/json"

	"github.com/example/homebrew-api-starter/internal/model"
)

func (q *Queries) ListAuditLogs(ctx context.Context, limit int, cursor int64, action, actorID string) ([]model.AuditLog, int64, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := q.db.Query(ctx,
		`SELECT id, occurred_at, request_id, COALESCE(trace_id, ''),
		        actor_id::text, actor_email, action, resource_type, resource_id,
		        result, status_code, COALESCE(host(ip), ''), user_agent,
		        changes, metadata, duration_ms
		 FROM audit_logs
		 WHERE ($1 = 0 OR id < $1)
		   AND ($2 = '' OR action = $2)
		   AND ($3 = '' OR actor_id::text = $3)
		 ORDER BY id DESC LIMIT $4`, cursor, action, actorID, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []model.AuditLog
	for rows.Next() {
		var a model.AuditLog
		var actorIDVal, actorEmail *string
		var changes, metadata []byte
		if err := rows.Scan(
			&a.ID, &a.OccurredAt, &a.RequestID, &a.TraceID,
			&actorIDVal, &actorEmail, &a.Action, &a.ResourceType, &a.ResourceID,
			&a.Result, &a.StatusCode, &a.IP, &a.UserAgent,
			&changes, &metadata, &a.DurationMS,
		); err != nil {
			return nil, 0, err
		}
		a.ActorID = actorIDVal
		a.ActorEmail = actorEmail
		if len(changes) > 0 {
			a.Changes = json.RawMessage(changes)
		}
		if len(metadata) > 0 {
			a.Metadata = json.RawMessage(metadata)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var next int64
	if len(out) == limit {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
