package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func seedUsageLogs(t *testing.T, clientID uuid.UUID, count int, baseTime time.Time) {
	t.Helper()
	for i := 0; i < count; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Minute)
		allowed := i%3 != 0
		latency := 10 + (i % 90)
		_, err := testPool.Exec(context.Background(), `
			INSERT INTO usage_logs (client_id, api, allowed, latency, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, clientID, "/api/test", allowed, latency, ts)
		if err != nil {
			t.Fatalf("seed usage log %d: %v", i, err)
		}
	}
}

func TestAggregateUsage_NoFilter(t *testing.T) {
	clientID := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedUsageLogs(t, clientID, 30, now)

	buckets, err := AggregateUsage(context.Background(), testPool, UsageFilter{
		Bucket: "day",
	})
	if err != nil {
		t.Fatalf("AggregateUsage: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected at least one bucket")
	}

	totalRows := 0
	for _, b := range buckets {
		totalRows += b.Allowed + b.Rejected
	}
	if totalRows != 30 {
		t.Errorf("expected 30 total rows, got %d", totalRows)
	}
}

func TestAggregateUsage_ClientFilter(t *testing.T) {
	clientA := seedClient(t)
	clientB := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedUsageLogs(t, clientA, 10, now)
	seedUsageLogs(t, clientB, 20, now)

	buckets, err := AggregateUsage(context.Background(), testPool, UsageFilter{
		ClientID: strPtr(clientA.String()),
		Bucket:   "day",
	})
	if err != nil {
		t.Fatalf("AggregateUsage: %v", err)
	}

	totalRows := 0
	for _, b := range buckets {
		totalRows += b.Allowed + b.Rejected
	}
	if totalRows != 10 {
		t.Errorf("expected 10 rows for clientA, got %d", totalRows)
	}
}

func TestAggregateUsage_StatusFilter(t *testing.T) {
	clientID := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedUsageLogs(t, clientID, 30, now)

	allowed := "allowed"
	buckets, err := AggregateUsage(context.Background(), testPool, UsageFilter{
		ClientID: strPtr(clientID.String()),
		Status:   &allowed,
		Bucket:   "day",
	})
	if err != nil {
		t.Fatalf("AggregateUsage: %v", err)
	}

	totalAllowed := 0
	totalRejected := 0
	for _, b := range buckets {
		totalAllowed += b.Allowed
		totalRejected += b.Rejected
	}
	if totalRejected != 0 {
		t.Errorf("expected 0 rejected with allowed filter, got %d", totalRejected)
	}
	if totalAllowed == 0 {
		t.Error("expected some allowed rows")
	}
}

func TestAggregateUsage_DateRange(t *testing.T) {
	clientID := seedClient(t)
	base := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedUsageLogs(t, clientID, 10, base)
	seedUsageLogs(t, clientID, 10, base.AddDate(0, 0, 1))

	from := base.Truncate(24 * time.Hour)
	to := from.Add(24*time.Hour - time.Nanosecond)
	buckets, err := AggregateUsage(context.Background(), testPool, UsageFilter{
		ClientID: strPtr(clientID.String()),
		From:     &from,
		To:       &to,
		Bucket:   "day",
	})
	if err != nil {
		t.Fatalf("AggregateUsage: %v", err)
	}

	totalRows := 0
	for _, b := range buckets {
		totalRows += b.Allowed + b.Rejected
	}
	if totalRows != 10 {
		t.Errorf("expected 10 rows for single day, got %d", totalRows)
	}
}

func TestAggregateLatency_NoFilter(t *testing.T) {
	clientID := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedUsageLogs(t, clientID, 30, now)

	buckets, err := AggregateLatency(context.Background(), testPool, LatencyFilter{
		Bucket: "day",
	})
	if err != nil {
		t.Fatalf("AggregateLatency: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected at least one bucket")
	}

	for _, b := range buckets {
		if b.AvgLatencyMS <= 0 {
			t.Errorf("expected positive avg_latency_ms, got %f", b.AvgLatencyMS)
		}
		if b.P95LatencyMS <= 0 {
			t.Errorf("expected positive p95_latency_ms, got %f", b.P95LatencyMS)
		}
		if b.P95LatencyMS < b.AvgLatencyMS {
			t.Errorf("expected p95 (%f) >= avg (%f)", b.P95LatencyMS, b.AvgLatencyMS)
		}
	}
}

func TestAggregateLatency_ClientFilter(t *testing.T) {
	clientA := seedClient(t)
	clientB := seedClient(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	seedUsageLogs(t, clientA, 10, now)
	seedUsageLogs(t, clientB, 15, now)

	buckets, err := AggregateLatency(context.Background(), testPool, LatencyFilter{
		ClientID: strPtr(clientA.String()),
		Bucket:   "day",
	})
	if err != nil {
		t.Fatalf("AggregateLatency: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected at least one bucket")
	}
}

func strPtr(s string) *string {
	return &s
}
