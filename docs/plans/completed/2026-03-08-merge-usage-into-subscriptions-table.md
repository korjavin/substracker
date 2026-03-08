---
# Merge Usage into Subscriptions Table

## Overview
Remove the separate "Usage Status" card from the Subscriptions tab and add usage columns directly to the subscriptions table. For services with cached usage data (claude, googleone), show status and usage inline per row. For others, show a dash.

## Context
- Files involved:
  - `web/index.html` - remove usage card HTML
  - `web/js/app.js` - merge usage fetching into subscription rendering
  - `web/css/style.css` - minor tweaks for inline usage display
  - `internal/repository/queries.go` - add ListProviderUsage query
  - `internal/api/handlers.go` - update cachedUsage to return all providers
- Related patterns: existing renderSubs, renderUsage, api() helper
- The current `/api/providers/usage/cached` only returns Claude usage (hardcoded). The scheduler already polls multiple providers and caches them, so we need to expose all cached rows.

## Development Approach
- Testing approach: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Add ListProviderUsage to repository and API

**Files:**
- Modify: `internal/repository/queries.go`
- Modify: `internal/api/handlers.go`
- Modify: `internal/repository/queries_test.go`
- Modify: `internal/api/handlers_test.go`

- [ ] Add `ListProviderUsage(ctx) ([]ProviderUsage, error)` to `queries.go` (SELECT all from provider_usage)
- [ ] Update `cachedUsage` handler in `handlers.go` to call `ListProviderUsage` and return an array instead of a single object (returns `[]ProviderUsage`, empty array if no rows)
- [ ] Add test in `queries_test.go` for `ListProviderUsage`
- [ ] Update `handlers_test.go` for the changed cachedUsage response shape (array)
- [ ] Run `go test ./...` - must pass before task 2

### Task 2: Update frontend to merge usage into subscriptions table

**Files:**
- Modify: `web/index.html`
- Modify: `web/js/app.js`
- Modify: `web/css/style.css`

- [ ] Remove the `#usage-container` card div from `index.html`
- [ ] Remove the `refresh-usage-btn` event listener and usage-only functions from `app.js` (or fold refresh into subscriptions area)
- [ ] Add a "Refresh Usage" button to the section header in `index.html` (next to "+ Add")
- [ ] In `app.js`, update `loadSubs` to fetch usage (`/api/providers/usage/cached`) in parallel with subscriptions using `Promise.all`
- [ ] Build a lookup map from provider_name to usage object (e.g. `{ claude: usageObj, googleone: usageObj }`) keyed by lowercased provider name stripped of spaces
- [ ] Add "Status" and "Usage" columns to the table header in `renderSubs`
- [ ] In each row, match subscription service to usage data: if found, show compact status (ACTIVE/BLOCKED badge) and usage text (e.g. "5.2h/100h" or just status if no numeric data); if not found, show "—"
- [ ] Update mobile card view similarly (add usage info to sub-card-details)
- [ ] Remove polling for usage separately; roll usage refresh into `loadSubs`
- [ ] Remove unused renderUsage, loadUsage, refreshUsage functions
- [ ] Add minimal CSS for inline usage display (compact bar or text style)
- [ ] Manually test: table shows usage inline for Claude row, dash for other services

### Task 3: Verify acceptance criteria

- [ ] manual test: subscriptions table shows Status and Usage columns; Claude row shows live usage status; non-tracked services show "—"
- [ ] manual test: "Refresh Usage" button fetches new usage and re-renders table
- [ ] run `go test ./...` - must pass
- [ ] run linter (if configured)

### Task 4: Update documentation

- [ ] update CLAUDE.md if internal patterns changed
- [ ] move this plan to `docs/plans/completed/`
