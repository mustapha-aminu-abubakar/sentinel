package worker_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sentinel/internal/analytics/event"
	"sentinel/internal/analytics/worker"
	"sentinel/internal/database"
	"sentinel/internal/dbtest"
)

func TestPipeline_SuccessfulWrite(t *testing.T) {
	pool, dsn, teardown, err := dbtest.StartPostgresWithDSN()
	require.NoError(t, err)
	defer teardown()

	runMigrations(t, dsn)

	clientID := uuid.New()
	insertClient(t, pool, clientID, "test-client")

	evt := event.CheckEvent{
		ClientID:  clientID.String(),
		API:       "/v1/test",
		Timestamp: time.Now().UTC(),
		LatencyMS: 42,
		Status:    event.StatusAllowed,
	}

	err = worker.Write(context.Background(), pool, evt)
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM usage_logs").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func insertClient(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, name string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"INSERT INTO clients (id, name, status) VALUES ($1, $2, 'active')", id, name)
	require.NoError(t, err)
}

func runMigrations(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()
	err = database.RunMigrations(db, "../../../migrations/")
	require.NoError(t, err, "migrations must succeed")
}
