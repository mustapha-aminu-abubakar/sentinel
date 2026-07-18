package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"sentinel/internal/limiter"
)

var ErrRuleNotFound = errors.New("cache: rule not found")

type RuleStore interface {
	GetRule(ctx context.Context, clientID string, api string) (limiter.Rule, error)
}

type RuleResolver struct {
	rdb   redis.Cmdable
	store RuleStore
	ttl   time.Duration
}

func NewRuleResolver(rdb redis.Cmdable, store RuleStore, ttl time.Duration) *RuleResolver {
	return &RuleResolver{rdb: rdb, store: store, ttl: ttl}
}

func (r *RuleResolver) Resolve(ctx context.Context, clientID, api string) (limiter.Rule, error) {
	key := fmt.Sprintf("cfg:rule:%s:%s", clientID, api)

	data, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil && !errors.Is(err, redis.Nil) {
		return limiter.Rule{}, fmt.Errorf("cache: redis get: %w", err)
	}

	if err == nil {
		var rule limiter.Rule
		if err := json.Unmarshal(data, &rule); err != nil {
			return limiter.Rule{}, fmt.Errorf("cache: unmarshal rule: %w", err)
		}
		return rule, nil
	}

	rule, err := r.store.GetRule(ctx, clientID, api)
	if err != nil {
		return limiter.Rule{}, err
	}

	data, err = json.Marshal(rule)
	if err != nil {
		return limiter.Rule{}, fmt.Errorf("cache: marshal rule: %w", err)
	}

	if err := r.rdb.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return limiter.Rule{}, fmt.Errorf("cache: redis set: %w", err)
	}

	return rule, nil
}
