package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewRouter builds a chi router pre-configured with the common middleware
// shared by every HTTP binary (api, and any future ui): panic recovery,
// request IDs, and OpenTelemetry HTTP instrumentation. Each binary mounts
// its own routes onto the returned mux.
func NewRouter(serviceName string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(otelhttp.NewMiddleware(serviceName))
	return r
}
