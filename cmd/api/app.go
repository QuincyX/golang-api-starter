package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	accountapi "github.com/example/homebrew-api-starter/internal/api/account"
	articleapi "github.com/example/homebrew-api-starter/internal/api/article"
	auditapi "github.com/example/homebrew-api-starter/internal/api/audit"
	authapi "github.com/example/homebrew-api-starter/internal/api/auth"
	healthapi "github.com/example/homebrew-api-starter/internal/api/health"
	auditpkg "github.com/example/homebrew-api-starter/internal/audit"
	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/middleware"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/platform/metrics"
	"github.com/example/homebrew-api-starter/internal/router"
	accountsvc "github.com/example/homebrew-api-starter/internal/service/account"
	articlesvc "github.com/example/homebrew-api-starter/internal/service/article"
	auditsvc "github.com/example/homebrew-api-starter/internal/service/auditlog"
	authsvc "github.com/example/homebrew-api-starter/internal/service/auth"
	"github.com/example/homebrew-api-starter/internal/store"
)

type dependencies struct {
	Logger      *slog.Logger
	JWT         *jwt.Manager
	DB          *pgxpool.Pool
	Metrics     *metrics.Metrics
	Permissions authpkg.PermissionStore
	Sessions    authpkg.SessionValidator
	Auth        authapi.Service
	Accounts    accountapi.Service
	Articles    articleapi.Service
	Audit       auditapi.Service
	AdminWebURL string
	Version     string
}

// newApp is the composition root. It is also the startup gate: route metadata
// is validated before the server can listen.
func newApp(deps dependencies) (http.Handler, *router.Registry, error) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.JWT == nil {
		return nil, nil, fmt.Errorf("jwt manager is required")
	}
	if deps.Version == "" {
		deps.Version = "dev"
	}
	if deps.Auth == nil || deps.Articles == nil {
		if deps.DB == nil {
			return nil, nil, fmt.Errorf("database is required to build services")
		}
		if err := buildServices(&deps); err != nil {
			return nil, nil, err
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestMeta())
	r.Use(middleware.BodyLimit(2 << 20))
	// Logger wraps Recover so a panic still produces one request log line with
	// status 500 instead of disappearing.
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Use(middleware.Recover(deps.Logger))
	if deps.Metrics != nil {
		r.Use(deps.Metrics.Middleware)
	}

	reg := router.NewRegistry()
	// The registry is empty here and populated below; the middleware reads it
	// per request, so public routes are known by the time traffic arrives.
	r.Use(middleware.Authenticate(reg, deps.JWT, deps.Permissions, deps.Sessions))

	api := humachi.New(r, humaConfig(deps.Version))

	if err := healthapi.New(deps.DB, deps.Version).Register(api, reg); err != nil {
		return nil, nil, err
	}
	if err := authapi.New(deps.Auth).Register(api, reg); err != nil {
		return nil, nil, err
	}
	if err := articleapi.New(deps.Articles).Register(api, reg); err != nil {
		return nil, nil, err
	}
	if deps.Accounts != nil {
		if err := accountapi.New(deps.Accounts).Register(api, reg); err != nil {
			return nil, nil, err
		}
	}
	if deps.Audit != nil {
		if err := auditapi.New(deps.Audit).Register(api, reg); err != nil {
			return nil, nil, err
		}
	}

	// These routes are registered by Huma itself. They are declared here so the
	// public-route contract and the auth middleware agree about them.
	metaRoutes := []router.Route{
		{Method: http.MethodGet, Path: "/openapi.json", OperationID: "docs.openapi.json", Public: true},
		{Method: http.MethodGet, Path: "/openapi.yaml", OperationID: "docs.openapi.yaml", Public: true},
		{Method: http.MethodGet, Path: "/docs", OperationID: "docs.ui", Public: true},
		{Method: http.MethodGet, Path: "/schemas/{schema}", OperationID: "docs.schema", Public: true},
	}
	for _, route := range metaRoutes {
		if err := reg.Add(route); err != nil {
			return nil, nil, err
		}
	}

	if err := verifyRouteCoverage(r, reg); err != nil {
		return nil, nil, err
	}
	if err := reg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("route contract: %w", err)
	}
	return r, reg, nil
}

func buildServices(deps *dependencies) error {
	st := store.New(deps.DB)
	recorder := auditpkg.NewRecorder(deps.DB)
	authSvc := authsvc.NewService(st, deps.JWT, recorder)
	deps.Auth = authSvc
	deps.Sessions = authSvc
	deps.Accounts = accountsvc.NewService(st, recorder, deps.AdminWebURL)
	deps.Articles = articlesvc.NewService(st, recorder)
	deps.Audit = auditsvc.NewService(st)
	return nil
}

// verifyRouteCoverage catches the one failure mode the registry cannot prevent
// by itself: someone calling huma.Register directly and bypassing the helper.
func verifyRouteCoverage(r chi.Router, reg *router.Registry) error {
	var missing []string
	walked := make(map[string]struct{})
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		walked[method+" "+route] = struct{}{}
		if isMetaRoute(route) {
			return nil
		}
		if !reg.Has(method, route) {
			missing = append(missing, method+" "+route)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk routes: %w", err)
	}

	// The reverse direction catches a declared route that never reached chi.
	for _, route := range reg.Routes() {
		if isMetaRoute(route.Path) {
			continue
		}
		key := route.Method + " " + route.Path
		if _, ok := walked[key]; !ok {
			missing = append(missing, "declared but not mounted: "+key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("route contract mismatch: %s", strings.Join(missing, ", "))
	}
	return nil
}

func isMetaRoute(route string) bool {
	return strings.HasPrefix(route, "/docs") ||
		strings.HasPrefix(route, "/openapi") ||
		strings.HasPrefix(route, "/schemas")
}

func humaConfig(version string) huma.Config {
	cfg := huma.DefaultConfig("Homebrew Admin API", version)
	cfg.Info.Description = "Management API: accounts, articles, audit."
	cfg.OpenAPIPath = "/openapi"
	cfg.DocsPath = "/docs"
	// DefaultConfig installs a transformer that appends `$schema` links to every
	// response body. That is useful for schema-aware clients, but it is noise for
	// the admin SPA and leaks internal URLs, so disable it.
	cfg.CreateHooks = nil
	return cfg
}
