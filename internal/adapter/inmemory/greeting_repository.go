package inmemory

import (
	"context"
	"sync"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/repository"
)

var _ repository.GreetingRepository = (*GreetingRepository)(nil)

type GreetingRepository struct {
	mu   sync.RWMutex
	data map[string]*domain.Greeting
}

func NewGreetingRepository() *GreetingRepository {
	return &GreetingRepository{
		data: make(map[string]*domain.Greeting),
	}
}

func (r *GreetingRepository) Save(_ context.Context, greeting *domain.Greeting) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[greeting.ID] = greeting
	return nil
}

func (r *GreetingRepository) FindByID(_ context.Context, id string) (*domain.Greeting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, ok := r.data[id]
	if !ok {
		return nil, domain.ErrGreetingNotFound
	}
	return g, nil
}

func (r *GreetingRepository) FindAll(_ context.Context) ([]*domain.Greeting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.Greeting, 0, len(r.data))
	for _, g := range r.data {
		result = append(result, g)
	}
	return result, nil
}
