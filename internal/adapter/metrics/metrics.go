// Package metrics provides the OpenTelemetry-backed implementation of the
// core GreetingMetrics driven port. These are business metrics — observability
// (latency, in-flight, throughput) is handled by the OTel HTTP middleware.
package metrics

import (
	"context"

	coremetrics "github.com/AlbertoDePena/go-api-poc/internal/core/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var _ coremetrics.GreetingMetrics = (*Metrics)(nil)

const meterName = "github.com/AlbertoDePena/go-api-poc/metrics"

// Metrics bundles the business-level instruments for the greetings service.
type Metrics struct {
	created metric.Int64Counter
	listed  metric.Int64Counter
	viewed  metric.Int64Counter
}

// New creates the greeting business instruments from the global MeterProvider.
// Call this AFTER otel.Setup has run, otherwise the instruments bind to the
// no-op provider and record nothing.
func NewMetrics() (*Metrics, error) {
	m := otel.Meter(meterName)

	created, err := m.Int64Counter(
		"greetings.created",
		metric.WithDescription("Total number of greetings created"),
		metric.WithUnit("{greeting}"),
	)
	if err != nil {
		return nil, err
	}

	listed, err := m.Int64Counter(
		"greetings.listed",
		metric.WithDescription("Total number of times greetings were listed"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	viewed, err := m.Int64Counter(
		"greetings.viewed",
		metric.WithDescription("Total number of individual greeting views"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{created: created, listed: listed, viewed: viewed}, nil
}

func (m *Metrics) GreetingCreated(ctx context.Context, success bool) {
	m.created.Add(ctx, 1, metric.WithAttributes(attribute.Bool("success", success)))
}

func (m *Metrics) GreetingsListed(ctx context.Context, success bool) {
	m.listed.Add(ctx, 1, metric.WithAttributes(attribute.Bool("success", success)))
}

func (m *Metrics) GreetingViewed(ctx context.Context, success bool) {
	m.viewed.Add(ctx, 1, metric.WithAttributes(attribute.Bool("success", success)))
}
