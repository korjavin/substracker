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
func (m *mockProvider) FetchUsageInfo(ctx context.Context, creds map[string]string) (*provider.UsageInfo, error) {
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
		CREATE TABLE subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 1,
			name TEXT NOT NULL,
			service TEXT NOT NULL,
			billing_day INTEGER NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			auth_token TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE subscription_usage (
			subscription_id INTEGER PRIMARY KEY REFERENCES subscriptions(id) ON DELETE CASCADE,
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

	// Insert subscription
	sub, err := repo.CreateSubscription(ctx, repository.CreateSubscriptionParams{
		UserID:     1,
		Name:       "Test Sub",
		Service:    "testprovider",
		BillingDay: 1,
		AuthToken:  "test_token",
	})
	if err != nil {
		t.Fatalf("failed to insert sub: %v", err)
	}

	scheduler := NewScheduler(repo, notif, logger, []provider.Provider{p}, time.Minute)

	// 1. Initial state (unblocked)
	scheduler.pollQuota(ctx)
	u, err := repo.GetSubscriptionUsage(ctx, sub.ID)
	if err != nil {
		t.Fatalf("expected usage to be saved: %v", err)
	}
	if u.IsBlocked {
		t.Errorf("expected is_blocked to be false")
	}

	// 2. Transition to blocked
	p.isBlocked = true
	scheduler.pollQuota(ctx)
	u, _ = repo.GetSubscriptionUsage(ctx, sub.ID)
	if !u.IsBlocked {
		t.Errorf("expected is_blocked to be true")
	}

	// 3. Transition to unblocked
	p.isBlocked = false
	scheduler.pollQuota(ctx)
	u, _ = repo.GetSubscriptionUsage(ctx, sub.ID)
	if u.IsBlocked {
		t.Errorf("expected is_blocked to be false")
	}
}
