package analyticshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/analytics/store"
	"sentinel/internal/dbtest"
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

	os.Exit(m.Run())
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

func seedLog(t *testing.T, clientID uuid.UUID, api string, allowed bool, latency int, ts time.Time) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `
		INSERT INTO usage_logs (client_id, api, allowed, latency, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, clientID, api, allowed, latency, ts)
	if err != nil {
		t.Fatalf("seed log: %v", err)
	}
}

func newHandler() *AnalyticsHandler {
	return NewAnalyticsHandler(testPool)
}

func muxForTest() http.Handler {
	mux := http.NewServeMux()
	h := newHandler()
	mux.HandleFunc("GET /analytics/usage", h.GetUsage)
	mux.HandleFunc("GET /analytics/latency", h.GetLatency)
	return mux
}

func TestGetUsage_ReturnsBuckets(t *testing.T) {
	clientID := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedLog(t, clientID, "/api/test", true, 10, now)
	seedLog(t, clientID, "/api/test", true, 20, now.Add(time.Hour))
	seedLog(t, clientID, "/api/test", false, 30, now.Add(2*time.Hour))

	mux := muxForTest()
	req := httptest.NewRequest(http.MethodGet, "/analytics/usage?client_id="+clientID.String(), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var buckets []store.UsageBucket
	if err := json.Unmarshal(w.Body.Bytes(), &buckets); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected at least one bucket")
	}

	allowed, rejected := 0, 0
	for _, b := range buckets {
		allowed += b.Allowed
		rejected += b.Rejected
	}
	if allowed != 2 {
		t.Errorf("expected 2 allowed, got %d", allowed)
	}
	if rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", rejected)
	}
}

func TestGetUsage_StatusFilter(t *testing.T) {
	clientID := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedLog(t, clientID, "/api/test", true, 10, now)
	seedLog(t, clientID, "/api/test", false, 20, now.Add(time.Hour))

	mux := muxForTest()
	req := httptest.NewRequest(http.MethodGet,
		"/analytics/usage?client_id="+clientID.String()+"&status=allowed", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var buckets []store.UsageBucket
	if err := json.Unmarshal(w.Body.Bytes(), &buckets); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	rejected := 0
	for _, b := range buckets {
		rejected += b.Rejected
	}
	if rejected != 0 {
		t.Errorf("expected 0 rejected with allowed filter, got %d", rejected)
	}
}

func TestGetUsage_400_BadBucket(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/analytics/usage?bucket=week", nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetUsage_400_BadStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/analytics/usage?status=unknown", nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetUsage_400_FromAfterTo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/analytics/usage?from=2026-07-20T00:00:00Z&to=2026-07-19T00:00:00Z", nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetUsage_400_BadFromFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/analytics/usage?from=not-a-date", nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetUsage_EmptyResult(t *testing.T) {
	clientID := uuid.New()
	req := httptest.NewRequest(http.MethodGet,
		"/analytics/usage?client_id="+clientID.String(), nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var buckets []store.UsageBucket
	if err := json.Unmarshal(w.Body.Bytes(), &buckets); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if buckets == nil {
		t.Fatal("expected empty array, got null")
	}
	if len(buckets) != 0 {
		t.Errorf("expected empty array, got %d buckets", len(buckets))
	}
}

func TestGetLatency_ReturnsBuckets(t *testing.T) {
	clientID := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedLog(t, clientID, "/api/test", true, 10, now)
	seedLog(t, clientID, "/api/test", true, 20, now.Add(time.Hour))
	seedLog(t, clientID, "/api/test", true, 30, now.Add(2*time.Hour))

	mux := muxForTest()
	req := httptest.NewRequest(http.MethodGet, "/analytics/latency?client_id="+clientID.String(), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var buckets []store.LatencyBucket
	if err := json.Unmarshal(w.Body.Bytes(), &buckets); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected at least one bucket")
	}
	for _, b := range buckets {
		if b.AvgLatencyMS <= 0 {
			t.Errorf("expected positive avg_latency_ms, got %f", b.AvgLatencyMS)
		}
	}
}

func TestGetLatency_400_BadBucket(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/analytics/latency?bucket=month", nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetLatency_400_FromAfterTo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/analytics/latency?from=2026-07-20T00:00:00Z&to=2026-07-19T00:00:00Z", nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetLatency_EmptyResult(t *testing.T) {
	clientID := uuid.New()
	req := httptest.NewRequest(http.MethodGet,
		"/analytics/latency?client_id="+clientID.String(), nil)
	w := httptest.NewRecorder()
	muxForTest().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var buckets []store.LatencyBucket
	if err := json.Unmarshal(w.Body.Bytes(), &buckets); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if buckets == nil {
		t.Fatal("expected empty array, got null")
	}
}
