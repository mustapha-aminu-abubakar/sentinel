package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"sentinel/internal/cache"
	"sentinel/internal/config"
	"sentinel/internal/database"
	"sentinel/internal/engine"
	"sentinel/internal/http/router"
	"sentinel/internal/limiter"
	"sentinel/internal/repository/postgres"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")
	stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")
	done <- true
}

func main() {
	cfg := config.Load()

	migrationDB := database.New()
	if err := database.RunMigrations(migrationDB.DB(), "migrations/"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	migrationDB.Close()

	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()

	clientRepo := postgres.NewClientRepository(pool)
	ruleRepo := postgres.NewRateRuleRepository(pool)

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisHost + ":" + cfg.RedisPort,
	})
	defer rdb.Close()

	ruleStore := cache.NewPostgresRuleStore(pool)
	ttlSec, err := strconv.Atoi(cfg.CacheRuleTTL)
	if err != nil || ttlSec <= 0 {
		log.Fatalf("invalid CACHED_RULE_TTL %q: must be a positive integer", cfg.CacheRuleTTL)
	}
	resolver := cache.NewRuleResolver(rdb, ruleStore, time.Duration(ttlSec)*time.Second)
	windowLimiter := limiter.New(rdb)
	pgLimiter := engine.NewPostgresLimiter(pool)
	eng := engine.New(windowLimiter, resolver, pgLimiter)

	handler := router.NewRouter(clientRepo, ruleRepo, eng)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	done := make(chan bool, 1)
	go gracefulShutdown(server, done)

	log.Printf("Server starting on port %s", cfg.HTTPPort)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	<-done
	log.Println("Graceful shutdown complete.")
}
