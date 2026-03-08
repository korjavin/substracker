package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/korjavin/substracker/internal/db"
	_ "modernc.org/sqlite"
)

func setupDB(t *testing.T) (*sql.DB, *Queries) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	if err := db.Migrate(database); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return database, New(database)
}

func TestSubscriptionIsolation(t *testing.T) {
	database, queries := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	sub1, err := queries.CreateSubscription(ctx, CreateSubscriptionParams{
		UserID:     1,
		Name:       "User 1 Sub",
		Service:    "claude",
		BillingDay: 10,
	})
	if err != nil {
		t.Fatalf("create sub1 failed: %v", err)
	}

	sub2, err := queries.CreateSubscription(ctx, CreateSubscriptionParams{
		UserID:     2,
		Name:       "User 2 Sub",
		Service:    "openai",
		BillingDay: 15,
	})
	if err != nil {
		t.Fatalf("create sub2 failed: %v", err)
	}

	subs1, err := queries.ListSubscriptions(ctx, 1)
	if err != nil || len(subs1) != 1 || subs1[0].ID != sub1.ID {
		t.Errorf("list subs for user 1 failed or returned wrong data")
	}

	subs2, err := queries.ListSubscriptions(ctx, 2)
	if err != nil || len(subs2) != 1 || subs2[0].ID != sub2.ID {
		t.Errorf("list subs for user 2 failed or returned wrong data")
	}

	_, err = queries.GetSubscription(ctx, sub1.ID, 2)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows when user 2 tries to get user 1's sub, got %v", err)
	}
}

func TestWebPushSubscriptionIsolation(t *testing.T) {
	database, queries := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	err := queries.UpsertWebPushSubscription(ctx, WebpushSubscriptionParams{
		UserID:   1,
		Endpoint: "endpoint1",
		P256dh:   "key1",
		Auth:     "auth1",
	})
	if err != nil {
		t.Fatalf("upsert sub1 failed: %v", err)
	}

	err = queries.UpsertWebPushSubscription(ctx, WebpushSubscriptionParams{
		UserID:   2,
		Endpoint: "endpoint2",
		P256dh:   "key2",
		Auth:     "auth2",
	})
	if err != nil {
		t.Fatalf("upsert sub2 failed: %v", err)
	}

	subs1, err := queries.ListWebPushSubscriptions(ctx, 1)
	if err != nil || len(subs1) != 1 || subs1[0].Endpoint != "endpoint1" {
		t.Errorf("list webpush subs for user 1 failed or returned wrong data")
	}

	err = queries.DeleteWebPushSubscription(ctx, "endpoint1", 2)
	if err != nil {
		t.Fatalf("delete sub error: %v", err)
	}

	subs1After, _ := queries.ListWebPushSubscriptions(ctx, 1)
	if len(subs1After) == 0 {
		t.Errorf("user 2 should not be able to delete user 1's webpush subscription")
	}

	// Test Ownership Transfer
	err = queries.UpsertWebPushSubscription(ctx, WebpushSubscriptionParams{
		UserID:   2,
		Endpoint: "endpoint1", // Same endpoint
		P256dh:   "new_key",
		Auth:     "new_auth",
	})
	if err != nil {
		t.Fatalf("upsert ownership transfer failed: %v", err)
	}

	subs1AfterTransfer, _ := queries.ListWebPushSubscriptions(ctx, 1)
	if len(subs1AfterTransfer) != 0 {
		t.Errorf("user 1 should no longer own endpoint1")
	}

	subs2AfterTransfer, _ := queries.ListWebPushSubscriptions(ctx, 2)
	if len(subs2AfterTransfer) != 2 { // Now owns endpoint2 and endpoint1
		t.Errorf("user 2 should own 2 endpoints, got %d", len(subs2AfterTransfer))
	}
}

func TestTelegramChatIsolation(t *testing.T) {
	database, queries := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	err := queries.CreateTelegramChat(ctx, "chat1", 1)
	if err != nil {
		t.Fatalf("create chat1 failed: %v", err)
	}

	err = queries.CreateTelegramChat(ctx, "chat2", 2)
	if err != nil {
		t.Fatalf("create chat2 failed: %v", err)
	}

	chats1, err := queries.ListTelegramChats(ctx, 1)
	if err != nil || len(chats1) != 1 || chats1[0].ChatID != "chat1" {
		t.Errorf("list chats for user 1 failed or returned wrong data")
	}

	err = queries.DeleteTelegramChat(ctx, "chat1", 2)
	if err != nil {
		t.Fatalf("delete chat error: %v", err)
	}

	chats1After, _ := queries.ListTelegramChats(ctx, 1)
	if len(chats1After) == 0 {
		t.Errorf("user 2 should not be able to delete user 1's telegram chat")
	}
}

func TestNotificationLogIsolation(t *testing.T) {
	database, queries := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	sub1, _ := queries.CreateSubscription(ctx, CreateSubscriptionParams{
		UserID:     1,
		Name:       "User 1 Sub",
		Service:    "claude",
		BillingDay: 10,
	})

	sub2, _ := queries.CreateSubscription(ctx, CreateSubscriptionParams{
		UserID:     2,
		Name:       "User 2 Sub",
		Service:    "openai",
		BillingDay: 15,
	})

	err := queries.CreateNotificationLog(ctx, CreateNotificationLogParams{
		SubscriptionID: sub1.ID,
		Channel:        "telegram",
		Message:        "Test 1",
	})
	if err != nil {
		t.Fatalf("create log1 failed: %v", err)
	}

	err = queries.CreateNotificationLog(ctx, CreateNotificationLogParams{
		SubscriptionID: sub2.ID,
		Channel:        "webpush",
		Message:        "Test 2",
	})
	if err != nil {
		t.Fatalf("create log2 failed: %v", err)
	}

	logs1, err := queries.ListNotificationLogs(ctx, 1)
	if err != nil || len(logs1) != 1 || logs1[0].Message != "Test 1" {
		t.Errorf("list logs for user 1 failed or returned wrong data")
	}

	logs2, err := queries.ListNotificationLogs(ctx, 2)
	if err != nil || len(logs2) != 1 || logs2[0].Message != "Test 2" {
		t.Errorf("list logs for user 2 failed or returned wrong data")
	}
}

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
