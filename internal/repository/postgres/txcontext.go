package postgres

import (
	"context"
	"database/sql"
)

type txCtxKey struct{}

// withTx returns a child context carrying the given transaction.
func withTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// txFromContext retrieves the transaction from context, or nil.
func txFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txCtxKey{}).(*sql.Tx)
	return tx
}

// execFrom returns the transaction from context if present, otherwise falls
// back to the pool. Reads and writes both route through it, so a query issued
// inside WithinTx participates in the active transaction.
func execFrom(ctx context.Context, fallback *sql.DB) Executor {
	if tx := txFromContext(ctx); tx != nil {
		return tx
	}
	return fallback
}
