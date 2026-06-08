package http

import (
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/craneww/api-poc/internal/adapter/http/handler"
	customMiddleware "github.com/craneww/api-poc/internal/adapter/http/middleware"
	"github.com/craneww/api-poc/internal/core/usecase"
)

func NewServer(
	tokenVerifier *oidc.IDTokenVerifier,
	createGreeting usecase.CreateGreetingUseCase,
	getGreeting usecase.GetGreetingUseCase,
	listGreetings usecase.ListGreetingsUseCase,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(otelhttp.NewMiddleware("api-poc"))

	healthHandler := handler.NewHealthHandler()
	greetingHandler := handler.NewGreetingHandler(createGreeting, getGreeting, listGreetings)

	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(customMiddleware.RequireAuth(tokenVerifier))
		r.Post("/greetings", greetingHandler.CreateGreeting)
		r.Get("/greetings", greetingHandler.ListGreetings)
		r.Get("/greetings/{id}", greetingHandler.GetGreeting)
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.UIConfig(map[string]string{
			"supportedSubmitMethods": "[]",
		}),
	))

	return r
}
