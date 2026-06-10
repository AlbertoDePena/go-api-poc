package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr                 string
	AzureTenantID        string
	AzureClientID        string
	OtelExporter         string
	OtelExporterEndpoint string
	DatabasePath         string
}

func Load() Config {
	_ = godotenv.Load()

	addr := os.Getenv("ADDR")
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	otelExporter := os.Getenv("OTEL_EXPORTER")
	otelEndpoint := os.Getenv("OTEL_EXPORTER_ENDPOINT")
	databasePath := os.Getenv("DATABASE_PATH")
	if addr == "" {
		addr = ":8080"
	}
	if tenantID == "" {
		panic("AZURE_TENANT_ID environment variable is required")
	}
	if clientID == "" {
		panic("AZURE_CLIENT_ID environment variable is required")
	}
	if otelExporter == "" {
		otelExporter = "stdout"
	}
	if otelEndpoint == "" {
		otelEndpoint = "localhost:18889"
	}
	if databasePath == "" {
		databasePath = "api-poc.db"
	}
	return Config{
		Addr:                 addr,
		AzureTenantID:        tenantID,
		AzureClientID:        clientID,
		OtelExporter:         otelExporter,
		OtelExporterEndpoint: otelEndpoint,
		DatabasePath:         databasePath,
	}
}
