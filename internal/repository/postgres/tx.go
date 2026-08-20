package postgres

import (
	"context"
	"fmt"
)

// Transactor implements atomic operations using a PostgreSQL transaction.
type Transactor struct {
	db *DB
}

func NewTransactor(db *DB) *Transactor {
	return &Transactor{db: db}
}

func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// If already inside a transaction, reuse it rather than opening a nested
	// one (Postgres has no true nested transactions without savepoints).
	if txFromContext(ctx) != nil {
		return fn(ctx)
	}

	tx, err := t.db.pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = fmt.Errorf("commit tx: %w", commitErr)
		}
	}()

	err = fn(withTx(ctx, tx))
	return err
}
