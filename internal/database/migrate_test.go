package database

import (
	"database/sql"
	"log"
	"testing"
)

func mustGetTestDB(t *testing.T) *sql.DB {
	t.Helper()

	if dbInstance == nil {
		_, err := mustStartPostgresContainer()
		if err != nil {
			t.Fatalf("could not start postgres container: %v", err)
		}
	}

	srv := New()
	type dbProvider interface{ DB() *sql.DB }
	if d, ok := srv.(dbProvider); ok {
		return d.DB()
	}
	// Fallback: reconstruct from New() internals by calling the unexported field
	// This shouldn't happen since we added DB() in this subtask.
	t.Fatal("Service does not implement DB() accessor")
	return nil
}

func TestRunMigrations_CreatesTables(t *testing.T) {
	db := mustGetTestDB(t)

	if err := RunMigrations(db, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	var tableCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name IN ('clients', 'rate_rules')
	`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("query tables failed: %v", err)
	}
	if tableCount != 2 {
		t.Fatalf("expected 2 tables, got %d", tableCount)
	}

	var colCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'clients'
	`).Scan(&colCount)
	if err != nil {
		t.Fatalf("query columns failed: %v", err)
	}
	if colCount != 5 {
		t.Fatalf("expected 5 columns in clients, got %d", colCount)
	}

	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'rate_rules'
	`).Scan(&colCount)
	if err != nil {
		t.Fatalf("query columns failed: %v", err)
	}
	if colCount != 7 {
		t.Fatalf("expected 7 columns in rate_rules, got %d", colCount)
	}

	// Re-run must be idempotent
	if err := RunMigrations(db, "../../migrations"); err != nil {
		t.Fatalf("second RunMigrations failed (not idempotent): %v", err)
	}
}

func TestRunMigrations_FKCascade(t *testing.T) {
	db := mustGetTestDB(t)

	if err := RunMigrations(db, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	var clientID string
	err := db.QueryRow(`INSERT INTO clients (name) VALUES ('test-client') RETURNING id`).Scan(&clientID)
	if err != nil {
		t.Fatalf("insert client failed: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO rate_rules (client_id, api, requests_allowed, window_seconds) VALUES ($1, 'test-api', 100, 60)`,
		clientID,
	)
	if err != nil {
		t.Fatalf("insert rate_rule failed: %v", err)
	}

	_, err = db.Exec(`DELETE FROM clients WHERE id = $1`, clientID)
	if err != nil {
		t.Fatalf("delete client failed: %v", err)
	}

	var ruleCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM rate_rules WHERE client_id = $1`, clientID).Scan(&ruleCount)
	if err != nil {
		t.Fatalf("count rules failed: %v", err)
	}
	if ruleCount != 0 {
		t.Fatalf("expected 0 rules after cascade delete, got %d", ruleCount)
	}
}

func TestRunMigrations_CheckConstraints(t *testing.T) {
	db := mustGetTestDB(t)

	if err := RunMigrations(db, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	var clientID string
	err := db.QueryRow(`INSERT INTO clients (name) VALUES ('test-client') RETURNING id`).Scan(&clientID)
	if err != nil {
		t.Fatalf("insert client failed: %v", err)
	}

	// requests_allowed = 0 should be rejected
	_, err = db.Exec(
		`INSERT INTO rate_rules (client_id, api, requests_allowed, window_seconds) VALUES ($1, 'test-api', 0, 60)`,
		clientID,
	)
	if err == nil {
		t.Fatal("expected error for requests_allowed=0, got nil")
	}
	log.Printf("expected error for requests_allowed=0: %v", err)

	// window_seconds = 0 should be rejected
	_, err = db.Exec(
		`INSERT INTO rate_rules (client_id, api, requests_allowed, window_seconds) VALUES ($1, 'test-api', 100, 0)`,
		clientID,
	)
	if err == nil {
		t.Fatal("expected error for window_seconds=0, got nil")
	}
	log.Printf("expected error for window_seconds=0: %v", err)
}

func TestRunMigrations_UniqueConstraint(t *testing.T) {
	db := mustGetTestDB(t)

	if err := RunMigrations(db, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	var clientID string
	err := db.QueryRow(`INSERT INTO clients (name) VALUES ('test-client') RETURNING id`).Scan(&clientID)
	if err != nil {
		t.Fatalf("insert client failed: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO rate_rules (client_id, api, requests_allowed, window_seconds) VALUES ($1, 'openai', 100, 60)`,
		clientID,
	)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Duplicate (client_id, api) should be rejected
	_, err = db.Exec(
		`INSERT INTO rate_rules (client_id, api, requests_allowed, window_seconds) VALUES ($1, 'openai', 200, 120)`,
		clientID,
	)
	if err == nil {
		t.Fatal("expected error for duplicate (client_id, api), got nil")
	}
	log.Printf("expected error for duplicate: %v", err)
}
