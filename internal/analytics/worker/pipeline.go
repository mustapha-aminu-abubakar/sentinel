package worker

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/analytics/event"
	"sentinel/internal/analytics/store"
)

// Write inserts a check event into Postgres with exponential backoff retries for transient errors.
func Write(ctx context.Context, pool *pgxpool.Pool, evt event.CheckEvent) error {
	clientID, err := uuid.Parse(evt.ClientID)
	if err != nil {
		return err
	}

	allowed := evt.Status == event.StatusAllowed

	log := store.UsageLog{
		ID:        uuid.New(),
		ClientID:  clientID,
		API:       evt.API,
		Allowed:   allowed,
		LatencyMS: evt.LatencyMS,
		CreatedAt: evt.Timestamp,
	}

	backoff := 100 * time.Millisecond
	maxBackoff := 30 * time.Second

	for {
		err := store.InsertUsageLog(ctx, pool, log)
		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			return err
		}

		pgUnavailable.Inc()
		slog.Warn("analytics-worker: insert failed, retrying", "error", err, "backoff", backoff)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// isRetryable returns true for connection-level errors that may succeed on retry.
func isRetryable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "connection") ||
		strings.Contains(msg, "refused") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "closed") ||
		strings.Contains(msg, "broken")
}
