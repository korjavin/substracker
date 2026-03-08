---
# Implement Telegram Web Authentication for SubsTracker

## Overview
Add Telegram Login Widget authentication to protect the web app from unauthenticated access. Users authenticate via Telegram, get a session cookie, and see only their own subscriptions. All data tables gain a `user_id` column to isolate data per Telegram user.

## Context
- Files involved:
  - `cmd/server/main.go` - server setup, new config vars
  - `internal/api/handlers.go` - all HTTP handlers
  - `internal/repository/queries.go` - all DB queries
  - `internal/db/migrations/` - new migration
  - `web/index.html` - redirect if unauthenticated
  - `web/js/app.js` - pass auth via session cookie (automatic)
  - New: `internal/auth/auth.go` - Telegram validation + session logic
  - New: `internal/api/auth_handlers.go` - login/logout endpoints
  - New: `web/login.html` - login page with Telegram widget
- Related patterns: follows medicationtrackerbot auth.go exactly
- Dependencies: none new (uses stdlib crypto/hmac, crypto/sha256)

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: DB Migration - Add user_id to data tables

**Files:**
- Create: `internal/db/migrations/002_add_user_id.sql`

- [ ] Add migration: `ALTER TABLE subscriptions ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0`
- [ ] Add migration: `ALTER TABLE webpush_subscriptions ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0`
- [ ] Add migration: `ALTER TABLE telegram_chats ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0`
- [ ] Add indexes on user_id for subscriptions and the other tables
- [ ] Verify migration applies cleanly on fresh DB
- [ ] Write test: migration runs without error, columns exist

### Task 2: Auth Package - Telegram Login Widget Validation + Sessions

**Files:**
- Create: `internal/auth/auth.go`

- [ ] Implement `ValidateTelegramLogin(botToken string, data url.Values) (bool, *TelegramUser, error)` - HMAC-SHA256 of sorted key=value pairs using bot token directly (not "WebAppData" prefix, this is the widget flow)
- [ ] Implement `CreateSessionToken(userID int64, secret string) string` - HMAC-signed token
- [ ] Implement `VerifySessionToken(token, secret string) (int64, bool)` - verify and extract user ID
- [ ] Define `TelegramUser` struct with ID, FirstName, LastName, Username
- [ ] Define `UserCtxKey` context key, `UserFromContext(ctx) (*TelegramUser, bool)` helper
- [ ] Define `AuthMiddleware(secret string) func(http.Handler) http.Handler` - checks `auth_session` cookie, puts user in context, returns 401 JSON for API paths if unauthenticated
- [ ] Write unit tests for ValidateTelegramLogin (valid case, hash mismatch, expired auth_date)
- [ ] Write unit tests for CreateSessionToken + VerifySessionToken (roundtrip, invalid token)
- [ ] Run tests - must pass

### Task 3: Auth HTTP Handlers + Route Protection

**Files:**
- Create: `internal/api/auth_handlers.go`
- Modify: `internal/api/handlers.go`
- Modify: `cmd/server/main.go`

- [ ] Add handler `telegramCallback` - validates widget data, sets `auth_session` cookie (30 day, HttpOnly, Secure, SameSite=Lax), redirects to `/`
- [ ] Add handler `logout` - clears `auth_session` cookie, redirects to `/login`
- [ ] Add handler `authStatus` GET `/api/auth/me` - returns current user info (used by frontend to check auth)
- [ ] Register routes in handlers.go: `GET /auth/logout`, `POST /auth/telegram/callback`, `GET /api/auth/me`
- [ ] Add `SESSION_SECRET` and `TELEGRAM_BOT_USERNAME` to server config struct and env var loading in main.go
- [ ] Wrap all `/api/` routes (except `/api/auth/me` check and auth routes) with `AuthMiddleware`
- [ ] Serve `web/login.html` at `GET /login` (unauthenticated)
- [ ] Serve `web/index.html` protected: redirect to `/login` if no valid session cookie
- [ ] Write handler tests for telegramCallback (valid data sets cookie, invalid data returns 403)
- [ ] Run tests - must pass

### Task 4: Repository Updates - Filter by user_id

**Files:**
- Modify: `internal/repository/queries.go`

- [ ] Update `ListSubscriptions` signature to accept `userID int64`, add `WHERE user_id = ?` filter
- [ ] Update `CreateSubscription` to accept `userID int64` in params, insert it
- [ ] Update `GetSubscription` to accept `userID int64`, add `AND user_id = ?` (returns 404 if not owned)
- [ ] Update `UpdateSubscription` to accept `userID int64`, add `AND user_id = ?`
- [ ] Update `DeleteSubscription` to accept `userID int64`, add `AND user_id = ?`
- [ ] Update webpush subscription queries (list, create, delete) to filter by userID
- [ ] Update telegram_chats queries (list, add, delete) to filter by userID
- [ ] Update notification log query to filter by userID (join through subscriptions)
- [ ] Update `CreateSubscriptionParams` and other param structs to include `UserID int64`
- [ ] Write repository tests verifying user isolation (user A cannot see user B's data)
- [ ] Run tests - must pass

### Task 5: API Handler Updates - Extract user_id from Context

**Files:**
- Modify: `internal/api/handlers.go`

- [ ] In every subscription handler: extract user via `auth.UserFromContext(r.Context())`, pass userID to repo calls
- [ ] In every webpush handler: extract userID, pass to repo
- [ ] In every telegram_chats handler: extract userID, pass to repo
- [ ] In notification log handler: extract userID, pass to repo
- [ ] Verify no handler calls a repo method without userID
- [ ] Write handler integration tests: unauthenticated request returns 401, authenticated returns only own data
- [ ] Run tests - must pass

### Task 6: Frontend - Login Page and Auth Check

**Files:**
- Create: `web/login.html`
- Modify: `web/index.html`
- Modify: `web/js/app.js`

- [ ] Create `web/login.html` with Telegram Login Widget (reads bot username from `window.BOT_USERNAME`)
  - Widget data-auth-url points to `/auth/telegram/callback`
  - Minimal styling consistent with existing app
- [ ] In `web/index.html`: add `<script>window.BOT_USERNAME = "{{.BotUsername}}"</script>` (or fetch from `/api/auth/me`)
- [ ] In `web/js/app.js`: on startup call `/api/auth/me`; if 401 redirect to `/login`; else show the app with user name in header
- [ ] Add logout button in the UI header that calls `GET /auth/logout`
- [ ] Session cookie is sent automatically by browser - no change needed to existing API call logic
- [ ] Manual test: unauthenticated browser -> redirects to `/login`; after login -> sees only own subscriptions
- [ ] Run tests - must pass

### Task 7: Verify acceptance criteria

- [ ] manual test: open app in fresh browser, get redirected to login page
- [ ] manual test: login with Telegram, see own subscriptions
- [ ] manual test: session persists across page refresh (30-day cookie)
- [ ] manual test: logout clears session and redirects to login
- [ ] manual test: API calls without cookie return 401
- [ ] run full test suite (go test ./...)
- [ ] run linter (go vet ./...)
- [ ] verify test coverage meets 80%+

### Task 8: Update documentation

- [ ] update README.md: add SESSION_SECRET and TELEGRAM_BOT_USERNAME to env var docs
- [ ] update CLAUDE.md if auth pattern should be noted
- [ ] move this plan to `docs/plans/completed/`
