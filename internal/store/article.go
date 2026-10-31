package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/example/homebrew-api-starter/internal/model"
)

const articleColumns = `id::text, slug, title, summary, content_md, content_html,
	COALESCE(cover_url, ''), category_id::text, author_id::text, status,
	published_at, version, created_at, updated_at`

func scanArticle(row pgx.Row) (*model.Article, error) {
	var a model.Article
	var categoryID *string
	err := row.Scan(
		&a.ID, &a.Slug, &a.Title, &a.Summary, &a.ContentMD, &a.ContentHTML,
		&a.CoverURL, &categoryID, &a.AuthorID, &a.Status, &a.PublishedAt,
		&a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CategoryID = categoryID
	return &a, nil
}

func (q *Queries) ListArticles(ctx context.Context, status, query string, limit, offset int) ([]model.Article, int64, error) {
	var total int64
	if err := q.db.QueryRow(ctx,
		`SELECT count(*) FROM articles
		 WHERE deleted_at IS NULL
		   AND ($1 = '' OR status = $1)
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%' OR slug ILIKE '%' || $2 || '%')`,
		status, query).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := q.db.Query(ctx,
		`SELECT `+articleColumns+` FROM articles
		 WHERE deleted_at IS NULL
		   AND ($1 = '' OR status = $1)
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%' OR slug ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, status, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []model.Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *a)
	}
	return out, total, rows.Err()
}

func (q *Queries) GetArticle(ctx context.Context, id string) (*model.Article, error) {
	return scanArticle(q.db.QueryRow(ctx,
		`SELECT `+articleColumns+` FROM articles WHERE id = $1::text::uuid AND deleted_at IS NULL`, id))
}

func (q *Queries) CreateArticle(ctx context.Context, a *model.Article) (*model.Article, error) {
	var categoryID any
	if a.CategoryID != nil {
		categoryID = *a.CategoryID
	}
	row := q.db.QueryRow(ctx,
		`INSERT INTO articles (slug, title, summary, content_md, content_html, cover_url, category_id, author_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::text::uuid, $8::text::uuid, 'draft')
		 RETURNING `+articleColumns,
		a.Slug, a.Title, a.Summary, a.ContentMD, a.ContentHTML, a.CoverURL, categoryID, a.AuthorID)
	out, err := scanArticle(row)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return out, err
}

func (q *Queries) UpdateArticle(ctx context.Context, id string, a *model.Article, expectedVersion int) (*model.Article, error) {
	var categoryID any
	if a.CategoryID != nil {
		categoryID = *a.CategoryID
	}
	row := q.db.QueryRow(ctx,
		`UPDATE articles
		 SET slug = $2, title = $3, summary = $4, content_md = $5, content_html = $6,
		     cover_url = $7, category_id = $8::text::uuid, version = version + 1, updated_at = now()
		 WHERE id = $1::text::uuid AND version = $9 AND deleted_at IS NULL
		 RETURNING `+articleColumns,
		id, a.Slug, a.Title, a.Summary, a.ContentMD, a.ContentHTML, a.CoverURL, categoryID, expectedVersion)
	out, err := scanArticle(row)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrConflict
	}
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return out, err
}

func (q *Queries) DeleteArticle(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE articles SET deleted_at = now(), updated_at = now()
		 WHERE id = $1::text::uuid AND deleted_at IS NULL`, id)
	return err
}

func (q *Queries) PublishArticle(ctx context.Context, id string) (*model.Article, error) {
	out, err := scanArticle(q.db.QueryRow(ctx,
		`UPDATE articles
		 SET status = 'published', published_at = COALESCE(published_at, now()),
		     version = version + 1, updated_at = now()
		 WHERE id = $1::text::uuid AND status = 'draft' AND deleted_at IS NULL
		 RETURNING `+articleColumns, id))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrConflict
	}
	return out, err
}

func (q *Queries) UnpublishArticle(ctx context.Context, id string) (*model.Article, error) {
	out, err := scanArticle(q.db.QueryRow(ctx,
		`UPDATE articles
		 SET status = 'draft', version = version + 1, updated_at = now()
		 WHERE id = $1::text::uuid AND status = 'published' AND deleted_at IS NULL
		 RETURNING `+articleColumns, id))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrConflict
	}
	return out, err
}

func (q *Queries) InsertArticleRevision(ctx context.Context, rev *model.ArticleRevision) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO article_revisions (article_id, version, title, content_md, editor_id, change_note)
		 VALUES ($1::text::uuid, $2, $3, $4, $5::text::uuid, $6)`,
		rev.ArticleID, rev.Version, rev.Title, rev.ContentMD, rev.EditorID, rev.ChangeNote)
	return err
}

func (q *Queries) ListArticleRevisions(ctx context.Context, articleID string) ([]model.ArticleRevision, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, article_id::text, version, title, content_md, editor_id::text, change_note, created_at
		 FROM article_revisions WHERE article_id = $1::text::uuid ORDER BY version DESC`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.ArticleRevision
	for rows.Next() {
		var rev model.ArticleRevision
		if err := rows.Scan(&rev.ID, &rev.ArticleID, &rev.Version, &rev.Title, &rev.ContentMD,
			&rev.EditorID, &rev.ChangeNote, &rev.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}
