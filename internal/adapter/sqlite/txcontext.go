package sqlite

import (
	"context"
	"database/sql"
)

type ctxKey struct{}

// withTx returns a child context carrying the given transaction.
func withTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, ctxKey{}, tx)
}

// txFromContext retrieves the transaction from context, or nil.
func txFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(ctxKey{}).(*sql.Tx)
	return tx
}

// writerFrom returns the transaction from context if present,
// otherwise falls back to the provided *sql.DB.
func writerFrom(ctx context.Context, fallback *sql.DB) Executor {
	if tx := txFromContext(ctx); tx != nil {
		return tx
	}
	return fallback
}
