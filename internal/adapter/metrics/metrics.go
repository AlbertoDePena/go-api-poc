// Package metrics defines the custom OpenTelemetry instruments for the
// greetings service. Construct it with New after otel.Setup has installed
// the global MeterProvider, then call the helpers from your handlers.
package metrics

import (
	"context"
	"time"

	coremetrics "github.com/craneww/api-poc/internal/core/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var _ coremetrics.GreetingMetrics = (*Metrics)(nil)

// meterName identifies the instrumentation scope for these metrics.
const meterName = "github.com/craneww/api-poc/metrics"

// Metrics bundles the instruments used by the greetings service.
type Metrics struct {
	served   metric.Int64Counter       // total greetings served
	duration metric.Float64Histogram   // greeting handling latency (ms)
	inFlight metric.Int64UpDownCounter // greetings currently being handled
}

// New creates the greetings instruments from the global MeterProvider.
// Call this AFTER otel.Setup has run, otherwise the instruments bind to the
// no-op provider and record nothing.
func New() (*Metrics, error) {
	m := otel.Meter(meterName)

	served, err := m.Int64Counter(
		"greetings.served",
		metric.WithDescription("Total number of greetings served"),
		metric.WithUnit("{greeting}"),
	)
	if err != nil {
		return nil, err
	}

	duration, err := m.Float64Histogram(
		"greetings.duration",
		metric.WithDescription("Time taken to handle a greeting"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	inFlight, err := m.Int64UpDownCounter(
		"greetings.in_flight",
		metric.WithDescription("Greetings currently being handled"),
		metric.WithUnit("{greeting}"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{served: served, duration: duration, inFlight: inFlight}, nil
}

// RecordGreeting instruments a single greeting. Wrap your handler logic with
// the returned done func, which records latency and the served count once the
// work completes.
//
//	done := m.RecordGreeting(ctx, "en")
//	defer done(nil) // pass the handler error, or nil on success
func (m *Metrics) RecordGreeting(ctx context.Context, language string) (done func(err error)) {
	start := time.Now()
	m.inFlight.Add(ctx, 1, metric.WithAttributes(attribute.String("language", language)))

	return func(err error) {
		attrs := metric.WithAttributes(
			attribute.String("language", language),
			attribute.Bool("success", err == nil),
		)
		m.served.Add(ctx, 1, attrs)
		m.duration.Record(ctx, float64(time.Since(start).Milliseconds()), attrs)
		m.inFlight.Add(ctx, -1, metric.WithAttributes(attribute.String("language", language)))
	}
}
