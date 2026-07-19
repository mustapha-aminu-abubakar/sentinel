package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/limiter"
)

// PostgresLimiterImpl implements PostgresLimiter by counting successful requests in usage_logs.
type PostgresLimiterImpl struct {
	pool *pgxpool.Pool
}

// NewPostgresLimiter creates a fallback limiter that queries Postgres usage_logs directly.
func NewPostgresLimiter(pool *pgxpool.Pool) *PostgresLimiterImpl {
	return &PostgresLimiterImpl{pool: pool}
}

// Check evaluates a rate-limit decision by querying the usage_logs table for the sliding window count.
func (p *PostgresLimiterImpl) Check(ctx context.Context, clientID, api string, rule limiter.Rule) (limiter.Decision, error) {
	var count int
	err := p.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM usage_logs
		WHERE client_id = $1 AND api = $2 AND allowed = true
		  AND created_at > NOW() - ($3 || ' seconds')::INTERVAL
	`, clientID, api, rule.WindowSeconds).Scan(&count)
	if err != nil {
		return limiter.Decision{}, fmt.Errorf("postgres limiter: count query: %w", err)
	}

	if count < rule.RequestsAllowed {
		remaining := rule.RequestsAllowed - count - 1
		if remaining < 0 {
			remaining = 0
		}
		return limiter.Decision{Allowed: true, Remaining: remaining, RetryAfter: 0}, nil
	}

	var oldestNano time.Time
	err = p.pool.QueryRow(ctx, `
		SELECT MIN(created_at)
		FROM usage_logs
		WHERE client_id = $1 AND api = $2 AND allowed = true
		  AND created_at > NOW() - ($3 || ' seconds')::INTERVAL
	`, clientID, api, rule.WindowSeconds).Scan(&oldestNano)
	if err != nil {
		return limiter.Decision{}, fmt.Errorf("postgres limiter: oldest usage query: %w", err)
	}

	retryAfter := int(time.Until(oldestNano.Add(time.Duration(rule.WindowSeconds) * time.Second)).Seconds())
	if retryAfter < 0 {
		retryAfter = 0
	}
	return limiter.Decision{Allowed: false, Remaining: 0, RetryAfter: retryAfter}, nil
}
