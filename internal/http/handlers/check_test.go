package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sentinel/internal/engine"
	"sentinel/internal/http/dto"
	"sentinel/internal/http/router"
	"sentinel/internal/limiter"
	"sentinel/internal/repository/fake"
)

type fakeLimiter struct {
	dec limiter.Decision
	err error
}

func (f *fakeLimiter) Check(ctx context.Context, clientID, api string, rule limiter.Rule) (limiter.Decision, error) {
	return f.dec, f.err
}

type fakeResolver struct {
	rule limiter.Rule
	err  error
}

func (f *fakeResolver) Resolve(ctx context.Context, clientID, api string) (limiter.Rule, error) {
	return f.rule, f.err
}

type fakePostgresLimiter struct{}

func (f *fakePostgresLimiter) Check(ctx context.Context, clientID, api string, rule limiter.Rule) (limiter.Decision, error) {
	return limiter.Decision{}, errors.New("not used")
}

func routerWithEngine(dec limiter.Decision) http.Handler {
	eng := engine.New(
		&fakeLimiter{dec: dec},
		&fakeResolver{rule: limiter.Rule{RequestsAllowed: 10, WindowSeconds: 60}},
		&fakePostgresLimiter{},
	)
	clientRepo := fake.NewClientRepository()
	ruleRepo := fake.NewRateRuleRepository()
	return router.NewRouter(clientRepo, ruleRepo, eng)
}

func TestCheck_Allowed(t *testing.T) {
	r := routerWithEngine(limiter.Decision{Allowed: true, Remaining: 5})

	body := `{"client_id":"550e8400-e29b-41d4-a716-446655440000","api":"/api/test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.CheckAllowedResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Allowed {
		t.Error("expected allowed=true")
	}
	if resp.Remaining != 5 {
		t.Errorf("expected remaining=5, got %d", resp.Remaining)
	}
}

func TestCheck_Rejected(t *testing.T) {
	r := routerWithEngine(limiter.Decision{Allowed: false, RetryAfter: 30})

	body := `{"client_id":"550e8400-e29b-41d4-a716-446655440000","api":"/api/test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.CheckRejectedResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Allowed {
		t.Error("expected allowed=false")
	}
	if resp.RetryAfter != 30 {
		t.Errorf("expected retry_after=30, got %d", resp.RetryAfter)
	}
}

func TestCheck_400_MissingFields(t *testing.T) {
	r := routerWithEngine(limiter.Decision{Allowed: true, Remaining: 5})

	tests := []struct {
		name string
		body string
	}{
		{"empty object", `{}`},
		{"missing api", `{"client_id":"abc"}`},
		{"missing client_id", `{"api":"/test"}`},
		{"empty client_id", `{"client_id":"","api":"/test"}`},
		{"empty api", `{"client_id":"abc","api":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/check", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}

			var errResp dto.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("failed to decode error: %v", err)
			}
			if errResp.Error == "" {
				t.Error("expected non-empty error message")
			}
		})
	}
}

func TestCheck_400_MalformedJSON(t *testing.T) {
	r := routerWithEngine(limiter.Decision{Allowed: true, Remaining: 5})

	req := httptest.NewRequest(http.MethodPost, "/v1/check", strings.NewReader(`{bad json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp dto.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error: %v", err)
	}
	if errResp.Error == "" {
		t.Error("expected non-empty error message")
	}
}
