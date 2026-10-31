package articleapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/example/homebrew-api-starter/internal/router"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

type ListInput struct {
	Page     int `query:"page"`
	PageSize int `query:"pageSize"`
}

type ListOutput struct {
	Body struct {
		Items    []any `json:"items"`
		Page     int   `json:"page"`
		PageSize int   `json:"pageSize"`
		Total    int64 `json:"total"`
	}
}

func (h *Handler) Register(api huma.API, reg *router.Registry) error {
	return router.Register(reg, api, router.Route{
		Method:      http.MethodGet,
		Path:        "/api/admin/v1/articles",
		OperationID: "article.list",
		Summary:     "List articles",
		Tags:        []string{"article"},
		Permission:  "article:read",
	}, h.list)
}

func (h *Handler) list(_ context.Context, in *ListInput) (*ListOutput, error) {
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	out := &ListOutput{}
	out.Body.Items = []any{}
	out.Body.Page = page
	out.Body.PageSize = pageSize
	out.Body.Total = 0
	return out, nil
}
