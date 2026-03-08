package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/korjavin/substracker/internal/db"
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
