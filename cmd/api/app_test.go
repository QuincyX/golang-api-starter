package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/config"
	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/router"
	accountsvc "github.com/example/homebrew-api-starter/internal/service/account"
	articlesvc "github.com/example/homebrew-api-starter/internal/service/article"
	authsvc "github.com/example/homebrew-api-starter/internal/service/auth"
	"github.com/example/homebrew-api-starter/internal/store"
)

func TestRouteContractAndAuth(t *testing.T) {
	cfg := testConfig()
	manager, err := jwt.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTTL)
	if err != nil {
		t.Fatalf("jwt.New: %v", err)
	}

	app, reg, err := newApp(dependencies{
		Logger:      testLogger(),
		JWT:         manager,
		Permissions: authpkg.NewStaticPermissionStore([]string{"article:read"}),
		Auth:        fakeAuth{},
		Accounts:    fakeAccount{},
		Articles:    fakeArticle{},
		Audit:       fakeAudit{},
		Version:     "test",
	})
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}

	if openapi := request(t, app, http.MethodGet, "/openapi.json", ""); openapi.Code != http.StatusOK {
		t.Fatalf("openapi status = %d, body = %s", openapi.Code, openapi.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/auth/login", strings.NewReader(`{"email":"a@b.c","password":"bad"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	app.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d, want 401", loginRec.Code)
	}
	if cookie := loginRec.Header().Get("Set-Cookie"); cookie != "" {
		t.Fatalf("login must not set cookies: %s", cookie)
	}

	if reg.Count() < 6 {
		t.Fatalf("expected at least 6 routes, got %d", reg.Count())
	}
	for _, route := range reg.Routes() {
		if route.Method != http.MethodGet && route.Method != http.MethodHead &&
			route.Method != http.MethodOptions && route.AuditAction == "" {
			t.Errorf("route %s %s has no audit action", route.Method, route.Path)
		}
		if route.Public && route.Permission != "" {
			t.Errorf("route %s %s is public but requires permission", route.Method, route.Path)
		}
	}

	if got := request(t, app, http.MethodGet, "/healthz", "").Code; got != http.StatusOK {
		t.Errorf("healthz status = %d, want 200", got)
	}
	if got := request(t, app, http.MethodGet, "/readyz", "").Code; got != http.StatusServiceUnavailable {
		t.Errorf("readyz status = %d, want 503 without a database", got)
	}
	if got := request(t, app, http.MethodGet, "/api/admin/v1/auth/me", "").Code; got != http.StatusUnauthorized {
		t.Errorf("me without token = %d, want 401", got)
	}

	token, _, err := manager.Sign("user-1", "session-1", 3)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	me := request(t, app, http.MethodGet, "/api/admin/v1/auth/me", token)
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", me.Code, me.Body.String())
	}
	if !strings.Contains(me.Body.String(), `"userId":"user-1"`) {
		t.Errorf("me body does not contain user id: %s", me.Body.String())
	}

	if got := request(t, app, http.MethodGet, "/api/admin/v1/articles", token).Code; got != http.StatusOK {
		t.Errorf("articles with permission = %d, want 200", got)
	}
	if got := request(t, app, http.MethodPost, "/api/admin/v1/auth/logout", token).Code; got != http.StatusOK {
		t.Errorf("logout = %d, want 200", got)
	}

	noPerms, _, err := newApp(dependencies{
		Logger:      testLogger(),
		JWT:         manager,
		Permissions: authpkg.NewStaticPermissionStore(nil),
		Auth:        fakeAuth{},
		Accounts:    fakeAccount{},
		Articles:    fakeArticle{},
		Audit:       fakeAudit{},
		Version:     "test",
	})
	if err != nil {
		t.Fatalf("newApp without permissions: %v", err)
	}
	if got := request(t, noPerms, http.MethodGet, "/api/admin/v1/articles", token).Code; got != http.StatusForbidden {
		t.Errorf("articles without permission = %d, want 403", got)
	}
}

