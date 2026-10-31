-- +goose Up
CREATE TABLE audit_logs (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    request_id    text NOT NULL,
    trace_id      text,
    actor_id      uuid,
    actor_email   text,
    action        text NOT NULL,
    resource_type text NOT NULL,
    resource_id   text,
    result        text NOT NULL CHECK (result IN ('success', 'failure')),
    status_code   integer,
    ip            inet,
    user_agent    text,
    changes       jsonb,
    metadata      jsonb,
    duration_ms   integer
);
CREATE INDEX audit_logs_occurred_idx ON audit_logs (occurred_at DESC);
CREATE INDEX audit_logs_actor_idx ON audit_logs (actor_id, occurred_at DESC);
CREATE INDEX audit_logs_resource_idx ON audit_logs (resource_type, resource_id, occurred_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
