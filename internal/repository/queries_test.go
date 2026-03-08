package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestProviderUsage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	// Run bare minimum schema for testing
	_, err = db.Exec(`
		CREATE TABLE provider_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_name TEXT UNIQUE NOT NULL,
			current_usage_seconds INTEGER NOT NULL DEFAULT 0,
			total_limit_seconds INTEGER NOT NULL DEFAULT 0,
			is_blocked INTEGER NOT NULL DEFAULT 0,
			fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	repo := New(db)
	ctx := context.Background()

	// 1. Get non-existent
	_, err = repo.GetProviderUsage(ctx, "Claude")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	// 2. Insert
	err = repo.UpsertProviderUsage(ctx, UpsertProviderUsageParams{
		ProviderName:        "Claude",
		CurrentUsageSeconds: 3600,
		TotalLimitSeconds:   7200,
		IsBlocked:           false,
	})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	u, err := repo.GetProviderUsage(ctx, "Claude")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if u.ProviderName != "Claude" || u.CurrentUsageSeconds != 3600 || u.TotalLimitSeconds != 7200 || u.IsBlocked != false {
		t.Errorf("unexpected values after insert: %+v", u)
	}

	// 3. Update
	err = repo.UpsertProviderUsage(ctx, UpsertProviderUsageParams{
		ProviderName:        "Claude",
		CurrentUsageSeconds: 7200,
		TotalLimitSeconds:   7200,
		IsBlocked:           true,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	u, err = repo.GetProviderUsage(ctx, "Claude")
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if u.CurrentUsageSeconds != 7200 || !u.IsBlocked {
		t.Errorf("unexpected values after update: %+v", u)
	}

	// 4. List
	err = repo.UpsertProviderUsage(ctx, UpsertProviderUsageParams{
		ProviderName:        "GoogleOne",
		CurrentUsageSeconds: 100,
		TotalLimitSeconds:   1000,
		IsBlocked:           false,
	})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	usages, err := repo.ListProviderUsage(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(usages) != 2 {
		t.Fatalf("expected 2 usages, got %d", len(usages))
	}
	if usages[0].ProviderName != "Claude" || usages[1].ProviderName != "GoogleOne" {
		t.Errorf("unexpected list results: %+v", usages)
	}
}

func TestProviderCredentials(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE provider_credentials (
			provider_name TEXT NOT NULL,
			credential_key TEXT NOT NULL,
			credential_value TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider_name, credential_key)
		);
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	repo := New(db)
	ctx := context.Background()

	// 1. Get non-existent
	_, err = repo.GetProviderCredential(ctx, "Claude", "session_key")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	// 2. Insert
	err = repo.UpsertProviderCredential(ctx, "Claude", "session_key", "test_value")
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	val, err := repo.GetProviderCredential(ctx, "Claude", "session_key")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", val)
	}

	// 3. Update
	err = repo.UpsertProviderCredential(ctx, "Claude", "session_key", "new_value")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	val, err = repo.GetProviderCredential(ctx, "Claude", "session_key")
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if val != "new_value" {
		t.Errorf("expected 'new_value', got '%s'", val)
	}
}