func TestRouteCoverageRejectsUnregisteredRoute(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/admin/v1/rogue", func(http.ResponseWriter, *http.Request) {})

	if err := verifyRouteCoverage(r, router.NewRegistry()); err == nil {
		t.Fatal("expected a route registered outside the contract to be rejected")
	}
}

func testConfig() config.Config {
	return config.Config{
		AppEnv:      "test",
		JWTSecret:   strings.Repeat("s", 32),
		JWTIssuer:   "homebrew-api",
		JWTAudience: "admin-web",
		AccessTTL:   15 * time.Minute,
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func request(t *testing.T, h http.Handler, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type fakeAuth struct{}

func (fakeAuth) Login(context.Context, string, string) (*authsvc.LoginResult, error) {
	return nil, authsvc.ErrInvalidCredentials
}

func (fakeAuth) Logout(context.Context, authpkg.Principal) error { return nil }

func (fakeAuth) ChangePassword(context.Context, string, string, string) error { return nil }

type fakeArticle struct{}

func (fakeArticle) List(_ context.Context, _ string, _ string, page, pageSize int) (*articlesvc.ListResult, error) {
	return &articlesvc.ListResult{Items: []model.Article{}, Page: page, PageSize: pageSize}, nil
}

func (fakeArticle) Get(context.Context, string) (*model.Article, error) {
	return nil, store.ErrNotFound
}

func (fakeArticle) Create(context.Context, authpkg.Principal, articlesvc.CreateInput) (*model.Article, error) {
	return nil, store.ErrNotFound
}

func (fakeArticle) Update(context.Context, authpkg.Principal, articlesvc.UpdateInput) (*model.Article, error) {
	return nil, store.ErrNotFound
}

func (fakeArticle) Publish(context.Context, authpkg.Principal, string) (*model.Article, error) {
	return nil, store.ErrNotFound
}

func (fakeArticle) Unpublish(context.Context, authpkg.Principal, string) (*model.Article, error) {
	return nil, store.ErrNotFound
}

func (fakeArticle) Delete(context.Context, authpkg.Principal, string) error { return nil }

func (fakeArticle) Revisions(context.Context, string) ([]model.ArticleRevision, error) {
	return []model.ArticleRevision{}, nil
}

type fakeAccount struct{}

func (fakeAccount) List(_ context.Context, _ string, page, pageSize int) (*accountsvc.ListResult, error) {
	return &accountsvc.ListResult{Items: []model.User{}, Page: page, PageSize: pageSize}, nil
}

func (fakeAccount) Create(context.Context, authpkg.Principal, accountsvc.CreateInput) (*accountsvc.CreateResult, error) {
	return &accountsvc.CreateResult{User: &model.User{}, ActivationURL: "http://localhost/activate?token=test"}, nil
}

func (fakeAccount) SetStatus(context.Context, authpkg.Principal, string, string) error  { return nil }
func (fakeAccount) SetRoles(context.Context, authpkg.Principal, string, []string) error { return nil }
func (fakeAccount) ResetPassword(context.Context, authpkg.Principal, string) (string, error) {
	return "http://localhost/activate?token=test", nil
}
func (fakeAccount) Sessions(context.Context, string) ([]model.Session, error) {
	return []model.Session{}, nil
}
func (fakeAccount) RevokeSession(context.Context, authpkg.Principal, string, string) error {
	return nil
}
func (fakeAccount) Roles(context.Context) ([]map[string]any, error) { return []map[string]any{}, nil }
func (fakeAccount) Permissions(context.Context) ([]map[string]any, error) {
	return []map[string]any{}, nil
}
func (fakeAccount) Activate(context.Context, string, string) error { return nil }

type fakeAudit struct{}

func (fakeAudit) List(context.Context, int, int64, string, string) ([]model.AuditLog, int64, error) {
	return []model.AuditLog{}, 0, nil
}
