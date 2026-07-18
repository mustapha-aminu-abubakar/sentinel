package limiter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestLimiter_RaceCondition(t *testing.T) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("redis container failed to start: %v", err)
	}
	defer container.Terminate(ctx)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatal(err)
	}

	rdb := goredis.NewClient(&goredis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}

	l := New(rdb)
	rule := Rule{
		ClientID:        "race-client",
		API:             "race-api",
		RequestsAllowed: 100,
		WindowSeconds:   60,
	}

	var allowed atomic.Int64
	var rejected atomic.Int64

	rdb.Del(ctx, "rl:race-client:race-api")

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dec, err := l.Check(ctx, "race-client", "race-api", rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if dec.Allowed {
				allowed.Add(1)
			} else {
				rejected.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(100), allowed.Load(), "expected exactly 100 allowed")
	assert.Equal(t, int64(900), rejected.Load(), "expected exactly 900 rejected")
}
