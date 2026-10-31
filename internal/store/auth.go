package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/example/homebrew-api-starter/internal/model"
)

const userColumns = `id::text, email, password_hash, display_name, status,
	must_change_password, token_version, last_login_at, created_at`

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Status,
		&u.MustChangePassword, &u.TokenVersion, &u.LastLoginAt, &u.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (q *Queries) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	return scanUser(q.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM admin_users
		 WHERE lower(email) = lower($1) AND deleted_at IS NULL`, email))
}

func (q *Queries) UserByID(ctx context.Context, id string) (*model.User, error) {
	return scanUser(q.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM admin_users
		 WHERE id = $1::text::uuid AND deleted_at IS NULL`, id))
}

func (q *Queries) CreateSession(ctx context.Context, userID, userAgent, ip string, expiresAt time.Time) (string, error) {
	var ipArg any
	if strings.TrimSpace(ip) != "" {
		ipArg = ip
	}
	var id string
	err := q.db.QueryRow(ctx,
		`INSERT INTO admin_sessions (user_id, user_agent, ip, expires_at)
		 VALUES ($1::text::uuid, $2, $3::text::inet, $4) RETURNING id::text`,
		userID, userAgent, ipArg, expiresAt).Scan(&id)
	return id, err
}

func (q *Queries) ValidateSession(ctx context.Context, sessionID, userID string, tokenVersion int) (*model.User, error) {
	var u model.User
	err := q.db.QueryRow(ctx,
		`SELECT u.id::text, u.email, u.display_name, u.status, u.token_version
		 FROM admin_sessions s
		 JOIN admin_users u ON u.id = s.user_id
		 WHERE s.id = $1::text::uuid
		   AND u.id = $2::text::uuid
		   AND s.revoked_at IS NULL
		   AND s.expires_at > now()
		   AND u.status = 'active'
		   AND u.deleted_at IS NULL
		   AND u.token_version = $3`,
		sessionID, userID, tokenVersion).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.Status, &u.TokenVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (q *Queries) RevokeSession(ctx context.Context, sessionID, reason string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_sessions SET revoked_at = now(), revoke_reason = $2
		 WHERE id = $1::text::uuid AND revoked_at IS NULL`, sessionID, reason)
	return err
}

func (q *Queries) RevokeUserSessions(ctx context.Context, userID, reason string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_sessions SET revoked_at = now(), revoke_reason = $2
		 WHERE user_id = $1::text::uuid AND revoked_at IS NULL`, userID, reason)
	return err
}

func (q *Queries) PermissionsForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := q.db.Query(ctx,
		`SELECT DISTINCT p.code
		 FROM permissions p
		 JOIN role_permissions rp ON rp.permission_code = p.code
		 JOIN admin_user_roles ur ON ur.role_id = rp.role_id
		 WHERE ur.user_id = $1::text::uuid
		 ORDER BY p.code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

func (q *Queries) RolesForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := q.db.Query(ctx,
		`SELECT r.code FROM roles r
		 JOIN admin_user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = $1::text::uuid ORDER BY r.code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

func (q *Queries) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_users SET last_login_at = now() WHERE id = $1::text::uuid`, userID)
	return err
}

func (q *Queries) IsLocked(ctx context.Context, userID string) (bool, error) {
	var locked bool
	err := q.db.QueryRow(ctx,
		`SELECT locked_until IS NOT NULL AND locked_until > now()
		 FROM admin_users WHERE id = $1::text::uuid`, userID).Scan(&locked)
	return locked, err
}

func (q *Queries) RecordLoginFailure(ctx context.Context, userID string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_users
		 SET failed_login_count = failed_login_count + 1,
		     locked_until = CASE
		       WHEN failed_login_count + 1 >= 10 THEN now() + interval '15 minutes'
		       ELSE locked_until
		     END,
		     updated_at = now()
		 WHERE id = $1::text::uuid`, userID)
	return err
}

func (q *Queries) ResetLoginFailures(ctx context.Context, userID string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_users
		 SET failed_login_count = 0, locked_until = NULL, updated_at = now()
		 WHERE id = $1::text::uuid`, userID)
	return err
}

func (q *Queries) CreateUser(ctx context.Context, email, passwordHash, displayName string, mustChangePassword bool) (*model.User, error) {
	var u model.User
	err := q.db.QueryRow(ctx,
		`INSERT INTO admin_users (email, password_hash, display_name, status, must_change_password)
		 VALUES (lower($1), $2, $3, 'active', $4)
		 RETURNING `+userColumns,
		email, passwordHash, displayName, mustChangePassword).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Status,
		&u.MustChangePassword, &u.TokenVersion, &u.LastLoginAt, &u.CreatedAt,
	)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return &u, err
}

func (q *Queries) SetUserStatus(ctx context.Context, userID, status string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_users SET status = $2, updated_at = now()
		 WHERE id = $1::text::uuid AND deleted_at IS NULL`, userID, status)
	return err
}

