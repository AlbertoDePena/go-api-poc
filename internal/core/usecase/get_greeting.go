package usecase

import (
	"context"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/repository"
)

// GetGreetingUseCase is a driving port.
type GetGreetingUseCase interface {
	Execute(ctx context.Context, id string) (*domain.Greeting, error)
}

type GetGreeting struct {
	greetingRepo repository.GreetingRepository
}

func NewGetGreetingUseCase(repo repository.GreetingRepository) *GetGreeting {
	return &GetGreeting{greetingRepo: repo}
}

func (uc *GetGreeting) Execute(ctx context.Context, id string) (*domain.Greeting, error) {
	return uc.greetingRepo.FindByID(ctx, id)
}
