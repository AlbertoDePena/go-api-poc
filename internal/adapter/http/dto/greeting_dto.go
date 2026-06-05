package dto

import (
	"time"

	"github.com/craneww/api-poc/internal/core/domain"
	"github.com/craneww/api-poc/internal/core/usecase"
)

// CreateGreetingRequest is the HTTP request body for creating a greeting.
type CreateGreetingRequest struct {
	Name string `json:"name" example:"World"`
}

// ToParams translates the HTTP DTO into use case params.
func (r CreateGreetingRequest) ToParams() usecase.CreateGreetingParams {
	return usecase.CreateGreetingParams{
		Name: r.Name,
	}
}

// GreetingResponse is the HTTP response body for a greeting.
type GreetingResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string    `json:"name" example:"World"`
	Message   string    `json:"message" example:"Hello, World!"`
	CreatedAt time.Time `json:"created_at" example:"2026-06-04T12:00:00Z"`
}

// GreetingFromDomain translates a domain entity into an HTTP response DTO.
func GreetingFromDomain(g *domain.Greeting) GreetingResponse {
	return GreetingResponse{
		ID:        g.ID,
		Name:      g.Name,
		Message:   g.Message,
		CreatedAt: g.CreatedAt,
	}
}

// GreetingsFromDomain translates a slice of domain entities.
func GreetingsFromDomain(greetings []*domain.Greeting) []GreetingResponse {
	result := make([]GreetingResponse, len(greetings))
	for i, g := range greetings {
		result[i] = GreetingFromDomain(g)
	}
	return result
}

// ErrorResponse is a generic error response.
type ErrorResponse struct {
	Error string `json:"error" example:"greeting not found"`
}
