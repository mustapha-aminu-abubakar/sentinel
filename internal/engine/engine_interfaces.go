// Package engine implements the decision engine that orchestrates rate-limit checks with graceful degradation.
package engine

import (
	"context"

	"sentinel/internal/limiter"
)

// Limiter is the contract for evaluating a rate-limit decision against a rule.
type Limiter interface {
	// Check evaluates a rate-limit decision for the given client, API, and rule.
	Check(ctx context.Context, clientID, api string, rule limiter.Rule) (limiter.Decision, error)
}

// RuleResolver is the contract for resolving a rate-limit rule for a client-API pair.
type RuleResolver interface {
	// Resolve returns the rate-limit rule for the given client and API.
	Resolve(ctx context.Context, clientID, api string) (limiter.Rule, error)
}

// PostgresLimiter is the contract for fallback rate-limit checks when Redis is unavailable.
type PostgresLimiter interface {
	// Check evaluates a rate-limit decision directly from Postgres usage logs.
	Check(ctx context.Context, clientID, api string, rule limiter.Rule) (limiter.Decision, error)
}
