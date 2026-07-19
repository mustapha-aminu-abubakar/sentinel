// Package cache provides a Redis-backed cache for rate-limit rules with a Postgres fallback store.
package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/limiter"
)

// PostgresRuleStore implements a RuleStore that reads rate-limit rules directly from Postgres.
type PostgresRuleStore struct {
	pool *pgxpool.Pool
}

// NewPostgresRuleStore creates a store that queries rules from Postgres via the given pool.
func NewPostgresRuleStore(pool *pgxpool.Pool) *PostgresRuleStore {
	return &PostgresRuleStore{pool: pool}
}

// GetRule retrieves the active rate-limit rule for a client-API pair from Postgres.
func (s *PostgresRuleStore) GetRule(ctx context.Context, clientID, api string) (limiter.Rule, error) {
	query := `
		SELECT r.requests_allowed, r.window_seconds
		FROM rate_rules r
		JOIN clients c ON c.id = r.client_id
		WHERE c.id = $1 AND r.api = $2 AND c.status = 'active'
	`

	var requestsAllowed, windowSeconds int32
	err := s.pool.QueryRow(ctx, query, clientID, api).Scan(&requestsAllowed, &windowSeconds)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return limiter.Rule{}, ErrRuleNotFound
		}
		return limiter.Rule{}, fmt.Errorf("cache: get rule from pg: %w", err)
	}

	return limiter.Rule{
		ClientID:        clientID,
		API:             api,
		RequestsAllowed: int(requestsAllowed),
		WindowSeconds:   int(windowSeconds),
	}, nil
}
