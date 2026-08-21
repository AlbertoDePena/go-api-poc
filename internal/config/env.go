package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// loadDotEnv loads a local .env file when present. It is a no-op in
// environments where the deploy platform injects env vars directly.
func loadDotEnv() {
	_ = godotenv.Load()
}

// getEnv returns the value of key, or def when the var is unset or empty.
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// mustEnv returns the value of key, or appends key to missing when unset.
// Collecting into a slice lets Load* report every missing var at once
// instead of failing on the first one.
func mustEnv(key string, missing *[]string) string {
	v := os.Getenv(key)
	if v == "" {
		*missing = append(*missing, key)
	}
	return v
}

// mustEnvFallback returns the value of the first set (non-empty) key in keys,
// letting a binary prefer a dedicated var but fall back to a shared one (e.g.
// MIGRATION_DATABASE_URL, then DATABASE_URL). If every key is empty it records
// the first key in missing so Load* can report it alongside other missing vars.
func mustEnvFallback(keys []string, missing *[]string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	*missing = append(*missing, keys[0])
	return ""
}

// getEnvDuration returns the duration parsed from key, or def when unset or
// unparseable.
func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

// getEnvInt returns the int parsed from key, or def when unset or unparseable.
func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
