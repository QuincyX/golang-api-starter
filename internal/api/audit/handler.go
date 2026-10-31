package auditapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/router"
)

type Service interface {
	List(ctx context.Context, limit int, cursor int64, action, actorID string) ([]model.AuditLog, int64, error)
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

type ListInput struct {
	Limit   int    `query:"limit"`
	Cursor  int64  `query:"cursor"`
	Action  string `query:"action"`
	ActorID string `query:"actorId"`
}

type AuditDTO struct {
	ID           int64     `json:"id"`
	OccurredAt   time.Time `json:"occurredAt"`
	RequestID    string    `json:"requestId"`
	TraceID      string    `json:"traceId,omitempty"`
	ActorID      *string   `json:"actorId"`
	ActorEmail   *string   `json:"actorEmail"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resourceType"`
	ResourceID   *string   `json:"resourceId"`
	Result       string    `json:"result"`
	StatusCode   *int      `json:"statusCode"`
	IP           string    `json:"ip,omitempty"`
	UserAgent    *string   `json:"userAgent,omitempty"`
	Changes      any       `json:"changes,omitempty"`
	Metadata     any       `json:"metadata,omitempty"`
	DurationMS   *int      `json:"durationMs"`
}

type ListOutput struct {
	Body struct {
		Items      []AuditDTO `json:"items"`
		NextCursor int64      `json:"nextCursor,omitempty"`
	}
}

func (h *Handler) Register(api huma.API, reg *router.Registry) error {
	return router.Register(reg, api, router.Route{
		Method:      http.MethodGet,
		Path:        "/api/admin/v1/audit-logs",
		OperationID: "audit.list",
		Summary:     "List operation audit logs",
		Tags:        []string{"audit"},
		Permission:  "audit:read",
	}, h.list)
}

func (h *Handler) list(ctx context.Context, in *ListInput) (*ListOutput, error) {
	items, next, err := h.svc.List(ctx, in.Limit, in.Cursor, in.Action, in.ActorID)
	if err != nil {
		return nil, &huma.ErrorModel{
			Type: "about:blank", Title: "Internal Server Error",
			Status: http.StatusInternalServerError, Detail: "list audit logs failed",
		}
	}

	out := &ListOutput{}
	out.Body.Items = make([]AuditDTO, 0, len(items))
	for i := range items {
		out.Body.Items = append(out.Body.Items, toDTO(&items[i]))
	}
	out.Body.NextCursor = next
	return out, nil
}

func toDTO(a *model.AuditLog) AuditDTO {
	dto := AuditDTO{
		ID: a.ID, OccurredAt: a.OccurredAt, RequestID: a.RequestID, TraceID: a.TraceID,
		ActorID: a.ActorID, ActorEmail: a.ActorEmail, Action: a.Action,
		ResourceType: a.ResourceType, ResourceID: a.ResourceID, Result: a.Result,
		StatusCode: a.StatusCode, UserAgent: a.UserAgent, DurationMS: a.DurationMS,
	}
	if a.IP != nil {
		dto.IP = *a.IP
	}
	if len(a.Changes) > 0 {
		_ = json.Unmarshal(a.Changes, &dto.Changes)
	}
	if len(a.Metadata) > 0 {
		_ = json.Unmarshal(a.Metadata, &dto.Metadata)
	}
	return dto
}
