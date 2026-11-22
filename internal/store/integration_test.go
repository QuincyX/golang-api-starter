//go:build integration

package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestIntegrationDatabaseReady is the starter smoke test for the integration
// layer. Set TEST_DATABASE_URL to a migrated PostgreSQL 18 database, then run:
//
//	TEST_DATABASE_URL=postgres://... go test -tags=integration ./...
//
// It is intentionally read-only; add seeded auth/article/audit tests next to it
// as the integration suite grows.
func TestIntegrationDatabaseReady(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	var id string
	if err := pool.QueryRow(ctx, "select uuidv7()::text").Scan(&id); err != nil {
		t.Fatalf("uuidv7() is required (PostgreSQL 18): %v", err)
	}

	for _, table := range []string{"admin_users", "admin_sessions", "articles", "audit_logs"} {
		var exists bool
		err := pool.QueryRow(ctx, `select exists (
			select 1 from information_schema.tables
			where table_schema = 'public' and table_name = $1
		)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("table %s is missing; run `go run ./cmd/migrate` against TEST_DATABASE_URL first", table)
		}
	}
}
