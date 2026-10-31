// Command token mints a development access token. It is a local debugging
// tool, not a production login endpoint.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/example/homebrew-api-starter/internal/config"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
)

func main() {
	sub := flag.String("sub", "", "user id (default: new UUID)")
	sid := flag.String("sid", "", "session id (default: new UUID)")
	ver := flag.Int("ver", 0, "token version")
	flag.Parse()

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

	if *sub == "" {
		*sub = newUUID()
	}
	if *sid == "" {
		*sid = newUUID()
	}

	token, expiresAt, err := manager.Sign(*sub, *sid, *ver)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sign:", err)
		os.Exit(1)
	}

	fmt.Println(token)
	fmt.Fprintf(os.Stderr, "sub=%s sid=%s ver=%d expires_at=%s\n",
		*sub, *sid, *ver, expiresAt.Format(time.RFC3339))
}

func newUUID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
