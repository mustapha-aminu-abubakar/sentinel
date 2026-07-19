// Package store provides direct Postgres access for analytics data (usage logs and aggregation queries).
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/domain"
)

// UsageLog represents a single row in the usage_logs table.
type UsageLog struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	API       string
	Allowed   bool
	LatencyMS int
	CreatedAt time.Time
}

// InsertUsageLog inserts a usage log row and returns domain errors on constraint violations.
func InsertUsageLog(ctx context.Context, pool *pgxpool.Pool, log UsageLog) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO usage_logs (client_id, api, allowed, latency, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, log.ClientID, log.API, log.Allowed, log.LatencyMS, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert usage log: %w", mapInsertError(err))
	}
	return nil
}

// mapInsertError converts pgx errors to domain sentinel errors for insert operations.
func mapInsertError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return domain.ErrConflict
	case "23503", "23514":
		return domain.ErrValidation
	default:
		return err
	}
}
