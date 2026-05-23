package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolOptions tunes the pgxpool sizing for an environment. Zero values fall
// back to the package defaults (MaxConns=25, MinConns=5).
type PoolOptions struct {
	MaxConns int32
	MinConns int32
}

// NewPostgres opens a pgxpool against databaseURL and verifies connectivity
// with a Ping. Use NewPostgresWithOptions to override pool sizing from env.
func NewPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return NewPostgresWithOptions(ctx, databaseURL, PoolOptions{})
}

func NewPostgresWithOptions(ctx context.Context, databaseURL string, opts PoolOptions) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	if opts.MaxConns > 0 {
		cfg.MaxConns = opts.MaxConns
	}
	if opts.MinConns > 0 {
		cfg.MinConns = opts.MinConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}
