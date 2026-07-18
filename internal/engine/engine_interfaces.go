package engine

import (
	"context"

	"sentinel/internal/limiter"
)

type Limiter interface {
	Check(ctx context.Context, clientID, api string, rule limiter.Rule) (limiter.Decision, error)
}

type RuleResolver interface {
	Resolve(ctx context.Context, clientID, api string) (limiter.Rule, error)
}

type PostgresLimiter interface {
	Check(ctx context.Context, clientID, api string, rule limiter.Rule) (limiter.Decision, error)
}
