-- +goose Up
-- PostgreSQL 18 provides uuidv7() natively. Keep timestamps in UTC (timestamptz).
CREATE TABLE admin_users (
    id                   uuid PRIMARY KEY DEFAULT uuidv7(),
    email                text NOT NULL,
    password_hash        text NOT NULL,
    display_name         text NOT NULL,
    status               text NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active', 'disabled', 'locked')),
    must_change_password boolean NOT NULL DEFAULT false,
    token_version        integer NOT NULL DEFAULT 0,
    failed_login_count   integer NOT NULL DEFAULT 0,
    locked_until         timestamptz,
    last_login_at        timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);
CREATE UNIQUE INDEX admin_users_email_uq
    ON admin_users (lower(email)) WHERE deleted_at IS NULL;

CREATE TABLE roles (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    is_system   boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    code text PRIMARY KEY,
    name text NOT NULL,
    grp  text NOT NULL
);

CREATE TABLE role_permissions (
    role_id         uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_code text NOT NULL REFERENCES permissions(code) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_code)
);

CREATE TABLE admin_user_roles (
    user_id uuid NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE admin_sessions (
    id            uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id       uuid NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    user_agent    text NOT NULL DEFAULT '',
    ip            inet,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    revoked_at    timestamptz,
    revoke_reason text
);
CREATE INDEX admin_sessions_user_idx
    ON admin_sessions (user_id) WHERE revoked_at IS NULL;

CREATE TABLE refresh_tokens (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    session_id  uuid NOT NULL REFERENCES admin_sessions(id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    rotated_at  timestamptz,
    replaced_by uuid REFERENCES refresh_tokens(id)
);
CREATE INDEX refresh_tokens_session_idx ON refresh_tokens (session_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS admin_user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS admin_users;
