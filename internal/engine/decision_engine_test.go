package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sentinel/internal/cache"
	"sentinel/internal/limiter"
)

type fakeLimiter struct {
	dec limiter.Decision
	err error
}

func (f *fakeLimiter) Check(_ context.Context, _, _ string, _ limiter.Rule) (limiter.Decision, error) {
	return f.dec, f.err
}

type fakeRuleResolver struct {
	rule limiter.Rule
	err  error
}

func (f *fakeRuleResolver) Resolve(_ context.Context, _, _ string) (limiter.Rule, error) {
	return f.rule, f.err
}

type fakePostgresLimiter struct {
	dec limiter.Decision
	err error
}

func (f *fakePostgresLimiter) Check(_ context.Context, _, _ string, _ limiter.Rule) (limiter.Decision, error) {
	return f.dec, f.err
}

var testRule = limiter.Rule{
	ClientID:        "c1",
	API:             "test-api",
	RequestsAllowed: 10,
	WindowSeconds:   60,
}

func TestDecisionEngine_NormalPath(t *testing.T) {
	eng := New(
		&fakeLimiter{dec: limiter.Decision{Allowed: true, Remaining: 9}},
		&fakeRuleResolver{rule: testRule},
		&fakePostgresLimiter{},
	)

	before := testutil.ToFloat64(degradedDecisions.WithLabelValues("Normal"))
	dec, state, err := eng.Decide(context.Background(), "c1", "test-api")
	after := testutil.ToFloat64(degradedDecisions.WithLabelValues("Normal"))

	require.NoError(t, err)
	assert.True(t, dec.Allowed)
	assert.Equal(t, 9, dec.Remaining)
	assert.Equal(t, Normal, state)
	assert.Equal(t, before, after, "Normal should not increment degraded counter")
}

func TestDecisionEngine_NoRule(t *testing.T) {
	eng := New(
		nil,
		&fakeRuleResolver{err: cache.ErrRuleNotFound},
		nil,
	)

	before := testutil.ToFloat64(degradedDecisions.WithLabelValues("NoRule"))
	dec, state, err := eng.Decide(context.Background(), "c1", "unknown-api")
	after := testutil.ToFloat64(degradedDecisions.WithLabelValues("NoRule"))

	require.NoError(t, err)
	assert.True(t, dec.Allowed)
	assert.Equal(t, -1, dec.Remaining)
	assert.Equal(t, NoRule, state)
	assert.Equal(t, before+1, after, "NoRule should increment degraded counter")
}

func TestDecisionEngine_RedisDown_PostgresOK(t *testing.T) {
	eng := New(
		&fakeLimiter{err: errors.New("redis connection refused")},
		&fakeRuleResolver{rule: testRule},
		&fakePostgresLimiter{dec: limiter.Decision{Allowed: true, Remaining: 5}},
	)

	before := testutil.ToFloat64(degradedDecisions.WithLabelValues("RedisDown"))
	dec, state, err := eng.Decide(context.Background(), "c1", "test-api")
	after := testutil.ToFloat64(degradedDecisions.WithLabelValues("RedisDown"))

	require.NoError(t, err)
	assert.True(t, dec.Allowed)
	assert.Equal(t, 5, dec.Remaining)
	assert.Equal(t, RedisDown, state)
	assert.Equal(t, before+1, after, "RedisDown should increment degraded counter")
}

func TestDecisionEngine_FailOpen_BothDown(t *testing.T) {
	eng := New(
		&fakeLimiter{err: errors.New("redis connection refused")},
		&fakeRuleResolver{rule: testRule},
		&fakePostgresLimiter{err: errors.New("postgres connection refused")},
	)

	before := testutil.ToFloat64(degradedDecisions.WithLabelValues("FailOpen"))
	dec, state, err := eng.Decide(context.Background(), "c1", "test-api")
	after := testutil.ToFloat64(degradedDecisions.WithLabelValues("FailOpen"))

	require.NoError(t, err)
	assert.True(t, dec.Allowed)
	assert.Equal(t, FailOpen, state)
	assert.Equal(t, before+1, after, "FailOpen should increment degraded counter")
}

func TestDecisionEngine_FailOpen_100Iterations(t *testing.T) {
	eng := New(
		&fakeLimiter{err: errors.New("redis down")},
		&fakeRuleResolver{rule: testRule},
		&fakePostgresLimiter{err: errors.New("postgres down")},
	)

	before := testutil.ToFloat64(degradedDecisions.WithLabelValues("FailOpen"))

	for i := 0; i < 100; i++ {
		dec, state, err := eng.Decide(context.Background(), "c1", "test-api")
		require.NoError(t, err, "iteration %d: expected no error in fail-open", i)
		assert.True(t, dec.Allowed, "iteration %d: expected Allowed=true in fail-open", i)
		assert.Equal(t, FailOpen, state, "iteration %d: expected FailOpen state", i)
	}

	after := testutil.ToFloat64(degradedDecisions.WithLabelValues("FailOpen"))
	assert.Equal(t, before+100, after, "expected 100 FailOpen increments")
}

func TestDecisionEngine_RedisDown_PostgresRejected(t *testing.T) {
	eng := New(
		&fakeLimiter{err: errors.New("redis timeout")},
		&fakeRuleResolver{rule: testRule},
		&fakePostgresLimiter{dec: limiter.Decision{Allowed: false, Remaining: 0, RetryAfter: 30}},
	)

	dec, state, err := eng.Decide(context.Background(), "c1", "test-api")

	require.NoError(t, err)
	assert.False(t, dec.Allowed)
	assert.Equal(t, 30, dec.RetryAfter)
	assert.Equal(t, RedisDown, state)
}

func TestDecisionEngine_ResolverError_FailOpen(t *testing.T) {
	eng := New(
		nil,
		&fakeRuleResolver{err: errors.New("unexpected resolver error")},
		nil,
	)

	before := testutil.ToFloat64(degradedDecisions.WithLabelValues("FailOpen"))
	dec, state, err := eng.Decide(context.Background(), "c1", "test-api")
	after := testutil.ToFloat64(degradedDecisions.WithLabelValues("FailOpen"))

	require.NoError(t, err)
	assert.True(t, dec.Allowed)
	assert.Equal(t, FailOpen, state)
	assert.Equal(t, before+1, after)
}
