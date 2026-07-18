package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*Limiter, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	l := New(rdb)
	t.Cleanup(func() { rdb.Close() })
	return l, mr, rdb
}

func TestLimiter_UnderLimit(t *testing.T) {
	l, _, _ := setupTest(t)
	ctx := context.Background()
	rule := Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 5, WindowSeconds: 60}

	for i := 0; i < 3; i++ {
		dec, err := l.Check(ctx, "c1", "test-api", rule)
		require.NoError(t, err)
		assert.True(t, dec.Allowed)
		assert.Equal(t, 5-i-1, dec.Remaining)
		assert.Equal(t, 0, dec.RetryAfter)
	}
}

func TestLimiter_AtLimit(t *testing.T) {
	l, _, _ := setupTest(t)
	ctx := context.Background()
	rule := Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 3, WindowSeconds: 60}

	for i := 0; i < 3; i++ {
		dec, err := l.Check(ctx, "c1", "test-api", rule)
		require.NoError(t, err)
		assert.True(t, dec.Allowed)
	}

	dec, err := l.Check(ctx, "c1", "test-api", rule)
	require.NoError(t, err)
	assert.False(t, dec.Allowed)
	assert.Equal(t, 0, dec.Remaining)
	assert.Greater(t, dec.RetryAfter, 0)
}

func TestLimiter_OverLimit(t *testing.T) {
	l, _, _ := setupTest(t)
	ctx := context.Background()
	rule := Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 2, WindowSeconds: 60}

	for i := 0; i < 2; i++ {
		dec, err := l.Check(ctx, "c1", "test-api", rule)
		require.NoError(t, err)
		assert.True(t, dec.Allowed)
	}

	dec, err := l.Check(ctx, "c1", "test-api", rule)
	require.NoError(t, err)
	assert.False(t, dec.Allowed)
	assert.Greater(t, dec.RetryAfter, 0)

	dec, err = l.Check(ctx, "c1", "test-api", rule)
	require.NoError(t, err)
	assert.False(t, dec.Allowed)
}

func TestLimiter_WindowExpiration_PurgesOldEntries(t *testing.T) {
	l, _, rdb := setupTest(t)
	ctx := context.Background()
	rule := Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 3, WindowSeconds: 10}

	oldNano := time.Now().UnixNano() - 15*1e9
	err := rdb.ZAdd(ctx, "rl:c1:test-api", redis.Z{Score: float64(oldNano), Member: oldNano}).Err()
	require.NoError(t, err)

	dec, err := l.Check(ctx, "c1", "test-api", rule)
	require.NoError(t, err)
	assert.True(t, dec.Allowed)
	assert.Equal(t, 2, dec.Remaining)
}

func TestLimiter_ZeroAllowedRule(t *testing.T) {
	l, _, _ := setupTest(t)
	ctx := context.Background()
	rule := Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 0, WindowSeconds: 60}

	dec, err := l.Check(ctx, "c1", "test-api", rule)
	require.NoError(t, err)
	assert.False(t, dec.Allowed)
}

func TestLimiter_ClientIsolation(t *testing.T) {
	l, _, _ := setupTest(t)
	ctx := context.Background()
	rule := Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 1, WindowSeconds: 60}

	dec, err := l.Check(ctx, "c1", "test-api", rule)
	require.NoError(t, err)
	assert.True(t, dec.Allowed)

	dec, err = l.Check(ctx, "c2", "test-api", rule)
	require.NoError(t, err)
	assert.True(t, dec.Allowed)
}

func TestLimiter_RetryAfterCalculation(t *testing.T) {
	l, _, rdb := setupTest(t)
	ctx := context.Background()

	rule := Rule{ClientID: "c1", API: "test-api", RequestsAllowed: 1, WindowSeconds: 10}

	entryNano := time.Now().UnixNano() - 3*1e9
	err := rdb.ZAdd(ctx, "rl:c1:test-api", redis.Z{Score: float64(entryNano), Member: entryNano}).Err()
	require.NoError(t, err)

	dec, err := l.Check(ctx, "c1", "test-api", rule)
	require.NoError(t, err)
	assert.False(t, dec.Allowed)

	assert.Equal(t, 7, dec.RetryAfter)
}

func TestLimiter_MultipleAPIs(t *testing.T) {
	l, _, _ := setupTest(t)
	ctx := context.Background()
	rule1 := Rule{ClientID: "c1", API: "openai", RequestsAllowed: 1, WindowSeconds: 60}
	rule2 := Rule{ClientID: "c1", API: "stripe", RequestsAllowed: 3, WindowSeconds: 60}

	dec, err := l.Check(ctx, "c1", "openai", rule1)
	require.NoError(t, err)
	assert.True(t, dec.Allowed)

	dec, err = l.Check(ctx, "c1", "openai", rule1)
	require.NoError(t, err)
	assert.False(t, dec.Allowed)

	dec, err = l.Check(ctx, "c1", "stripe", rule2)
	require.NoError(t, err)
	assert.True(t, dec.Allowed)
}
