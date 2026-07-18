package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/dbtest"
	"sentinel/internal/domain"
	"sentinel/internal/repository"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	if _, ok := os.LookupEnv("DATABASE_URL_TEST"); ok {
		testPool = mustConnect(os.Getenv("DATABASE_URL_TEST"))
	} else {
		var teardown func()
		var err error
		testPool, teardown, err = dbtest.StartPostgres()
		if err != nil {
			panic("failed to start postgres: " + err.Error())
		}
		defer teardown()
	}

	if err := dbtest.RunMigrations(dsn(), "../../../migrations"); err != nil {
		panic("failed to run migrations: " + err.Error())
	}

	m.Run()
}

func dsn() string {
	if dsn, ok := os.LookupEnv("DATABASE_URL_TEST"); ok {
		return dsn
	}
	return ""
}

func mustConnect(dsn string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		panic("failed to connect: " + err.Error())
	}
	return pool
}

func seedClient(t *testing.T, name string) domain.Client {
	t.Helper()
	repo := NewClientRepository(testPool)
	client, err := repo.Create(context.Background(), domain.Client{
		ID:     uuid.New(),
		Name:   name,
		Status: domain.ClientStatusActive,
	})
	if err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return client
}

func TestClientRepository_CreateAndGet(t *testing.T) {
	repo := NewClientRepository(testPool)

	id := uuid.New()
	created, err := repo.Create(context.Background(), domain.Client{
		ID:     id,
		Name:   "test-client",
		Status: domain.ClientStatusActive,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if created.ID != id {
		t.Fatalf("expected id %v, got %v", id, created.ID)
	}
	if created.Name != "test-client" {
		t.Fatalf("expected name 'test-client', got %q", created.Name)
	}
	if created.Status != domain.ClientStatusActive {
		t.Fatalf("expected status active, got %q", created.Status)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
	if created.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero UpdatedAt")
	}

	got, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test-client" {
		t.Fatalf("Get name: expected 'test-client', got %q", got.Name)
	}
}

func TestClientRepository_Get_NotFound(t *testing.T) {
	repo := NewClientRepository(testPool)

	_, err := repo.Get(context.Background(), uuid.New())
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClientRepository_List_FilterByStatus(t *testing.T) {
	repo := NewClientRepository(testPool)

	c1 := seedClient(t, "active-client-1")
	c2 := seedClient(t, "active-client-2")

	active := domain.ClientStatusActive
	all, err := repo.List(context.Background(), repository.ListClientsParams{
		Status: &active,
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := 0
	for _, c := range all {
		if c.ID == c1.ID || c.ID == c2.ID {
			found++
		}
	}
	if found < 2 {
		t.Fatalf("expected at least 2 active clients, found %d", found)
	}
}

func TestClientRepository_Update(t *testing.T) {
	repo := NewClientRepository(testPool)

	client := seedClient(t, "before-update")

	inactive := domain.ClientStatusInactive
	name := "after-update"
	updated, err := repo.Update(context.Background(), client.ID, repository.ClientUpdate{
		Name:   &name,
		Status: &inactive,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "after-update" {
		t.Fatalf("expected name 'after-update', got %q", updated.Name)
	}
	if updated.Status != domain.ClientStatusInactive {
		t.Fatalf("expected status inactive, got %q", updated.Status)
	}
	if !updated.UpdatedAt.After(client.UpdatedAt) {
		t.Fatal("expected updated_at to be bumped")
	}
}

func TestClientRepository_Deactivate(t *testing.T) {
	repo := NewClientRepository(testPool)

	client := seedClient(t, "to-deactivate")

	deactivated, err := repo.Deactivate(context.Background(), client.ID)
	if err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	if deactivated.Status != domain.ClientStatusInactive {
		t.Fatalf("expected inactive, got %q", deactivated.Status)
	}
	if !deactivated.UpdatedAt.After(client.UpdatedAt) {
		t.Fatal("expected updated_at to be bumped")
	}

	second, err := repo.Deactivate(context.Background(), client.ID)
	if err != nil {
		t.Fatalf("second Deactivate: %v", err)
	}
	if second.Status != domain.ClientStatusInactive {
		t.Fatalf("expected still inactive, got %q", second.Status)
	}
}

func TestClientRepository_Create_InvalidName(t *testing.T) {
	repo := NewClientRepository(testPool)

	_, err := repo.Create(context.Background(), domain.Client{
		ID:     uuid.New(),
		Name:   "",
		Status: domain.ClientStatusActive,
	})
	if err != domain.ErrValidation {
		t.Fatalf("expected ErrValidation for empty name, got %v", err)
	}
}
