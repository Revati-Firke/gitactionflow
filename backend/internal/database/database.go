package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultConnectTimeout = 5 * time.Second
	defaultMaxConns       = 10
	defaultMinConns       = 1
	defaultMaxConnIdle    = 5 * time.Minute
)

// NewPool creates a pgx connection pool from DATABASE_URL.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	cfg.MaxConns = defaultMaxConns
	cfg.MinConns = defaultMinConns
	cfg.MaxConnIdleTime = defaultMaxConnIdle
	cfg.ConnConfig.ConnectTimeout = defaultConnectTimeout

	connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer pingCancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Ping checks that the database is reachable.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	pingCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()
	return pool.Ping(pingCtx)
}
