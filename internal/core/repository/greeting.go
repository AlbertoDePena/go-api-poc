package repository

import (
	"context"

	"github.com/AlbertoDePena/go-api-poc/internal/core/domain"
)

// GreetingRepository is a driven port.
type GreetingRepository interface {
	Save(ctx context.Context, greeting *domain.Greeting) error

	FindByID(ctx context.Context, id string) (*domain.Greeting, error)

	FindAll(ctx context.Context) ([]*domain.Greeting, error)
}
