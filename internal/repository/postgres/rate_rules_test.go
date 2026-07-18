package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"sentinel/internal/domain"
	"sentinel/internal/repository"
)

func seedRateRule(t *testing.T, clientID uuid.UUID, api string) domain.RateRule {
	t.Helper()
	repo := NewRateRuleRepository(testPool)
	rule, err := repo.Create(context.Background(), domain.RateRule{
		ID:              uuid.New(),
		ClientID:        clientID,
		API:             api,
		RequestsAllowed: 100,
		WindowSeconds:   60,
	})
	if err != nil {
		t.Fatalf("seed rate rule: %v", err)
	}
	return rule
}

func TestRateRuleRepository_CreateAndGet(t *testing.T) {
	client := seedClient(t, "rule-test-client")
	repo := NewRateRuleRepository(testPool)

	created, err := repo.Create(context.Background(), domain.RateRule{
		ID:              uuid.New(),
		ClientID:        client.ID,
		API:             "test-api",
		RequestsAllowed: 100,
		WindowSeconds:   60,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if created.API != "test-api" {
		t.Fatalf("expected api 'test-api', got %q", created.API)
	}
	if created.RequestsAllowed != 100 {
		t.Fatalf("expected requests_allowed 100, got %d", created.RequestsAllowed)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}

	got, err := repo.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.API != "test-api" {
		t.Fatalf("Get api: expected 'test-api', got %q", got.API)
	}
}

func TestRateRuleRepository_ListByClient_Isolation(t *testing.T) {
	clientA := seedClient(t, "client-a")
	clientB := seedClient(t, "client-b")

	ruleA := seedRateRule(t, clientA.ID, "api-for-a")
	seedRateRule(t, clientB.ID, "api-for-b")

	repo := NewRateRuleRepository(testPool)

	rules, err := repo.ListByClient(context.Background(), clientA.ID)
	if err != nil {
		t.Fatalf("ListByClient: %v", err)
	}

	for _, r := range rules {
		if r.ClientID != clientA.ID {
			t.Fatalf("rule %v belongs to client %v, not client A %v", r.ID, r.ClientID, clientA.ID)
		}
	}

	foundA := false
	for _, r := range rules {
		if r.ID == ruleA.ID {
			foundA = true
			break
		}
	}
	if !foundA {
		t.Fatal("expected ruleA to be returned for client A")
	}
}

func TestRateRuleRepository_Update_BumpsUpdatedAt(t *testing.T) {
	client := seedClient(t, "update-rule-client")
	rule := seedRateRule(t, client.ID, "update-test")

	repo := NewRateRuleRepository(testPool)

	allowed := 200
	window := 120
	updated, err := repo.Update(context.Background(), rule.ID, repository.RateRuleUpdate{
		RequestsAllowed: &allowed,
		WindowSeconds:   &window,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.RequestsAllowed != 200 {
		t.Fatalf("expected requests_allowed 200, got %d", updated.RequestsAllowed)
	}
	if updated.WindowSeconds != 120 {
		t.Fatalf("expected window_seconds 120, got %d", updated.WindowSeconds)
	}
	if !updated.UpdatedAt.After(rule.UpdatedAt) {
		t.Fatal("expected updated_at to be bumped")
	}
}

func TestRateRuleRepository_Create_DuplicateReturnsConflict(t *testing.T) {
	client := seedClient(t, "dup-rule-client")
	repo := NewRateRuleRepository(testPool)

	_, err := repo.Create(context.Background(), domain.RateRule{
		ID:              uuid.New(),
		ClientID:        client.ID,
		API:             "duplicate-api",
		RequestsAllowed: 100,
		WindowSeconds:   60,
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = repo.Create(context.Background(), domain.RateRule{
		ID:              uuid.New(),
		ClientID:        client.ID,
		API:             "duplicate-api",
		RequestsAllowed: 200,
		WindowSeconds:   120,
	})
	if err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict for duplicate, got %v", err)
	}
}

func TestRateRuleRepository_Create_InvalidInput(t *testing.T) {
	client := seedClient(t, "invalid-rule-client")
	repo := NewRateRuleRepository(testPool)

	_, err := repo.Create(context.Background(), domain.RateRule{
		ID:              uuid.New(),
		ClientID:        client.ID,
		API:             "",
		RequestsAllowed: 100,
		WindowSeconds:   60,
	})
	if err != domain.ErrValidation {
		t.Fatalf("expected ErrValidation for empty api, got %v", err)
	}

	_, err = repo.Create(context.Background(), domain.RateRule{
		ID:              uuid.New(),
		ClientID:        client.ID,
		API:             "valid-api",
		RequestsAllowed: 0,
		WindowSeconds:   60,
	})
	if err != domain.ErrValidation {
		t.Fatalf("expected ErrValidation for zero requests_allowed, got %v", err)
	}
}

func TestRateRuleRepository_GetByClientAndAPI(t *testing.T) {
	client := seedClient(t, "lookup-rule-client")
	rule := seedRateRule(t, client.ID, "lookup-key")

	repo := NewRateRuleRepository(testPool)

	found, err := repo.GetByClientAndAPI(context.Background(), client.ID, "lookup-key")
	if err != nil {
		t.Fatalf("GetByClientAndAPI: %v", err)
	}
	if found.ID != rule.ID {
		t.Fatalf("expected rule id %v, got %v", rule.ID, found.ID)
	}

	_, err = repo.GetByClientAndAPI(context.Background(), client.ID, "nonexistent")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound for missing (client, api), got %v", err)
	}
}

func TestRateRuleRepository_CascadeDelete(t *testing.T) {
	client := seedClient(t, "cascade-delete-client")
	seedRateRule(t, client.ID, "cascade-rule")

	clientRepo := NewClientRepository(testPool)
	ruleRepo := NewRateRuleRepository(testPool)

	_, err := clientRepo.Deactivate(context.Background(), client.ID)
	if err != nil {
		t.Fatalf("Deactivate client: %v", err)
	}

	rules, err := ruleRepo.ListByClient(context.Background(), client.ID)
	if err != nil {
		t.Fatalf("ListByClient after deactivate: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after deactivate (cascade on delete only), got %d", len(rules))
	}
}
