package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/AlbertoDePena/go-api-poc/internal/config"
	"github.com/AlbertoDePena/go-api-poc/internal/otel"
	"github.com/AlbertoDePena/go-api-poc/internal/repository/postgres"
)

// serviceName identifies this binary in traces/metrics/logs.
//
// migrate is deliberately its own binary rather than logic inside cmd/api: it
// runs as a one-shot job with a privileged, schema-changing service account
// (MIGRATION_DATABASE_URL), whereas the API and outbox relay connect with a
// least-privilege runtime account that has no DDL rights. Deploy it as an
// init container / pre-deploy job that must complete before the runtime
// binaries start.
const serviceName = "migrate"

func main() {
	if err := run(); err != nil {
		// No logger is guaranteed on the failure path, so write straight to
		// stderr to be sure the error reaches the console; exit non-zero so a
		// deploy pipeline halts before starting the app against an un-migrated
		// schema.
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadMigrate()
	if err != nil {
		return err
	}
	ctx := context.Background()

	// --- OpenTelemetry ---
	otelShutdown, err := otel.Setup(ctx, otel.Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		ExporterType:   cfg.OtelExporter,
		OTLPEndpoint:   cfg.OtelExporterEndpoint,
	})
	if err != nil {
		return fmt.Errorf("initialise OpenTelemetry: %w", err)
	}
	// Flush telemetry before the process exits — a one-shot job would
	// otherwise drop batched spans/logs on a fast exit.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			slog.Default().ErrorContext(ctx, "otel shutdown", "error", err)
		}
	}()

	logger := slog.Default()
	logger.InfoContext(ctx, "applying database migrations")

	if err := postgres.Migrate(ctx, cfg.DatabaseURL); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	logger.InfoContext(ctx, "database migrations complete")
	return nil
}
