-- name: CreateSubscription :one
INSERT INTO subscriptions (name, service, billing_day, notes)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetSubscription :one
SELECT * FROM subscriptions WHERE id = ? LIMIT 1;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions ORDER BY name;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET name = ?, service = ?, billing_day = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE id = ?;
