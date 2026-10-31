package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config is the complete runtime configuration. Everything comes from the
// environment so the same binary runs locally, in CI, and on the single VM.
type Config struct {
	AppEnv      string `env:"APP_ENV" envDefault:"development"`
	HTTPAddr    string `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL string `env:"DATABASE_URL"`

	JWTSecret   string        `env:"JWT_SECRET,required"`
	JWTIssuer   string        `env:"JWT_ISSUER" envDefault:"homebrew-api"`
	JWTAudience string        `env:"JWT_AUDIENCE" envDefault:"admin-web"`
	AccessTTL   time.Duration `env:"ACCESS_TTL" envDefault:"15m"`

	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// DevPermissions is only used when APP_ENV != production. It lets the P0
	// skeleton exercise permission checks before the RBAC tables are wired in.
	DevPermissions []string `env:"DEV_PERMISSIONS" envSeparator:","`
}

func Load() (Config, error) {
	// Local convenience only: a .env file never overrides real environment
	// variables, and production deployments should inject env directly.
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes (got %d)", len(c.JWTSecret))
	}
	if strings.TrimSpace(c.JWTIssuer) == "" {
		return fmt.Errorf("JWT_ISSUER is required")
	}
	if strings.TrimSpace(c.JWTAudience) == "" {
		return fmt.Errorf("JWT_AUDIENCE is required")
	}
	if c.AccessTTL <= 0 {
		return fmt.Errorf("ACCESS_TTL must be positive")
	}
	if c.IsProduction() && strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required in production")
	}
	if c.IsProduction() && len(c.DevPermissions) > 0 {
		return fmt.Errorf("DEV_PERMISSIONS must not be set in production")
	}
	return nil
}

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}
