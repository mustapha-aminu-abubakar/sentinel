package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBDatabase string
	DBUsername string
	DBPassword string
	DBSchema   string
	HTTPPort   string
}

func Load() Config {
	return Config{
		DBHost:     getEnv("BLUEPRINT_DB_HOST", "localhost"),
		DBPort:     getEnv("BLUEPRINT_DB_PORT", "5432"),
		DBDatabase: getEnv("BLUEPRINT_DB_DATABASE", "sentinel"),
		DBUsername: getEnv("BLUEPRINT_DB_USERNAME", "postgres"),
		DBPassword: getEnv("BLUEPRINT_DB_PASSWORD", "postgres"),
		DBSchema:   getEnv("BLUEPRINT_DB_SCHEMA", "public"),
		HTTPPort:   getEnv("PORT", "8080"),
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		c.DBUsername, c.DBPassword, c.DBHost, c.DBPort, c.DBDatabase, c.DBSchema,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
