package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/example/homebrew-api-starter/internal/audit"
	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/platform/requestmeta"
	"github.com/example/homebrew-api-starter/internal/store"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRateLimited        = errors.New("too many login attempts")
)

type LoginResult struct {
	User        *model.User
	Permissions []string
	Roles       []string
	AccessToken string
	ExpiresIn   int64
	ExpiresAt   time.Time
}

type Service struct {
	store   *store.Store
	jwt     *jwt.Manager
	audit   *audit.Recorder
	limiter *loginLimiter
	now     func() time.Time
}

func NewService(st *store.Store, manager *jwt.Manager, recorder *audit.Recorder) *Service {
	return &Service{
		store:   st,
		jwt:     manager,
		audit:   recorder,
		limiter: newLoginLimiter(),
		now:     time.Now,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	meta := requestmeta.From(ctx)
	email = strings.ToLower(strings.TrimSpace(email))

	if !s.limiter.allow("ip:"+meta.IP, 10, time.Minute) ||
		!s.limiter.allow("email:"+email, 10, time.Hour) {
		s.auditFailure(ctx, "", email, "rate_limited")
		return nil, ErrRateLimited
	}

	user, err := s.store.Queries().UserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		// Pay the hashing cost for unknown users too.
		_, _ = VerifyPassword(DummyHash(), password)
		s.auditFailure(ctx, "", email, "unknown_user")
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	locked, err := s.store.Queries().IsLocked(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if locked {
		s.auditFailure(ctx, user.ID, email, "locked")
		return nil, ErrInvalidCredentials
	}

	ok, err := VerifyPassword(user.PasswordHash, password)
	if err != nil || !ok {
		_ = s.store.Queries().RecordLoginFailure(ctx, user.ID)
		s.auditFailure(ctx, user.ID, email, "bad_password")
		return nil, ErrInvalidCredentials
	}
	if user.Status != "active" {
		s.auditFailure(ctx, user.ID, email, "inactive")
		return nil, ErrInvalidCredentials
	}
	_ = s.store.Queries().ResetLoginFailures(ctx, user.ID)

	permissions, err := s.store.Queries().PermissionsForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roles, err := s.store.Queries().RolesForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	expiresAt := s.now().Add(s.jwt.TTL())
	var sessionID string
	err = s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		sessionID, err = q.CreateSession(ctx, user.ID, meta.UserAgent, meta.IP, expiresAt)
		if err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "auth.login",
			ResourceType: "session",
			ResourceID:   sessionID,
			Result:       "success",
			ActorID:      user.ID,
			ActorEmail:   user.Email,
		})
	})
	if err != nil {
		return nil, err
	}

	token, _, err := s.jwt.Sign(user.ID, sessionID, user.TokenVersion)
	if err != nil {
		_ = s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
			return q.RevokeSession(ctx, sessionID, "token_sign_failed")
		})
		return nil, err
	}

	_ = s.store.Queries().UpdateLastLogin(ctx, user.ID)

	// A stale hash (e.g. after raising Argon2 parameters) is upgraded on the
	// next successful login without changing the credential.
	if NeedsRehash(user.PasswordHash) {
		if hash, err := HashPassword(password); err == nil {
			_ = s.store.Queries().UpdatePasswordHash(ctx, user.ID, hash)
		}
	}

	return &LoginResult{
		User:        user,
		Permissions: permissions,
		Roles:       roles,
		AccessToken: token,
		ExpiresIn:   int64(s.jwt.TTL().Seconds()),
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *Service) ValidateSession(ctx context.Context, sessionID, userID string, tokenVersion int) (authpkg.Principal, error) {
	user, err := s.store.Queries().ValidateSession(ctx, sessionID, userID, tokenVersion)
	if errors.Is(err, store.ErrNotFound) {
		return authpkg.Principal{}, authpkg.ErrUnauthenticated
	}
	if err != nil {
		return authpkg.Principal{}, err
	}
	permissions, err := s.store.Queries().PermissionsForUser(ctx, user.ID)
	if err != nil {
		return authpkg.Principal{}, err
	}
	principal := authpkg.NewPrincipal(user.ID, sessionID, user.TokenVersion, permissions)
	principal.Email = user.Email
	return principal, nil
}

func (s *Service) Logout(ctx context.Context, principal authpkg.Principal) error {
	return s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.RevokeSession(ctx, principal.SessionID, "logout"); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "auth.logout",
			ResourceType: "session",
			ResourceID:   principal.SessionID,
			Result:       "success",
			ActorID:      principal.UserID,
			ActorEmail:   principal.Email,
		})
	})
}

func (s *Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.store.Queries().UserByID(ctx, userID)
	if err != nil {
		return err
	}
	ok, err := VerifyPassword(user.PasswordHash, oldPassword)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.UpdatePassword(ctx, userID, hash); err != nil {
			return err
		}
		if err := q.RevokeUserSessions(ctx, userID, "password_changed"); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "auth.password_change",
			ResourceType: "admin_user",
			ResourceID:   userID,
			Result:       "success",
			ActorID:      userID,
			ActorEmail:   user.Email,
		})
	})
}

func (s *Service) auditFailure(ctx context.Context, userID, email, reason string) {
	s.audit.RecordBestEffort(ctx, audit.Event{
		Action:       "auth.login_failed",
		ResourceType: "admin_user",
		Result:       "failure",
		ActorID:      userID,
		ActorEmail:   email,
		Metadata:     map[string]string{"reason": reason},
	})
}

type loginLimiter struct {
	mu      sync.Mutex
	windows map[string]limitWindow
}

type limitWindow struct {
	count int
	reset time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{windows: make(map[string]limitWindow)}
}

func (l *loginLimiter) allow(key string, limit int, period time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if len(l.windows) > 10000 {
		l.windows = make(map[string]limitWindow)
	}
	w, ok := l.windows[key]
	if !ok || now.After(w.reset) {
		w = limitWindow{reset: now.Add(period)}
	}
	w.count++
	l.windows[key] = w
	return w.count <= limit
}
