package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UsageFilter struct {
	ClientID *string
	API      *string
	From     *time.Time
	To       *time.Time
	Status   *string
	Bucket   string
}

type UsageBucket struct {
	Bucket   time.Time `json:"bucket"`
	Allowed  int       `json:"allowed"`
	Rejected int       `json:"rejected"`
}

type LatencyFilter struct {
	ClientID *string
	API      *string
	From     *time.Time
	To       *time.Time
	Bucket   string
}

type LatencyBucket struct {
	Bucket       time.Time `json:"bucket"`
	AvgLatencyMS float64   `json:"avg_latency_ms"`
	P95LatencyMS float64   `json:"p95_latency_ms"`
}

func AggregateUsage(ctx context.Context, pool *pgxpool.Pool, filter UsageFilter) ([]UsageBucket, error) {
	query, args := buildUsageQuery(filter)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate usage: %w", err)
	}
	defer rows.Close()

	var buckets []UsageBucket
	for rows.Next() {
		var b UsageBucket
		if err := rows.Scan(&b.Bucket, &b.Allowed, &b.Rejected); err != nil {
			return nil, fmt.Errorf("scan usage bucket: %w", err)
		}
		buckets = append(buckets, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return buckets, nil
}

func AggregateLatency(ctx context.Context, pool *pgxpool.Pool, filter LatencyFilter) ([]LatencyBucket, error) {
	query, args := buildLatencyQuery(filter)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate latency: %w", err)
	}
	defer rows.Close()

	var buckets []LatencyBucket
	for rows.Next() {
		var b LatencyBucket
		if err := rows.Scan(&b.Bucket, &b.AvgLatencyMS, &b.P95LatencyMS); err != nil {
			return nil, fmt.Errorf("scan latency bucket: %w", err)
		}
		buckets = append(buckets, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return buckets, nil
}

type queryParams struct {
	args    []any
	clauses []string
	argIdx  int
}

func newQueryParams(bucket string) queryParams {
	return queryParams{args: []any{bucket}, argIdx: 1}
}

func (qp *queryParams) add(value any, clause string) {
	qp.argIdx++
	qp.args = append(qp.args, value)
	qp.clauses = append(qp.clauses, fmt.Sprintf(clause, qp.argIdx))
}

func (qp *queryParams) whereClause() string {
	if len(qp.clauses) > 0 {
		return "WHERE " + strings.Join(qp.clauses, " AND ")
	}
	return ""
}

func buildUsageQuery(filter UsageFilter) (string, []any) {
	qp := newQueryParams(filter.Bucket)

	if filter.ClientID != nil {
		qp.add(*filter.ClientID, "client_id = $%d")
	}
	if filter.API != nil {
		qp.add(*filter.API, "api = $%d")
	}
	if filter.From != nil {
		qp.add(*filter.From, "created_at >= $%d")
	}
	if filter.To != nil {
		qp.add(*filter.To, "created_at <= $%d")
	}
	if filter.Status != nil && *filter.Status != "all" {
		qp.add(*filter.Status == "allowed", "allowed = $%d")
	}

	query := fmt.Sprintf(`
SELECT
    date_trunc($1::text, created_at) AS bucket,
    COUNT(*) FILTER (WHERE allowed) AS allowed,
    COUNT(*) FILTER (WHERE NOT allowed) AS rejected
FROM usage_logs
%s
GROUP BY bucket
ORDER BY bucket`, qp.whereClause())

	return query, qp.args
}

func buildLatencyQuery(filter LatencyFilter) (string, []any) {
	qp := newQueryParams(filter.Bucket)

	if filter.ClientID != nil {
		qp.add(*filter.ClientID, "client_id = $%d")
	}
	if filter.API != nil {
		qp.add(*filter.API, "api = $%d")
	}
	if filter.From != nil {
		qp.add(*filter.From, "created_at >= $%d")
	}
	if filter.To != nil {
		qp.add(*filter.To, "created_at <= $%d")
	}

	query := fmt.Sprintf(`
SELECT
    date_trunc($1::text, created_at) AS bucket,
    AVG(latency)::float8 AS avg_latency_ms,
    percentile_cont(0.95) WITHIN GROUP (ORDER BY latency) AS p95_latency_ms
FROM usage_logs
%s
GROUP BY bucket
ORDER BY bucket`, qp.whereClause())

	return query, qp.args
}
