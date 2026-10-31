package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Postgres holds the connection settings. DATABASE_URL is an optional override
// for managed databases or environments that already provide a DSN.
type Postgres struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	Database string `env:"POSTGRES_DB" envDefault:"admin"`
	User     string `env:"POSTGRES_USER" envDefault:"admin"`
	Password string `env:"POSTGRES_PASSWORD"`
	SSLMode  string `env:"POSTGRES_SSLMODE" envDefault:"disable"`
	URL      string `env:"DATABASE_URL"`
}

func (p Postgres) DSN() (string, error) {
	if strings.TrimSpace(p.URL) != "" {
		return p.URL, nil
	}
	if err := p.validateParts(); err != nil {
		return "", err
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:   "/" + p.Database,
	}
	if p.Password != "" {
		u.User = url.UserPassword(p.User, p.Password)
	} else {
		u.User = url.User(p.User)
	}
	q := url.Values{}
	q.Set("sslmode", p.SSLMode)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p Postgres) Validate() error {
	if strings.TrimSpace(p.URL) != "" {
		return nil
	}
	return p.validateParts()
}

func (p Postgres) validateParts() error {
	if strings.TrimSpace(p.Host) == "" {
		return fmt.Errorf("POSTGRES_HOST is required")
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("POSTGRES_PORT must be between 1 and 65535")
	}
	if strings.TrimSpace(p.Database) == "" {
		return fmt.Errorf("POSTGRES_DB is required")
	}
	if strings.TrimSpace(p.User) == "" {
		return fmt.Errorf("POSTGRES_USER is required")
	}
	switch p.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("POSTGRES_SSLMODE has unsupported value %q", p.SSLMode)
	}
	return nil
}

// Redis is reserved for P1 caching/rate limiting. P0 validates it but does not
// open a connection.
type Redis struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT" envDefault:"6379"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
	Username string `env:"REDIS_USERNAME"`
	Password string `env:"REDIS_PASSWORD"`
}

func (r Redis) Addr() string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

func (r Redis) Validate() error {
	if strings.TrimSpace(r.Host) == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	if r.Port < 1 || r.Port > 65535 {
		return fmt.Errorf("REDIS_PORT must be between 1 and 65535")
	}
	if r.DB < 0 {
		return fmt.Errorf("REDIS_DB must be >= 0")
	}
	return nil
}

// Config is the complete runtime configuration. Everything comes from the
// environment so the same binary runs locally, in CI, and on the single VM.
type Config struct {
	AppEnv      string `env:"APP_ENV" envDefault:"development"`
	HTTPAddr    string `env:"HTTP_ADDR" envDefault:":8080"`
	MetricsAddr string `env:"METRICS_ADDR" envDefault:"127.0.0.1:9090"`

	Postgres Postgres
	Redis    Redis

	// AdminWebURL is used to build one-time activation links.
	AdminWebURL string `env:"ADMIN_WEB_URL" envDefault:"http://localhost:5173"`

	JWTSecret   string        `env:"JWT_SECRET,required"`
	JWTIssuer   string        `env:"JWT_ISSUER" envDefault:"homebrew-api"`
	JWTAudience string        `env:"JWT_AUDIENCE" envDefault:"admin-web"`
	AccessTTL   time.Duration `env:"ACCESS_TTL" envDefault:"8h"`

	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

func Load() (Config, error) {
	loadDotEnv()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadPostgres is used by cmd/migrate, which needs database settings but not
// JWT settings.
func LoadPostgres() (Postgres, error) {
	loadDotEnv()

	var pg Postgres
	if err := env.Parse(&pg); err != nil {
		return Postgres{}, fmt.Errorf("parse postgres environment: %w", err)
	}
	if err := pg.Validate(); err != nil {
		return Postgres{}, err
	}
	return pg, nil
}

func (c Config) Validate() error {
	if err := c.Postgres.Validate(); err != nil {
		return err
	}
	if err := c.Redis.Validate(); err != nil {
		return err
	}
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
	if c.IsProduction() && c.Postgres.URL == "" && c.Postgres.Password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD or DATABASE_URL is required in production")
	}
	return nil
}

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func loadDotEnv() {
	// Local convenience only: a .env file never overrides real environment
	// variables, and production deployments should inject env directly.
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}
}
