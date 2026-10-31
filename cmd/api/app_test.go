package main

import (
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
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/router"
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
		Version:     "test",
	})
	if err != nil {
		t.Fatalf("newApp: %v", err)
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
