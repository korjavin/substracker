package api

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
	"github.com/korjavin/substracker/internal/repository"
	_ "modernc.org/sqlite"
)

// mockProvider implements provider.Provider for testing API endpoints
type mockProvider struct {
	sessionKey string
	shouldFail bool
}

func (m *mockProvider) Name() string {
	return "MockClaude"
}

func (m *mockProvider) FetchUsageInfo(ctx context.Context, credentials map[string]string) (*provider.UsageInfo, error) {
	if m.shouldFail {
		return nil, provider.ErrUnauthorized
	}
	sessionKey := credentials["session_key"]
	if sessionKey == "" {
		return nil, provider.ErrUnauthorized
	}
	return &provider.UsageInfo{
		ResetDate:           time.Now(),
		CurrentUsageSeconds: 3600,
		TotalLimitSeconds:   7200,
		IsBlocked:           true,
	}, nil
}

func setupTestDB(t *testing.T) *repository.Queries {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE provider_credentials (
			provider_name TEXT NOT NULL,
			credential_key TEXT NOT NULL,
			credential_value TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider_name, credential_key)
		);

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
	return repository.New(db)
}
