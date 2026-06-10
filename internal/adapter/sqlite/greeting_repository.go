package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/repository"
)

var _ repository.GreetingRepository = (*GreetingRepository)(nil)

// GreetingRepository is a SQLite-backed implementation of the
// repository.GreetingRepository driven port.
type GreetingRepository struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

// NewGreetingRepository returns a GreetingRepository backed by this DB.
func NewGreetingRepository(db *DB) *GreetingRepository {
	return &GreetingRepository{readDB: db.readDB, writeDB: db.writeDB}
}

func (r *GreetingRepository) Save(ctx context.Context, greeting *domain.Greeting) error {
	const query = `
		INSERT INTO greetings (id, name, message, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name    = excluded.name,
			message = excluded.message`

	_, err := writerFrom(ctx, r.writeDB).ExecContext(ctx, query,
		greeting.ID,
		greeting.Name,
		greeting.Message,
		greeting.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

func (r *GreetingRepository) FindByID(ctx context.Context, id string) (*domain.Greeting, error) {
	const query = `SELECT id, name, message, created_at FROM greetings WHERE id = ?`

	g, err := scanGreeting(r.readDB.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrGreetingNotFound
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (r *GreetingRepository) FindAll(ctx context.Context) ([]*domain.Greeting, error) {
	const query = `SELECT id, name, message, created_at FROM greetings ORDER BY created_at DESC`

	rows, err := r.readDB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Greeting
	for rows.Next() {
		g, err := scanGreeting(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanGreeting(s scanner) (*domain.Greeting, error) {
	var g domain.Greeting
	var createdAt string
	if err := s.Scan(&g.ID, &g.Name, &g.Message, &createdAt); err != nil {
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	g.CreatedAt = t
	return &g, nil
}
