package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/craneww/api-poc/internal/core/repository"
)

// MessageHandler processes a dispatched outbox message.
// Implementations must be idempotent — the relay guarantees
// at-least-once delivery.
type MessageHandler func(ctx context.Context, aggregateType, aggregateID, eventType string, payload []byte) error

// Relay polls the outbox table for pending messages and dispatches them
// via the configured MessageHandler.
type Relay struct {
	outboxRepo repository.OutboxRepository
	handler    MessageHandler
	interval   time.Duration
	batchSize  int
}

func NewRelay(
	outboxRepo repository.OutboxRepository,
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
	slog.Info("outbox relay started", "interval", r.interval, "batch_size", r.batchSize)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox relay stopped")
			return
		case <-ticker.C:
			r.poll(ctx)
		}
	}
}

func (r *Relay) poll(ctx context.Context) {
	messages, err := r.outboxRepo.FindPending(ctx, r.batchSize)
	if err != nil {
		slog.Error("outbox relay: fetch pending", "error", err)
		return
	}

	for _, msg := range messages {
		if err := r.handler(ctx, msg.AggregateType, msg.AggregateID, msg.EventType, msg.Payload); err != nil {
			slog.Error("outbox relay: handle message",
				"id", msg.ID,
				"event_type", msg.EventType,
				"error", err,
			)
			continue
		}

		if err := r.outboxRepo.MarkProcessed(ctx, msg.ID); err != nil {
			slog.Error("outbox relay: mark processed",
				"id", msg.ID,
				"error", err,
			)
		}

		slog.Info("outbox relay: dispatched",
			"id", msg.ID,
			"aggregate_type", msg.AggregateType,
			"aggregate_id", msg.AggregateID,
			"event_type", msg.EventType,
		)
	}
}
