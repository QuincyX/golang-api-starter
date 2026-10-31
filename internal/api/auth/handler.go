package authapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/router"
	authsvc "github.com/example/homebrew-api-starter/internal/service/auth"
)

type Service interface {
	Login(ctx context.Context, email, password string) (*authsvc.LoginResult, error)
	Logout(ctx context.Context, principal auth.Principal) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

type LoginInput struct {
	Body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
}

type LoginUserDTO struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	DisplayName        string `json:"displayName"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

type LoginOutput struct {
	Body struct {
		AccessToken string       `json:"accessToken"`
		TokenType   string       `json:"tokenType"`
		ExpiresIn   int64        `json:"expiresIn"`
		User        LoginUserDTO `json:"user"`
		Permissions []string     `json:"permissions"`
		Roles       []string     `json:"roles"`
	}
}

type MeOutput struct {
	Body struct {
		UserID       string   `json:"userId"`
		SessionID    string   `json:"sessionId"`
		TokenVersion int      `json:"tokenVersion"`
		Permissions  []string `json:"permissions"`
	}
}

type PasswordInput struct {
	Body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
}

type OKOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func (h *Handler) Register(api huma.API, reg *router.Registry) error {
	if err := router.Register(reg, api, router.Route{
		Method:      http.MethodPost,
		Path:        "/api/admin/v1/auth/login",
		OperationID: "auth.login",
		Summary:     "Log in with email and password",
		Tags:        []string{"auth"},
		Public:      true,
		AuditAction: "auth.login",
	}, h.login); err != nil {
		return err
	}

	if err := router.Register(reg, api, router.Route{
		Method:      http.MethodGet,
		Path:        "/api/admin/v1/auth/me",
		OperationID: "auth.me",
		Summary:     "Current administrator and effective permissions",
		Tags:        []string{"auth"},
	}, h.me); err != nil {
		return err
	}

	if err := router.Register(reg, api, router.Route{
		Method:      http.MethodPost,
		Path:        "/api/admin/v1/auth/logout",
		OperationID: "auth.logout",
		Summary:     "Log out the current session",
		Tags:        []string{"auth"},
		AuditAction: "auth.logout",
	}, h.logout); err != nil {
		return err
	}

	return router.Register(reg, api, router.Route{
		Method:      http.MethodPut,
		Path:        "/api/admin/v1/auth/password",
		OperationID: "auth.password",
		Summary:     "Change the current administrator password",
		Tags:        []string{"auth"},
		AuditAction: "auth.password_change",
	}, h.password)
}

func (h *Handler) login(ctx context.Context, in *LoginInput) (*LoginOutput, error) {
	if in.Body.Email == "" || in.Body.Password == "" {
		return nil, problem(http.StatusUnprocessableEntity, "Validation failed", "email and password are required")
	}

	result, err := h.svc.Login(ctx, in.Body.Email, in.Body.Password)
	if errors.Is(err, authsvc.ErrRateLimited) {
		return nil, problem(http.StatusTooManyRequests, "Too Many Requests", "too many login attempts, try again later")
	}
	if errors.Is(err, authsvc.ErrInvalidCredentials) {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "invalid email or password")
	}
	if err != nil {
		return nil, problem(http.StatusInternalServerError, "Internal Server Error", "login failed")
	}

	out := &LoginOutput{}
	out.Body.AccessToken = result.AccessToken
	out.Body.TokenType = "Bearer"
	out.Body.ExpiresIn = result.ExpiresIn
	out.Body.User = LoginUserDTO{
		ID:                 result.User.ID,
		Email:              result.User.Email,
		DisplayName:        result.User.DisplayName,
		MustChangePassword: result.User.MustChangePassword,
	}
	out.Body.Permissions = result.Permissions
	out.Body.Roles = result.Roles
	return out, nil
}

func (h *Handler) me(ctx context.Context, _ *struct{}) (*MeOutput, error) {
	principal, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}

	out := &MeOutput{}
	out.Body.UserID = principal.UserID
	out.Body.SessionID = principal.SessionID
	out.Body.TokenVersion = principal.TokenVersion
	out.Body.Permissions = principal.Permissions()
	return out, nil
}

func (h *Handler) logout(ctx context.Context, _ *struct{}) (*OKOutput, error) {
	principal, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}
	if err := h.svc.Logout(ctx, principal); err != nil {
		return nil, problem(http.StatusInternalServerError, "Internal Server Error", "logout failed")
	}
	out := &OKOutput{}
	out.Body.OK = true
	return out, nil
}

func (h *Handler) password(ctx context.Context, in *PasswordInput) (*OKOutput, error) {
	principal, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}
	if len(in.Body.NewPassword) < 8 {
		return nil, problem(http.StatusUnprocessableEntity, "Validation failed", "new password must be at least 8 characters")
	}
	if err := h.svc.ChangePassword(ctx, principal.UserID, in.Body.CurrentPassword, in.Body.NewPassword); err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) {
			return nil, problem(http.StatusUnauthorized, "Unauthorized", "current password is incorrect")
		}
		return nil, problem(http.StatusInternalServerError, "Internal Server Error", "password change failed")
	}
	out := &OKOutput{}
	out.Body.OK = true
	return out, nil
}

func problem(status int, title, detail string) *huma.ErrorModel {
	return &huma.ErrorModel{Type: "about:blank", Title: title, Status: status, Detail: detail}
}
