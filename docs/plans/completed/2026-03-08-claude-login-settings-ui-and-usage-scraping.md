---
# Claude Login Settings UI + Usage Data from claude.ai

## Overview
Implement the missing settings UI for Claude login (session cookie entry), persist credentials to the database so they survive server restarts, scrape claude.ai/settings/usage to extract session and weekly usage percentages, and update the frontend to display this data as progress bars.

## Context
- Files involved:
  - `internal/provider/claudeprovider/claudeprovider.go` - update FetchUsageInfo to scrape usage page
  - `internal/provider/provider.go` - extend UsageInfo if needed for multiple quotas
  - `internal/api/handlers.go` - update login handler to persist credentials
  - `internal/db/migrations/` - add migration for credential storage
  - `internal/repository/` - add credential storage repo methods
  - `web/index.html` - add settings modal markup
  - `web/js/app.js` - add settings open/close logic and login form submission
  - `web/css/style.css` - add modal styles if not already present
- Related patterns: provider RWMutex pattern, DB migration with goose, UpsertProviderUsage pattern
- Dependencies: none new (standard lib html parsing is sufficient)

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: DB migration and repository for credential persistence

**Files:**
- Create: `internal/db/migrations/003_provider_credentials.sql`
- Modify: `internal/repository/repository.go` (or relevant repo file)

- [ ] Create migration adding `provider_credentials` table with fields: `provider_name TEXT UNIQUE, credential_key TEXT, credential_value TEXT, updated_at DATETIME`
- [ ] Add `UpsertProviderCredential(ctx, providerName, key, value string)` repo method
- [ ] Add `GetProviderCredential(ctx, providerName, key string) (string, error)` repo method
- [ ] Write tests for both repo methods using in-memory SQLite
- [ ] Run `go test ./internal/repository/...` - must pass

### Task 2: Load and persist Claude credentials through the API

**Files:**
- Modify: `internal/api/handlers.go`
- Modify: `cmd/server/main.go`

- [ ] Update `claudeLogin` handler to also call `UpsertProviderCredential` after successful `provider.Login()`
- [ ] On server startup in `main.go`, load saved Claude session key from DB and call `provider.Login()` if present
- [ ] Write handler test: POST login → credential persisted → next server startup loads it
- [ ] Run `go test ./internal/api/...` - must pass

### Task 3: Scrape claude.ai/settings/usage for session and weekly usage

**Files:**
- Modify: `internal/provider/claudeprovider/claudeprovider.go`
- Modify: `internal/provider/provider.go` if UsageInfo needs extra fields

- [ ] Research what API endpoint claude.ai/settings/usage calls (check network tab pattern - likely `/api/usage_limits` or similar); implement HTTP fetch with session cookie and parse JSON response
- [ ] Add `SessionUsagePct`, `WeeklyUsagePct float64` and `SessionResetsAt`, `WeeklyResetsAt time.Time` fields to `UsageInfo` struct (or add a second entry approach)
- [ ] Update `FetchUsageInfo` to fetch and parse the usage page data, populating new fields
- [ ] Write tests for new parsing logic using `httptest.Server` mocks
- [ ] Run `go test ./internal/provider/claudeprovider/...` - must pass

### Task 4: Frontend settings modal for Claude login

**Files:**
- Modify: `web/index.html`
- Modify: `web/js/app.js`
- Modify: `web/css/style.css` (if modal styles missing)

- [ ] Add a settings gear icon/button to the usage card header (next to Refresh button)
- [ ] Add settings modal HTML with: instruction text (go to claude.ai, open DevTools > Application > Cookies, copy sessionKey), an input field for the session cookie value, and a Save button
- [ ] Add CSS for modal overlay (backdrop, centered card, close on backdrop click)
- [ ] In `app.js`, wire up open/close modal logic
- [ ] On Save: POST to `/api/providers/claude/login` with `{"session_key": "..."}`, close modal on success, trigger `refreshUsage()`
- [ ] Update the "Login required" message to include a clickable "Go to settings" link that opens the modal (instead of plain text)
- [ ] Manual test: no automated frontend tests needed here; verify behavior manually

### Task 5: Display Claude session and weekly usage as progress bars

**Files:**
- Modify: `web/js/app.js`
- Modify: `internal/api/handlers.go` (ensure new fields are returned in usage JSON response)

- [ ] Update usage API response to include `session_usage_pct`, `weekly_usage_pct`, `session_resets_at`, `weekly_resets_at`
- [ ] Update `renderUsage()` in `app.js` to show two progress bars for Claude when percentage data is present: "Current session: X% used, resets in Y" and "Weekly (all models): X% used, resets Mon HH:MM"
- [ ] Fallback: if percentage data absent, keep existing ACTIVE/BLOCKED display
- [ ] Run `go test ./internal/api/...` - must pass

### Task 6: Verify acceptance criteria

- [ ] Manual test: start fresh server, open app, see "Login required", click settings link, enter session cookie, save, usage loads with progress bars
- [ ] Manual test: restart server, usage still loads (credentials persisted)
- [ ] Run full test suite: `go test ./...` - must pass
- [ ] Run linter: `go vet ./...` - must pass

### Task 7: Update documentation

- [ ] Update README.md with Claude login instructions if user-facing
- [ ] Move this plan to `docs/plans/completed/`
