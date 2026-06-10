package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/repository"
)

var _ repository.OutboxRepository = (*OutboxRepository)(nil)

// OutboxRepository is a SQLite-backed implementation of the
// repository.OutboxRepository driven port.
type OutboxRepository struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

// NewOutboxRepository returns an OutboxRepository backed by this DB.
func NewOutboxRepository(db *DB) *OutboxRepository {
	return &OutboxRepository{readDB: db.readDB, writeDB: db.writeDB}
}

func (r *OutboxRepository) Save(ctx context.Context, msg *domain.OutboxMessage) error {
	const query = `
		INSERT INTO outbox_messages (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := writerFrom(ctx, r.writeDB).ExecContext(ctx, query,
		msg.ID,
		msg.AggregateType,
		msg.AggregateID,
		msg.EventType,
		msg.Payload,
		msg.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

func (r *OutboxRepository) FindPending(ctx context.Context, limit int) ([]*domain.OutboxMessage, error) {
	const query = `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at
		FROM outbox_messages
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT ?`

	rows, err := r.readDB.QueryContext(ctx, query, limit)
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
	const query = `UPDATE outbox_messages SET processed_at = ? WHERE id = ?`
	_, err := r.writeDB.ExecContext(ctx, query, time.Now().Format(time.RFC3339Nano), id)
	return err
}

func scanOutboxMessage(s scanner) (*domain.OutboxMessage, error) {
	var msg domain.OutboxMessage
	var createdAt string
	var processedAt sql.NullString

	if err := s.Scan(
		&msg.ID,
		&msg.AggregateType,
		&msg.AggregateID,
		&msg.EventType,
		&msg.Payload,
		&createdAt,
		&processedAt,
	); err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	msg.CreatedAt = t

	if processedAt.Valid {
		pt, err := time.Parse(time.RFC3339Nano, processedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse processed_at: %w", err)
		}
		msg.ProcessedAt = &pt
	}

	return &msg, nil
}
