---
# Implement Google One Provider

## Overview
Add a Google One (cloud storage subscription) provider to SubsTracker that fetches the billing cycle reset date from Google's account APIs using a session cookie, following the same patterns as the existing ClaudeProvider.

## Context
- Files involved:
  - Create: `internal/provider/googleoneprovider/googleoneprovider.go`
  - Create: `internal/provider/googleoneprovider/googleoneprovider_test.go`
  - Modify: `internal/api/handlers.go` (register provider, add endpoints)
  - Modify: `internal/api/handlers_test.go` (add API handler tests)
- Related patterns: ClaudeProvider (session cookie auth, httptest mock, mutex, baseURL override for testing)
- Dependencies: none new; uses standard library net/http, encoding/json, sync, time

## Development Approach
- **Testing approach**: Regular (implement, then write tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Create GoogleOneProvider

**Files:**
- Create: `internal/provider/googleoneprovider/googleoneprovider.go`

Implement the Provider interface following ClaudeProvider patterns:

- [ ] Define `GoogleOneProvider` struct with fields: `mu sync.RWMutex`, `sessionCookie string`, `baseURL string`
- [ ] Implement `NewGoogleOneProvider()` constructor with `baseURL = "https://one.google.com"`
- [ ] Implement `Name()` returning `"Google One"`
- [ ] Implement `Login(ctx, credentials)` accepting `credentials["session_cookie"]`; store in struct with mutex; return error if empty
- [ ] Implement `FetchUsageInfo(ctx)`:
  - Return `ErrUnauthorized` if no session cookie stored
  - Make GET request to `{baseURL}/api/subscriptions` (or discovered endpoint) with `Cookie: SID={session_cookie}` header
  - Return `ErrUnauthorized` on 401/403 responses
  - Parse billing period end date from response JSON
  - Return `&UsageInfo{ResetDate: parsedDate}` on success
- [ ] Use 10-second HTTP client timeout (same as ClaudeProvider)
- [ ] Note: the exact Google One API endpoint and response shape will need to be verified by inspecting network traffic on one.google.com; document the endpoint and JSON path in code comments

### Task 2: Write unit tests for GoogleOneProvider

**Files:**
- Create: `internal/provider/googleoneprovider/googleoneprovider_test.go`

- [ ] `TestGoogleOneProvider_Name` - verify Name() returns "Google One"
- [ ] `TestGoogleOneProvider_Login` - test valid credential storage, empty credential error
- [ ] `TestGoogleOneProvider_FetchUsageInfo` subtests using httptest.Server with baseURL override:
  - `NoSessionCookie` - returns ErrUnauthorized when not logged in
  - `Unauthorized` - returns ErrUnauthorized on 401 response
  - `UnexpectedStatus` - returns error on 500 response
  - `Success` - parses reset date correctly from mock JSON response
  - `InvalidJSON` - returns error on malformed response
- [ ] Run: `go test ./internal/provider/googleoneprovider/...`

### Task 3: Register provider and add API endpoints

**Files:**
- Modify: `internal/api/handlers.go`

- [ ] Add `googleOneProvider provider.Provider` field to `Handler` struct
- [ ] Initialize in `NewHandler()`: `googleOneProvider: googleoneprovider.NewGoogleOneProvider()`
- [ ] Add import for `googleoneprovider` package
- [ ] Implement handler `googleOneLoginInfo(w, r)` - GET, returns JSON with instructions to obtain session cookie from browser dev tools
- [ ] Implement handler `googleOneLogin(w, r)` - POST with `{"session_cookie": "..."}`, calls `Login()`, returns 200 or 400/500
- [ ] Implement handler `googleOneUsage(w, r)` - GET, calls `FetchUsageInfo()`, returns reset date JSON or 401 on ErrUnauthorized
- [ ] Register routes in the router: `/api/providers/googleone/login-info`, `/api/providers/googleone/login`, `/api/providers/googleone/usage`
- [ ] Write handler tests in `internal/api/handlers_test.go` using the mock provider pattern (implement `provider.Provider` interface in test with controllable behavior)
- [ ] Run: `go test ./internal/api/...`

### Task 4: Verify acceptance criteria

- [ ] manual test: start server, POST session cookie to `/api/providers/googleone/login`, GET `/api/providers/googleone/usage`, verify reset date returned
- [ ] run full test suite: `go test ./...`
- [ ] run linter: `go vet ./...`
- [ ] verify test coverage meets 80%+

### Task 5: Update documentation

- [ ] update CLAUDE.md if internal patterns changed
- [ ] move this plan to `docs/plans/completed/`
---
