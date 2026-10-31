package accountapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/router"
	accountsvc "github.com/example/homebrew-api-starter/internal/service/account"
)

type Service interface {
	List(ctx context.Context, query string, page, pageSize int) (*accountsvc.ListResult, error)
	Create(ctx context.Context, actor authpkg.Principal, in accountsvc.CreateInput) (*accountsvc.CreateResult, error)
	SetStatus(ctx context.Context, actor authpkg.Principal, userID, status string) error
	SetRoles(ctx context.Context, actor authpkg.Principal, userID string, roleCodes []string) error
	ResetPassword(ctx context.Context, actor authpkg.Principal, userID string) (string, error)
	Sessions(ctx context.Context, userID string) ([]model.Session, error)
	RevokeSession(ctx context.Context, actor authpkg.Principal, userID, sessionID string) error
	Roles(ctx context.Context) ([]map[string]any, error)
	Permissions(ctx context.Context) ([]map[string]any, error)
	Activate(ctx context.Context, rawToken, newPassword string) error
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

type AccountUserDTO struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	DisplayName        string     `json:"displayName"`
	Status             string     `json:"status"`
	MustChangePassword bool       `json:"mustChangePassword"`
	Roles              []string   `json:"roles"`
	LastLoginAt        *time.Time `json:"lastLoginAt"`
	CreatedAt          time.Time  `json:"createdAt"`
}

type ListInput struct {
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
	Query    string `query:"q"`
}

type ListOutput struct {
	Body struct {
		Items    []AccountUserDTO `json:"items"`
		Page     int              `json:"page"`
		PageSize int              `json:"pageSize"`
		Total    int64            `json:"total"`
	}
}

type CreateInput struct {
	Body struct {
		Email       string   `json:"email"`
		DisplayName string   `json:"displayName"`
		Roles       []string `json:"roles" required:"false"`
	}
}

type CreateOutput struct {
	Body struct {
		User          AccountUserDTO `json:"user"`
		ActivationURL string         `json:"activationUrl"`
	}
}

type GetInput struct {
	ID string `path:"id"`
}

type UpdateInput struct {
	ID   string `path:"id"`
	Body struct {
		Status string `json:"status"`
	}
}

type RolesInput struct {
	ID   string `path:"id"`
	Body struct {
		Roles []string `json:"roles"`
	}
}

type ResetOutput struct {
	Body struct {
		ActivationURL string `json:"activationUrl"`
	}
}

type SessionsOutput struct {
	Body struct {
		Items []SessionDTO `json:"items"`
	}
}

type SessionDTO struct {
	ID        string     `json:"id"`
	UserAgent string     `json:"userAgent"`
	IP        string     `json:"ip"`
	CreatedAt time.Time  `json:"createdAt"`
	LastSeen  time.Time  `json:"lastSeenAt"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt"`
}

type ActivateInput struct {
	Body struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
}

type OKOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func (h *Handler) Register(api huma.API, reg *router.Registry) error {
	routes := []struct {
		route   router.Route
		handler any
	}{
		{router.Route{Method: http.MethodGet, Path: "/api/admin/v1/accounts", OperationID: "account.list", Tags: []string{"account"}, Permission: "account:read"}, h.list},
		{router.Route{Method: http.MethodPost, Path: "/api/admin/v1/accounts", OperationID: "account.create", Tags: []string{"account"}, Permission: "account:write", AuditAction: "account.create"}, h.create},
		{router.Route{Method: http.MethodPatch, Path: "/api/admin/v1/accounts/{id}", OperationID: "account.update", Tags: []string{"account"}, Permission: "account:write", AuditAction: "account.update"}, h.update},
		{router.Route{Method: http.MethodPut, Path: "/api/admin/v1/accounts/{id}/roles", OperationID: "account.roles", Tags: []string{"account"}, Permission: "role:assign", AuditAction: "account.role_update"}, h.roles},
		{router.Route{Method: http.MethodPost, Path: "/api/admin/v1/accounts/{id}/reset-password", OperationID: "account.reset_password", Tags: []string{"account"}, Permission: "account:write", AuditAction: "account.reset_password"}, h.reset},
		{router.Route{Method: http.MethodGet, Path: "/api/admin/v1/accounts/{id}/sessions", OperationID: "account.sessions", Tags: []string{"account"}, Permission: "account:read"}, h.sessions},
		{router.Route{Method: http.MethodDelete, Path: "/api/admin/v1/accounts/{id}/sessions/{sid}", OperationID: "account.session_revoke", Tags: []string{"account"}, Permission: "account:write", AuditAction: "account.session_revoke"}, h.revokeSession},
		{router.Route{Method: http.MethodGet, Path: "/api/admin/v1/roles", OperationID: "role.list", Tags: []string{"account"}, Permission: "role:read"}, h.rolesList},
		{router.Route{Method: http.MethodGet, Path: "/api/admin/v1/permissions", OperationID: "permission.list", Tags: []string{"account"}, Permission: "role:read"}, h.permissions},
		{router.Route{Method: http.MethodPost, Path: "/api/admin/v1/auth/activate", OperationID: "auth.activate", Tags: []string{"auth"}, Public: true, AuditAction: "auth.activate"}, h.activate},
	}
	for _, item := range routes {
		if err := register(reg, api, item.route, item.handler); err != nil {
			return err
		}
	}
	return nil
}

func register[T any](reg *router.Registry, api huma.API, route router.Route, handler T) error {
	switch h := any(handler).(type) {
	case func(context.Context, *ListInput) (*ListOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *CreateInput) (*CreateOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *UpdateInput) (*OKOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *RolesInput) (*OKOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *GetInput) (*ResetOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *GetInput) (*SessionsOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *SessionRevokeInput) (*OKOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *struct{}) (*RolesOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *struct{}) (*PermissionsOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *ActivateInput) (*OKOutput, error):
		return router.Register(reg, api, route, h)
	default:
		panic("unsupported account handler type")
	}
}

type SessionRevokeInput struct {
	ID  string `path:"id"`
	SID string `path:"sid"`
}

type RolesOutput struct {
	Body struct {
		Items []map[string]any `json:"items"`
	}
}

type PermissionsOutput struct {
	Body struct {
		Items []map[string]any `json:"items"`
	}
}

func (h *Handler) list(ctx context.Context, in *ListInput) (*ListOutput, error) {
	result, err := h.svc.List(ctx, in.Query, in.Page, in.PageSize)
	if err != nil {
		return nil, problem(http.StatusInternalServerError, "list accounts failed")
	}
	out := &ListOutput{}
	out.Body.Items = make([]AccountUserDTO, 0, len(result.Items))
	for i := range result.Items {
		out.Body.Items = append(out.Body.Items, toAccountUserDTO(&result.Items[i]))
	}
	out.Body.Page = result.Page
	out.Body.PageSize = result.PageSize
	out.Body.Total = result.Total
	return out, nil
}

func (h *Handler) create(ctx context.Context, in *CreateInput) (*CreateOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "authentication required")
	}
	if in.Body.Email == "" || in.Body.DisplayName == "" {
		return nil, problem(http.StatusUnprocessableEntity, "email and displayName are required")
	}
	result, err := h.svc.Create(ctx, actor, accountsvc.CreateInput{
		Email: in.Body.Email, DisplayName: in.Body.DisplayName, Roles: in.Body.Roles,
	})
	if err != nil {
		return nil, accountError(err)
	}
	out := &CreateOutput{}
	out.Body.User = toAccountUserDTO(result.User)
	out.Body.ActivationURL = result.ActivationURL
	return out, nil
}

func (h *Handler) update(ctx context.Context, in *UpdateInput) (*OKOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "authentication required")
	}
	if err := h.svc.SetStatus(ctx, actor, in.ID, in.Body.Status); err != nil {
		return nil, accountError(err)
	}
	return okOutput(), nil
}

