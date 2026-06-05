package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds the settings needed to initialise the OTel providers.
type Config struct {
	ServiceName    string
	ServiceVersion string
	ExporterType   string // "stdout" (default) or "otlp"
	OTLPEndpoint   string // e.g. "localhost:18889"
}

// Shutdown is a function that flushes and shuts down an OTel provider.
type Shutdown func(ctx context.Context) error

// Setup initialises the OpenTelemetry TracerProvider, MeterProvider and
// LoggerProvider. The exporter is selected by Config.ExporterType:
//   - "stdout" (default): pretty-printed JSON to stdout, suitable for local dev.
//   - "otlp": gRPC OTLP exporter targeting Config.OTLPEndpoint (e.g. Aspire Dashboard).
//
// It returns a combined shutdown function that the caller must invoke on
// application exit to flush pending telemetry.
func Setup(ctx context.Context, cfg Config) (Shutdown, error) {
	var shutdowns []Shutdown

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	// --- Traces ---
	traceExp, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("otel trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	shutdowns = append(shutdowns, tp.Shutdown)

	// --- Metrics ---
	metricExp, err := newMetricExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("otel metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(30*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)

	// --- Logs ---
	logExp, err := newLogExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("otel log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)
	shutdowns = append(shutdowns, lp.Shutdown)

	return combinedShutdown(shutdowns), nil
}

func newTraceExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	switch cfg.ExporterType {
	case "otlp":
		return otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(),
		)
	default:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
}

func newMetricExporter(ctx context.Context, cfg Config) (sdkmetric.Exporter, error) {
	switch cfg.ExporterType {
	case "otlp":
		return otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
	default:
		return stdoutmetric.New()
	}
}

func newLogExporter(ctx context.Context, cfg Config) (sdklog.Exporter, error) {
	switch cfg.ExporterType {
	case "otlp":
		return otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlploggrpc.WithInsecure(),
		)
	default:
		return stdoutlog.New()
	}
}

func combinedShutdown(fns []Shutdown) Shutdown {
	return func(ctx context.Context) error {
		var first error
		for _, fn := range fns {
			if err := fn(ctx); err != nil && first == nil {
				first = err
			}
		}
		return first
	}
}
