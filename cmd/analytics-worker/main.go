package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"sentinel/internal/analytics/worker"
	"sentinel/internal/config"
	"sentinel/internal/database"
)

// main starts the analytics worker: runs migrations, connects to Postgres and Kafka, and consumes events.
func main() {
	cfg := config.Load()

	migrationDB := database.New()
	if err := database.RunMigrations(migrationDB.DB(), "migrations/"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	migrationDB.Close()

	pool, err := database.NewPool(context.Background())
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	brokers := strings.Split(cfg.KafkaBrokers, ",")
	consumer := worker.NewConsumer(brokers, pool)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":9091", nil); err != nil {
			slog.Warn("metrics server stopped", "error", err)
		}
	}()

	slog.Info("starting analytics-worker",
		"brokers", cfg.KafkaBrokers,
	)

	if err := consumer.Run(ctx); err != nil {
		slog.Error("consumer exited with error", "error", err)
		os.Exit(1)
	}

	slog.Info("consumer stopped gracefully, shutting down")
	if err := consumer.Close(); err != nil {
		slog.Error("failed to close consumer", "error", err)
	}

	pool.Close()
	slog.Info("analytics-worker shut down complete")
}
