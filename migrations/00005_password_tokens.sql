-- +goose Up
CREATE TABLE admin_password_tokens (
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id    uuid NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_password_tokens_user_idx ON admin_password_tokens (user_id);

-- +goose Down
DROP TABLE IF EXISTS admin_password_tokens;
