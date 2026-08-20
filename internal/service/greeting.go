package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AlbertoDePena/go-api-poc/internal/domain"
	"github.com/google/uuid"
)

// greetingRepository is declared here, next to its only caller — not in
// internal/repository and not in internal/domain.
type greetingRepository interface {
	Save(ctx context.Context, greeting *domain.Greeting) error
	FindByID(ctx context.Context, id string) (*domain.Greeting, error)
	FindAll(ctx context.Context) ([]*domain.Greeting, error)
}

// outboxWriter is all greetingService needs — just the atomic write side.
type outboxWriter interface {
	Save(ctx context.Context, message *domain.OutboxMessage) error
}

// greetingMetrics records business-level greeting metrics.
type greetingMetrics interface {
	GreetingCreated(ctx context.Context, success bool)
	GreetingsListed(ctx context.Context, success bool)
	GreetingViewed(ctx context.Context, success bool)
}

// GreetingService is the business-logic seam shared across every binary
// (cmd/api handlers, and any future cmd/ui or cmd/worker). Handlers and
// consumers depend on this concrete type, never on repository/queue directly.
type GreetingService struct {
	metrics      greetingMetrics
	tx           transactor
	greetingRepo greetingRepository
	outboxRepo   outboxWriter
}

type CreateGreetingParams struct {
	Name string
}

func NewGreetingService(
	metrics greetingMetrics,
	tx transactor,
	greetingRepo greetingRepository,
	outboxRepo outboxWriter,
) *GreetingService {
	return &GreetingService{
		metrics:      metrics,
		tx:           tx,
		greetingRepo: greetingRepo,
		outboxRepo:   outboxRepo,
	}
}

func (s *GreetingService) CreateGreeting(ctx context.Context, params CreateGreetingParams) (*domain.Greeting, error) {
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

func (s *GreetingService) GetGreeting(ctx context.Context, id string) (*domain.Greeting, error) {
	greeting, err := s.greetingRepo.FindByID(ctx, id)
	if err != nil {
		s.metrics.GreetingViewed(ctx, false)
		return nil, err
	}

	s.metrics.GreetingViewed(ctx, true)
	return greeting, nil
}

func (s *GreetingService) ListGreetings(ctx context.Context) ([]*domain.Greeting, error) {
	greetings, err := s.greetingRepo.FindAll(ctx)
	if err != nil {
		s.metrics.GreetingsListed(ctx, false)
		return nil, err
	}

	s.metrics.GreetingsListed(ctx, true)
	return greetings, nil
}
