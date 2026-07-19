package engine

import (
	"context"
	"log/slog"

	"sentinel/internal/cache"
	"sentinel/internal/limiter"
)

// DegradedState represents the operational mode of the decision engine.
type DegradedState int

const (
	// Normal indicates full Redis + rule resolution capability.
	Normal DegradedState = iota
	// NoRule indicates the client-API pair has no configured rule; requests are allowed.
	NoRule
	// RedisDown indicates Redis is unavailable and the Postgres fallback was used.
	RedisDown
	// FailOpen indicates both Redis and Postgres failed; all requests are allowed.
	FailOpen
)

// String returns a human-readable label for the degraded state.
func (s DegradedState) String() string {
	switch s {
	case Normal:
		return "Normal"
	case NoRule:
		return "NoRule"
	case RedisDown:
		return "RedisDown"
	case FailOpen:
		return "FailOpen"
	default:
		return "Unknown"
	}
}

// DecisionEngine orchestrates rate-limit checks with graceful degradation through RedisDown -> FailOpen.
type DecisionEngine struct {
	limiter        Limiter
	resolver       RuleResolver
	postgresLim    PostgresLimiter
}

// New creates a DecisionEngine with the given primary limiter, rule resolver, and Postgres fallback.
func New(lim Limiter, resolver RuleResolver, postgresLim PostgresLimiter) *DecisionEngine {
	return &DecisionEngine{
		limiter:     lim,
		resolver:    resolver,
		postgresLim: postgresLim,
	}
}

// Decide evaluates a rate-limit decision, degrading gracefully through Redis fallback and fail-open.
func (e *DecisionEngine) Decide(ctx context.Context, clientID, api string) (limiter.Decision, DegradedState, error) {
	rule, err := e.resolver.Resolve(ctx, clientID, api)
	if err != nil {
		if err == cache.ErrRuleNotFound {
			incDegraded(NoRule)
			return limiter.Decision{Allowed: true, Remaining: -1}, NoRule, nil
		}
		incDegraded(FailOpen)
		slog.Error("rule resolution failed, failing open",
			"client_id", clientID, "api", api, "error", err,
		)
		return limiter.Decision{Allowed: true}, FailOpen, nil
	}

	dec, err := e.limiter.Check(ctx, clientID, api, rule)
	if err == nil {
		return dec, Normal, nil
	}

	slog.Warn("redis limiter failed, falling back to postgres",
		"client_id", clientID, "api", api, "error", err,
	)

	fallbackDec, fallbackErr := e.postgresLim.Check(ctx, clientID, api, rule)
	if fallbackErr == nil {
		incDegraded(RedisDown)
		slog.Warn("degraded state: redis down, using postgres fallback",
			"client_id", clientID, "api", api,
			"redis_error", err,
		)
		return fallbackDec, RedisDown, nil
	}

	incDegraded(FailOpen)
	slog.Error("both redis and postgres failed, failing open",
		"client_id", clientID, "api", api,
		"redis_error", err,
		"postgres_error", fallbackErr,
	)
	return limiter.Decision{Allowed: true}, FailOpen, nil
}