func (h *Handler) roles(ctx context.Context, in *RolesInput) (*OKOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "authentication required")
	}
	if err := h.svc.SetRoles(ctx, actor, in.ID, in.Body.Roles); err != nil {
		return nil, accountError(err)
	}
	return okOutput(), nil
}

func (h *Handler) reset(ctx context.Context, in *GetInput) (*ResetOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "authentication required")
	}
	url, err := h.svc.ResetPassword(ctx, actor, in.ID)
	if err != nil {
		return nil, accountError(err)
	}
	out := &ResetOutput{}
	out.Body.ActivationURL = url
	return out, nil
}

func (h *Handler) sessions(ctx context.Context, in *GetInput) (*SessionsOutput, error) {
	items, err := h.svc.Sessions(ctx, in.ID)
	if err != nil {
		return nil, accountError(err)
	}
	out := &SessionsOutput{}
	out.Body.Items = make([]SessionDTO, 0, len(items))
	for _, session := range items {
		out.Body.Items = append(out.Body.Items, SessionDTO{
			ID: session.ID, UserAgent: session.UserAgent, IP: session.IP,
			CreatedAt: session.CreatedAt, LastSeen: session.LastSeenAt,
			ExpiresAt: session.ExpiresAt, RevokedAt: session.RevokedAt,
		})
	}
	return out, nil
}

func (h *Handler) revokeSession(ctx context.Context, in *SessionRevokeInput) (*OKOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "authentication required")
	}
	if err := h.svc.RevokeSession(ctx, actor, in.ID, in.SID); err != nil {
		return nil, accountError(err)
	}
	return okOutput(), nil
}

func (h *Handler) rolesList(ctx context.Context, _ *struct{}) (*RolesOutput, error) {
	items, err := h.svc.Roles(ctx)
	if err != nil {
		return nil, problem(http.StatusInternalServerError, "list roles failed")
	}
	out := &RolesOutput{}
	out.Body.Items = items
	return out, nil
}

func (h *Handler) permissions(ctx context.Context, _ *struct{}) (*PermissionsOutput, error) {
	items, err := h.svc.Permissions(ctx)
	if err != nil {
		return nil, problem(http.StatusInternalServerError, "list permissions failed")
	}
	out := &PermissionsOutput{}
	out.Body.Items = items
	return out, nil
}

func (h *Handler) activate(ctx context.Context, in *ActivateInput) (*OKOutput, error) {
	if in.Body.Token == "" || len(in.Body.NewPassword) < 8 {
		return nil, problem(http.StatusUnprocessableEntity, "token and a password of at least 8 characters are required")
	}
	if err := h.svc.Activate(ctx, in.Body.Token, in.Body.NewPassword); err != nil {
		if errors.Is(err, accountsvc.ErrInvalidToken) {
			return nil, problem(http.StatusBadRequest, "invalid or expired activation token")
		}
		return nil, problem(http.StatusInternalServerError, "activation failed")
	}
	return okOutput(), nil
}

func toAccountUserDTO(u *model.User) AccountUserDTO {
	if u == nil {
		return AccountUserDTO{}
	}
	return AccountUserDTO{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Status: u.Status,
		MustChangePassword: u.MustChangePassword, Roles: u.Roles,
		LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt,
	}
}

func okOutput() *OKOutput {
	out := &OKOutput{}
	out.Body.OK = true
	return out
}

func accountError(err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return problem(http.StatusNotFound, "account not found")
	case errors.Is(err, model.ErrConflict):
		return problem(http.StatusConflict, "email is already in use")
	case errors.Is(err, model.ErrLastSuperAdmin):
		return problem(http.StatusConflict, "cannot disable or demote the last super admin")
	case errors.Is(err, accountsvc.ErrInvalidStatus):
		return problem(http.StatusUnprocessableEntity, "invalid status")
	default:
		return problem(http.StatusInternalServerError, "account operation failed")
	}
}

func problem(status int, detail string) *huma.ErrorModel {
	return &huma.ErrorModel{Type: "about:blank", Title: http.StatusText(status), Status: status, Detail: detail}
}
