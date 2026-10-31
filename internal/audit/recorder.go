package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/homebrew-api-starter/internal/platform/requestmeta"
)

type Event struct {
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
	ActorID      string
	ActorEmail   string
	StatusCode   int
	DurationMS   int
	Changes      any
	Metadata     any
}

type Recorder struct {
	pool *pgxpool.Pool
}

func NewRecorder(pool *pgxpool.Pool) *Recorder {
	return &Recorder{pool: pool}
}

// Record writes inside the caller's transaction. Mutations should use this so
// the audit row and the business change commit or roll back together.
func (r *Recorder) Record(ctx context.Context, tx pgx.Tx, event Event) error {
	if event.Action == "" {
		return nil
	}
	meta := requestmeta.From(ctx)
	if event.Result == "" {
		event.Result = "success"
	}

	changes, err := marshal(event.Changes)
	if err != nil {
		return err
	}
	metadata, err := marshal(event.Metadata)
	if err != nil {
		return err
	}

	var actorID any
	if event.ActorID != "" {
		actorID = event.ActorID
	}
	var statusCode any
	if event.StatusCode > 0 {
		statusCode = event.StatusCode
	}
	var duration any
	if event.DurationMS > 0 {
		duration = event.DurationMS
	}
	var ip any
	if meta.IP != "" {
		ip = meta.IP
	}
	var actorEmail any
	if event.ActorEmail != "" {
		actorEmail = event.ActorEmail
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO audit_logs
		 (request_id, trace_id, actor_id, actor_email, action, resource_type,
		  resource_id, result, status_code, ip, user_agent, changes, metadata, duration_ms)
		 VALUES ($1, $2, $3::text::uuid, $4, $5, $6, $7, $8, $9, $10::text::inet, $11, $12, $13, $14)`,
		meta.RequestID, nil, actorID, actorEmail, event.Action, event.ResourceType,
		event.ResourceID, event.Result, statusCode, ip, meta.UserAgent, changes, metadata, duration,
	)
	return err
}

// RecordBestEffort is for events without a business transaction, such as a
// failed login. It must never turn a 401 into a 500.
func (r *Recorder) RecordBestEffort(ctx context.Context, event Event) {
	if r == nil || r.pool == nil {
		return
	}
	meta := requestmeta.From(ctx)
	if event.Result == "" {
		event.Result = "failure"
	}
	changes, _ := marshal(event.Changes)
	metadata, _ := marshal(event.Metadata)

	var actorID any
	if event.ActorID != "" {
		actorID = event.ActorID
	}
	var ip any
	if meta.IP != "" {
		ip = meta.IP
	}
	var statusCode any
	if event.StatusCode > 0 {
		statusCode = event.StatusCode
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs
		 (request_id, actor_id, actor_email, action, resource_type, resource_id,
		  result, status_code, ip, user_agent, changes, metadata)
		 VALUES ($1, $2::text::uuid, $3, $4, $5, $6, $7, $8, $9::text::inet, $10, $11, $12)`,
		meta.RequestID, actorID, event.ActorEmail, event.Action, event.ResourceType,
		event.ResourceID, event.Result, statusCode, ip, meta.UserAgent, changes, metadata,
	)
	if err != nil {
		slog.ErrorContext(ctx, "audit write failed", "action", event.Action, "error", err)
	}
}

func marshal(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func Timestamp() time.Time {
	return time.Now().UTC()
}
