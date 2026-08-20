package main

import (
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/AlbertoDePena/go-api-poc/cmd/api/handler"
	"github.com/AlbertoDePena/go-api-poc/internal/httpserver"
	"github.com/AlbertoDePena/go-api-poc/internal/httpserver/middleware"
	"github.com/AlbertoDePena/go-api-poc/internal/service"
)

// newRouter builds the API's chi router: the shared middleware from
// httpserver.NewRouter, plus this binary's own route mounts and handlers.
func newRouter(
	serviceName string,
	tokenVerifier *oidc.IDTokenVerifier,
	pinger handler.Pinger,
	greetingSvc *service.GreetingService,
) *chi.Mux {
	r := httpserver.NewRouter(serviceName)

	healthHandler := handler.NewHealthHandler(pinger)
	greetingHandler := handler.NewGreetingHandler(greetingSvc, greetingSvc)

	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RequireAuth(tokenVerifier))
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
