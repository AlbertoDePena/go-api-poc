package service

import "context"

// transactor is declared here because service is the consumer.
type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
