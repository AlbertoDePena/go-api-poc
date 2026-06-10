package usecase

import (
	"context"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/metrics"
	"github.com/craneww/api-poc/internal/core/repository"
)

// ListGreetingsUseCase is a driving port.
type ListGreetingsUseCase interface {
	Execute(ctx context.Context) ([]*domain.Greeting, error)
}

type ListGreetings struct {
	metrics      metrics.GreetingMetrics
	greetingRepo repository.GreetingRepository
}

func NewListGreetingsUseCase(metrics metrics.GreetingMetrics, repo repository.GreetingRepository) *ListGreetings {
	return &ListGreetings{metrics: metrics, greetingRepo: repo}
}

func (uc *ListGreetings) Execute(ctx context.Context) ([]*domain.Greeting, error) {
	greetings, err := uc.greetingRepo.FindAll(ctx)
	if err != nil {
		uc.metrics.GreetingsListed(ctx, false)
		return nil, err
	}

	uc.metrics.GreetingsListed(ctx, true)
	return greetings, nil
}
