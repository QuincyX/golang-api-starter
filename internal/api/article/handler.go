package articleapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/router"
	articlesvc "github.com/example/homebrew-api-starter/internal/service/article"
)

type Service interface {
	List(ctx context.Context, status, query string, page, pageSize int) (*articlesvc.ListResult, error)
	Get(ctx context.Context, id string) (*model.Article, error)
	Create(ctx context.Context, actor authpkg.Principal, in articlesvc.CreateInput) (*model.Article, error)
	Update(ctx context.Context, actor authpkg.Principal, in articlesvc.UpdateInput) (*model.Article, error)
	Publish(ctx context.Context, actor authpkg.Principal, id string) (*model.Article, error)
	Unpublish(ctx context.Context, actor authpkg.Principal, id string) (*model.Article, error)
	Delete(ctx context.Context, actor authpkg.Principal, id string) error
	Revisions(ctx context.Context, id string) ([]model.ArticleRevision, error)
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

type ArticleDTO struct {
	ID          string     `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	ContentMD   string     `json:"contentMd"`
	ContentHTML string     `json:"contentHtml"`
	CoverURL    string     `json:"coverUrl"`
	CategoryID  *string    `json:"categoryId"`
	AuthorID    string     `json:"authorId"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"publishedAt"`
	Version     int        `json:"version"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type ListInput struct {
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
	Status   string `query:"status"`
	Query    string `query:"q"`
}

type ListOutput struct {
	Body struct {
		Items    []ArticleDTO `json:"items"`
		Page     int          `json:"page"`
		PageSize int          `json:"pageSize"`
		Total    int64        `json:"total"`
	}
}

type GetInput struct {
	ID string `path:"id"`
}

type CreateInput struct {
	Body struct {
		Slug       string  `json:"slug" required:"false"`
		Title      string  `json:"title"`
		Summary    string  `json:"summary" required:"false"`
		ContentMD  string  `json:"contentMd"`
		CoverURL   string  `json:"coverUrl" required:"false"`
		CategoryID *string `json:"categoryId" required:"false"`
	}
}

type UpdateInput struct {
	ID   string `path:"id"`
	Body struct {
		Version    int     `json:"version"`
		Slug       string  `json:"slug" required:"false"`
		Title      string  `json:"title"`
		Summary    string  `json:"summary" required:"false"`
		ContentMD  string  `json:"contentMd"`
		CoverURL   string  `json:"coverUrl" required:"false"`
		CategoryID *string `json:"categoryId" required:"false"`
	}
}

type ArticleOutput struct {
	Body ArticleDTO
}

type RevisionsOutput struct {
	Body struct {
		Items []RevisionDTO `json:"items"`
	}
}

type RevisionDTO struct {
	ID         int64     `json:"id"`
	Version    int       `json:"version"`
	Title      string    `json:"title"`
	ContentMD  string    `json:"contentMd"`
	EditorID   string    `json:"editorId"`
	ChangeNote string    `json:"changeNote"`
	CreatedAt  time.Time `json:"createdAt"`
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
		{router.Route{Method: http.MethodGet, Path: "/api/admin/v1/articles", OperationID: "article.list", Summary: "List articles", Tags: []string{"article"}, Permission: "article:read"}, h.list},
		{router.Route{Method: http.MethodGet, Path: "/api/admin/v1/articles/{id}", OperationID: "article.get", Summary: "Get an article", Tags: []string{"article"}, Permission: "article:read"}, h.get},
		{router.Route{Method: http.MethodPost, Path: "/api/admin/v1/articles", OperationID: "article.create", Summary: "Create a draft", Tags: []string{"article"}, Permission: "article:write", AuditAction: "article.create"}, h.create},
		{router.Route{Method: http.MethodPut, Path: "/api/admin/v1/articles/{id}", OperationID: "article.update", Summary: "Update an article", Tags: []string{"article"}, Permission: "article:write", AuditAction: "article.update"}, h.update},
		{router.Route{Method: http.MethodDelete, Path: "/api/admin/v1/articles/{id}", OperationID: "article.delete", Summary: "Soft-delete an article", Tags: []string{"article"}, Permission: "article:delete", AuditAction: "article.delete"}, h.delete},
		{router.Route{Method: http.MethodPost, Path: "/api/admin/v1/articles/{id}/publish", OperationID: "article.publish", Summary: "Publish an article", Tags: []string{"article"}, Permission: "article:publish", AuditAction: "article.publish"}, h.publish},
		{router.Route{Method: http.MethodPost, Path: "/api/admin/v1/articles/{id}/unpublish", OperationID: "article.unpublish", Summary: "Unpublish an article", Tags: []string{"article"}, Permission: "article:publish", AuditAction: "article.unpublish"}, h.unpublish},
		{router.Route{Method: http.MethodGet, Path: "/api/admin/v1/articles/{id}/revisions", OperationID: "article.revisions", Summary: "List article revisions", Tags: []string{"article"}, Permission: "article:read"}, h.revisions},
	}

	for _, item := range routes {
		if err := register(reg, api, item.route, item.handler); err != nil {
			return err
		}
	}
	return nil
}

// register keeps the table above readable while preserving typed Huma handlers.
func register[T any](reg *router.Registry, api huma.API, route router.Route, handler T) error {
	switch h := any(handler).(type) {
	case func(context.Context, *ListInput) (*ListOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *GetInput) (*ArticleOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *CreateInput) (*ArticleOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *UpdateInput) (*ArticleOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *GetInput) (*OKOutput, error):
		return router.Register(reg, api, route, h)
	case func(context.Context, *GetInput) (*RevisionsOutput, error):
		return router.Register(reg, api, route, h)
	default:
		panic("unsupported article handler type")
	}
}

func (h *Handler) list(ctx context.Context, in *ListInput) (*ListOutput, error) {
	result, err := h.svc.List(ctx, in.Status, in.Query, in.Page, in.PageSize)
	if err != nil {
		return nil, problem(http.StatusInternalServerError, "Internal Server Error", "list articles failed")
	}
	out := &ListOutput{}
	out.Body.Items = make([]ArticleDTO, 0, len(result.Items))
	for i := range result.Items {
		out.Body.Items = append(out.Body.Items, toDTO(&result.Items[i]))
	}
	out.Body.Page = result.Page
	out.Body.PageSize = result.PageSize
	out.Body.Total = result.Total
	return out, nil
}

func (h *Handler) get(ctx context.Context, in *GetInput) (*ArticleOutput, error) {
	a, err := h.svc.Get(ctx, in.ID)
	if err != nil {
		return nil, articleError(err)
	}
	return &ArticleOutput{Body: toDTO(a)}, nil
}

func (h *Handler) create(ctx context.Context, in *CreateInput) (*ArticleOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}
	if in.Body.Title == "" || in.Body.ContentMD == "" {
		return nil, problem(http.StatusUnprocessableEntity, "Validation failed", "title and contentMd are required")
	}
	a, err := h.svc.Create(ctx, actor, articlesvc.CreateInput{
		Slug: in.Body.Slug, Title: in.Body.Title, Summary: in.Body.Summary,
		ContentMD: in.Body.ContentMD, CoverURL: in.Body.CoverURL, CategoryID: in.Body.CategoryID,
	})
	if err != nil {
		return nil, articleError(err)
	}
	return &ArticleOutput{Body: toDTO(a)}, nil
}

