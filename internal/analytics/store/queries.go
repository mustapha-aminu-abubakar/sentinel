package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageFilter defines the parameters for the usage aggregation query.
type UsageFilter struct {
	ClientID *string
	API      *string
	From     *time.Time
	To       *time.Time
	Status   *string
	Bucket   string
}

// UsageBucket represents one time bucket of allowed/rejected counts.
type UsageBucket struct {
	Bucket   time.Time `json:"bucket"`
	Allowed  int       `json:"allowed"`
	Rejected int       `json:"rejected"`
}

// LatencyFilter defines the parameters for the latency aggregation query.
type LatencyFilter struct {
	ClientID *string
	API      *string
	From     *time.Time
	To       *time.Time
	Bucket   string
}

// LatencyBucket represents one time bucket of average and P95 latency.
type LatencyBucket struct {
	Bucket       time.Time `json:"bucket"`
	AvgLatencyMS float64   `json:"avg_latency_ms"`
	P95LatencyMS float64   `json:"p95_latency_ms"`
}

// AggregateUsage computes allowed/rejected counts bucketed by hour or day.
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

// AggregateLatency computes average and P95 latency bucketed by hour or day.
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

// queryParams is a builder for parameterised SQL queries with dynamic WHERE clauses.
type queryParams struct {
	args    []any
	clauses []string
	argIdx  int
}

// newQueryParams initialises a queryParams with the first argument (bucket).
func newQueryParams(bucket string) queryParams {
	return queryParams{args: []any{bucket}, argIdx: 1}
}

// add appends a filter condition with its parameter placeholder.
func (qp *queryParams) add(value any, clause string) {
	qp.argIdx++
	qp.args = append(qp.args, value)
	qp.clauses = append(qp.clauses, fmt.Sprintf(clause, qp.argIdx))
}

// whereClause returns the assembled WHERE clause, or empty string if no filters.
func (qp *queryParams) whereClause() string {
	if len(qp.clauses) > 0 {
		return "WHERE " + strings.Join(qp.clauses, " AND ")
	}
	return ""
}

// buildUsageQuery constructs the parameterised SQL and arguments for usage aggregation.
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

// buildLatencyQuery constructs the parameterised SQL and arguments for latency aggregation.
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
