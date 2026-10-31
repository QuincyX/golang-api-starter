package article

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"

	"github.com/example/homebrew-api-starter/internal/audit"
	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/store"
)

var (
	ErrNotFound = errors.New("article not found")
	ErrConflict = errors.New("article conflict")
)

type Service struct {
	store  *store.Store
	audit  *audit.Recorder
	md     goldmark.Markdown
	policy *bluemonday.Policy
	now    func() time.Time
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, store.ErrConflict) {
		return ErrConflict
	}
	return err
}

func NewService(st *store.Store, recorder *audit.Recorder) *Service {
	return &Service{
		store:  st,
		audit:  recorder,
		md:     goldmark.New(goldmark.WithExtensions()),
		policy: bluemonday.UGCPolicy(),
		now:    time.Now,
	}
}

type CreateInput struct {
	Slug       string
	Title      string
	Summary    string
	ContentMD  string
	CoverURL   string
	CategoryID *string
}

type UpdateInput struct {
	ID         string
	Version    int
	Slug       string
	Title      string
	Summary    string
	ContentMD  string
	CoverURL   string
	CategoryID *string
}

type ListResult struct {
	Items    []model.Article
	Total    int64
	Page     int
	PageSize int
}

func (s *Service) List(ctx context.Context, status, query string, page, pageSize int) (*ListResult, error) {
	page, pageSize = normalizePage(page, pageSize)
	items, total, err := s.store.Queries().ListArticles(ctx, status, query, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &ListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) Get(ctx context.Context, id string) (*model.Article, error) {
	a, err := s.store.Queries().GetArticle(ctx, id)
	return a, mapStoreError(err)
}

func (s *Service) Create(ctx context.Context, actor authpkg.Principal, in CreateInput) (*model.Article, error) {
	slug := in.Slug
	if strings.TrimSpace(slug) == "" {
		slug = slugify(in.Title)
	}
	html, err := s.render(in.ContentMD)
	if err != nil {
		return nil, err
	}
	a := &model.Article{
		Slug:        slug,
		Title:       in.Title,
		Summary:     in.Summary,
		ContentMD:   in.ContentMD,
		ContentHTML: html,
		CoverURL:    in.CoverURL,
		CategoryID:  in.CategoryID,
		AuthorID:    actor.UserID,
	}
	var created *model.Article
	err = s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		created, err = q.CreateArticle(ctx, a)
		if err != nil {
			return err
		}
		if err := q.InsertArticleRevision(ctx, &model.ArticleRevision{
			ArticleID: created.ID, Version: created.Version, Title: created.Title,
			ContentMD: created.ContentMD, EditorID: actor.UserID, ChangeNote: "created",
		}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "article.create",
			ResourceType: "article",
			ResourceID:   created.ID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Metadata:     map[string]any{"slug": created.Slug},
		})
	})
	return created, mapStoreError(err)
}

func (s *Service) Update(ctx context.Context, actor authpkg.Principal, in UpdateInput) (*model.Article, error) {
	before, err := s.store.Queries().GetArticle(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	html, err := s.render(in.ContentMD)
	if err != nil {
		return nil, err
	}
	next := &model.Article{
		Slug:        in.Slug,
		Title:       in.Title,
		Summary:     in.Summary,
		ContentMD:   in.ContentMD,
		ContentHTML: html,
		CoverURL:    in.CoverURL,
		CategoryID:  in.CategoryID,
	}
	if strings.TrimSpace(next.Slug) == "" {
		next.Slug = before.Slug
	}

	var updated *model.Article
	err = s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		updated, err = q.UpdateArticle(ctx, in.ID, next, in.Version)
		if err != nil {
			return err
		}
		if err := q.InsertArticleRevision(ctx, &model.ArticleRevision{
			ArticleID: updated.ID, Version: updated.Version, Title: updated.Title,
			ContentMD: updated.ContentMD, EditorID: actor.UserID, ChangeNote: "updated",
		}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "article.update",
			ResourceType: "article",
			ResourceID:   updated.ID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Changes:      articleDiff(before, updated),
		})
	})
	return updated, mapStoreError(err)
}

func (s *Service) Publish(ctx context.Context, actor authpkg.Principal, id string) (*model.Article, error) {
	return s.transition(ctx, actor, id, "article.publish", true)
}

func (s *Service) Unpublish(ctx context.Context, actor authpkg.Principal, id string) (*model.Article, error) {
	return s.transition(ctx, actor, id, "article.unpublish", false)
}

func (s *Service) Delete(ctx context.Context, actor authpkg.Principal, id string) error {
	err := s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.DeleteArticle(ctx, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       "article.delete",
			ResourceType: "article",
			ResourceID:   id,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
		})
	})
	return mapStoreError(err)
}

func (s *Service) Revisions(ctx context.Context, id string) ([]model.ArticleRevision, error) {
	items, err := s.store.Queries().ListArticleRevisions(ctx, id)
	return items, mapStoreError(err)
}

type transitionFunc func(context.Context, string) (*model.Article, error)

func (s *Service) transition(ctx context.Context, actor authpkg.Principal, id, action string, publish bool) (*model.Article, error) {
	var out *model.Article
	err := s.store.WithTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		before, err := q.GetArticle(ctx, id)
		if err != nil {
			return err
		}
		if publish {
			out, err = q.PublishArticle(ctx, id)
		} else {
			out, err = q.UnpublishArticle(ctx, id)
		}
		if err != nil {
			return err
		}
		if err := q.InsertArticleRevision(ctx, &model.ArticleRevision{
			ArticleID: out.ID, Version: out.Version, Title: out.Title,
			ContentMD: out.ContentMD, EditorID: actor.UserID, ChangeNote: action,
		}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			Action:       action,
			ResourceType: "article",
			ResourceID:   out.ID,
			Result:       "success",
			ActorID:      actor.UserID,
			ActorEmail:   actor.Email,
			Changes:      articleDiff(before, out),
		})
	})
	return out, mapStoreError(err)
}

func (s *Service) render(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return string(s.policy.SanitizeBytes(buf.Bytes())), nil
}

func articleDiff(before, after *model.Article) map[string]any {
	diff := map[string]any{}
	if before.Title != after.Title {
		diff["title"] = map[string]string{"from": before.Title, "to": after.Title}
	}
	if before.Slug != after.Slug {
		diff["slug"] = map[string]string{"from": before.Slug, "to": after.Slug}
	}
	if before.Status != after.Status {
		diff["status"] = map[string]string{"from": before.Status, "to": after.Status}
	}
	if before.ContentMD != after.ContentMD {
		diff["content"] = map[string]string{
			"from": hash(before.ContentMD),
			"to":   hash(after.ContentMD),
		}
	}
	return diff
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = slugPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = fmt.Sprintf("article-%d", time.Now().UnixNano())
	}
	return slug
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
