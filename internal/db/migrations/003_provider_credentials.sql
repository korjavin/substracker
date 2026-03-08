-- +goose Up
CREATE TABLE provider_credentials (
    provider_name TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (provider_name, key)
);

-- +goose Down
DROP TABLE provider_credentials;
