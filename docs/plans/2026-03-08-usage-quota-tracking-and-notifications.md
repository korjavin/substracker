# Usage/Quota Tracking and Blocked-State Notifications

## Overview
Extend the provider interface and Claude implementation to fetch and track actual usage/quota data (hours used vs. weekly limit). Add a periodic polling scheduler that detects when quota goes from blocked to unblocked and sends notifications. Update the web UI to show usage status on the main page.

## Context
- Files involved:
  - `internal/provider/provider.go` - extend UsageInfo struct
  - `internal/provider/claudeprovider/claudeprovider.go` - fetch real usage data
  - `internal/provider/testprovider/testprovider.go` - add dummy usage data
  - `internal/db/migrations/` - new migration for usage cache table
  - `internal/repository/models.go` + `queries.go` - usage cache model + queries
  - `internal/service/scheduler.go` - add periodic quota polling job
  - `internal/service/notification.go` - add quota unblock notification message
  - `internal/api/handlers.go` - update usage endpoint response
  - `web/index.html` - add usage display section
  - `web/js/app.js` - fetch and render usage data
  - `web/css/style.css` - usage bar and blocked indicator styles
- Related patterns: existing scheduler runs at 00:05 daily via time.Sleep loop; providers use ctx + credential map; notifications via SendAll
- Dependencies: none new (existing webpush + telegram channels used)

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Extend Provider Interface with Usage/Quota Fields

**Files:**
- Modify: `internal/provider/provider.go`
- Modify: `internal/provider/testprovider/testprovider.go`

- [ ] Add fields to UsageInfo: `CurrentUsageSeconds int64`, `TotalLimitSeconds int64`, `IsBlocked bool`
- [ ] Keep `ResetDate time.Time` as-is for backward compat with billing cycle display
- [ ] Update testprovider.FetchUsageInfo to return realistic dummy values (e.g. 3h used of 5h limit, not blocked)
- [ ] Write unit tests for UsageInfo field calculations (e.g. PercentUsed helper if added)
- [ ] Run test suite - must pass before task 2

### Task 2: Update Claude Provider to Fetch Real Usage Data

**Files:**
- Modify: `internal/provider/claudeprovider/claudeprovider.go`

- [ ] Explore Claude API endpoints available with session key to find usage/quota data (likely `/api/organizations/{id}/usage` or similar)
- [ ] Parse response to extract: hours/messages used this week, total weekly limit, whether quota is currently blocked
- [ ] Populate `CurrentUsageSeconds`, `TotalLimitSeconds`, `IsBlocked` in returned UsageInfo
- [ ] Handle case where usage endpoint is unavailable (return partial data, not error)
- [ ] Write unit tests with mocked HTTP responses for usage parsing
- [ ] Run test suite - must pass before task 3

### Task 3: Add Database Migration for Usage Cache

**Files:**
- Create: `internal/db/migrations/002_usage_cache.sql`
- Modify: `internal/repository/models.go`
- Modify: `internal/repository/queries.go`

- [ ] Create migration adding `provider_usage` table: `id`, `provider_name TEXT UNIQUE`, `current_usage_seconds INTEGER`, `total_limit_seconds INTEGER`, `is_blocked INTEGER`, `fetched_at DATETIME`
- [ ] Add `ProviderUsage` struct to models.go
- [ ] Add queries: `UpsertProviderUsage(ctx, name, usage)` and `GetProviderUsage(ctx, name)`
- [ ] Write repository tests
- [ ] Run test suite - must pass before task 4

### Task 4: Add Periodic Quota Polling to Scheduler

**Files:**
- Modify: `internal/service/scheduler.go`
- Modify: `internal/service/notification.go`

- [ ] Add a second ticker in scheduler running every 15 minutes (configurable via env var `QUOTA_POLL_INTERVAL`, default 15m)
- [ ] Each tick: call provider.FetchUsageInfo, compare `is_blocked` with last stored state from DB
- [ ] If transition detected (was blocked → now unblocked): call SendAll with message "Your {provider} quota is unblocked! You can use it again."
- [ ] If newly blocked (unblocked → blocked): optionally send "Your {provider} quota has been reached for this week." notification
- [ ] Save updated usage to DB after each poll
- [ ] Write tests for state transition detection logic
- [ ] Run test suite - must pass before task 5

### Task 5: Update API Usage Endpoint Response

**Files:**
- Modify: `internal/api/handlers.go`

- [ ] Update the existing GET `/api/providers/claude/usage` handler to return full usage data: `current_usage_seconds`, `total_limit_seconds`, `is_blocked`, `reset_date`, `fetched_at`
- [ ] Add GET `/api/providers/usage/cached` endpoint returning last cached usage from DB (no live fetch, fast for UI polling)
- [ ] Write handler tests
- [ ] Run test suite - must pass before task 6

### Task 6: Update Web Interface to Show Usage/Quota

**Files:**
- Modify: `web/index.html`
- Modify: `web/js/app.js`
- Modify: `web/css/style.css`

- [ ] Add a "Usage Status" section on the main subscriptions page (below or alongside the subscription list)
- [ ] Fetch from `/api/providers/usage/cached` on page load and every 5 minutes
- [ ] Display per-provider: progress bar showing used/total, hours remaining text, "BLOCKED" badge if is_blocked, reset date
- [ ] Style: use existing orange theme; blocked state uses red accent; progress bar fills with orange, turns red when >90%
- [ ] Handle missing data gracefully (show "No usage data - login required" if no cache)
- [ ] Manual "Refresh Usage" button that calls live fetch endpoint
- [ ] Run test suite - must pass before task 7

### Task 7: Verify Acceptance Criteria

- [ ] manual test: login to Claude, verify usage data appears on main page with correct hours
- [ ] manual test: verify polling runs and logged in scheduler output
- [ ] manual test: verify notification sent when quota state changes (can use testprovider toggling)
- [ ] run full test suite: `go test ./...`
- [ ] run linter: `golangci-lint run` or `go vet ./...`
- [ ] verify test coverage meets 80%+

### Task 8: Update Documentation

- [ ] update README.md if user-facing changes (new env var QUOTA_POLL_INTERVAL)
- [ ] update CLAUDE.md if internal patterns changed
- [ ] move this plan to `docs/plans/completed/`
