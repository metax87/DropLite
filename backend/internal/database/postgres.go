package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"droplite/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect 建立到 PostgreSQL 的连接并执行基础健康检查。
func Connect(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	db, err := sql.Open("pgx", cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
