package analyticshttp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/analytics/store"
	"sentinel/internal/domain"
	"sentinel/internal/http/httperr"
)

type AnalyticsHandler struct {
	pool *pgxpool.Pool
}

func NewAnalyticsHandler(pool *pgxpool.Pool) *AnalyticsHandler {
	return &AnalyticsHandler{pool: pool}
}

func (h *AnalyticsHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	filter, err := parseUsageParams(r)
	if err != nil {
		httperr.WriteError(w, fmt.Errorf("%w: %v", domain.ErrValidation, err))
		return
	}

	buckets, err := store.AggregateUsage(r.Context(), h.pool, filter)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	if buckets == nil {
		buckets = []store.UsageBucket{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buckets); err != nil {
		log.Printf("encode usage response: %v", err)
	}
}

func (h *AnalyticsHandler) GetLatency(w http.ResponseWriter, r *http.Request) {
	filter, err := parseLatencyParams(r)
	if err != nil {
		httperr.WriteError(w, fmt.Errorf("%w: %v", domain.ErrValidation, err))
		return
	}

	buckets, err := store.AggregateLatency(r.Context(), h.pool, filter)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	if buckets == nil {
		buckets = []store.LatencyBucket{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buckets); err != nil {
		log.Printf("encode latency response: %v", err)
	}
}

type commonParams struct {
	clientID *string
	api      *string
	bucket   string
	from     *time.Time
	to       *time.Time
}

func parseCommonParams(q map[string][]string) (commonParams, error) {
	p := commonParams{bucket: "day"}

	if v := getFirst(q, "client_id"); v != "" {
		p.clientID = &v
	}
	if v := getFirst(q, "api"); v != "" {
		p.api = &v
	}
	if v := getFirst(q, "bucket"); v != "" {
		if v != "hour" && v != "day" {
			return p, fmt.Errorf("invalid bucket %q: must be hour or day", v)
		}
		p.bucket = v
	}
	if v := getFirst(q, "from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return p, fmt.Errorf("invalid from %q: expected ISO8601 (RFC3339)", v)
		}
		p.from = &t
	}
	if v := getFirst(q, "to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return p, fmt.Errorf("invalid to %q: expected ISO8601 (RFC3339)", v)
		}
		p.to = &t
	}
	if p.from != nil && p.to != nil && p.from.After(*p.to) {
		return p, fmt.Errorf("from must not be after to")
	}
	return p, nil
}

func getFirst(q map[string][]string, key string) string {
	vals, ok := q[key]
	if !ok || len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func parseUsageParams(r *http.Request) (store.UsageFilter, error) {
	cp, err := parseCommonParams(r.URL.Query())
	if err != nil {
		return store.UsageFilter{}, err
	}

	filter := store.UsageFilter{
		ClientID: cp.clientID,
		API:      cp.api,
		Bucket:   cp.bucket,
		From:     cp.from,
		To:       cp.to,
	}

	if v := getFirst(r.URL.Query(), "status"); v != "" {
		if v != "allowed" && v != "rejected" && v != "all" {
			return filter, fmt.Errorf("invalid status %q: must be allowed, rejected, or all", v)
		}
		filter.Status = &v
	}

	return filter, nil
}

func parseLatencyParams(r *http.Request) (store.LatencyFilter, error) {
	cp, err := parseCommonParams(r.URL.Query())
	if err != nil {
		return store.LatencyFilter{}, err
	}

	return store.LatencyFilter{
		ClientID: cp.clientID,
		API:      cp.api,
		Bucket:   cp.bucket,
		From:     cp.from,
		To:       cp.to,
	}, nil
}
