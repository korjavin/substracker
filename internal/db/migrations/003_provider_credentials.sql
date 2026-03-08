-- +goose Up
CREATE TABLE provider_credentials (
    provider_name    TEXT NOT NULL,
    credential_key   TEXT NOT NULL,
    credential_value TEXT NOT NULL,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_name, credential_key)
);

-- +goose Down
DROP TABLE provider_credentials;
