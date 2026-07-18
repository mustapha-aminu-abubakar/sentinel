package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sentinel/internal/limiter"
)

type fakeRuleStore struct {
	rule limiter.Rule
	err  error
	callCount int
}

func (f *fakeRuleStore) GetRule(_ context.Context, clientID, api string) (limiter.Rule, error) {
	f.callCount++
	if f.err != nil {
		return limiter.Rule{}, f.err
	}
	r := f.rule
	r.ClientID = clientID
	r.API = api
	return r, nil
}

func setupTest(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return mr, rdb
}

func TestResolve_CacheHit_DoesNotCallStore(t *testing.T) {
	mr, rdb := setupTest(t)
	ctx := context.Background()

	expected := limiter.Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 10, WindowSeconds: 60}
	data, err := json.Marshal(expected)
	require.NoError(t, err)
	mr.Set("cfg:rule:c1:test-api", string(data))

	store := &fakeRuleStore{err: errors.New("should not be called")}
	resolver := NewRuleResolver(rdb, store, time.Minute)

	rule, err := resolver.Resolve(ctx, "c1", "test-api")
	require.NoError(t, err)
	assert.Equal(t, expected, rule)
	assert.Equal(t, 0, store.callCount, "store must not be called on cache hit")
}

func TestResolve_CacheMiss_PopulatesRedis(t *testing.T) {
	mr, rdb := setupTest(t)
	ctx := context.Background()

	stored := limiter.Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 5, WindowSeconds: 30}
	store := &fakeRuleStore{rule: stored}
	resolver := NewRuleResolver(rdb, store, 30*time.Second)

	rule, err := resolver.Resolve(ctx, "c1", "test-api")
	require.NoError(t, err)
	assert.Equal(t, stored, rule)
	assert.Equal(t, 1, store.callCount, "store must be called once on cache miss")

	raw, err := mr.Get("cfg:rule:c1:test-api")
	require.NoError(t, err, "key must exist in redis after populate")

	var cached limiter.Rule
	err = json.Unmarshal([]byte(raw), &cached)
	require.NoError(t, err)
	assert.Equal(t, stored, cached)

	ttl := mr.TTL("cfg:rule:c1:test-api")
	assert.InDelta(t, 30*time.Second, ttl, float64(time.Second), "ttl must be ~30s")
}

func TestResolve_MissingRule_ReturnsErrRuleNotFound(t *testing.T) {
	mr, rdb := setupTest(t)
	ctx := context.Background()

	store := &fakeRuleStore{err: ErrRuleNotFound}
	resolver := NewRuleResolver(rdb, store, time.Minute)

	_, err := resolver.Resolve(ctx, "c1", "missing-api")
	assert.ErrorIs(t, err, ErrRuleNotFound)
	assert.Equal(t, 1, store.callCount)

	exists := mr.Exists("cfg:rule:c1:missing-api")
	assert.Equal(t, false, exists, "no key should be written for missing rule")
}

type errCmdable struct {
	redis.Cmdable
	errKey string
}

func (e *errCmdable) Get(ctx context.Context, key string) *redis.StringCmd {
	if key == e.errKey {
		return redis.NewStringResult("", errors.New("redis connection refused"))
	}
	return e.Cmdable.Get(ctx, key)
}

func TestResolve_RedisError_Surfaces(t *testing.T) {
	_, rdb := setupTest(t)
	ctx := context.Background()

	wrapped := &errCmdable{Cmdable: rdb, errKey: "cfg:rule:c1:boom"}
	store := &fakeRuleStore{err: errors.New("should not be called")}
	resolver := NewRuleResolver(wrapped, store, time.Minute)

	_, err := resolver.Resolve(ctx, "c1", "boom")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis connection refused")
	assert.Equal(t, 0, store.callCount)
}
