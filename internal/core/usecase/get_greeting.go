package usecase

import (
	"context"

	"github.com/AlbertoDePena/go-api-poc/internal/core/domain"
	"github.com/AlbertoDePena/go-api-poc/internal/core/metrics"
	"github.com/AlbertoDePena/go-api-poc/internal/core/repository"
)

// GetGreetingUseCase is a driving port.
type GetGreetingUseCase interface {
	Execute(ctx context.Context, id string) (*domain.Greeting, error)
}

type GetGreeting struct {
	metrics      metrics.GreetingMetrics
	greetingRepo repository.GreetingRepository
}

func NewGetGreetingUseCase(metrics metrics.GreetingMetrics, repo repository.GreetingRepository) *GetGreeting {
	return &GetGreeting{metrics: metrics, greetingRepo: repo}
}

func (uc *GetGreeting) Execute(ctx context.Context, id string) (*domain.Greeting, error) {
	greeting, err := uc.greetingRepo.FindByID(ctx, id)
	if err != nil {
		uc.metrics.GreetingViewed(ctx, false)
		return nil, err
	}

	uc.metrics.GreetingViewed(ctx, true)
	return greeting, nil
}
