package db

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func NewPool(dsn string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("db: parse config: %v", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnIdleTime = 20 * time.Second
	cfg.ConnConfig.ConnectTimeout = 10 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("db: connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("db: ping: %v", err)
	}

	runMigrations(pool)

	return pool
}

func runMigrations(pool *pgxpool.Pool) {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func(db *sql.DB) {
		if err := db.Close(); err != nil {
			log.Printf("db: migration conn close: %v", err)
		}
	}(sqlDB)

	goose.SetBaseFS(Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("db: goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		log.Fatalf("db: migrate: %v", err)
	}
}
