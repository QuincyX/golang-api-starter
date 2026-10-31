package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID                 string
	Email              string
	PasswordHash       string
	DisplayName        string
	Status             string
	MustChangePassword bool
	TokenVersion       int
	LastLoginAt        *time.Time
	CreatedAt          time.Time
	Roles              []string
	Permissions        []string
}

type Session struct {
	ID         string
	UserID     string
	UserAgent  string
	IP         string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

type Article struct {
	ID          string
	Slug        string
	Title       string
	Summary     string
	ContentMD   string
	ContentHTML string
	CoverURL    string
	CategoryID  *string
	AuthorID    string
	Status      string
	PublishedAt *time.Time
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ArticleRevision struct {
	ID         int64
	ArticleID  string
	Version    int
	Title      string
	ContentMD  string
	EditorID   string
	ChangeNote string
	CreatedAt  time.Time
}

type AuditLog struct {
	ID           int64
	OccurredAt   time.Time
	RequestID    string
	TraceID      string
	ActorID      *string
	ActorEmail   *string
	Action       string
	ResourceType string
	ResourceID   *string
	Result       string
	StatusCode   *int
	IP           *string
	UserAgent    *string
	Changes      json.RawMessage
	Metadata     json.RawMessage
	DurationMS   *int
}

type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}
