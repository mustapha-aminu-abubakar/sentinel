package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"sentinel/internal/domain"
	"sentinel/internal/http/dto"
	"sentinel/internal/repository/fake"

	"sentinel/internal/http/router"
)

func routerWithFakes() (http.Handler, *fake.ClientRepository, *fake.RateRuleRepository) {
	clientRepo := fake.NewClientRepository()
	ruleRepo := fake.NewRateRuleRepository()
	return router.NewRouter(clientRepo, ruleRepo), clientRepo, ruleRepo
}

func mustCreateClient(t *testing.T, repo *fake.ClientRepository, name string) domain.Client {
	t.Helper()
	c, err := repo.Create(context.Background(), domain.Client{
		ID:     uuid.New(),
		Name:   name,
		Status: domain.ClientStatusActive,
	})
	if err != nil {
		t.Fatalf("setup create client: %v", err)
	}
	return c
}

func TestClients_Create_201(t *testing.T) {
	router, _, _ := routerWithFakes()

	body := `{"name":"Acme"}`
	req := httptest.NewRequest(http.MethodPost, "/clients", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var resp dto.ClientResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected non-empty id")
	}
	if resp.Name != "Acme" {
		t.Errorf("expected name 'Acme', got %q", resp.Name)
	}
	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.Status)
	}
	if resp.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
	if resp.UpdatedAt == "" {
		t.Error("expected non-empty updated_at")
	}
}

func TestClients_Create_400_EmptyName(t *testing.T) {
	router, _, _ := routerWithFakes()

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/clients", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

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

func TestClients_Get_200(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()
	created := mustCreateClient(t, clientRepo, "TestCo")

	req := httptest.NewRequest(http.MethodGet, "/clients/"+created.ID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp dto.ClientResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Name != "TestCo" {
		t.Errorf("expected 'TestCo', got %q", resp.Name)
	}
}

func TestClients_Get_404(t *testing.T) {
	router, _, _ := routerWithFakes()

	req := httptest.NewRequest(http.MethodGet, "/clients/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestClients_Update_Deactivate(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()
	created := mustCreateClient(t, clientRepo, "TestCo")

	body := `{"status":"inactive"}`
	req := httptest.NewRequest(http.MethodPatch, "/clients/"+created.ID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.ClientResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Status != "inactive" {
		t.Errorf("expected status 'inactive', got %q", resp.Status)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/clients/"+created.ID.String(), nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	var getResp dto.ClientResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}
	if getResp.Status != "inactive" {
		t.Errorf("expected GET to show 'inactive', got %q", getResp.Status)
	}
}

func TestClients_List_FilterByStatus(t *testing.T) {
	router, clientRepo, _ := routerWithFakes()

	mustCreateClient(t, clientRepo, "ActiveCo")
	toDeactivate := mustCreateClient(t, clientRepo, "ToDeactivate")

	clientRepo.Deactivate(context.Background(), toDeactivate.ID)

	req := httptest.NewRequest(http.MethodGet, "/clients?status=active", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string][]dto.ClientResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	clients := body["clients"]
	if len(clients) != 1 {
		t.Errorf("expected 1 active client, got %d", len(clients))
	}
	if len(clients) > 0 && clients[0].Name != "ActiveCo" {
		t.Errorf("expected 'ActiveCo', got %q", clients[0].Name)
	}
}
