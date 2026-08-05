package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/AlbertoDePena/go-api-poc/docs"
	httpserver "github.com/AlbertoDePena/go-api-poc/internal/adapter/http"
	"github.com/AlbertoDePena/go-api-poc/internal/adapter/metrics"
	appOtel "github.com/AlbertoDePena/go-api-poc/internal/adapter/otel"
	"github.com/AlbertoDePena/go-api-poc/internal/adapter/outbox"
	sqlite "github.com/AlbertoDePena/go-api-poc/internal/adapter/sqlite"
	"github.com/AlbertoDePena/go-api-poc/internal/config"
	"github.com/AlbertoDePena/go-api-poc/internal/core/usecase"
	"github.com/coreos/go-oidc/v3/oidc"
)

// @title           API POC
// @version         1.0
// @description     A proof-of-concept REST API built with Go, Chi, and hexagonal architecture.

// @host      localhost:8080
// @BasePath  /

const serviceName = "api-poc"

func main() {
	cfg := config.Load()
	ctx := context.Background()

	oidcProvider, err := oidc.NewProvider(ctx, cfg.AzureIssuer)
	if err != nil {
		log.Fatalf("oidc discovery failed: %v", err)
	}

	idTokenVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: cfg.AzureAudience})

	// --- OpenTelemetry ---
	otelShutdown, err := appOtel.Setup(ctx, appOtel.Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		ExporterType:   cfg.OtelExporter,
		OTLPEndpoint:   cfg.OtelExporterEndpoint,
	})
	if err != nil {
		slog.Error("failed to initialise OpenTelemetry", "error", err)
		os.Exit(1)
	}

	metrics, err := metrics.NewMetrics()
	if err != nil {
		slog.Error("failed to initialise metrics", "error", err)
		os.Exit(1)
	}

	// Wire driven adapters
	sqliteDB, err := sqlite.Open(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	greetingRepo := sqlite.NewGreetingRepository(sqliteDB)
	outboxRepo := sqlite.NewOutboxRepository(sqliteDB)
	unitOfWork := sqlite.NewUnitOfWork(sqliteDB)

	// Wire use cases — inject driven ports
	createGreeting := usecase.NewCreateGreetingUseCase(metrics, unitOfWork, greetingRepo, outboxRepo)
	getGreeting := usecase.NewGetGreetingUseCase(metrics, greetingRepo)
	listGreetings := usecase.NewListGreetingsUseCase(metrics, greetingRepo)

	// Wire driving adapter (HTTP server) — inject use case interfaces
	router := httpserver.NewServer(serviceName, idTokenVerifier, sqliteDB, createGreeting, getGreeting, listGreetings)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	// Start outbox relay
	relayCtx, relayCancel := context.WithCancel(context.Background())
	relay := outbox.NewRelay(outboxRepo, func(ctx context.Context, aggType, aggID, evtType string, payload []byte) error {
		slog.Info("outbox dispatch",
			"aggregate_type", aggType,
			"aggregate_id", aggID,
			"event_type", evtType,
			"payload", string(payload),
		)
		return nil
	}, 5*time.Second, 50)
	go relay.Start(relayCtx)

	// Listen for SIGINT/SIGTERM in the background
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("Starting server", "addr", cfg.Addr)
		slog.Info("Swagger UI available", "url", "http://localhost"+cfg.Addr+"/swagger/index.html")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen failed", "error", err)
			os.Exit(1)
		}
	}()

	// Block until signal received
	<-sigCtx.Done()
	stop()
	slog.Info("Shutting down gracefully...")

	// Stop relay first so it stops issuing queries
	relayCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "error", err)
	}

	if err := sqliteDB.Close(); err != nil {
		slog.Error("database close", "error", err)
	}

	if err := otelShutdown(shutdownCtx); err != nil {
		slog.Error("otel shutdown", "error", err)
	}

	slog.Info("Server stopped")
}
