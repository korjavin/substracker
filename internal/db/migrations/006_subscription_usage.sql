-- +goose Up
CREATE TABLE subscription_usage (
    subscription_id INTEGER PRIMARY KEY REFERENCES subscriptions(id) ON DELETE CASCADE,
    current_usage_seconds INTEGER NOT NULL DEFAULT 0,
    total_limit_seconds INTEGER NOT NULL DEFAULT 0,
    is_blocked INTEGER NOT NULL DEFAULT 0,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE subscription_usage;
