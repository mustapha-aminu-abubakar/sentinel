// Package config loads environment-based configuration for the Sentinel API and analytics worker.
package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration values sourced from environment variables with defaults.
type Config struct {
	DBHost        string // Postgres host address.
	DBPort        string // Postgres port.
	DBDatabase    string // Postgres database name.
	DBUsername    string // Postgres login username.
	DBPassword    string // Postgres login password.
	DBSchema      string // Postgres schema (e.g. public).
	HTTPPort      string // HTTP listen port for the API server.
	RedisHost     string // Redis host address.
	RedisPort     string // Redis port.
	CacheRuleTTL  string // Cache TTL in seconds for rate-limit rules.
	KafkaBrokers  string // Comma-separated list of Kafka broker addresses.
}

// Load reads environment variables and returns a Config with defaults applied.
func Load() Config {
	return Config{
		DBHost:        getEnv("BLUEPRINT_DB_HOST", "localhost"),
		DBPort:        getEnv("BLUEPRINT_DB_PORT", "5432"),
		DBDatabase:    getEnv("BLUEPRINT_DB_DATABASE", "sentinel"),
		DBUsername:    getEnv("BLUEPRINT_DB_USERNAME", "postgres"),
		DBPassword:    getEnv("BLUEPRINT_DB_PASSWORD", "postgres"),
		DBSchema:      getEnv("BLUEPRINT_DB_SCHEMA", "public"),
		HTTPPort:      getEnv("PORT", "8080"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		CacheRuleTTL:  getEnv("CACHED_RULE_TTL", "60"),
		KafkaBrokers:  getEnv("KAFKA_BROKERS", "localhost:9092"),
	}
}

// DSN returns the Postgres connection string built from Config fields.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		c.DBUsername, c.DBPassword, c.DBHost, c.DBPort, c.DBDatabase, c.DBSchema,
	)
}

// getEnv returns the value of an environment variable or a fallback default.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
