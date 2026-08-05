package sqlite

import (
	"context"
	"fmt"

	"github.com/AlbertoDePena/go-api-poc/internal/core/repository"
)

var _ repository.UnitOfWork = (*UnitOfWork)(nil)

// UnitOfWork implements repository.UnitOfWork using a SQLite transaction
// on the write connection pool.
type UnitOfWork struct {
	db *DB
}

func NewUnitOfWork(db *DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// If already inside a transaction, just execute without nesting.
	// Nesting would deadlock on the single-writer connection pool.
	if txFromContext(ctx) != nil {
		return fn(ctx)
	}

	tx, err := u.db.writeDB.BeginTx(ctx, nil)
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
