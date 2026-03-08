-- +goose Up
ALTER TABLE subscriptions ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE webpush_subscriptions ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE telegram_chats ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;

-- Legacy subscriptions and webpush_subscriptions are left with user_id = 0.
-- We can backfill telegram_chats safely because chat_id is inherently the user's ID in personal chats.
UPDATE telegram_chats SET user_id = CAST(chat_id AS INTEGER) WHERE user_id = 0;

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_webpush_subscriptions_user_id ON webpush_subscriptions(user_id);
CREATE INDEX idx_telegram_chats_user_id ON telegram_chats(user_id);

-- +goose Down
DROP INDEX idx_telegram_chats_user_id;
DROP INDEX idx_webpush_subscriptions_user_id;
DROP INDEX idx_subscriptions_user_id;

ALTER TABLE telegram_chats DROP COLUMN user_id;
ALTER TABLE webpush_subscriptions DROP COLUMN user_id;
ALTER TABLE subscriptions DROP COLUMN user_id;
