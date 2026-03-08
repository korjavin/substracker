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
}

func TestProviderCredential(t *testing.T) {
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
	_, err = repo.GetProviderCredential(ctx, "Google One", "session_cookie")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	// 2. Insert
	err = repo.UpsertProviderCredential(ctx, UpsertProviderCredentialParams{
		ProviderName:    "Google One",
		CredentialKey:   "session_cookie",
		CredentialValue: "test_cookie_123",
	})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	c, err := repo.GetProviderCredential(ctx, "Google One", "session_cookie")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if c.ProviderName != "Google One" || c.CredentialKey != "session_cookie" || c.CredentialValue != "test_cookie_123" {
		t.Errorf("unexpected values after insert: %+v", c)
	}

	// 3. Update
	err = repo.UpsertProviderCredential(ctx, UpsertProviderCredentialParams{
		ProviderName:    "Google One",
		CredentialKey:   "session_cookie",
		CredentialValue: "new_cookie_456",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	c, err = repo.GetProviderCredential(ctx, "Google One", "session_cookie")
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if c.CredentialValue != "new_cookie_456" {
		t.Errorf("unexpected values after update: %+v", c)
	}
}