func (q *Queries) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_users
		 SET password_hash = $2, must_change_password = false,
		     token_version = token_version + 1, updated_at = now()
		 WHERE id = $1::text::uuid AND deleted_at IS NULL`, userID, passwordHash)
	return err
}

// UpdatePasswordHash is used for transparent Argon2 rehashing on login. It
// must not bump token_version, otherwise the token just issued would be invalid.
func (q *Queries) UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE admin_users SET password_hash = $2, updated_at = now()
		 WHERE id = $1::text::uuid AND deleted_at IS NULL`, userID, passwordHash)
	return err
}

func (q *Queries) SetUserRoles(ctx context.Context, userID string, roleIDs []string) error {
	if _, err := q.db.Exec(ctx, `DELETE FROM admin_user_roles WHERE user_id = $1::text::uuid`, userID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if _, err := q.db.Exec(ctx,
			`INSERT INTO admin_user_roles (user_id, role_id) VALUES ($1::text::uuid, $2::text::uuid)`,
			userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queries) CountActiveSuperAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx,
		`SELECT count(DISTINCT u.id)
		 FROM admin_users u
		 JOIN admin_user_roles ur ON ur.user_id = u.id
		 JOIN roles r ON r.id = ur.role_id
		 WHERE r.code = 'super_admin' AND u.status = 'active' AND u.deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (q *Queries) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx, `SELECT count(*) FROM admin_users WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (q *Queries) HasRole(ctx context.Context, userID, roleCode string) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx,
		`SELECT EXISTS (
		   SELECT 1 FROM admin_user_roles ur
		   JOIN roles r ON r.id = ur.role_id
		   WHERE ur.user_id = $1::text::uuid AND r.code = $2
		 )`, userID, roleCode).Scan(&exists)
	return exists, err
}

func (q *Queries) ListUsers(ctx context.Context, query string, limit, offset int) ([]model.User, int64, error) {
	var total int64
	if err := q.db.QueryRow(ctx,
		`SELECT count(*) FROM admin_users
		 WHERE deleted_at IS NULL
		   AND ($1 = '' OR lower(email) LIKE '%' || lower($1) || '%' OR display_name ILIKE '%' || $1 || '%')`,
		query).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := q.db.Query(ctx,
		`SELECT `+userColumns+` FROM admin_users
		 WHERE deleted_at IS NULL
		   AND ($1 = '' OR lower(email) LIKE '%' || lower($1) || '%' OR display_name ILIKE '%' || $1 || '%')
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Status,
			&u.MustChangePassword, &u.TokenVersion, &u.LastLoginAt, &u.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func (q *Queries) ListSessions(ctx context.Context, userID string) ([]model.Session, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id::text, user_agent, COALESCE(host(ip), ''), created_at, last_seen_at, expires_at, revoked_at
		 FROM admin_sessions WHERE user_id = $1::text::uuid ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Session
	for rows.Next() {
		var s model.Session
		s.UserID = userID
		if err := rows.Scan(&s.ID, &s.UserAgent, &s.IP, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt, &s.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (q *Queries) CreatePasswordToken(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO admin_password_tokens (user_id, token_hash, expires_at)
		 VALUES ($1::text::uuid, $2, $3)`, userID, tokenHash, expiresAt)
	return err
}

func (q *Queries) ConsumePasswordToken(ctx context.Context, tokenHash []byte) (string, error) {
	var userID string
	err := q.db.QueryRow(ctx,
		`UPDATE admin_password_tokens
		 SET used_at = now()
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
		 RETURNING user_id::text`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}

func (q *Queries) RoleIDsByCodes(ctx context.Context, codes []string) ([]string, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id::text FROM roles WHERE code = ANY($1::text[]) ORDER BY code`, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (q *Queries) ListRoles(ctx context.Context) ([]map[string]any, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id::text, code, name, description, is_system FROM roles ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id, code, name, description string
		var system bool
		if err := rows.Scan(&id, &code, &name, &description, &system); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "code": code, "name": name,
			"description": description, "isSystem": system,
		})
	}
	return out, rows.Err()
}

func (q *Queries) ListPermissions(ctx context.Context) ([]map[string]any, error) {
	rows, err := q.db.Query(ctx,
		`SELECT code, name, grp FROM permissions ORDER BY grp, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var code, name, grp string
		if err := rows.Scan(&code, &name, &grp); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"code": code, "name": name, "group": grp})
	}
	return out, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
