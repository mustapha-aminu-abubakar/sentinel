package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"sentinel/internal/http/dto"
)

func mustCreateRule(t *testing.T, router http.Handler, clientID uuid.UUID, api string, allowed int, window int) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"client_id":"` + clientID.String() + `","api":"` + api + `","requests_allowed":` + strconv.Itoa(allowed) + `,"window_seconds":` + strconv.Itoa(window) + `}`
	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRules_Create_201(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()
	client := mustCreateClient(t, clientRepo, "TestCo")

	w := mustCreateRule(t, router, client.ID, "users-api", 1000, 60)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.RuleResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected non-empty id")
	}
	if resp.ClientID != client.ID.String() {
		t.Errorf("expected client_id %q, got %q", client.ID.String(), resp.ClientID)
	}
	if resp.API != "users-api" {
		t.Errorf("expected api 'users-api', got %q", resp.API)
	}
	if resp.RequestsAllowed != 1000 {
		t.Errorf("expected 1000, got %d", resp.RequestsAllowed)
	}
	if resp.WindowSeconds != 60 {
		t.Errorf("expected 60, got %d", resp.WindowSeconds)
	}
}

func TestRules_Create_400_InvalidRequestsAllowed(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()
	client := mustCreateClient(t, clientRepo, "TestCo")

	w := mustCreateRule(t, router, client.ID, "users-api", 0, 60)

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

func TestRules_Create_409_Duplicate(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()
	client := mustCreateClient(t, clientRepo, "TestCo")

	w1 := mustCreateRule(t, router, client.ID, "users-api", 1000, 60)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first create expected 201, got %d", w1.Code)
	}

	w2 := mustCreateRule(t, router, client.ID, "users-api", 2000, 120)
	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRules_Patch_200(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()
	client := mustCreateClient(t, clientRepo, "TestCo")
	w := mustCreateRule(t, router, client.ID, "users-api", 1000, 60)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup create expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created dto.RuleResponse
	json.Unmarshal(w.Body.Bytes(), &created)

	patchBody := `{"requests_allowed":5000,"window_seconds":60}`
	req := httptest.NewRequest(http.MethodPatch, "/rules/"+created.ID, strings.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	pw := httptest.NewRecorder()
	router.ServeHTTP(pw, req)

	if pw.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", pw.Code, pw.Body.String())
	}

	var updated dto.RuleResponse
	if err := json.Unmarshal(pw.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if updated.RequestsAllowed != 5000 {
		t.Errorf("expected requests_allowed 5000, got %d", updated.RequestsAllowed)
	}
	if updated.WindowSeconds != 60 {
		t.Errorf("expected window_seconds 60, got %d", updated.WindowSeconds)
	}
}

func TestRules_List_ByClientID(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()
	clientA := mustCreateClient(t, clientRepo, "ClientA")
	clientB := mustCreateClient(t, clientRepo, "ClientB")

	mustCreateRule(t, router, clientA.ID, "api-a1", 100, 60)
	mustCreateRule(t, router, clientA.ID, "api-a2", 200, 60)
	mustCreateRule(t, router, clientB.ID, "api-b1", 300, 60)

	req := httptest.NewRequest(http.MethodGet, "/rules?client_id="+clientA.ID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string][]dto.RuleResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	rules := body["rules"]
	if len(rules) != 2 {
		t.Errorf("expected 2 rules for client A, got %d", len(rules))
	}

	for _, rule := range rules {
		if rule.ClientID != clientA.ID.String() {
			t.Errorf("expected rule for client A, got client_id %q", rule.ClientID)
		}
	}
}
