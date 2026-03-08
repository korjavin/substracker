---
# Web Panel Redesign: Orange Color Scheme + Enhanced UX

## Overview
Redesign the existing web panel (web/) with a Memrise-like orange color scheme, a card-based mobile-friendly subscription list, and a proper subscription detail panel. All backend API endpoints already exist. Only frontend files change.

## Context
- Files involved: `web/css/style.css`, `web/index.html`, `web/js/app.js`
- Existing: subscription list (table), add/edit modal, delete, notification settings, activity log tabs
- Current accent color: purple #6c63ff — needs to become orange (~#FF6200)
- Subscription model fields: id, name, service, billing_day, notes, created_at, updated_at
- No backend changes needed

## Development Approach
- **Testing approach**: Regular (implement, then manual visual verification)
- Complete each task fully before moving to the next
- No build tooling — plain HTML/CSS/JS, served as static files from `web/`
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Orange color scheme in CSS

**Files:**
- Modify: `web/css/style.css`

- [ ] Replace `--accent: #6c63ff` with `--accent: #ff6200` (Memrise-style orange)
- [ ] Replace `--accent-hover: #8880ff` with `--accent-hover: #ff7c2a`
- [ ] Update `.tab-btn.active` background to use `rgba(255,98,0,0.12)`
- [ ] Update `.btn-primary` to use orange accent
- [ ] Update `.form-group input:focus` border to orange accent
- [ ] Update `.modal-backdrop` focus ring, alert-info colors to orange
- [ ] Update `header h1` color to orange
- [ ] Update `tr:hover td` background to orange tint `rgba(255,98,0,0.04)`
- [ ] Update `.badge-claude` to use a warmer orange (align with new accent)
- [ ] Verify all `var(--accent)` usages look correct after change
- [ ] Manual: open browser, confirm orange accent throughout, no purple remnants

### Task 2: Mobile-friendly subscription cards

**Files:**
- Modify: `web/css/style.css`
- Modify: `web/index.html`
- Modify: `web/js/app.js`

- [ ] Add `.sub-card` CSS: card block per subscription, visible on mobile (≤600px), hidden on desktop
- [ ] Keep existing table for desktop (≥601px), hide cards on desktop
- [ ] Add media query: `@media (max-width: 600px)` — show cards, hide table
- [ ] In `web/js/app.js` `renderSubs()`: generate both table rows AND card markup in same render pass; show/hide via CSS
- [ ] Card layout: service badge top-right, name bold, billing day + next reset below, actions row at bottom
- [ ] Make header nav scrollable horizontally on mobile (add `overflow-x: auto; -webkit-overflow-scrolling: touch` to nav)
- [ ] Manual: verify at 375px viewport — cards visible, table hidden, tabs scrollable

### Task 3: Subscription detail panel

**Files:**
- Modify: `web/index.html`
- Modify: `web/css/style.css`
- Modify: `web/js/app.js`

- [ ] Add a slide-in detail panel to `index.html`: `<aside id="detail-panel">` with close button, fields for name/service/billing day/notes/created date, and Edit + Delete action buttons
- [ ] CSS: detail panel fixed right side on desktop (width 340px, full height), full-screen overlay on mobile; `transform: translateX(100%)` hidden, `translateX(0)` open; transition 0.2s
- [ ] In `app.js`: add `openDetail(id)` function — populates panel fields from `subs` array, opens panel
- [ ] Wire subscription name/row click to `openDetail(id)` (instead of inline edit)
- [ ] Keep "Edit" button in detail panel opening the existing edit modal; keep "Delete" button calling `deleteSub(id)` then closing panel
- [ ] Add `closeDetail()` function; wire to close button and backdrop click
- [ ] Manual: click a subscription → panel slides in with correct data; Edit opens modal; Delete works; panel closes properly

### Task 4: Polish and verification

**Files:**
- Modify: `web/css/style.css` (minor tweaks)

- [ ] Ensure empty state uses orange accent for icon or CTA
- [ ] Ensure `+ Add` button in header area is prominent on mobile (full-width or larger tap target)
- [ ] Run full manual test: add subscription → appears in list → click to view detail → edit → save → delete
- [ ] Verify mobile at 375px: all interactions work, no horizontal overflow
- [ ] Verify desktop at 1200px: table view, detail panel slides in from right

### Task 5: Update documentation

- [ ] Update README.md if there are user-facing screenshots or UI description
- [ ] Move this plan to `docs/plans/completed/`
---
