package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/example/homebrew-api-starter/internal/audit"
	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/model"
	authsvc "github.com/example/homebrew-api-starter/internal/service/auth"
	"github.com/example/homebrew-api-starter/internal/store"
)

var (
	ErrInvalidToken  = errors.New("invalid or expired token")
	ErrInvalidStatus = errors.New("invalid status")
	ErrWeakPassword  = errors.New("password must be at least 8 characters")
)

type Service struct {
	store   *store.Store
	audit   *audit.Recorder
	baseURL string
	now     func() time.Time
}

func NewService(st *store.Store, recorder *audit.Recorder, baseURL string) *Service {
	return &Service{store: st, audit: recorder, baseURL: strings.TrimRight(baseURL, "/"), now: time.Now}
}

type CreateInput struct {
	Email       string
	DisplayName string
	Roles       []string
}

type CreateResult struct {
	User          *model.User
	ActivationURL string
}

type ListResult struct {
	Items    []model.User
	Total    int64
	Page     int
	PageSize int
}

func (s *Service) Create(ctx context.Context, actor authpkg.Principal, in CreateInput) (*CreateResult, error) {
	placeholder, err := randomToken()
	if err != nil {
		return nil, err
	}
	hash, err := authsvc.HashPassword(placeholder)
	if err != nil {
		return nil, err
	}
	roleIDs, err := s.store.Queries().RoleIDsByCodes(ctx, in.Roles)
	if err != nil {
		return nil, err
	}

	rawToken, err := randomToken()
	if err != nil {
		return nil, err
	}
	tokenHash := hashToken(rawToken)

	var created *model.User
	err = s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		created, err = q.CreateUser(ctx, in.Email, hash, in.DisplayName, true)
		if err != nil {
			return err
		}
		if err := q.SetUserRoles(ctx, created.ID, roleIDs); err != nil {
			return err
		}
		if err := q.CreatePasswordToken(ctx, created.ID, tokenHash, s.now().Add(30*time.Minute)); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "account.create",
			ResourceType: "admin_user",
			ResourceID:   created.ID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Metadata:     map[string]any{"email": in.Email, "roles": in.Roles},
		})
	})
	if err != nil {
		return nil, err
	}

	created.Roles = append([]string(nil), in.Roles...)
	return &CreateResult{User: created, ActivationURL: s.activationURL(rawToken)}, nil
}

func (s *Service) List(ctx context.Context, query string, page, pageSize int) (*ListResult, error) {
	page, pageSize = normalizePage(page, pageSize)
	users, total, err := s.store.Queries().ListUsers(ctx, query, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	for i := range users {
		roles, err := s.store.Queries().RolesForUser(ctx, users[i].ID)
		if err != nil {
			return nil, err
		}
		users[i].Permissions = nil
		users[i].Roles = roles
	}
	return &ListResult{Items: users, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) SetStatus(ctx context.Context, actor authpkg.Principal, userID, status string) error {
	if status != "active" && status != "disabled" {
		return ErrInvalidStatus
	}
	user, err := s.store.Queries().UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if status != "active" {
		isSuper, err := s.store.Queries().HasRole(ctx, userID, "super_admin")
		if err != nil {
			return err
		}
		if isSuper {
			count, err := s.store.Queries().CountActiveSuperAdmins(ctx)
			if err != nil {
				return err
			}
			if count <= 1 {
				return store.ErrLastSuperAdmin
			}
		}
	}

	return s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.SetUserStatus(ctx, userID, status); err != nil {
			return err
		}
		if status != "active" {
			if err := q.RevokeUserSessions(ctx, userID, "account_disabled"); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "account.disable",
			ResourceType: "admin_user",
			ResourceID:   userID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Metadata:     map[string]any{"status": status, "email": user.Email},
		})
	})
}

