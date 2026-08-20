package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps a single PostgreSQL connection pool. Unlike the SQLite adapter,
// Postgres has no single-writer constraint, so one pool serves both reads
// and writes concurrently.
type DB struct {
	pool *sql.DB
}

// Open connects to the PostgreSQL database at databaseURL (a standard DSN,
// e.g. postgres://user:pass@host:5432/db?sslmode=disable), verifies
// connectivity, and runs all pending schema migrations before returning.
//
// ctx governs the whole startup sequence, so a caller that cancels it (e.g. on
// SIGTERM during boot) interrupts a slow connect or migration. Connectivity is
// additionally bounded by a short internal timeout; migrations run under ctx
// directly so a large migration set is not capped by that connectivity budget.
func Open(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	maxConns := max(4, runtime.NumCPU())
	pool.SetMaxOpenConns(maxConns)
	pool.SetMaxIdleConns(maxConns)
	pool.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &DB{pool: pool}, nil
}

// Close closes the connection pool.
func (db *DB) Close() error {
	return db.pool.Close()
}

// Ping verifies that the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.PingContext(ctx)
}
