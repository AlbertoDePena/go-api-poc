package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

// API holds configuration for the cmd/api binary.
type API struct {
	Addr                 string
	AzureIssuer          string
	AzureAudience        string
	OtelExporter         string
	OtelExporterEndpoint string
	DatabasePath         string
}

// OutboxRelay holds configuration for the cmd/outbox-relay binary. It has no
// Azure/HTTP settings — the relay is a loop, not an HTTP server.
type OutboxRelay struct {
	OtelExporter         string
	OtelExporterEndpoint string
	DatabasePath         string
	PollInterval         time.Duration
	BatchSize            int
}

// LoadAPI reads the environment configuration for the API binary.
func LoadAPI() API {
	loadDotEnv()

	issuer := os.Getenv("AZURE_ISSUER")
	if issuer == "" {
		panic("AZURE_ISSUER environment variable is required")
	}
	audience := os.Getenv("AZURE_AUDIENCE")
	if audience == "" {
		panic("AZURE_AUDIENCE environment variable is required")
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	otelExporter, otelEndpoint := otelEnv()

	return API{
		Addr:                 addr,
		AzureIssuer:          issuer,
		AzureAudience:        audience,
		OtelExporter:         otelExporter,
		OtelExporterEndpoint: otelEndpoint,
		DatabasePath:         databasePath(),
	}
}

// LoadOutboxRelay reads the environment configuration for the outbox relay binary.
func LoadOutboxRelay() OutboxRelay {
	loadDotEnv()

	otelExporter, otelEndpoint := otelEnv()

	return OutboxRelay{
		OtelExporter:         otelExporter,
		OtelExporterEndpoint: otelEndpoint,
		DatabasePath:         databasePath(),
		PollInterval:         5 * time.Second,
		BatchSize:            50,
	}
}

func loadDotEnv() {
	_ = godotenv.Load()
}

func otelEnv() (exporter, endpoint string) {
	exporter = os.Getenv("OTEL_EXPORTER")
	if exporter == "" {
		exporter = "stdout"
	}
	endpoint = os.Getenv("OTEL_EXPORTER_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:18889"
	}
	return exporter, endpoint
}

func databasePath() string {
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		panic("DATABASE_PATH environment variable is required")
	}
	return path
}
