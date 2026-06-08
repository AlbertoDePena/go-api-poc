package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	_ "github.com/craneww/api-poc/docs"
	httpserver "github.com/craneww/api-poc/internal/adapter/http"
	"github.com/craneww/api-poc/internal/adapter/inmemory"
	appOtel "github.com/craneww/api-poc/internal/adapter/otel"
	"github.com/craneww/api-poc/internal/config"
	"github.com/craneww/api-poc/internal/core/usecase"
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

	issuer := fmt.Sprintf("https://sts.windows.net/%s/", cfg.AzureTenantID)

	oidcProvider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Fatalf("oidc discovery failed: %v", err)
	}

	idTokenVerifier := oidcProvider.Verifier(&oidc.Config{ClientID: cfg.AzureClientID})

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

	// Wire driven adapters
	greetingRepo := inmemory.NewGreetingRepository()

	// Wire use cases — inject driven ports
	createGreeting := usecase.NewCreateGreetingUseCase(greetingRepo)
	getGreeting := usecase.NewGetGreetingUseCase(greetingRepo)
	listGreetings := usecase.NewListGreetingsUseCase(greetingRepo)

	// Wire driving adapter (HTTP server) — inject use case interfaces
	router := httpserver.NewServer(serviceName, idTokenVerifier, createGreeting, getGreeting, listGreetings)

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
			slog.Error("listen failed", "error", err)
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

	if err := otelShutdown(shutdownCtx); err != nil {
		slog.Error("otel shutdown", "error", err)
	}

	slog.Info("Server stopped")
}
