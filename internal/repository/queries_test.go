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
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (provider_name, key)
		);
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	repo := New(db)
	ctx := context.Background()

	// 1. Get empty
	creds, err := repo.GetProviderCredentials(ctx, "Claude")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected empty map, got %v", creds)
	}

	// 2. Insert
	err = repo.UpsertProviderCredential(ctx, UpsertProviderCredentialParams{
		ProviderName: "Claude",
		Key:          "session_key",
		Value:        "key123",
	})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	creds, err = repo.GetProviderCredentials(ctx, "Claude")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(creds) != 1 || creds["session_key"] != "key123" {
		t.Errorf("unexpected values after insert: %+v", creds)
	}

	// 3. Update
	err = repo.UpsertProviderCredential(ctx, UpsertProviderCredentialParams{
		ProviderName: "Claude",
		Key:          "session_key",
		Value:        "key456",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	creds, err = repo.GetProviderCredentials(ctx, "Claude")
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if creds["session_key"] != "key456" {
		t.Errorf("unexpected values after update: %+v", creds)
	}
}
