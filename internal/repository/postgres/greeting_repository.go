package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/AlbertoDePena/go-api-poc/internal/domain"
)

// GreetingRepository is a PostgreSQL-backed implementation of greeting persistence.
type GreetingRepository struct {
	db *sql.DB
}

// NewGreetingRepository returns a GreetingRepository backed by this DB.
func NewGreetingRepository(db *DB) *GreetingRepository {
	return &GreetingRepository{db: db.pool}
}

func (r *GreetingRepository) Save(ctx context.Context, greeting *domain.Greeting) error {
	const query = `
		INSERT INTO greetings (id, name, message, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name    = excluded.name,
			message = excluded.message`

	_, err := execFrom(ctx, r.db).ExecContext(ctx, query,
		greeting.ID,
		greeting.Name,
		greeting.Message,
		greeting.CreatedAt,
	)
	return err
}

func (r *GreetingRepository) FindByID(ctx context.Context, id string) (*domain.Greeting, error) {
	const query = `SELECT id, name, message, created_at FROM greetings WHERE id = $1`

	g, err := scanGreeting(execFrom(ctx, r.db).QueryRowContext(ctx, query, id))
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

	rows, err := execFrom(ctx, r.db).QueryContext(ctx, query)
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
	if err := s.Scan(&g.ID, &g.Name, &g.Message, &g.CreatedAt); err != nil {
		return nil, err
	}
	return &g, nil
}
