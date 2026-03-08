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
		`INSERT INTO subscriptions (user_id, name, service, billing_day, notes)
		 VALUES (?, ?, ?, ?, ?)
		 RETURNING id, user_id, name, service, billing_day, notes, created_at, updated_at`,
		arg.UserID, arg.Name, arg.Service, arg.BillingDay, arg.Notes,
	)
	return scanSubscriptionRow(row)
}

func (q *Queries) GetSubscription(ctx context.Context, id, userID int64) (Subscription, error) {
	row := q.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, service, billing_day, notes, created_at, updated_at
		 FROM subscriptions WHERE id = ? AND user_id = ? LIMIT 1`,
		id, userID,
	)
	return scanSubscriptionRow(row)
}

func (q *Queries) ListSubscriptions(ctx context.Context, userID int64) ([]Subscription, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, user_id, name, service, billing_day, notes, created_at, updated_at
		 FROM subscriptions WHERE user_id = ? ORDER BY name`,
		userID,
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

func (q *Queries) ListAllSubscriptions(ctx context.Context) ([]Subscription, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, user_id, name, service, billing_day, notes, created_at, updated_at
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
		 WHERE id=? AND user_id=?
		 RETURNING id, user_id, name, service, billing_day, notes, created_at, updated_at`,
		arg.Name, arg.Service, arg.BillingDay, arg.Notes, arg.ID, arg.UserID,
	)
	return scanSubscriptionRow(row)
}

func (q *Queries) DeleteSubscription(ctx context.Context, id, userID int64) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=? AND user_id=?`, id, userID)
	return err
}

// --- Web Push ---

func (q *Queries) UpsertWebPushSubscription(ctx context.Context, arg WebpushSubscriptionParams) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO webpush_subscriptions (user_id, endpoint, p256dh, auth)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(endpoint) DO UPDATE SET p256dh=excluded.p256dh, auth=excluded.auth`,
		arg.UserID, arg.Endpoint, arg.P256dh, arg.Auth,
	)
	return err
}

func (q *Queries) ListWebPushSubscriptions(ctx context.Context, userID int64) ([]WebpushSubscription, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, user_id, endpoint, p256dh, auth, created_at FROM webpush_subscriptions WHERE user_id=?`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []WebpushSubscription
	for rows.Next() {
		var s WebpushSubscription
		var createdAt string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &createdAt); err != nil {
			return nil, err
		}
		s.CreatedAt = parseTime(createdAt)
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (q *Queries) DeleteWebPushSubscription(ctx context.Context, endpoint string, userID int64) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM webpush_subscriptions WHERE endpoint=? AND user_id=?`, endpoint, userID)
	return err
}

// --- Telegram Chats ---

func (q *Queries) CreateTelegramChat(ctx context.Context, chatID string, userID int64) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO telegram_chats (user_id, chat_id) VALUES (?, ?) ON CONFLICT(chat_id) DO NOTHING`,
		userID, chatID,
	)
	return err
}

func (q *Queries) ListTelegramChats(ctx context.Context, userID int64) ([]TelegramChat, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id, user_id, chat_id, created_at FROM telegram_chats WHERE user_id=? ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []TelegramChat
	for rows.Next() {
		var c TelegramChat
		var createdAt string
		if err := rows.Scan(&c.ID, &c.UserID, &c.ChatID, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTime(createdAt)
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func (q *Queries) DeleteTelegramChat(ctx context.Context, chatID string, userID int64) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM telegram_chats WHERE chat_id=? AND user_id=?`, chatID, userID)
	return err
}

// --- Notification Log ---

func (q *Queries) CreateNotificationLog(ctx context.Context, arg CreateNotificationLogParams) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO notification_log (subscription_id, channel, message) VALUES (?, ?, ?)`,
		arg.SubscriptionID, arg.Channel, arg.Message,
	)
	return err
}

func (q *Queries) ListNotificationLogs(ctx context.Context, userID int64) ([]NotificationLog, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT nl.id, nl.subscription_id, nl.channel, nl.message, nl.sent_at
		 FROM notification_log nl
		 JOIN subscriptions s ON nl.subscription_id = s.id
		 WHERE s.user_id = ?
		 ORDER BY nl.sent_at DESC LIMIT 100`,
		userID,
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
	err := row.Scan(&s.ID, &s.UserID, &s.Name, &s.Service, &s.BillingDay, &s.Notes, &createdAt, &updatedAt)
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
	err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Service, &s.BillingDay, &s.Notes, &createdAt, &updatedAt)
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
