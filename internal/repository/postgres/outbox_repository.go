package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/AlbertoDePena/go-api-poc/internal/domain"
)

// OutboxRepository is a PostgreSQL-backed implementation of outbox message persistence.
type OutboxRepository struct {
	db *sql.DB
}

// NewOutboxRepository returns an OutboxRepository backed by this DB.
func NewOutboxRepository(db *DB) *OutboxRepository {
	return &OutboxRepository{db: db.pool}
}

func (r *OutboxRepository) Save(ctx context.Context, msg *domain.OutboxMessage) error {
	const query = `
		INSERT INTO outbox_messages (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := execFrom(ctx, r.db).ExecContext(ctx, query,
		msg.ID,
		msg.AggregateType,
		msg.AggregateID,
		msg.EventType,
		msg.Payload,
		msg.CreatedAt,
	)
	return err
}

func (r *OutboxRepository) FindPending(ctx context.Context, limit int) ([]*domain.OutboxMessage, error) {
	const query = `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at
		FROM outbox_messages
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := execFrom(ctx, r.db).QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.OutboxMessage
	for rows.Next() {
		msg, err := scanOutboxMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id string) error {
	const query = `UPDATE outbox_messages SET processed_at = $1 WHERE id = $2`
	_, err := execFrom(ctx, r.db).ExecContext(ctx, query, time.Now(), id)
	return err
}

func scanOutboxMessage(s scanner) (*domain.OutboxMessage, error) {
	var msg domain.OutboxMessage
	var processedAt sql.NullTime

	if err := s.Scan(
		&msg.ID,
		&msg.AggregateType,
		&msg.AggregateID,
		&msg.EventType,
		&msg.Payload,
		&msg.CreatedAt,
		&processedAt,
	); err != nil {
		return nil, err
	}

	if processedAt.Valid {
		msg.ProcessedAt = &processedAt.Time
	}

	return &msg, nil
}
