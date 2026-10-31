package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/config"
	"github.com/example/homebrew-api-starter/internal/platform/db"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/platform/logx"
)

// version is injected with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := logx.New(cfg.LogLevel)
	slog.SetDefault(logger)

	manager, err := jwt.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTTL)
	if err != nil {
		return fmt.Errorf("create jwt manager: %w", err)
	}

	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err = db.Open(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		defer pool.Close()
	} else {
		logger.Warn("DATABASE_URL is empty; /readyz will report unavailable")
	}

	var permissions authpkg.PermissionStore
	if len(cfg.DevPermissions) > 0 {
		if cfg.IsProduction() {
			return errors.New("DEV_PERMISSIONS must not be set in production")
		}
		permissions = authpkg.NewStaticPermissionStore(cfg.DevPermissions)
		logger.Warn("static dev permissions enabled", "permissions", cfg.DevPermissions)
	}

	handler, reg, err := newApp(dependencies{
		Logger:      logger,
		JWT:         manager,
		DB:          pool,
		Permissions: permissions,
		Version:     version,
	})
	if err != nil {
		return fmt.Errorf("build app: %w", err)
	}
	logger.Info("route contract validated", "routes", reg.Count())

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.HTTPAddr, "env", cfg.AppEnv, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-stopCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("server stopped")
	return nil
}
