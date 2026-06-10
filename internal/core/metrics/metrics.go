package metrics

import "context"

// GreetingMetrics is a driven port for recording greeting-related
// business metrics. The infrastructure adapter decides the backing
// telemetry system (e.g. OpenTelemetry, Prometheus, etc.).
type GreetingMetrics interface {
	// RecordGreeting begins instrumenting a single greeting operation.
	// The returned done function must be called when the operation
	// completes, passing any error that occurred.
	RecordGreeting(ctx context.Context, language string) (done func(err error))
}
