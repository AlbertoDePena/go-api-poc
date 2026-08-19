package metrics

import "context"

// GreetingMetrics is a driven port for recording greeting-related
// business metrics. The infrastructure adapter decides the backing
// telemetry system (e.g. OpenTelemetry, Prometheus, etc.).
//
// These track business events, not observability (latency, in-flight,
// throughput) — that is handled by the OTel HTTP middleware.
type GreetingMetrics interface {
	GreetingCreated(ctx context.Context, success bool)

	GreetingsListed(ctx context.Context, success bool)

	GreetingViewed(ctx context.Context, success bool)
}
