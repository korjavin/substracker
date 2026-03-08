---
# Z.ai Login Settings UI + Credential Persistence + Usage Scraping

## Overview
Implement a Z.ai provider following the same pattern as Claude and Google One: create the zaiprovider package, add API routes for login/usage, persist credentials to the DB so they survive restarts, investigate the actual Z.ai usage API endpoint, and add a frontend settings modal matching the existing provider UI patterns.

## Context
- Files involved:
  - `internal/provider/zaiprovider/zaiprovider.go` - new provider implementing Provider interface
  - `internal/provider/zaiprovider/zaiprovider_test.go` - new provider tests
  - `internal/api/handlers.go` - add zai login-info, login, and usage handlers + register routes
  - `internal/db/migrations/003_provider_credentials.sql` - may already exist from Claude plan implementation
  - `internal/repository/queries.go` - add credential storage methods if not already added
  - `cmd/server/main.go` - load saved Z.ai credentials on startup; expose zai provider to scheduler if needed
  - `web/index.html` - add Z.ai settings modal markup
  - `web/js/app.js` - add Z.ai settings modal open/close logic, form submission, usage refresh
- Related patterns: provider RWMutex pattern, DB migration with goose, UpsertProviderCredential pattern (from Claude/Google One plans)
- Dependencies: none new; migration 003 may already exist

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: DB migration and repository for credential persistence

**Files:**
- Create: `internal/db/migrations/003_provider_credentials.sql` (only if not already created by Claude/Google One plan)
- Modify: `internal/repository/queries.go`
- Modify: `internal/repository/models.go`

- [ ] Check if `003_provider_credentials.sql` already exists; if not, create it with `provider_credentials` table: `provider_name TEXT UNIQUE, credential_key TEXT, credential_value TEXT, updated_at DATETIME`
- [ ] Check if `UpsertProviderCredential` and `GetProviderCredential` repo methods already exist; if not, add them to `queries.go` and corresponding model/param types to `models.go`
- [ ] Write tests for both repo methods (if not already tested) using in-memory SQLite
- [ ] Run `go test ./internal/repository/...` - must pass

### Task 2: Create Z.ai provider package

**Files:**
- Create: `internal/provider/zaiprovider/zaiprovider.go`
- Create: `internal/provider/zaiprovider/zaiprovider_test.go`

- [ ] Implement `ZAIProvider` struct with `sync.RWMutex`, `sessionCookie string`, `baseURL string`, and `*http.Client` fields matching the Google One pattern
- [ ] Implement `Name() string` returning `"Z.ai"`
- [ ] Implement `Login(ctx, credentials)` accepting `session_cookie` key, storing it under lock
- [ ] Investigate the actual Z.ai usage API endpoint by inspecting network traffic on z.ai (likely something under `/api/` or a GraphQL endpoint); implement `FetchUsageInfo` to fetch subscription/usage data and return `*provider.UsageInfo` with `ResetDate` populated; return `provider.ErrUnauthorized` on 401/403
- [ ] If the endpoint requires specific cookie headers beyond a single session cookie value (e.g. multiple cookies), update `Login` to accept them as separate keys
- [ ] Write tests using `httptest.Server` mocks covering: successful fetch, unauthorized response, malformed response
- [ ] Run `go test ./internal/provider/zaiprovider/...` - must pass

### Task 3: API handlers and routes for Z.ai

**Files:**
- Modify: `internal/api/handlers.go`

- [ ] Add `zaiProvider provider.Provider` field to `Handler` struct; initialize it with `zaiprovider.NewZAIProvider()` in `NewHandler`
- [ ] Add `GetZAIProvider()` accessor if needed by scheduler
- [ ] Implement `zaiLoginInfo` handler returning `{"url": "https://z.ai/", "instructions": "..."}`
- [ ] Implement `zaiLogin` handler: decode `session_cookie`, call `h.zaiProvider.Login()`, call `UpsertProviderCredential("Z.ai", "session_cookie", ...)`, return `{"status": "logged_in"}`
- [ ] Implement `zaiUsage` handler: call `FetchUsageInfo`, handle `ErrUnauthorized` → 401, update `UpsertProviderUsage`, return info JSON
- [ ] Register routes: `GET /api/providers/zai/login-info`, `POST /api/providers/zai/login`, `GET /api/providers/zai/usage`
- [ ] Write handler tests for login (persists credential) and usage (success + unauthorized paths)
- [ ] Run `go test ./internal/api/...` - must pass

### Task 4: Load Z.ai credentials on server startup

**Files:**
- Modify: `cmd/server/main.go`

- [ ] On startup, after `handler` is created, call `repo.GetProviderCredential(ctx, "Z.ai", "session_cookie")`; if found, call `handler.GetZAIProvider().Login(ctx, ...)` to restore session
- [ ] Add zai provider to scheduler `providers` slice if Z.ai usage polling is desired
- [ ] Run `go test ./...` - must pass (no unit tests for main.go needed)

### Task 5: Frontend settings modal for Z.ai login

**Files:**
- Modify: `web/index.html`
- Modify: `web/js/app.js`

- [ ] Add a Z.ai settings section/card to the usage tab (similar to Claude/Google One cards) with a settings gear icon/button and a "Login required" message
- [ ] Add settings modal HTML for Z.ai with: instruction text (go to z.ai, open DevTools > Application > Cookies, copy the session cookie value), an input field for the session cookie, and a Save button
- [ ] Reuse existing modal CSS (no new styles needed unless Z.ai modal requires distinct elements)
- [ ] In `app.js`, add open/close logic for the Z.ai settings modal (separate from Claude/Google One modals)
- [ ] On Save: POST to `/api/providers/zai/login` with `{"session_cookie": "..."}`, close modal on success, trigger Z.ai usage refresh
- [ ] Add `loadZAIUsage()` / `refreshZAIUsage()` functions fetching from `/api/providers/zai/usage` and rendering reset date
- [ ] Update the "Login required" message to include a clickable "Go to settings" link opening the Z.ai modal
- [ ] Manual test: no automated frontend tests; verify behavior manually

### Task 6: Verify acceptance criteria

- [ ] Manual test: start fresh server, open app, see Z.ai "Login required", click settings, enter session cookie, save, usage data loads
- [ ] Manual test: restart server, Z.ai usage still loads (credentials persisted)
- [ ] Run full test suite: `go test ./...` - must pass
- [ ] Run linter: `go vet ./...` - must pass

### Task 7: Update documentation

- [ ] Update README.md with Z.ai login instructions if user-facing
- [ ] Move this plan to `docs/plans/completed/`
