---
# OpenAI/Codex Login Settings UI + Credential Persistence + Usage Scraping

## Overview
Implement an OpenAI/Codex provider following the same pattern as Claude, Google One, and Z.ai: create the openaiprovider package, add API routes for login/usage, persist credentials to the DB so they survive restarts, investigate the actual OpenAI usage API endpoint, and add a frontend settings modal matching existing provider UI patterns.

## Context
- Files involved:
  - `internal/provider/openaiprovider/openaiprovider.go` - new provider implementing Provider interface
  - `internal/provider/openaiprovider/openaiprovider_test.go` - new provider tests
  - `internal/api/handlers.go` - add openai login-info, login, and usage handlers + register routes
  - `internal/db/migrations/003_provider_credentials.sql` - may already exist from Claude/Google One/Z.ai plan implementation
  - `internal/repository/queries.go` - add credential storage methods if not already added
  - `cmd/server/main.go` - load saved OpenAI credentials on startup; expose openai provider to scheduler if needed
  - `web/index.html` - add OpenAI settings modal markup
  - `web/js/app.js` - add OpenAI settings modal open/close logic, form submission, usage refresh
- Related patterns: provider RWMutex pattern, DB migration with goose, UpsertProviderCredential pattern (from Claude/Google One/Z.ai plans)
- Dependencies: none new; migration 003 may already exist

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: DB migration and repository for credential persistence

**Files:**
- Create: `internal/db/migrations/003_provider_credentials.sql` (only if not already created by prior plans)
- Modify: `internal/repository/queries.go`
- Modify: `internal/repository/models.go`

- [ ] Check if `003_provider_credentials.sql` already exists; if not, create it with `provider_credentials` table: `provider_name TEXT UNIQUE, credential_key TEXT, credential_value TEXT, updated_at DATETIME`
- [ ] Check if `UpsertProviderCredential` and `GetProviderCredential` repo methods already exist; if not, add them to `queries.go` and corresponding model/param types to `models.go`
- [ ] Write tests for both repo methods (if not already tested) using in-memory SQLite
- [ ] Run `go test ./internal/repository/...` - must pass

### Task 2: Create OpenAI provider package

**Files:**
- Create: `internal/provider/openaiprovider/openaiprovider.go`
- Create: `internal/provider/openaiprovider/openaiprovider_test.go`

- [ ] Implement `OpenAIProvider` struct with `sync.RWMutex`, `sessionToken string`, `baseURL string`, and `*http.Client` fields matching the Google One/Z.ai pattern
- [ ] Implement `Name() string` returning `"OpenAI"`
- [ ] Implement `Login(ctx, credentials)` accepting `session_token` key (the `__Secure-next-auth.session-token` cookie from platform.openai.com), storing it under lock
- [ ] Investigate the actual OpenAI usage API endpoint by inspecting network traffic on platform.openai.com/settings/organization/usage or the Codex subscription page (likely `https://api.openai.com/dashboard/billing/subscription` or similar); implement `FetchUsageInfo` to fetch subscription/usage data and return `*provider.UsageInfo` with `ResetDate` populated; return `provider.ErrUnauthorized` on 401/403
- [ ] Write tests using `httptest.Server` mocks covering: successful fetch, unauthorized response, malformed response
- [ ] Run `go test ./internal/provider/openaiprovider/...` - must pass

### Task 3: API handlers and routes for OpenAI

**Files:**
- Modify: `internal/api/handlers.go`

- [ ] Add `openaiProvider provider.Provider` field to `Handler` struct; initialize it with `openaiprovider.NewOpenAIProvider()` in `NewHandler`
- [ ] Add `GetOpenAIProvider()` accessor if needed by scheduler
- [ ] Implement `openaiLoginInfo` handler returning `{"url": "https://platform.openai.com/", "instructions": "Go to platform.openai.com, open DevTools > Application > Cookies, copy the __Secure-next-auth.session-token value"}`
- [ ] Implement `openaiLogin` handler: decode `session_token`, call `h.openaiProvider.Login()`, call `UpsertProviderCredential("OpenAI", "session_token", ...)`, return `{"status": "logged_in"}`
- [ ] Implement `openaiUsage` handler: call `FetchUsageInfo`, handle `ErrUnauthorized` → 401, update `UpsertProviderUsage`, return info JSON
- [ ] Register routes: `GET /api/providers/openai/login-info`, `POST /api/providers/openai/login`, `GET /api/providers/openai/usage`
- [ ] Write handler tests for login (persists credential) and usage (success + unauthorized paths)
- [ ] Run `go test ./internal/api/...` - must pass

### Task 4: Load OpenAI credentials on server startup

**Files:**
- Modify: `cmd/server/main.go`

- [ ] On startup, after `handler` is created, call `repo.GetProviderCredential(ctx, "OpenAI", "session_token")`; if found, call `handler.GetOpenAIProvider().Login(ctx, ...)` to restore session
- [ ] Add openai provider to scheduler `providers` slice if usage polling is desired
- [ ] Run `go test ./...` - must pass

### Task 5: Frontend settings modal for OpenAI login

**Files:**
- Modify: `web/index.html`
- Modify: `web/js/app.js`

- [ ] Add an OpenAI settings section/card to the usage tab (similar to existing provider cards) with a settings gear icon/button and a "Login required" message
- [ ] Add settings modal HTML for OpenAI with: instruction text (go to platform.openai.com, open DevTools > Application > Cookies, copy `__Secure-next-auth.session-token`), an input field for the session token, and a Save button
- [ ] Reuse existing modal CSS (no new styles needed unless distinct elements required)
- [ ] In `app.js`, add open/close logic for the OpenAI settings modal
- [ ] On Save: POST to `/api/providers/openai/login` with `{"session_token": "..."}`, close modal on success, trigger OpenAI usage refresh
- [ ] Add `loadOpenAIUsage()` / `refreshOpenAIUsage()` functions fetching from `/api/providers/openai/usage` and rendering reset date and subscription info
- [ ] Update the "Login required" message to include a clickable "Go to settings" link opening the OpenAI modal
- [ ] Manual test: no automated frontend tests; verify behavior manually

### Task 6: Verify acceptance criteria

- [ ] Manual test: start fresh server, open app, see OpenAI "Login required", click settings, enter session token, save, usage data loads
- [ ] Manual test: restart server, OpenAI usage still loads (credentials persisted)
- [ ] Run full test suite: `go test ./...` - must pass
- [ ] Run linter: `go vet ./...` - must pass

### Task 7: Update documentation

- [ ] Update README.md with OpenAI login instructions if user-facing
- [ ] Move this plan to `docs/plans/completed/`
