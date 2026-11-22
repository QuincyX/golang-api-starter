package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	authpkg "github.com/example/golang-api-starter/internal/auth"
	"github.com/example/golang-api-starter/internal/platform/jwt"
)

var (
	updateOpenAPI = flag.Bool("update-openapi", false, "rewrite the OpenAPI snapshot")
	openAPIOut    = flag.String("openapi-out", "", "write the OpenAPI document to this path without comparing")
)

// TestOpenAPISnapshot makes API contract changes visible in review. Regenerate
// the snapshot with `task api:update` after an intentional change, and compare
// against a base branch with `task api:breaking`.
func TestOpenAPISnapshot(t *testing.T) {
	app := newSnapshotApp(t)

	rec := request(t, app, http.MethodGet, "/openapi.json", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("openapi status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var doc any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}
	got, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("encode openapi: %v", err)
	}
	got = append(got, '\n')

	if *openAPIOut != "" {
		writeSnapshot(t, resolveRootPath(t, *openAPIOut), got)
		return
	}

	snapshot := filepath.Join("testdata", "openapi.json")
	if *updateOpenAPI {
		writeSnapshot(t, snapshot, got)
		return
	}

	want, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatalf("read OpenAPI snapshot: %v (run `task api:update`)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("OpenAPI snapshot changed; run `task api:update` after reviewing the contract diff")
	}
}

func newSnapshotApp(t *testing.T) http.Handler {
	t.Helper()

	cfg := testConfig()
	manager, err := jwt.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTTL)
	if err != nil {
		t.Fatalf("jwt.New: %v", err)
	}

	app, _, err := newApp(dependencies{
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
		t.Fatalf("newApp: %v", err)
	}
	return app
}

func resolveRootPath(t *testing.T, path string) string {
	t.Helper()
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Clean(filepath.Join("..", "..")), path)
}

func writeSnapshot(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create snapshot dir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
}
