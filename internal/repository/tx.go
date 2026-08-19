package repository

import "context"

// Transactor runs fn within a single transaction, grouping multiple
// repository operations into one atomic unit. The infrastructure adapter
// decides the isolation mechanism (e.g. a SQL transaction) and never leaks
// a driver type above this interface — callers only ever see a closure.
//
// If fn returns nil, the transaction is committed.
// If fn returns an error (or panics), the transaction is rolled back.
type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
