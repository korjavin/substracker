-- +goose Up
CREATE TABLE provider_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_name TEXT UNIQUE NOT NULL,
    current_usage_seconds INTEGER NOT NULL DEFAULT 0,
    total_limit_seconds INTEGER NOT NULL DEFAULT 0,
    is_blocked INTEGER NOT NULL DEFAULT 0,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE provider_usage;
