package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/dbtest"
	"sentinel/internal/domain"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	var dsn string
	if envDSN, ok := os.LookupEnv("DATABASE_URL_TEST"); ok {
		dsn = envDSN
		testPool = mustConnect(dsn)
	} else {
		var teardown func()
		var err error
		testPool, dsn, teardown, err = dbtest.StartPostgresWithDSN()
		if err != nil {
			panic("failed to start postgres: " + err.Error())
		}
		defer teardown()
	}

	if err := dbtest.RunMigrations(dsn, "../../../migrations"); err != nil {
		panic("failed to run migrations: " + err.Error())
	}

	m.Run()
}

func mustConnect(dsn string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		panic("failed to connect: " + err.Error())
	}
	return pool
}

func seedClient(t *testing.T) uuid.UUID {
	t.Helper()
	clientID := uuid.New()
	_, err := testPool.Exec(context.Background(), `
		INSERT INTO clients (id, name, status)
		VALUES ($1, $2, $3)
	`, clientID, "test-client-"+uuid.New().String(), "active")
	if err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return clientID
}

func TestInsertUsageLog_InsertAndSelect(t *testing.T) {
	clientID := seedClient(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	log := UsageLog{
		ClientID:  clientID,
		API:       "/api/test",
		Allowed:   true,
		LatencyMS: 42,
		CreatedAt: now,
	}

	if err := InsertUsageLog(context.Background(), testPool, log); err != nil {
		t.Fatalf("InsertUsageLog: %v", err)
	}

	var got UsageLog
	err := testPool.QueryRow(context.Background(), `
		SELECT id, client_id, api, allowed, latency, created_at
		FROM usage_logs
		WHERE client_id = $1 AND api = $2
	`, clientID, "/api/test").Scan(
		&got.ID, &got.ClientID, &got.API,
		&got.Allowed, &got.LatencyMS, &got.CreatedAt,
	)
	if err != nil {
		t.Fatalf("select back: %v", err)
	}

	if got.ClientID != clientID {
		t.Errorf("client_id: expected %v, got %v", clientID, got.ClientID)
	}
	if got.API != "/api/test" {
		t.Errorf("api: expected '/api/test', got %q", got.API)
	}
	if got.Allowed != true {
		t.Errorf("allowed: expected true, got %v", got.Allowed)
	}
	if got.LatencyMS != 42 {
		t.Errorf("latency: expected 42, got %d", got.LatencyMS)
	}
	if got.ID == uuid.Nil {
		t.Error("expected non-zero id")
	}
}

func TestInsertUsageLog_ForeignKeyViolation(t *testing.T) {
	log := UsageLog{
		ClientID:  uuid.New(),
		API:       "/api/test",
		Allowed:   true,
		LatencyMS: 10,
		CreatedAt: time.Now(),
	}

	err := InsertUsageLog(context.Background(), testPool, log)
	if err == nil {
		t.Fatal("expected error for foreign key violation, got nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestInsertUsageLog_DefaultValues(t *testing.T) {
	clientID := seedClient(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	log := UsageLog{
		ClientID:  clientID,
		API:       "/api/defaults",
		Allowed:   false,
		CreatedAt: now,
	}

	if err := InsertUsageLog(context.Background(), testPool, log); err != nil {
		t.Fatalf("InsertUsageLog: %v", err)
	}

	var gotLatency int
	var gotCreatedAt time.Time
	err := testPool.QueryRow(context.Background(), `
		SELECT latency, created_at
		FROM usage_logs
		WHERE client_id = $1 AND api = $2
	`, clientID, "/api/defaults").Scan(&gotLatency, &gotCreatedAt)
	if err != nil {
		t.Fatalf("select back: %v", err)
	}

	if gotLatency != 0 {
		t.Errorf("latency default: expected 0, got %d", gotLatency)
	}
	if gotCreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}