func (h *Handler) update(ctx context.Context, in *UpdateInput) (*ArticleOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}
	if in.Body.Version < 1 {
		return nil, problem(http.StatusUnprocessableEntity, "Validation failed", "version is required")
	}
	a, err := h.svc.Update(ctx, actor, articlesvc.UpdateInput{
		ID: in.ID, Version: in.Body.Version, Slug: in.Body.Slug, Title: in.Body.Title,
		Summary: in.Body.Summary, ContentMD: in.Body.ContentMD,
		CoverURL: in.Body.CoverURL, CategoryID: in.Body.CategoryID,
	})
	if err != nil {
		return nil, articleError(err)
	}
	return &ArticleOutput{Body: toDTO(a)}, nil
}

func (h *Handler) delete(ctx context.Context, in *GetInput) (*OKOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}
	if err := h.svc.Delete(ctx, actor, in.ID); err != nil {
		return nil, articleError(err)
	}
	out := &OKOutput{}
	out.Body.OK = true
	return out, nil
}

func (h *Handler) publish(ctx context.Context, in *GetInput) (*ArticleOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}
	a, err := h.svc.Publish(ctx, actor, in.ID)
	if err != nil {
		return nil, articleError(err)
	}
	return &ArticleOutput{Body: toDTO(a)}, nil
}

func (h *Handler) unpublish(ctx context.Context, in *GetInput) (*ArticleOutput, error) {
	actor, ok := authpkg.PrincipalFrom(ctx)
	if !ok {
		return nil, problem(http.StatusUnauthorized, "Unauthorized", "authentication required")
	}
	a, err := h.svc.Unpublish(ctx, actor, in.ID)
	if err != nil {
		return nil, articleError(err)
	}
	return &ArticleOutput{Body: toDTO(a)}, nil
}

func (h *Handler) revisions(ctx context.Context, in *GetInput) (*RevisionsOutput, error) {
	items, err := h.svc.Revisions(ctx, in.ID)
	if err != nil {
		return nil, articleError(err)
	}
	out := &RevisionsOutput{}
	out.Body.Items = make([]RevisionDTO, 0, len(items))
	for i := range items {
		rev := items[i]
		out.Body.Items = append(out.Body.Items, RevisionDTO{
			ID: rev.ID, Version: rev.Version, Title: rev.Title, ContentMD: rev.ContentMD,
			EditorID: rev.EditorID, ChangeNote: rev.ChangeNote, CreatedAt: rev.CreatedAt,
		})
	}
	return out, nil
}

func toDTO(a *model.Article) ArticleDTO {
	if a == nil {
		return ArticleDTO{}
	}
	return ArticleDTO{
		ID: a.ID, Slug: a.Slug, Title: a.Title, Summary: a.Summary,
		ContentMD: a.ContentMD, ContentHTML: a.ContentHTML, CoverURL: a.CoverURL,
		CategoryID: a.CategoryID, AuthorID: a.AuthorID, Status: a.Status,
		PublishedAt: a.PublishedAt, Version: a.Version,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func articleError(err error) error {
	switch {
	case errors.Is(err, articlesvc.ErrNotFound):
		return problem(http.StatusNotFound, "Not Found", "article not found")
	case errors.Is(err, articlesvc.ErrConflict):
		return problem(http.StatusConflict, "Conflict", "article was changed or slug is taken")
	default:
		return problem(http.StatusInternalServerError, "Internal Server Error", "article operation failed")
	}
}

func problem(status int, title, detail string) *huma.ErrorModel {
	return &huma.ErrorModel{Type: "about:blank", Title: title, Status: status, Detail: detail}
}
