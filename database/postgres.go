package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"order-management/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgresDB(cfg *config.Config) (*sql.DB, error) {
	escapedUser := url.QueryEscape(cfg.DBUser)
	escapedPassword := url.QueryEscape(cfg.DBPassword)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		escapedUser, escapedPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening DB pool: %w", err)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(15 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return db, nil
}
