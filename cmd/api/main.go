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

	_ "github.com/AlbertoDePena/go-api-poc/docs"
	"github.com/AlbertoDePena/go-api-poc/internal/config"
	"github.com/AlbertoDePena/go-api-poc/internal/metrics"
	"github.com/AlbertoDePena/go-api-poc/internal/otel"
	"github.com/AlbertoDePena/go-api-poc/internal/service"
	"github.com/AlbertoDePena/go-api-poc/internal/sqlite"
	"github.com/coreos/go-oidc/v3/oidc"
)

// @title           API POC
// @version         1.0
// @description     A proof-of-concept REST API built with Go, Chi, and hexagonal architecture.

// @host      localhost:8080
// @BasePath  /

const serviceName = "api-poc"

func main() {
	cfg := config.LoadAPI()
	ctx := context.Background()

	oidcProvider, err := oidc.NewProvider(ctx, cfg.AzureIssuer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise oidc: %v\n", err)
		os.Exit(1)
	}

	idTokenVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: cfg.AzureAudience})

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

	businessMetrics, err := metrics.NewMetrics()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise metrics: %v\n", err)
		os.Exit(1)
	}

	// Wire driven adapters
	sqliteDB, err := sqlite.Open(cfg.DatabasePath)
	if err != nil {
		// Write fatal startup errors straight to stderr so
		// they are guaranteed to reach the console.
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	greetingRepo := sqlite.NewGreetingRepository(sqliteDB)
	outboxRepo := sqlite.NewOutboxRepository(sqliteDB)
	transactor := sqlite.NewTransactor(sqliteDB)

	// Wire the shared service — the same constructor a cmd/ui or cmd/worker would use.
	greetingSvc := service.NewGreetingService(businessMetrics, transactor, greetingRepo, outboxRepo)

	// Wire the HTTP router for this binary.
	router := newRouter(serviceName, idTokenVerifier, sqliteDB, greetingSvc)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	// Listen for SIGINT/SIGTERM in the background
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("Starting server", "addr", cfg.Addr)
		slog.Info("Swagger UI available", "url", "http://localhost"+cfg.Addr+"/swagger/index.html")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Block until signal received
	<-sigCtx.Done()
	stop()
	slog.Info("Shutting down gracefully...")

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
