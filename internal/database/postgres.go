package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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

	// Neon's pooler endpoint (and any pgbouncer in transaction mode) cannot
	// reuse server-side prepared statements across pool checkouts. pgx's
	// default cache_statement mode prepares + caches by name, which breaks
	// against a transaction-mode pooler with errors like
	// "prepared statement \"stmtcache_<n>\" already exists" or
	// "prepared statement \"stmtcache_<n>\" does not exist".
	//
	// cache_describe is the most efficient pgbouncer-safe mode: it caches
	// the protocol-level statement *description* (column types, OIDs) on
	// the client but never issues a server-side PREPARE. Each query goes
	// out as a single Parse+Bind+Execute round-trip.
	//
	// We apply this automatically when the URL looks like a pooler endpoint
	// so direct-connection deployments retain full prepared-statement perf.
	if isPgBouncerPooler(databaseURL) {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
		// Statement cache capacity must be zero in this mode — pgx's default
		// statement-name cache is for cache_statement mode, not cache_describe.
		cfg.ConnConfig.StatementCacheCapacity = 0
		cfg.ConnConfig.DescriptionCacheCapacity = 512
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

// isPgBouncerPooler heuristically detects connection strings that target a
// transaction-mode pooler. Neon names the pooler host with a "-pooler"
// suffix; explicit ?pgbouncer=true is the supabase/cnpg convention.
func isPgBouncerPooler(databaseURL string) bool {
	lower := strings.ToLower(databaseURL)
	if strings.Contains(lower, "-pooler.") {
		return true
	}
	if strings.Contains(lower, "pgbouncer=true") {
		return true
	}
	return false
}
