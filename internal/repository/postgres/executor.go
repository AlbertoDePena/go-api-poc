package postgres

import (
	"context"
	"database/sql"
)

// Executor is the common subset of *sql.DB and *sql.Tx that repository
// methods need. This allows the same repository code to run inside or
// outside a transaction.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
