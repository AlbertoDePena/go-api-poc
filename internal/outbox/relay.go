package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/AlbertoDePena/go-api-poc/internal/domain"
)

// outboxReader is the minimal interface the relay needs from the outbox repository.
type outboxReader interface {
	FindPending(ctx context.Context, limit int) ([]*domain.OutboxMessage, error)
	MarkProcessed(ctx context.Context, id string) error
}

// MessageHandler processes a dispatched outbox message.
// Implementations must be idempotent — the relay guarantees
// at-least-once delivery.
type MessageHandler func(ctx context.Context, aggregateType, aggregateID, eventType string, payload []byte) error

// Relay polls the outbox table for pending messages and dispatches them
// via the configured MessageHandler.
type Relay struct {
	outboxRepo outboxReader
	handler    MessageHandler
	interval   time.Duration
	batchSize  int
}

func NewRelay(
	outboxRepo outboxReader,
	handler MessageHandler,
	interval time.Duration,
	batchSize int,
) *Relay {
	return &Relay{
		outboxRepo: outboxRepo,
		handler:    handler,
		interval:   interval,
		batchSize:  batchSize,
	}
}

// Start runs the polling loop until ctx is cancelled.
// It is intended to be called in a goroutine.
func (r *Relay) Start(ctx context.Context) {
	slog.InfoContext(ctx, "outbox relay started", "interval", r.interval, "batch_size", r.batchSize)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "outbox relay stopped")
			return
		case <-ticker.C:
			r.poll(ctx)
		}
	}
}

func (r *Relay) poll(ctx context.Context) {
	messages, err := r.outboxRepo.FindPending(ctx, r.batchSize)
	if err != nil {
		slog.ErrorContext(ctx, "outbox relay: fetch pending", "error", err)
		return
	}

	for _, msg := range messages {
		if err := r.handler(ctx, msg.AggregateType, msg.AggregateID, msg.EventType, msg.Payload); err != nil {
			slog.ErrorContext(ctx, "outbox relay: handle message",
				"id", msg.ID,
				"event_type", msg.EventType,
				"error", err,
			)
			continue
		}

		if err := r.outboxRepo.MarkProcessed(ctx, msg.ID); err != nil {
			slog.ErrorContext(ctx, "outbox relay: mark processed",
				"id", msg.ID,
				"error", err,
			)
		}

		slog.InfoContext(ctx, "outbox relay: dispatched",
			"id", msg.ID,
			"aggregate_type", msg.AggregateType,
			"aggregate_id", msg.AggregateID,
			"event_type", msg.EventType,
		)
	}
}
