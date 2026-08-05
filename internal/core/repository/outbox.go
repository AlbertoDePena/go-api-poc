package repository

import (
	"context"

	"github.com/AlbertoDePena/go-api-poc/internal/core/domain"
)

// OutboxRepository is a driven port for outbox message persistence.
type OutboxRepository interface {
	// Save persists a new outbox message. When called inside a UnitOfWork,
	// it participates in the same transaction.
	Save(ctx context.Context, message *domain.OutboxMessage) error

	// FindPending returns up to limit unprocessed messages ordered by
	// creation time (oldest first).
	FindPending(ctx context.Context, limit int) ([]*domain.OutboxMessage, error)

	// MarkProcessed sets the processed_at timestamp for the given message ID.
	MarkProcessed(ctx context.Context, id string) error
}
