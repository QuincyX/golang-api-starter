package authapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/router"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

type MeOutput struct {
	Body struct {
		UserID       string   `json:"userId"`
		SessionID    string   `json:"sessionId"`
		TokenVersion int      `json:"tokenVersion"`
		Permissions  []string `json:"permissions"`
	}
}

type LogoutOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func (h *Handler) Register(api huma.API, reg *router.Registry) error {
	if err := router.Register(reg, api, router.Route{
		Method:      http.MethodGet,
		Path:        "/api/admin/v1/auth/me",
		OperationID: "auth.me",
		Summary:     "Current administrator and effective permissions",
		Tags:        []string{"auth"},
	}, h.me); err != nil {
		return err
	}

	return router.Register(reg, api, router.Route{
		Method:      http.MethodPost,
		Path:        "/api/admin/v1/auth/logout",
		OperationID: "auth.logout",
		Summary:     "Log out the current session",
		Tags:        []string{"auth"},
		AuditAction: "auth.logout",
	}, h.logout)
}

func (h *Handler) me(ctx context.Context, _ *struct{}) (*MeOutput, error) {
	principal, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return nil, &huma.ErrorModel{
			Status: http.StatusUnauthorized,
			Title:  "Unauthorized",
			Detail: "authentication required",
		}
	}

	out := &MeOutput{}
	out.Body.UserID = principal.UserID
	out.Body.SessionID = principal.SessionID
	out.Body.TokenVersion = principal.TokenVersion
	out.Body.Permissions = principal.Permissions()
	return out, nil
}

func (h *Handler) logout(context.Context, *struct{}) (*LogoutOutput, error) {
	// P1: revoke the session row and record an audit event in the same
	// transaction. P0 only exercises the route contract and JWT middleware.
	out := &LogoutOutput{}
	out.Body.OK = true
	return out, nil
}
