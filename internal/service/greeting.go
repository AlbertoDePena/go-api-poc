package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AlbertoDePena/go-api-poc/internal/domain"
	"github.com/AlbertoDePena/go-api-poc/internal/metrics"
	"github.com/AlbertoDePena/go-api-poc/internal/repository"
	"github.com/google/uuid"
)

// GreetingService is the business-logic seam shared across every binary
// (cmd/api handlers, and any future cmd/ui or cmd/worker). Handlers and
// consumers depend on this interface, never on repository/queue directly.
type GreetingService interface {
	CreateGreeting(ctx context.Context, params CreateGreetingParams) (*domain.Greeting, error)
	GetGreeting(ctx context.Context, id string) (*domain.Greeting, error)
	ListGreetings(ctx context.Context) ([]*domain.Greeting, error)
}

type CreateGreetingParams struct {
	Name string
}

type greetingService struct {
	metrics      metrics.GreetingMetrics
	tx           repository.Transactor
	greetingRepo repository.GreetingRepository
	outboxRepo   repository.OutboxRepository
}

func NewGreetingService(
	metrics metrics.GreetingMetrics,
	tx repository.Transactor,
	greetingRepo repository.GreetingRepository,
	outboxRepo repository.OutboxRepository,
) GreetingService {
	return &greetingService{
		metrics:      metrics,
		tx:           tx,
		greetingRepo: greetingRepo,
		outboxRepo:   outboxRepo,
	}
}

func (s *greetingService) CreateGreeting(ctx context.Context, params CreateGreetingParams) (*domain.Greeting, error) {
	if params.Name == "" {
		return nil, domain.ErrNameRequired
	}

	greeting := domain.NewGreeting(uuid.New().String(), params.Name)

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.greetingRepo.Save(ctx, greeting); err != nil {
			return fmt.Errorf("save greeting: %w", err)
		}

		payload, err := json.Marshal(greeting)
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}

		outboxMsg := domain.NewOutboxMessage(
			uuid.New().String(),
			"greeting",
			greeting.ID,
			"greeting.created",
			payload,
		)

		if err := s.outboxRepo.Save(ctx, outboxMsg); err != nil {
			return fmt.Errorf("save outbox message: %w", err)
		}

		return nil
	})
	if err != nil {
		s.metrics.GreetingCreated(ctx, false)
		return nil, fmt.Errorf("create greeting: %w", err)
	}

	s.metrics.GreetingCreated(ctx, true)
	return greeting, nil
}

func (s *greetingService) GetGreeting(ctx context.Context, id string) (*domain.Greeting, error) {
	greeting, err := s.greetingRepo.FindByID(ctx, id)
	if err != nil {
		s.metrics.GreetingViewed(ctx, false)
		return nil, err
	}

	s.metrics.GreetingViewed(ctx, true)
	return greeting, nil
}

func (s *greetingService) ListGreetings(ctx context.Context) ([]*domain.Greeting, error) {
	greetings, err := s.greetingRepo.FindAll(ctx)
	if err != nil {
		s.metrics.GreetingsListed(ctx, false)
		return nil, err
	}

	s.metrics.GreetingsListed(ctx, true)
	return greetings, nil
}