func (s *Service) SetRoles(ctx context.Context, actor authpkg.Principal, userID string, roleCodes []string) error {
	roleIDs, err := s.store.Queries().RoleIDsByCodes(ctx, roleCodes)
	if err != nil {
		return err
	}
	isSuper, err := s.store.Queries().HasRole(ctx, userID, "super_admin")
	if err != nil {
		return err
	}
	willBeSuper := false
	for _, code := range roleCodes {
		if code == "super_admin" {
			willBeSuper = true
			break
		}
	}
	if isSuper && !willBeSuper {
		count, err := s.store.Queries().CountActiveSuperAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return store.ErrLastSuperAdmin
		}
	}

	return s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.SetUserRoles(ctx, userID, roleIDs); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "account.role_update",
			ResourceType: "admin_user",
			ResourceID:   userID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Metadata:     map[string]any{"roles": roleCodes},
		})
	})
}

func (s *Service) ResetPassword(ctx context.Context, actor authpkg.Principal, userID string) (string, error) {
	user, err := s.store.Queries().UserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	rawToken, err := randomToken()
	if err != nil {
		return "", err
	}
	return s.activationURL(rawToken), s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.CreatePasswordToken(ctx, userID, hashToken(rawToken), s.now().Add(30*time.Minute)); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "account.reset_password",
			ResourceType: "admin_user",
			ResourceID:   userID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Metadata:     map[string]any{"email": user.Email},
		})
	})
}

func (s *Service) Activate(ctx context.Context, rawToken, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrInvalidToken
	}
	var userID string
	err := s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		id, err := q.ConsumePasswordToken(ctx, hashToken(rawToken))
		if errors.Is(err, store.ErrNotFound) {
			return ErrInvalidToken
		}
		if err != nil {
			return err
		}
		userID = id
		hash, err := authsvc.HashPassword(newPassword)
		if err != nil {
			return err
		}
		if err := q.UpdatePassword(ctx, userID, hash); err != nil {
			return err
		}
		if err := q.RevokeUserSessions(ctx, userID, "password_activated"); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "auth.activate",
			ResourceType: "admin_user",
			ResourceID:   userID,
			Result:       "success",
			ActorID:      userID,
		})
	})
	return err
}

func (s *Service) Sessions(ctx context.Context, userID string) ([]model.Session, error) {
	return s.store.Queries().ListSessions(ctx, userID)
}

func (s *Service) RevokeSession(ctx context.Context, actor authpkg.Principal, userID, sessionID string) error {
	return s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.RevokeSession(ctx, sessionID, "admin_revoked"); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "account.session_revoke",
			ResourceType: "session",
			ResourceID:   sessionID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Metadata:     map[string]any{"user_id": userID},
		})
	})
}

func (s *Service) Roles(ctx context.Context) ([]map[string]any, error) {
	return s.store.Queries().ListRoles(ctx)
}

func (s *Service) Permissions(ctx context.Context) ([]map[string]any, error) {
	return s.store.Queries().ListPermissions(ctx)
}

func (s *Service) Seed(ctx context.Context, email, password, displayName string) (bool, error) {
	if len(password) < 8 {
		return false, ErrWeakPassword
	}
	count, err := s.store.Queries().CountUsers(ctx)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	hash, err := authsvc.HashPassword(password)
	if err != nil {
		return false, err
	}
	roleIDs, err := s.store.Queries().RoleIDsByCodes(ctx, []string{"super_admin"})
	if err != nil {
		return false, err
	}
	return true, s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		user, err := q.CreateUser(ctx, email, hash, displayName, false)
		if err != nil {
			return err
		}
		if err := q.SetUserRoles(ctx, user.ID, roleIDs); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "account.create",
			ResourceType: "admin_user",
			ResourceID:   user.ID,
			Result:       "success",
			Metadata:     map[string]any{"email": email, "seed": true},
		})
	})
}

func (s *Service) activationURL(rawToken string) string {
	return s.baseURL + "/activate?token=" + url.QueryEscape(rawToken)
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
