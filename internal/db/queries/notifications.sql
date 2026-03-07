-- name: UpsertWebPushSubscription :exec
INSERT INTO webpush_subscriptions (endpoint, p256dh, auth)
VALUES (?, ?, ?)
ON CONFLICT(endpoint) DO UPDATE SET p256dh = excluded.p256dh, auth = excluded.auth;

-- name: ListWebPushSubscriptions :many
SELECT * FROM webpush_subscriptions;

-- name: DeleteWebPushSubscription :exec
DELETE FROM webpush_subscriptions WHERE endpoint = ?;

-- name: CreateTelegramChat :exec
INSERT INTO telegram_chats (chat_id) VALUES (?) ON CONFLICT(chat_id) DO NOTHING;

-- name: ListTelegramChats :many
SELECT * FROM telegram_chats ORDER BY created_at;

-- name: DeleteTelegramChat :exec
DELETE FROM telegram_chats WHERE chat_id = ?;

-- name: CreateNotificationLog :exec
INSERT INTO notification_log (subscription_id, channel, message)
VALUES (?, ?, ?);

-- name: ListNotificationLogs :many
SELECT * FROM notification_log ORDER BY sent_at DESC LIMIT 100;
