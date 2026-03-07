# SubsTracker — Agent Guide

## Project Structure

```
cmd/server/main.go          — Entry point: wires DB, services, HTTP server
internal/
  db/
    db.go                   — Open SQLite, run migrations (goose)
    migrations/001_init.sql — Schema: subscriptions, webpush_subs, telegram_chats, notification_log
    queries/                — sqlc SQL query files (for regenerating repository/)
  repository/
    models.go               — Go structs for DB rows + param types
    queries.go              — Manual DB layer (database/sql wrappers)
  service/
    notification.go         — Sends via Web Push (webpush-go) and Telegram (telegram-bot-api)
    scheduler.go            — Daily check: notifies on reset day and day before
  api/
    handlers.go             — All HTTP handlers, registered on http.ServeMux
  middleware/
    ratelimit.go            — IP-based rate limiter (golang.org/x/time/rate)
    logging.go              — Structured request logging (slog)
web/
  index.html                — Single-page app (tabs: Subscriptions, Notifications, Log)
  css/style.css             — Dark theme styles
  js/app.js                 — Vanilla JS: fetch API, tab switching, modal, push subscription
  sw.js                     — Service Worker for Web Push
```

## Adding a New API Endpoint

1. Add the handler method to `internal/api/handlers.go`
2. Register it in `Handler.Register()` using Go 1.22 routing (`"METHOD /path/{param}"`)
3. Add any DB queries to `internal/repository/queries.go`

## Adding a New Notification Channel

1. Add config fields to `service.NotificationConfig` and read from env in `cmd/server/main.go`
2. Add a `sendXxx()` method in `internal/service/notification.go`
3. Call it from `SendAll()`

## DB Schema Changes

1. Add a new migration file: `internal/db/migrations/002_your_change.sql` (goose format)
2. Update `internal/repository/models.go` for new structs
3. Add query methods to `internal/repository/queries.go`
4. (Optional) update sqlc query files and run `sqlc generate`

## Key Dependencies

- `modernc.org/sqlite` — pure Go SQLite (no CGO, CGO_ENABLED=0)
- `github.com/pressly/goose/v3` — SQL migrations
- `github.com/SherClockHolmes/webpush-go` — Web Push / VAPID
- `github.com/go-telegram-bot-api/telegram-bot-api` — Telegram (v4, no commands, just Send)
- `golang.org/x/time/rate` — rate limiting

## Important Notes

- SQLite is opened with WAL mode + foreign keys enabled
- `db.SetMaxOpenConns(1)` — required for SQLite write safety
- Timestamps in SQLite are stored as TEXT (`CURRENT_TIMESTAMP` format: `"2006-01-02 15:04:05"`); `parseTime()` in queries.go handles parsing
- Web Push only works with HTTPS in production (localhost is fine for dev)
- Telegram bot sends messages only — no command handling, no webhook/polling
