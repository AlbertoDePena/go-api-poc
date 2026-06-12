package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr                 string
	AzureIssuer          string
	AzureAudience        string
	OtelExporter         string
	OtelExporterEndpoint string
	DatabasePath         string
}

func Load() Config {
	_ = godotenv.Load()

	addr := os.Getenv("ADDR")
	issuer := os.Getenv("AZURE_ISSUER")
	audience := os.Getenv("AZURE_AUDIENCE")
	otelExporter := os.Getenv("OTEL_EXPORTER")
	otelEndpoint := os.Getenv("OTEL_EXPORTER_ENDPOINT")
	databasePath := os.Getenv("DATABASE_PATH")
	if addr == "" {
		addr = ":8080"
	}
	if issuer == "" {
		panic("AZURE_ISSUER environment variable is required")
	}
	if audience == "" {
		panic("AZURE_AUDIENCE environment variable is required")
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
		AzureIssuer:          issuer,
		AzureAudience:        audience,
		OtelExporter:         otelExporter,
		OtelExporterEndpoint: otelEndpoint,
		DatabasePath:         databasePath,
	}
}
