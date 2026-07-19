package limiter

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed sliding_window.lua
var slidingWindowScript string

// ErrRedis is returned when a Redis operation in the limiter fails.
var ErrRedis = errors.New("limiter: redis error")

// Limiter implements a sliding-window rate limiter backed by a Redis Lua script.
type Limiter struct {
	rdb     redis.Cmdable
	mu      sync.Mutex
	sha     string
	counter atomic.Int64
}

// New creates a Limiter backed by the given Redis client.
func New(rdb redis.Cmdable) *Limiter {
	return &Limiter{rdb: rdb}
}

// Check evaluates a rate-limit decision for the given client, API, and rule against the sliding window in Redis.
func (l *Limiter) Check(ctx context.Context, clientID, api string, rule Rule) (Decision, error) {
	l.mu.Lock()
	if l.sha == "" {
		sha, err := l.rdb.ScriptLoad(ctx, slidingWindowScript).Result()
		if err != nil {
			l.mu.Unlock()
			return Decision{}, fmt.Errorf("%w: script load: %w", ErrRedis, err)
		}
		l.sha = sha
	}
	l.mu.Unlock()

	key := fmt.Sprintf("rl:%s:%s", clientID, api)
	nowNano := time.Now().UnixNano()
	member := fmt.Sprintf("%d:%d", nowNano, l.counter.Add(1))
	windowSec := fmt.Sprintf("%d", rule.WindowSeconds)
	allowed := fmt.Sprintf("%d", rule.RequestsAllowed)

	result, err := l.rdb.EvalSha(ctx, l.sha, []string{key}, fmt.Sprintf("%d", nowNano), windowSec, allowed, member).Result()
	if err != nil {
		if strings.Contains(err.Error(), "NOSCRIPT") {
			sha, reloadErr := l.rdb.ScriptLoad(ctx, slidingWindowScript).Result()
			if reloadErr != nil {
				return Decision{}, fmt.Errorf("%w: script reload: %w", ErrRedis, reloadErr)
			}
			l.mu.Lock()
			l.sha = sha
			l.mu.Unlock()
			result, err = l.rdb.EvalSha(ctx, sha, []string{key}, fmt.Sprintf("%d", nowNano), windowSec, allowed, member).Result()
			if err != nil {
				return Decision{}, fmt.Errorf("%w: evalsha after reload: %w", ErrRedis, err)
			}
		} else {
			return Decision{}, fmt.Errorf("%w: evalsha: %w", ErrRedis, err)
		}
	}

	vals, ok := result.([]interface{})
	if !ok || len(vals) < 3 {
		return Decision{}, fmt.Errorf("%w: unexpected redis response type: %T", ErrRedis, result)
	}

	allowedInt := toInt(vals[0])
	remaining := toInt(vals[1])
	retryAfter := toInt(vals[2])

	return Decision{
		Allowed:    allowedInt == 1,
		Remaining:  remaining,
		RetryAfter: retryAfter,
	}, nil
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	case string:
		if n == "" {
			return 0
		}
		neg := false
		pos := 0
		if n[0] == '-' {
			neg = true
			pos = 1
		}
		i := 0
		for ; pos < len(n); pos++ {
			c := n[pos]
			if c >= '0' && c <= '9' {
				i = i*10 + int(c-'0')
			} else {
				break
			}
		}
		if neg {
			i = -i
		}
		return i
	default:
		return 0
	}
}
