package config

import (
	"net/url"
	"strings"
	"testing"

	"github.com/caarlos0/env/v11"
)

func TestPostgresDSNEscapesCredentials(t *testing.T) {
	pg := Postgres{
		Host:     "db.internal",
		Port:     5432,
		Database: "homebrew",
		User:     "admin",
		Password: "p@ss:word",
		SSLMode:  "require",
	}

	dsn, err := pg.DSN()
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	if u.Host != "db.internal:5432" || u.Path != "/homebrew" {
		t.Fatalf("unexpected DSN parts: %s", dsn)
	}
	if u.User.Username() != "admin" {
		t.Fatalf("username = %q", u.User.Username())
	}
	if password, _ := u.User.Password(); password != "p@ss:word" {
		t.Fatalf("password = %q", password)
	}
	if got := u.Query().Get("sslmode"); got != "require" {
		t.Fatalf("sslmode = %q", got)
	}
}

func TestPostgresDSNURLOverride(t *testing.T) {
	const override = "postgres://user:pass@example.com:6543/app?sslmode=verify-full"
	pg := Postgres{URL: override}
	dsn, err := pg.DSN()
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	if dsn != override {
		t.Fatalf("dsn = %q, want override", dsn)
	}
}

func TestPostgresValidateRejectsBadPort(t *testing.T) {
	pg := Postgres{Host: "localhost", Port: 0, Database: "admin", User: "admin", SSLMode: "disable"}
	if err := pg.Validate(); err == nil {
		t.Fatal("expected invalid port to fail")
	}
}

func TestRedisAddrAndValidate(t *testing.T) {
	redis := Redis{Host: "cache.internal", Port: 6380, DB: 2}
	if got := redis.Addr(); got != "cache.internal:6380" {
		t.Fatalf("Addr() = %q", got)
	}
	if err := redis.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	redis.Port = 70000
	if err := redis.Validate(); err == nil {
		t.Fatal("expected invalid port to fail")
	}
}

func TestNestedEnvParsing(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "db.internal")
	t.Setenv("POSTGRES_PORT", "6543")
	t.Setenv("POSTGRES_DB", "homebrew")
	t.Setenv("POSTGRES_USER", "app")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("REDIS_HOST", "redis.internal")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		t.Fatalf("env.Parse: %v", err)
	}
	if cfg.Postgres.Host != "db.internal" || cfg.Postgres.Port != 6543 || cfg.Postgres.Database != "homebrew" {
		t.Fatalf("unexpected postgres config: %+v", cfg.Postgres)
	}
	if cfg.Redis.Host != "redis.internal" || cfg.Redis.Port != 6380 || cfg.Redis.DB != 3 {
		t.Fatalf("unexpected redis config: %+v", cfg.Redis)
	}
}
