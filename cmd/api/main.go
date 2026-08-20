package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/AlbertoDePena/go-api-poc/api/docs"
	"github.com/AlbertoDePena/go-api-poc/internal/config"
	internalotel "github.com/AlbertoDePena/go-api-poc/internal/otel"
	"github.com/AlbertoDePena/go-api-poc/internal/repository/postgres"
	"github.com/AlbertoDePena/go-api-poc/internal/service"
	"github.com/coreos/go-oidc/v3/oidc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// @title           API POC
// @version         1.0
// @description     A proof-of-concept REST API built with Go, Chi, and hexagonal architecture.

// @host      localhost:8080
// @BasePath  /

const serviceName = "api-poc"

func main() {
	cfg, err := config.LoadAPI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	oidcProvider, err := oidc.NewProvider(ctx, cfg.AzureIssuer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise oidc: %v\n", err)
		os.Exit(1)
	}

	idTokenVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: cfg.AzureAudience})

	// --- OpenTelemetry ---
	otelShutdown, err := internalotel.Setup(ctx, internalotel.Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		ExporterType:   cfg.OtelExporter,
		OTLPEndpoint:   cfg.OtelExporterEndpoint,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise OpenTelemetry: %v\n", err)
		os.Exit(1)
	}

	// Business metrics — same OTel MeterProvider set up above.
	businessMetrics, err := newBusinessMetrics()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise metrics: %v\n", err)
		os.Exit(1)
	}

	// Wire driven adapters
	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		// Write fatal startup errors straight to stderr so
		// they are guaranteed to reach the console.
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	greetingRepo := postgres.NewGreetingRepository(db)
	outboxRepo := postgres.NewOutboxRepository(db)
	transactor := postgres.NewTransactor(db)

	// Wire the shared service — the same constructor a cmd/ui or cmd/worker would use.
	greetingSvc := service.NewGreetingService(businessMetrics, transactor, greetingRepo, outboxRepo)

	// Wire the HTTP router for this binary.
	router := newRouter(serviceName, idTokenVerifier, db, greetingSvc)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	// Listen for SIGINT/SIGTERM in the background
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.Default()

	go func() {
		logger.InfoContext(ctx, "Starting server", "addr", cfg.Addr)
		logger.InfoContext(ctx, "Swagger UI available", "url", "http://localhost"+cfg.Addr+"/swagger/index.html")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Block until signal received
	<-sigCtx.Done()
	stop()
	logger.InfoContext(ctx, "Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(ctx, "forced shutdown", "error", err)
	}

	if err := db.Close(); err != nil {
		logger.ErrorContext(ctx, "database close", "error", err)
	}

	if err := otelShutdown(shutdownCtx); err != nil {
		logger.ErrorContext(ctx, "otel shutdown", "error", err)
	}

	logger.InfoContext(ctx, "Server stopped")
}

// businessMetrics bundles the OTel instruments for greeting business metrics.
type businessMetrics struct {
	created otelmetric.Int64Counter
	listed  otelmetric.Int64Counter
	viewed  otelmetric.Int64Counter
}

const meterName = "github.com/AlbertoDePena/go-api-poc/metrics"

func newBusinessMetrics() (*businessMetrics, error) {
	m := otel.Meter(meterName)

	created, err := m.Int64Counter(
		"greetings.created",
		otelmetric.WithDescription("Total number of greetings created"),
		otelmetric.WithUnit("{greeting}"),
	)
	if err != nil {
		return nil, err
	}

	listed, err := m.Int64Counter(
		"greetings.listed",
		otelmetric.WithDescription("Total number of times greetings were listed"),
		otelmetric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	viewed, err := m.Int64Counter(
		"greetings.viewed",
		otelmetric.WithDescription("Total number of individual greeting views"),
		otelmetric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	return &businessMetrics{created: created, listed: listed, viewed: viewed}, nil
}

func (m *businessMetrics) GreetingCreated(ctx context.Context, success bool) {
	m.created.Add(ctx, 1, otelmetric.WithAttributes(attribute.Bool("success", success)))
}

func (m *businessMetrics) GreetingsListed(ctx context.Context, success bool) {
	m.listed.Add(ctx, 1, otelmetric.WithAttributes(attribute.Bool("success", success)))
}

func (m *businessMetrics) GreetingViewed(ctx context.Context, success bool) {
	m.viewed.Add(ctx, 1, otelmetric.WithAttributes(attribute.Bool("success", success)))
}
