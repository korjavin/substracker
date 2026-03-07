# Completed Plan: Add Claude Provider Implementation

## Task 1: Extend provider interface with ErrUnauthorized
- Added `var ErrUnauthorized = errors.New("unauthorized: relogin required")` to `internal/provider/provider.go`.
- Updated `internal/provider/testprovider/testprovider.go` to return `fmt.Errorf("%w", provider.ErrUnauthorized)`.
- Updated `internal/provider/testprovider/testprovider_test.go` to check for `provider.ErrUnauthorized`.
- Tested `go test ./internal/provider/...`

## Task 2: Create ClaudeProvider and implement Login
- Created `internal/provider/claudeprovider/claudeprovider.go`.
- Added struct `ClaudeProvider` with `sessionKey string`.
- Implemented `Name()` -> "Claude".
- Implemented `Login(ctx, map[string]string)` saving the sessionKey.
- Implemented stub `FetchUsageInfo()`.
- Created tests in `internal/provider/claudeprovider/claudeprovider_test.go`.
- Tested `go test ./internal/provider/claudeprovider/...`

## Task 3: Implement FetchUsageInfo
- Modified `FetchUsageInfo` in `claudeprovider.go` to hit `https://claude.ai/api/organizations` and then fetch billing info.
- Checked for 401/403 and returned `provider.ErrUnauthorized`.
- Parsed the billing cycle `end_date` as RFC3339 and fallback parsing.
- Added extensive mock server tests handling success, missing tokens, and HTTP failure paths.

## Task 4: Add API endpoints for Claude provider
- Created `h.claudeProvider` property in API `Handler` in `internal/api/handlers.go`.
- Added `GET /api/providers/claude/login-info`, `POST /api/providers/claude/login`, and `GET /api/providers/claude/usage`.
- Created unit tests `TestClaudeLoginInfo`, `TestClaudeLogin`, and `TestClaudeUsage` using `httptest`.

## Task 5: Verify acceptance criteria
- Completed manual review of structure.
- Passed full test suite: `go test ./...`
- Passed linter check: `go vet ./...`
- Verified test coverage for claudeprovider > 80% (Currently 86.8%).

## Task 6: Update documentation
- Moved the completed plan into `docs/plans/completed/claude_provider.md`.
