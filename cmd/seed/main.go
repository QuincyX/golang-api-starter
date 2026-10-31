package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/example/homebrew-api-starter/internal/audit"
	"github.com/example/homebrew-api-starter/internal/config"
	"github.com/example/homebrew-api-starter/internal/platform/db"
	"github.com/example/homebrew-api-starter/internal/platform/logx"
	accountsvc "github.com/example/homebrew-api-starter/internal/service/account"
	"github.com/example/homebrew-api-starter/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger := logx.New(cfg.LogLevel)

	email := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	name := os.Getenv("BOOTSTRAP_ADMIN_NAME")
	if name == "" {
		name = "Admin"
	}
	if email == "" || password == "" {
		logger.Info("bootstrap admin env is empty; nothing to seed")
		return nil
	}

	dsn, err := cfg.Postgres.DSN()
	if err != nil {
		return fmt.Errorf("build postgres dsn: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	svc := accountsvc.NewService(store.New(pool), audit.NewRecorder(pool), cfg.AdminWebURL)
	seeded, err := svc.Seed(ctx, email, password, name)
	if err != nil {
		return err
	}
	if seeded {
		logger.Info("bootstrap super admin created", "email", email)
	} else {
		logger.Info("admin users already exist; seed skipped")
	}
	return nil
}
