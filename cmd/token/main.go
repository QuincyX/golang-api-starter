// Command token mints a development access token backed by a real session.
// It is a local debugging tool, not a production login endpoint.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/example/homebrew-api-starter/internal/config"
	"github.com/example/homebrew-api-starter/internal/platform/db"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/store"
)

func main() {
	email := flag.String("email", "", "administrator email (required)")
	ua := flag.String("ua", "cmd-token", "session user agent")
	flag.Parse()

	if *email == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/token -email admin@example.com")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	manager, err := jwt.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTTL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "jwt:", err)
		os.Exit(1)
	}
	dsn, err := cfg.Postgres.DSN()
	if err != nil {
		fmt.Fprintln(os.Stderr, "postgres:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	defer pool.Close()

	st := store.New(pool)
	user, err := st.Queries().UserByEmail(ctx, *email)
	if err != nil {
		fmt.Fprintln(os.Stderr, "user:", err)
		os.Exit(1)
	}
	if user.Status != "active" {
		fmt.Fprintln(os.Stderr, "user is not active")
		os.Exit(1)
	}
	sessionID, err := st.Queries().CreateSession(ctx, user.ID, *ua, "", time.Now().Add(cfg.AccessTTL))
	if err != nil {
		fmt.Fprintln(os.Stderr, "session:", err)
		os.Exit(1)
	}
	token, expiresAt, err := manager.Sign(user.ID, sessionID, user.TokenVersion)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sign:", err)
		os.Exit(1)
	}

	fmt.Println(token)
	fmt.Fprintf(os.Stderr, "email=%s session=%s expires_at=%s\n", user.Email, sessionID, expiresAt.Format(time.RFC3339))
}
