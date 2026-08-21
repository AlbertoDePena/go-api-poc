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
// e.g. postgres://user:pass@host:5432/db?sslmode=disable) and verifies
// connectivity. It deliberately does NOT run schema migrations: the runtime
// binaries (cmd/api, cmd/outbox-relay) pass a least-privilege app account that
// has no DDL rights. Migrations are applied out-of-band by the cmd/migrate
// binary via Migrate, which uses a separate privileged DSN.
//
// ctx governs the connect; connectivity is additionally bounded by a short
// internal timeout.
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
