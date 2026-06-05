package usecase

import (
	"context"
	"fmt"

	"github.com/craneww/api-poc/internal/core/domain"
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
	greetingRepo repository.GreetingRepository
}

func NewCreateGreetingUseCase(repo repository.GreetingRepository) *CreateGreeting {
	return &CreateGreeting{greetingRepo: repo}
}

func (uc *CreateGreeting) Execute(ctx context.Context, params CreateGreetingParams) (*domain.Greeting, error) {
	if params.Name == "" {
		return nil, domain.ErrNameRequired
	}

	greeting := domain.NewGreeting(uuid.New().String(), params.Name)

	if err := uc.greetingRepo.Save(ctx, greeting); err != nil {
		return nil, fmt.Errorf("create greeting: %w", err)
	}

	return greeting, nil
}
