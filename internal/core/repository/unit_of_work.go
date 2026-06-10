package repository

import "context"

// UnitOfWork is a driven port that groups multiple repository
// operations into a single atomic unit. The infrastructure adapter
// decides the isolation mechanism (e.g. SQL transaction).
//
// If the function returns nil, the unit is committed.
// If the function returns an error (or panics), the unit is rolled back.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
