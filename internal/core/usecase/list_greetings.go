package usecase

import (
	"context"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/repository"
)

// ListGreetingsUseCase is a driving port.
type ListGreetingsUseCase interface {
	Execute(ctx context.Context) ([]*domain.Greeting, error)
}

type ListGreetings struct {
	greetingRepo repository.GreetingRepository
}

func NewListGreetingsUseCase(repo repository.GreetingRepository) *ListGreetings {
	return &ListGreetings{greetingRepo: repo}
}

func (uc *ListGreetings) Execute(ctx context.Context) ([]*domain.Greeting, error) {
	return uc.greetingRepo.FindAll(ctx)
}
