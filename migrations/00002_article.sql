-- +goose Up
CREATE TABLE categories (
    id   uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    sort integer NOT NULL DEFAULT 0
);

CREATE TABLE tags (
    id   uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL,
    slug text NOT NULL UNIQUE
);

CREATE TABLE articles (
    id              uuid PRIMARY KEY DEFAULT uuidv7(),
    slug            text NOT NULL,
    title           text NOT NULL,
    summary         text NOT NULL DEFAULT '',
    content_md      text NOT NULL,
    content_html    text NOT NULL DEFAULT '',
    cover_url       text,
    category_id     uuid REFERENCES categories(id) ON DELETE SET NULL,
    author_id       uuid NOT NULL REFERENCES admin_users(id),
    status          text NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'published', 'archived')),
    published_at    timestamptz,
    version         integer NOT NULL DEFAULT 1,
    seo_title       text,
    seo_description text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);
CREATE UNIQUE INDEX articles_slug_uq
    ON articles (slug) WHERE deleted_at IS NULL;
CREATE INDEX articles_list_idx
    ON articles (status, published_at DESC NULLS LAST) WHERE deleted_at IS NULL;

CREATE TABLE article_tags (
    article_id uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag_id     uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, tag_id)
);

CREATE TABLE article_revisions (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    article_id  uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    version     integer NOT NULL,
    title       text NOT NULL,
    content_md  text NOT NULL,
    editor_id   uuid NOT NULL REFERENCES admin_users(id),
    change_note text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (article_id, version)
);

-- +goose Down
DROP TABLE IF EXISTS article_revisions;
DROP TABLE IF EXISTS article_tags;
DROP TABLE IF EXISTS articles;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS categories;
