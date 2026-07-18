//go:build integration

package worker_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"sentinel/internal/analytics/event"
	"sentinel/internal/analytics/worker"
)

func TestConsumer_NoDataLossDuringPostgresOutage(t *testing.T) {
	ctx := context.Background()

	// --- Start Postgres ---
	pgContainer, pgPool, pgDSN := startPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)

	runMigrations(t, pgDSN)

	// Seed a client row so FK constraint on usage_logs is satisfied
	clientID := uuid.New()
	_, err := pgPool.Exec(ctx,
		"INSERT INTO clients (id, name, status) VALUES ($1, $2, 'active')",
		clientID, "test-load-client",
	)
	require.NoError(t, err)

	// --- Start Kafka ---
	kafkaContainer, brokers := startKafkaContainer(t, ctx)
	defer kafkaContainer.Terminate(ctx)

	brokerAddr := brokers[0]
	createTopic(t, brokerAddr, event.TopicCheckEvents, 1)

	// --- Start consumer in background ---
	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	consumer := worker.NewConsumer(brokers, pgPool)
	consumerErr := make(chan error, 1)
	go func() {
		consumerErr <- consumer.Run(consumerCtx)
	}()

	// --- Publish 100 events ---
	publishEvents(t, brokerAddr, event.TopicCheckEvents, 100, 0, clientID)
	waitForRowCount(t, pgPool, 100, 30*time.Second)
	t.Log("phase 1: first 100 events written successfully")

	// --- Stop Postgres (not terminate — data must survive) ---
	stopCtx, stopCancel := context.WithTimeout(ctx, 10*time.Second)
	defer stopCancel()
	err = pgContainer.Stop(stopCtx, nil)
	require.NoError(t, err)
	t.Log("phase 2: postgres container stopped")

	// --- Publish 100 more events while PG is down ---
	publishEvents(t, brokerAddr, event.TopicCheckEvents, 100, 100, clientID)
	t.Log("phase 3: published 100 more events while postgres was down")

	// Brief pause to ensure consumer has tried and failed
	time.Sleep(2 * time.Second)

	// --- Restart the same Postgres container (data preserved) ---
	t.Log("phase 4: restarting postgres...")
	err = pgContainer.Start(ctx)
	require.NoError(t, err)

	// Wait for pool to reconnect — pgxpool reconnects lazily
	waitForPoolReady(t, pgPool, 30*time.Second)

	// Wait for consumer to catch up
	waitForRowCount(t, pgPool, 200, 60*time.Second)
	t.Log("phase 5: all 200 events written after postgres recovery")

	// --- Assertions ---
	var count int
	err = pgPool.QueryRow(ctx, "SELECT COUNT(*) FROM usage_logs").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 200, count, "zero data loss: all 200 events must be in usage_logs")

	// Clean shutdown
	cancelConsumer()
	select {
	case err := <-consumerErr:
		if err != nil {
			t.Logf("consumer exited with error (expected on ctx cancel): %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not exit in time")
	}
	require.NoError(t, consumer.Close())
}

func startPostgresContainer(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, *pgxpool.Pool, string) {
	t.Helper()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("sentinel_test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://user:password@%s:%s/sentinel_test?sslmode=disable", host, port.Port())
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	err = pool.Ping(ctx)
	require.NoError(t, err)

	return container, pool, dsn
}

func startKafkaContainer(t *testing.T, ctx context.Context) (*tckafka.KafkaContainer, []string) {
	t.Helper()

	container, err := tckafka.Run(ctx,
		"confluentinc/confluent-local:7.7.0",
		tckafka.WithClusterID("sentinel-test-cluster"),
	)
	require.NoError(t, err)

	brokers, err := container.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	return container, brokers
}

func createTopic(t *testing.T, brokerAddr, topic string, partitions int) {
	t.Helper()

	conn, err := kafka.Dial("tcp", brokerAddr)
	require.NoError(t, err)
	defer conn.Close()

	partitionCount := partitions
	if partitionCount <= 0 {
		partitionCount = 1
	}

	controller, err := conn.Controller()
	require.NoError(t, err)

	controllerAddr := fmt.Sprintf("%s:%d", controller.Host, controller.Port)
	controllerConn, err := kafka.Dial("tcp", controllerAddr)
	require.NoError(t, err)
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitionCount,
		ReplicationFactor: 1,
	})
	require.NoError(t, err)
}

func publishEvents(t *testing.T, brokerAddr, topic string, count, startIdx int, clientID uuid.UUID) {
	t.Helper()

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddr),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	for i := 0; i < count; i++ {
		evt := event.CheckEvent{
			ClientID:  clientID.String(),
			API:       "/v1/check",
			Timestamp: time.Now().UTC(),
			LatencyMS: i + startIdx,
			Status:    event.StatusAllowed,
		}
		data, err := event.Encode(evt)
		require.NoError(t, err)

		err = writer.WriteMessages(context.Background(), kafka.Message{
			Value: data,
		})
		require.NoError(t, err)
	}
}

func waitForRowCount(t *testing.T, pool *pgxpool.Pool, expected int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int
		err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM usage_logs").Scan(&count)
		if err == nil && count >= expected {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	var actual int
	_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM usage_logs").Scan(&actual)
	t.Fatalf("timed out waiting for %d rows in usage_logs, got %d", expected, actual)
}

func waitForPoolReady(t *testing.T, pool *pgxpool.Pool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := pool.Ping(context.Background())
		if err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("timed out waiting for pool to reconnect")
}
