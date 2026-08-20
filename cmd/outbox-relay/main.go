package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlbertoDePena/go-api-poc/internal/config"
	"github.com/AlbertoDePena/go-api-poc/internal/otel"
	"github.com/AlbertoDePena/go-api-poc/internal/outbox"
	"github.com/AlbertoDePena/go-api-poc/internal/repository/sqlite"
)

// serviceName identifies this binary in traces/metrics/logs.
//
// The relay is deliberately its own binary rather than a goroutine inside
// cmd/api: it has a different scaling profile (run as a single active
// instance to avoid double-publishing) than the HTTP server.
const serviceName = "outbox-relay"

func main() {
	cfg, err := config.LoadOutboxRelay()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	logger := slog.Default()

	// --- OpenTelemetry ---
	otelShutdown, err := otel.Setup(ctx, otel.Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		ExporterType:   cfg.OtelExporter,
		OTLPEndpoint:   cfg.OtelExporterEndpoint,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise OpenTelemetry: %v\n", err)
		os.Exit(1)
	}

	sqliteDB, err := sqlite.Open(cfg.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	outboxRepo := sqlite.NewOutboxRepository(sqliteDB)

	// The dispatch handler is the plug-in point for a real queue
	// (queue.Enqueue). For now it logs — must stay idempotent-safe because
	// the relay guarantees at-least-once delivery.
	relay := outbox.NewRelay(outboxRepo, func(ctx context.Context, aggType, aggID, evtType string, payload []byte) error {
		logger.InfoContext(ctx, "outbox dispatch",
			"aggregate_type", aggType,
			"aggregate_id", aggID,
			"event_type", evtType,
			"payload", string(payload),
		)
		return nil
	}, cfg.PollInterval, cfg.BatchSize)

	// SIGTERM/SIGINT cancels ctx so an in-flight poll finishes and the loop
	// exits cleanly on deploy instead of being killed mid-publish.
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start blocks until sigCtx is cancelled.
	relay.Start(sigCtx)

	logger.InfoContext(ctx, "Shutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sqliteDB.Close(); err != nil {
		logger.ErrorContext(ctx, "database close", "error", err)
	}

	if err := otelShutdown(shutdownCtx); err != nil {
		logger.ErrorContext(ctx, "otel shutdown", "error", err)
	}

	logger.InfoContext(ctx, "Outbox relay stopped")
}
