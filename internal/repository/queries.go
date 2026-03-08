package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Queries struct {
	db *sql.DB
}

func New(db *sql.DB) *Queries {
	return &Queries{db: db}
}

// --- Subscriptions ---

func (q *Queries) CreateSubscription(ctx context.Context, arg CreateSubscriptionParams) (Subscription, error) {
	row := q.db.QueryRowContext(ctx,
		`INSERT INTO subscriptions (name, service, billing_day, notes)
		 VALUES (?, ?, ?, ?)
		 RETURNING id, name, service, billing_day, notes, created_at, updated_at`,
		arg.Name, arg.Service, arg.BillingDay, arg.Notes,
	)
	return scanSubscriptionRow(row)
}

func (q *Queries) GetSubscription(ctx context.Context, id int64) (Subscription, error) {
	row := q.db.QueryRowContext(ctx,
		`SELECT id, name, service, billing_day, notes, created_at, updated_at
		 FROM subscriptions WHERE id = ? LIMIT 1`,
		id,
	)
	return scanSubscriptionRow(row)
}

func (q *Queries) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, name, service, billing_day, notes, created_at, updated_at
		 FROM subscriptions ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		sub, err := scanSubscriptionRows(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (q *Queries) UpdateSubscription(ctx context.Context, arg UpdateSubscriptionParams) (Subscription, error) {
	row := q.db.QueryRowContext(ctx,
		`UPDATE subscriptions
		 SET name=?, service=?, billing_day=?, notes=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?
		 RETURNING id, name, service, billing_day, notes, created_at, updated_at`,
		arg.Name, arg.Service, arg.BillingDay, arg.Notes, arg.ID,
	)
	return scanSubscriptionRow(row)
}

func (q *Queries) DeleteSubscription(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=?`, id)
	return err
}

// --- Web Push ---

func (q *Queries) UpsertWebPushSubscription(ctx context.Context, arg WebpushSubscriptionParams) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO webpush_subscriptions (endpoint, p256dh, auth)
		 VALUES (?, ?, ?)
		 ON CONFLICT(endpoint) DO UPDATE SET p256dh=excluded.p256dh, auth=excluded.auth`,
		arg.Endpoint, arg.P256dh, arg.Auth,
	)
	return err
}

func (q *Queries) ListWebPushSubscriptions(ctx context.Context) ([]WebpushSubscription, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, endpoint, p256dh, auth, created_at FROM webpush_subscriptions`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []WebpushSubscription
	for rows.Next() {
		var s WebpushSubscription
		var createdAt string
		if err := rows.Scan(&s.ID, &s.Endpoint, &s.P256dh, &s.Auth, &createdAt); err != nil {
			return nil, err
		}
		s.CreatedAt = parseTime(createdAt)
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (q *Queries) DeleteWebPushSubscription(ctx context.Context, endpoint string) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM webpush_subscriptions WHERE endpoint=?`, endpoint)
	return err
}

// --- Telegram Chats ---

func (q *Queries) CreateTelegramChat(ctx context.Context, chatID string) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO telegram_chats (chat_id) VALUES (?) ON CONFLICT(chat_id) DO NOTHING`,
		chatID,
	)
	return err
}

func (q *Queries) ListTelegramChats(ctx context.Context) ([]TelegramChat, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, chat_id, created_at FROM telegram_chats ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []TelegramChat
	for rows.Next() {
		var c TelegramChat
		var createdAt string
		if err := rows.Scan(&c.ID, &c.ChatID, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTime(createdAt)
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func (q *Queries) DeleteTelegramChat(ctx context.Context, chatID string) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM telegram_chats WHERE chat_id=?`, chatID)
	return err
}

// --- Provider Credentials ---

