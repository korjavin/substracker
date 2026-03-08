---
# Use chatgpt.com/codex/settings/usage for OpenAI Provider Usage Scraping

## Overview
Update the OpenAI provider to scrape Codex usage data from chatgpt.com instead of (or in addition to) api.openai.com billing API. This includes discovering the actual API endpoint used by the page, updating the provider to parse quota/usage data, and adding clear per-service cookie instructions in the settings modal UI.

## Context
- Files involved:
  - `internal/provider/openaiprovider/openaiprovider.go` - update to use chatgpt.com endpoint, parse usage
  - `internal/provider/openaiprovider/openaiprovider_test.go` - update tests for new endpoint/response
  - `web/index.html` - add dynamic auth help section in subscription modal
  - `web/js/app.js` - show/hide service-specific cookie instructions based on selected service
- Related patterns: same Provider interface pattern as Z.ai (returns CurrentUsageSeconds, TotalLimitSeconds, IsBlocked, ResetDate)
- Auth mechanism: `__Secure-next-auth.session-token` cookie from chatgpt.com domain (same cookie name as api.openai.com but different domain, so different value)
- Current state: OpenAI provider only returns ResetDate, no usage tracking; settings UI has a generic auth token input with no instructions

## Development Approach
- Testing approach: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Discover and document the chatgpt.com/codex/settings/usage API endpoint

**Files:**
- No code changes - manual investigation step

- [ ] Open a browser, log in to chatgpt.com, navigate to https://chatgpt.com/codex/settings/usage
- [ ] Open DevTools > Network tab (filter: Fetch/XHR), refresh the page
- [ ] Identify the API endpoint(s) called to fetch usage data (likely something under /backend-api/ or /api/)
- [ ] Record: full URL, HTTP method, required request headers, cookie names in Cookie header, response JSON schema (fields: quota used, quota limit, reset date, etc.)
- [ ] Confirm which cookie name is needed (expected: __Secure-next-auth.session-token from chatgpt.com)
- [ ] Document findings as a comment block at the top of openaiprovider.go before any code changes

### Task 2: Update OpenAI provider to scrape Codex usage from chatgpt.com

**Files:**
- Modify: `internal/provider/openaiprovider/openaiprovider.go`
- Modify: `internal/provider/openaiprovider/openaiprovider_test.go`

- [ ] Change baseURL from https://api.openai.com to https://chatgpt.com
- [ ] Update FetchUsageInfo to call the discovered API endpoint (from Task 1) with the session token as the Cookie header value (__Secure-next-auth.session-token=<token>)
- [ ] Add response structs matching the actual JSON schema from Task 1
- [ ] Parse usage fields into UsageInfo: CurrentUsageSeconds (or equivalent request counts scaled to seconds), TotalLimitSeconds, IsBlocked (if quota exceeded), ResetDate
- [ ] Keep ErrUnauthorized on 401/403 responses
- [ ] If the chatgpt.com endpoint also provides billing reset date, use it; otherwise derive from the usage period
- [ ] Update/replace tests in openaiprovider_test.go with httptest.Server mock matching the new endpoint URL and response shape; cover: success with full usage data, unauthorized (401), unexpected response
- [ ] Run `go test ./internal/provider/openaiprovider/...` - must pass

### Task 3: Add service-specific cookie instructions in the settings modal

**Files:**
- Modify: `web/index.html`
- Modify: `web/js/app.js`

- [ ] In index.html, add a collapsible/hidden div inside the `sub-auth-group` form group (below the input), with id `sub-auth-help`, containing per-service instructions rendered as HTML steps
- [ ] For the OpenAI service, write clear step-by-step instructions:
  1. Go to https://chatgpt.com and sign in
  2. Open DevTools: press F12 (or Cmd+Option+I on Mac)
  3. Click the "Application" tab > "Cookies" in the left sidebar > select https://chatgpt.com
  4. Find the cookie named `__Secure-next-auth.session-token`
  5. Click the row and copy the full value from the "Value" column (it will be a long JWT string)
  6. Paste the value into the Auth Token field above
- [ ] In app.js, add a `updateAuthHelp(service)` function that sets the innerHTML of `sub-auth-help` based on the selected service (openai shows the ChatGPT steps above; claude shows instructions for claude.ai sessionKey; googleone shows SID from one.google.com; zai shows session_cookie from z.ai; other hides the help)
- [ ] Call `updateAuthHelp` on the service select `change` event and when the modal opens (with the current service value)
- [ ] Style: the help section should be visually distinct (e.g., light info box) using existing CSS variables; add minimal inline styles if no matching class exists
- [ ] Manual test: open Add Subscription modal, switch service dropdown, verify instructions update per service

### Task 4: Verify acceptance criteria

- [ ] Manual test: add openai subscription with chatgpt.com session token, verify usage data loads (quota used/total, blocked status, reset date)
- [ ] Manual test: switch service in modal between claude/openai/other, verify instructions update correctly
- [ ] Run full test suite: `go test ./...` - must pass
- [ ] Run linter: `go vet ./...` - must pass

### Task 5: Update documentation

- [ ] Update README.md: add OpenAI to the Tracked Providers section with note about getting `__Secure-next-auth.session-token` from chatgpt.com
- [ ] Move this plan to `docs/plans/completed/`
