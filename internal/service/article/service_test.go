package article

import (
	"errors"
	"strings"
	"testing"

	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/store"
)

func TestRenderSanitizesHTML(t *testing.T) {
	svc := NewService(nil, nil)
	html, err := svc.render("# Hello\n\n<script>alert('xss')</script>\n\n**bold**")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(strings.ToLower(html), "<script") {
		t.Fatalf("script tag survived sanitization: %s", html)
	}
	if !strings.Contains(html, "<h1") || !strings.Contains(html, "<strong>") {
		t.Fatalf("markdown was not rendered: %s", html)
	}
}

func TestSlugify(t *testing.T) {
	if got := slugify("Hello, World!"); got != "hello-world" {
		t.Fatalf("slugify = %q", got)
	}
	if got := slugify("你好"); !strings.HasPrefix(got, "article-") {
		t.Fatalf("non-latin title should fall back to article id, got %q", got)
	}
}

func TestMapStoreError(t *testing.T) {
	if !errors.Is(mapStoreError(store.ErrNotFound), ErrNotFound) {
		t.Fatal("store not-found should map to service not-found")
	}
	if !errors.Is(mapStoreError(store.ErrConflict), ErrConflict) {
		t.Fatal("store conflict should map to service conflict")
	}
	var unknown = errors.New("boom")
	if !errors.Is(mapStoreError(unknown), unknown) {
		t.Fatal("unknown errors must pass through")
	}
	if mapStoreError(nil) != nil {
		t.Fatal("nil must stay nil")
	}
}

func TestArticleDiff(t *testing.T) {
	before := &model.Article{Title: "Old", Slug: "old", Status: "draft", ContentMD: "a"}
	after := &model.Article{Title: "New", Slug: "new", Status: "published", ContentMD: "b"}
	diff := articleDiff(before, after)
	if _, ok := diff["title"]; !ok {
		t.Fatal("title change missing")
	}
	if _, ok := diff["status"]; !ok {
		t.Fatal("status change missing")
	}
	if _, ok := diff["content"]; !ok {
		t.Fatal("content change missing")
	}
}
