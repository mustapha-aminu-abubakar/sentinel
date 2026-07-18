package engine

import (
	"context"
	"log/slog"

	"sentinel/internal/cache"
	"sentinel/internal/limiter"
)

type DegradedState int

const (
	Normal DegradedState = iota
	NoRule
	RedisDown
	FailOpen
)

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

type DecisionEngine struct {
	limiter        Limiter
	resolver       RuleResolver
	postgresLim    PostgresLimiter
}

func New(lim Limiter, resolver RuleResolver, postgresLim PostgresLimiter) *DecisionEngine {
	return &DecisionEngine{
		limiter:     lim,
		resolver:    resolver,
		postgresLim: postgresLim,
	}
}

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
