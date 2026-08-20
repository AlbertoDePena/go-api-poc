package config

import (
	"fmt"
	"strings"
	"time"
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

// LoadAPI reads the environment configuration for the API binary. It reports
// every missing required var at once rather than failing on the first.
func LoadAPI() (API, error) {
	loadDotEnv()

	var missing []string
	c := API{
		Addr:                 getEnv("ADDR", ":8080"),
		AzureIssuer:          mustEnv("AZURE_ISSUER", &missing),
		AzureAudience:        mustEnv("AZURE_AUDIENCE", &missing),
		OtelExporter:         getEnv("OTEL_EXPORTER", "stdout"),
		OtelExporterEndpoint: getEnv("OTEL_EXPORTER_ENDPOINT", "localhost:18889"),
		DatabasePath:         mustEnv("DATABASE_PATH", &missing),
	}
	if len(missing) > 0 {
		return API{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

// LoadOutboxRelay reads the environment configuration for the outbox relay binary.
func LoadOutboxRelay() (OutboxRelay, error) {
	loadDotEnv()

	var missing []string
	c := OutboxRelay{
		OtelExporter:         getEnv("OTEL_EXPORTER", "stdout"),
		OtelExporterEndpoint: getEnv("OTEL_EXPORTER_ENDPOINT", "localhost:18889"),
		DatabasePath:         mustEnv("DATABASE_PATH", &missing),
		PollInterval:         getEnvDuration("OUTBOX_POLL_INTERVAL", 5*time.Second),
		BatchSize:            getEnvInt("OUTBOX_BATCH_SIZE", 50),
	}
	if len(missing) > 0 {
		return OutboxRelay{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return c, nil
}
