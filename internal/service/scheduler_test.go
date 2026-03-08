package service

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
	"github.com/korjavin/substracker/internal/repository"
	_ "modernc.org/sqlite"
)

type mockProvider struct {
	name      string
	isBlocked bool
	err       error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Login(ctx context.Context, creds map[string]string) error { return nil }
func (m *mockProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &provider.UsageInfo{
		ResetDate:           time.Now(),
		CurrentUsageSeconds: 0,
		TotalLimitSeconds:   0,
		IsBlocked:           m.isBlocked,
	}, nil
}

func TestSchedulerPollQuota(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE provider_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_name TEXT UNIQUE NOT NULL,
			current_usage_seconds INTEGER NOT NULL DEFAULT 0,
			total_limit_seconds INTEGER NOT NULL DEFAULT 0,
			is_blocked INTEGER NOT NULL DEFAULT 0,
			fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE webpush_subscriptions (id INTEGER PRIMARY KEY, endpoint TEXT, p256dh TEXT, auth TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE telegram_chats (id INTEGER PRIMARY KEY, chat_id TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE notification_log (id INTEGER PRIMARY KEY, subscription_id INTEGER, channel TEXT, message TEXT, sent_at DATETIME DEFAULT CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	repo := repository.New(db)
	notif := NewNotificationService(repo, NotificationConfig{})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx := context.Background()
	p := &mockProvider{name: "TestProvider", isBlocked: false}

	scheduler := NewScheduler(repo, notif, logger, []provider.Provider{p}, time.Minute)

	// 1. Initial state (unblocked)
	scheduler.pollQuota(ctx)
	u, err := repo.GetProviderUsage(ctx, "TestProvider")
	if err != nil {
		t.Fatalf("expected usage to be saved: %v", err)
	}
	if u.IsBlocked {
		t.Errorf("expected is_blocked to be false")
	}

	// 2. Transition to blocked
	p.isBlocked = true
	scheduler.pollQuota(ctx)
	u, _ = repo.GetProviderUsage(ctx, "TestProvider")
	if !u.IsBlocked {
		t.Errorf("expected is_blocked to be true")
	}

	// 3. Transition to unblocked
	p.isBlocked = false
	scheduler.pollQuota(ctx)
	u, _ = repo.GetProviderUsage(ctx, "TestProvider")
	if u.IsBlocked {
		t.Errorf("expected is_blocked to be false")
	}
}
