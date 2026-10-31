package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/example/homebrew-api-starter/internal/config"
	"github.com/example/homebrew-api-starter/migrations"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	pg, err := config.LoadPostgres()
	if err != nil {
		return fmt.Errorf("load postgres config: %w", err)
	}
	dsn, err := pg.DSN()
	if err != nil {
		return fmt.Errorf("build postgres dsn: %w", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	upCtx, cancelUp := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelUp()
	if err := goose.UpContext(upCtx, db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
