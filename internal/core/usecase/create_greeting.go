package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/metrics"
	"github.com/craneww/api-poc/internal/core/repository"
	"github.com/google/uuid"
)

// CreateGreetingUseCase is a driving port.
type CreateGreetingUseCase interface {
	Execute(ctx context.Context, params CreateGreetingParams) (*domain.Greeting, error)
}

type CreateGreetingParams struct {
	Name string
}

type CreateGreeting struct {
	metrics      metrics.GreetingMetrics
	unitOfWork   repository.UnitOfWork
	greetingRepo repository.GreetingRepository
	outboxRepo   repository.OutboxRepository
}

func NewCreateGreetingUseCase(
	metrics metrics.GreetingMetrics,
	uow repository.UnitOfWork,
	greetingRepo repository.GreetingRepository,
	outboxRepo repository.OutboxRepository,
) *CreateGreeting {
	return &CreateGreeting{
		metrics:      metrics,
		unitOfWork:   uow,
		greetingRepo: greetingRepo,
		outboxRepo:   outboxRepo,
	}
}

func (uc *CreateGreeting) Execute(ctx context.Context, params CreateGreetingParams) (*domain.Greeting, error) {
	if params.Name == "" {
		return nil, domain.ErrNameRequired
	}

	greeting := domain.NewGreeting(uuid.New().String(), params.Name)

	err := uc.unitOfWork.Do(ctx, func(ctx context.Context) error {
		if err := uc.greetingRepo.Save(ctx, greeting); err != nil {
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

		if err := uc.outboxRepo.Save(ctx, outboxMsg); err != nil {
			return fmt.Errorf("save outbox message: %w", err)
		}

		return nil
	})
	if err != nil {
		uc.metrics.GreetingCreated(ctx, false)
		return nil, fmt.Errorf("create greeting: %w", err)
	}

	uc.metrics.GreetingCreated(ctx, true)
	return greeting, nil
}
