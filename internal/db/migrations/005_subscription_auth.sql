-- +goose Up
ALTER TABLE subscriptions ADD COLUMN auth_token TEXT NOT NULL DEFAULT '';

-- +goose Down
-- We just keep the column or recreate table. SQLite drop column might not be supported well in older versions, but if needed we can drop.
ALTER TABLE subscriptions DROP COLUMN auth_token;