func (q *Queries) UpsertProviderCredential(ctx context.Context, arg UpsertProviderCredentialParams) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO provider_credentials (provider_name, credential_key, credential_value, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(provider_name, credential_key) DO UPDATE SET
			credential_value=excluded.credential_value,
			updated_at=CURRENT_TIMESTAMP`,
		arg.ProviderName, arg.CredentialKey, arg.CredentialValue,
	)
	return err
}

func (q *Queries) GetProviderCredential(ctx context.Context, providerName, credentialKey string) (ProviderCredential, error) {
	row := q.db.QueryRowContext(ctx,
		`SELECT provider_name, credential_key, credential_value, updated_at
		 FROM provider_credentials
		 WHERE provider_name = ? AND credential_key = ? LIMIT 1`,
		providerName, credentialKey,
	)
	var c ProviderCredential
	var updatedAt string
	err := row.Scan(&c.ProviderName, &c.CredentialKey, &c.CredentialValue, &updatedAt)
	if err != nil {
		return c, err
	}
	c.UpdatedAt = parseTime(updatedAt)
	return c, nil
}

// --- Provider Usage ---

func (q *Queries) UpsertProviderUsage(ctx context.Context, arg UpsertProviderUsageParams) error {
	isBlockedInt := 0
	if arg.IsBlocked {
		isBlockedInt = 1
	}
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO provider_usage (provider_name, current_usage_seconds, total_limit_seconds, is_blocked, fetched_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(provider_name) DO UPDATE SET
			current_usage_seconds=excluded.current_usage_seconds,
			total_limit_seconds=excluded.total_limit_seconds,
			is_blocked=excluded.is_blocked,
			fetched_at=CURRENT_TIMESTAMP`,
		arg.ProviderName, arg.CurrentUsageSeconds, arg.TotalLimitSeconds, isBlockedInt,
	)
	return err
}

func (q *Queries) GetProviderUsage(ctx context.Context, providerName string) (ProviderUsage, error) {
	row := q.db.QueryRowContext(ctx,
		`SELECT id, provider_name, current_usage_seconds, total_limit_seconds, is_blocked, fetched_at
		 FROM provider_usage WHERE provider_name = ? LIMIT 1`,
		providerName,
	)
	var u ProviderUsage
	var fetchedAt string
	var isBlockedInt int
	err := row.Scan(&u.ID, &u.ProviderName, &u.CurrentUsageSeconds, &u.TotalLimitSeconds, &isBlockedInt, &fetchedAt)
	if err != nil {
		return u, err
	}
	u.FetchedAt = parseTime(fetchedAt)
	u.IsBlocked = isBlockedInt == 1
	return u, nil
}

// --- Notification Log ---

func (q *Queries) CreateNotificationLog(ctx context.Context, arg CreateNotificationLogParams) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO notification_log (subscription_id, channel, message) VALUES (?, ?, ?)`,
		arg.SubscriptionID, arg.Channel, arg.Message,
	)
	return err
}

func (q *Queries) ListNotificationLogs(ctx context.Context) ([]NotificationLog, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, subscription_id, channel, message, sent_at
		 FROM notification_log ORDER BY sent_at DESC LIMIT 100`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []NotificationLog
	for rows.Next() {
		var l NotificationLog
		var sentAt string
		if err := rows.Scan(&l.ID, &l.SubscriptionID, &l.Channel, &l.Message, &sentAt); err != nil {
			return nil, err
		}
		l.SentAt = parseTime(sentAt)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// --- helpers ---

func scanSubscriptionRow(row *sql.Row) (Subscription, error) {
	var s Subscription
	var createdAt, updatedAt string
	err := row.Scan(&s.ID, &s.Name, &s.Service, &s.BillingDay, &s.Notes, &createdAt, &updatedAt)
	if err != nil {
		return s, err
	}
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	return s, nil
}

func scanSubscriptionRows(rows *sql.Rows) (Subscription, error) {
	var s Subscription
	var createdAt, updatedAt string
	err := rows.Scan(&s.ID, &s.Name, &s.Service, &s.BillingDay, &s.Notes, &createdAt, &updatedAt)
	if err != nil {
		return s, err
	}
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	return s, nil
}

func parseTime(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	// fallback: return zero time, log nothing to avoid import cycle
	_ = fmt.Sprintf("unparseable time: %s", s)
	return time.Time{}
}
