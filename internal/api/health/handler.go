package health

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/homebrew-api-starter/internal/router"
)

type Handler struct {
	db      *pgxpool.Pool
	version string
}

func New(db *pgxpool.Pool, version string) *Handler {
	return &Handler{db: db, version: version}
}

type HealthOutput struct {
	Body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
}

func (h *Handler) Register(api huma.API, reg *router.Registry) error {
	if err := router.Register(reg, api, router.Route{
		Method:      http.MethodGet,
		Path:        "/healthz",
		OperationID: "system.health",
		Summary:     "Liveness probe",
		Tags:        []string{"system"},
		Public:      true,
	}, h.health); err != nil {
		return err
	}

	return router.Register(reg, api, router.Route{
		Method:      http.MethodGet,
		Path:        "/readyz",
		OperationID: "system.ready",
		Summary:     "Readiness probe",
		Tags:        []string{"system"},
		Public:      true,
	}, h.ready)
}

func (h *Handler) health(context.Context, *struct{}) (*HealthOutput, error) {
	out := &HealthOutput{}
	out.Body.Status = "ok"
	out.Body.Version = h.version
	return out, nil
}

func (h *Handler) ready(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
	if h.db == nil {
		return nil, &huma.ErrorModel{
			Status: http.StatusServiceUnavailable,
			Title:  "Service Unavailable",
			Detail: "database is not configured",
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := h.db.Ping(pingCtx); err != nil {
		return nil, &huma.ErrorModel{
			Status: http.StatusServiceUnavailable,
			Title:  "Service Unavailable",
			Detail: "database is not reachable",
		}
	}

	out := &HealthOutput{}
	out.Body.Status = "ready"
	out.Body.Version = h.version
	return out, nil
}
