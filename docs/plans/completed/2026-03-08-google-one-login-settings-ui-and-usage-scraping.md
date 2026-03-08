---
# Google One Login Settings UI + Credential Persistence

## Overview
Implement settings UI for Google One login (session cookie entry via modal), persist credentials to the database so they survive server restarts, verify/fix the actual Google One API endpoint for subscription data, and update the frontend to display a settings gear icon and modal with instructions matching the Claude provider pattern.

## Context
- Files involved:
  - `internal/provider/googleoneprovider/googleoneprovider.go` - verify/fix FetchUsageInfo endpoint
  - `internal/api/handlers.go` - update googleOneLogin handler to persist credentials
  - `internal/db/migrations/003_provider_credentials.sql` - may already exist from Claude plan
  - `internal/repository/queries.go` - add credential storage methods if not already added
  - `cmd/server/main.go` - load saved Google One cookie on startup
  - `web/index.html` - add Google One settings modal markup
  - `web/js/app.js` - add Google One settings open/close logic and form submission
  - `web/css/style.css` - if Google One modal needs distinct styles
- Related patterns: provider RWMutex pattern, DB migration with goose, UpsertProviderCredential pattern (from Claude plan)
- Dependencies: none new; migration 003 may already exist from Claude plan implementation

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: DB migration and repository for credential persistence

**Files:**
- Create: `internal/db/migrations/003_provider_credentials.sql` (only if not already created by Claude plan)
- Modify: `internal/repository/queries.go`
- Modify: `internal/repository/models.go`

- [ ] Check if `003_provider_credentials.sql` already exists; if not, create it with `provider_credentials` table: `provider_name TEXT UNIQUE, credential_key TEXT, credential_value TEXT, updated_at DATETIME`
- [ ] Check if `UpsertProviderCredential` and `GetProviderCredential` repo methods already exist; if not, add them to `queries.go` and the corresponding model/param types to `models.go`
- [ ] Write tests for both repo methods (if not already tested) using in-memory SQLite
- [ ] Run `go test ./internal/repository/...` - must pass

### Task 2: Load and persist Google One credentials through the API

**Files:**
- Modify: `internal/api/handlers.go`
- Modify: `cmd/server/main.go`

- [ ] Update `googleOneLogin` handler to call `UpsertProviderCredential(ctx, "Google One", "session_cookie", req.SessionCookie)` after successful `provider.Login()`
- [ ] In `main.go`, on server startup load saved Google One session cookie from DB and call `googleOneProvider.Login()` if present (alongside existing Claude credential loading)
- [ ] Write handler test: POST googleOneLogin → credential persisted → next startup loads it
- [ ] Run `go test ./internal/api/...` - must pass

### Task 3: Verify and fix Google One usage scraping

**Files:**
- Modify: `internal/provider/googleoneprovider/googleoneprovider.go`
- Modify: `internal/provider/googleoneprovider/googleoneprovider_test.go`

- [ ] Investigate the actual API endpoint used by one.google.com (check network requests; likely something like `/u/0/_/OneMatrixUi/data/batchexecute` or a REST endpoint under `/api/`); update `FetchUsageInfo` with correct endpoint, headers, and response parsing
- [ ] If the endpoint requires additional cookies beyond SID (e.g. `HSID`, `SSID`, `APISID`), update `Login` to accept and store them, and update the handler accordingly
- [ ] Update tests to use the verified response shape with `httptest.Server` mocks
- [ ] Run `go test ./internal/provider/googleoneprovider/...` - must pass

### Task 4: Frontend settings modal for Google One login

**Files:**
- Modify: `web/index.html`
- Modify: `web/js/app.js`
- Modify: `web/css/style.css` (if Google One modal needs distinct styles)

- [ ] Add a settings gear icon/button to the Google One usage card area (similar to Claude card pattern)
- [ ] Add settings modal HTML for Google One with: instruction text (go to one.google.com, open DevTools > Application > Cookies, copy the SID cookie value), an input field for the session cookie, and a Save button
- [ ] Reuse existing modal CSS or add minimal new styles if needed
- [ ] In `app.js`, wire up open/close logic for Google One settings modal (separate from Claude modal)
- [ ] On Save: POST to `/api/providers/googleone/login` with `{"session_cookie": "..."}`, close modal on success, trigger Google One usage refresh
- [ ] Update the "Login required" message in the Google One card to include a "Go to settings" link that opens the Google One modal
- [ ] Manual test: no automated frontend tests needed; verify behavior manually

### Task 5: Verify acceptance criteria

- [ ] Manual test: start fresh server, open app, see Google One "Login required", click settings link, enter SID cookie, save, usage data loads with reset date
- [ ] Manual test: restart server, Google One usage still loads (credentials persisted)
- [ ] Run full test suite: `go test ./...` - must pass
- [ ] Run linter: `go vet ./...` - must pass

### Task 6: Update documentation

- [ ] Update README.md with Google One login instructions if user-facing
- [ ] Move this plan to `docs/plans/completed/`
