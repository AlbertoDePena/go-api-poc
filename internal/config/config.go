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
	DatabaseURL          string
}

// OutboxRelay holds configuration for the cmd/outbox-relay binary. It has no
// Azure/HTTP settings — the relay is a loop, not an HTTP server.
type OutboxRelay struct {
	OtelExporter         string
	OtelExporterEndpoint string
	DatabaseURL          string
	PollInterval         time.Duration
	BatchSize            int
}

// Migrate holds configuration for the cmd/migrate binary. It is a one-shot job
// that applies schema migrations with a privileged service account, so its
// DatabaseURL is intentionally separate from the runtime app account the API
// and relay use.
type Migrate struct {
	OtelExporter         string
	OtelExporterEndpoint string
	DatabaseURL          string
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
		DatabaseURL:          mustEnv("DATABASE_URL", &missing),
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
		DatabaseURL:          mustEnv("DATABASE_URL", &missing),
		PollInterval:         getEnvDuration("OUTBOX_POLL_INTERVAL", 5*time.Second),
		BatchSize:            getEnvInt("OUTBOX_BATCH_SIZE", 50),
	}
	if len(missing) > 0 {
		return OutboxRelay{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

// LoadMigrate reads the environment configuration for the migrate binary. It
// prefers MIGRATION_DATABASE_URL — a privileged, schema-changing DSN — and
// falls back to DATABASE_URL for local development where a single account is
// used for everything.
func LoadMigrate() (Migrate, error) {
	loadDotEnv()

	var missing []string
	c := Migrate{
		OtelExporter:         getEnv("OTEL_EXPORTER", "stdout"),
		OtelExporterEndpoint: getEnv("OTEL_EXPORTER_ENDPOINT", "localhost:18889"),
		DatabaseURL:          mustEnvFallback([]string{"MIGRATION_DATABASE_URL", "DATABASE_URL"}, &missing),
	}
	if len(missing) > 0 {
		return Migrate{}, fmt.Errorf("missing required env vars: %s (set MIGRATION_DATABASE_URL to a privileged DSN)", strings.Join(missing, ", "))
	}
	return c, nil
}
