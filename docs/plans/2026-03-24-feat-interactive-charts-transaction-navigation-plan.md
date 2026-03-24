---
title: "feat: Interactive Charts with Graph-to-Transaction Navigation"
type: feat
status: completed
date: 2026-03-24
origin: docs/brainstorms/2026-03-24-interactive-charts-brainstorm.md
---

# feat: Interactive Charts with Graph-to-Transaction Navigation

## Overview

Add rich interactive charting to the finance tracker web UI. All charts become clickable — clicking a data point (category, billing period, merchant, day) navigates to the Transactions page with appropriate filters pre-applied via URL search params. Includes a new dedicated Analytics page with three new chart types, and a category filter dropdown on the Transactions page.

## Problem Statement / Motivation

The current Dashboard has two static charts (pie + bar) with no interactivity. Users can see spending breakdowns but cannot drill into the underlying transactions. The Transactions page has basic filters (billing period + text search) but no category filter, despite the API supporting it. There's no way to visually explore spending patterns over time, by day, or by merchant.

This feature creates a drill-down workflow: **visual summary → filtered transaction detail**, making the data explorable and actionable.

## Proposed Solution

(see brainstorm: `docs/brainstorms/2026-03-24-interactive-charts-brainstorm.md`)

### Core approach: URL-driven filtering

All chart clicks use React Router's `useNavigate()` to go to `/transactions?category=X&start=Y&end=Z`. The Transactions page reads URL search params via `useSearchParams()` and applies them as filters. This gives us shareable URLs, browser back/forward support, and zero global state management.

### Five implementation phases

1. **Backend**: New API endpoints + data fixes
2. **Transactions refactor**: URL-driven filters + category dropdown
3. **Dashboard interactivity**: Make existing charts clickable
4. **Analytics page**: Three new interactive charts
5. **Polish**: Theme colors, hover states, responsive

## Technical Considerations

### Architecture impacts

- New `/analytics` route alongside existing `/analysis` (LLM reports). Different purposes — analytics is visual charts, analysis is AI-generated text reports.
- Three new API endpoints under `/api/analytics/` namespace.
- Transactions page state management changes from `useState` to `useSearchParams` for filter state.

### Key decisions from SpecFlow analysis

**Uncategorized transactions bug**: `CountByCategory` labels transactions with no category as `"Uncategorized"`, but the `List` filter matches against empty string `""`. Clicking the "Uncategorized" pie slice would return zero results. **Fix**: The transactions `List` filter must treat `category=Uncategorized` as equivalent to `category=""`.

**TrendPoint needs timestamps**: The bar chart click handler needs `start` and `end` Unix timestamps per bar, but `TrendPoint` only has `Label` and `Total`. **Fix**: Extend `TrendPoint` struct with `Start` and `End` fields.

**URL param scope**: `category`, `search`, `start`, `end`, `page` go in URL params. `sortBy` and `sortDir` stay as local `useState` (less important to share/bookmark).

**Search debouncing**: Use 300ms debounce with `replace` (not `push`) to avoid polluting browser history.

**Include positive**: Keep `include_positive=true` on Transactions page regardless of navigation source. Charts show expenses only, but the drill-down should show the full picture for that category/period.

**Billing period reconciliation**: When URL contains a date range not matching any billing period (e.g., single-day heatmap click), show all period buttons deselected with a "Custom range" indicator and a clear button.

**Merchant matching**: Top merchants chart uses raw `description` field — no merchant normalization (out of scope). Click uses `search` param for fuzzy match, which is acceptable.

**Accounts consistency**: Dashboard charts filter by included accounts (`is_included=1`). The Transactions `List` query currently shows all accounts. For consistency when navigating from charts, pass `included_only=true` in the URL when coming from Dashboard/Analytics chart clicks.

### Performance implications

- New SQL queries for daily totals and merchant aggregation — both simple GROUP BY queries, index on `posted` already exists.
- Category trend query iterates billing periods — could be a single query with date bucketing or multiple calls. Single query preferred.
- Heatmap renders up to ~90 day cells — lightweight.

### Security considerations

- URL params are user-controlled input — validate `start`/`end` are valid numbers, `page` is positive, `category` is string. Fall back to defaults for invalid values.
- Sort column whitelist already exists — no changes needed.

## System-Wide Impact

- **Interaction graph**: Chart click → `useNavigate()` → route change → Transactions component mounts → reads `useSearchParams` → fires React Query with new params → API call → renders filtered table. No side effects beyond navigation.
- **Error propagation**: Invalid URL params → fall back to defaults (no errors shown). API errors → React Query handles with existing error display pattern.
- **State lifecycle risks**: None — URL is the source of truth, React Query handles caching/refetching.
- **API surface parity**: New endpoints follow existing patterns. Chat tool-calling may benefit from analytics endpoints in the future but not required now.

## Acceptance Criteria

### Phase 1: Backend Preparation

- [x]`TrendPoint` struct extended with `Start int64` and `End int64` fields, populated in dashboard handler
- [x]`TransactionFilter` supports `IncludedOnly bool` field — when true, joins with accounts and filters by `is_included = 1`
- [x]Transaction `List` query treats `category=Uncategorized` as a filter for transactions with empty/null category
- [x]New endpoint `GET /api/analytics/category-trend?months=N` returns per-category spending totals across N billing periods
  - Response: `{ data: { categories: string[], periods: { label: string, start: number, end: number, totals: Record<string, number> }[] } }`
- [x]New endpoint `GET /api/analytics/daily-totals?start=X&end=Y` returns per-day expense totals
  - Response: `{ data: { days: { date: string, total: number }[] } }`
- [x]New endpoint `GET /api/analytics/top-merchants?start=X&end=Y&limit=N` returns top N merchants by total spending
  - Response: `{ data: { merchants: { name: string, total: number, count: number }[] } }`

### Phase 2: Transactions Page URL-Driven Filtering

- [x]Filter state (`category`, `search`, `start`, `end`, `page`) managed via `useSearchParams` instead of `useState`
- [x]Search input debounced at 300ms, uses `replace` for URL updates
- [x]Category filter dropdown added (shadcn Command/Combobox pattern) with "All categories" option, populated from `/api/categories/unique`
- [x]Billing period selector reconciles with URL `start`/`end` — highlights matching period or shows "Custom range" indicator with clear button
- [x]URL with pre-populated params (from chart click or shared link) correctly initializes all filter controls
- [x]Browser back/forward correctly restores filter state
- [x]CSV export includes current URL filter params

### Phase 3: Dashboard Chart Interactivity

- [x]Pie chart slices are clickable — navigating to `/transactions?category=X&start=Y&end=Z&included_only=true`
- [x]Bar chart bars are clickable — navigating to `/transactions?start=Y&end=Z`
- [x]Clickable chart elements show `cursor: pointer` and subtle hover effect
- [x]Tooltip still works alongside click handler (no interference)

### Phase 4: Analytics Page

- [x]New `/analytics` route with sidebar navigation (between Transactions and Analysis)
- [x]Period selector at the top of Analytics page (default: last 6 billing periods)
- [x]Category trend line/area chart — shows spending per category over billing periods
  - Click a data point → `/transactions?category=X&start=Y&end=Z`
  - Top 8 categories shown, rest grouped as "Other"
  - Legend clickable to toggle category visibility
- [x]Daily spending heatmap — calendar grid showing expense intensity per day
  - Click a day → `/transactions?start=<day_start>&end=<day_end>`
  - Color scale from light to dark based on spending amount
  - Shows last 90 days by default
- [x]Top merchants horizontal bar chart — top 10 merchants by spending
  - Click a bar → `/transactions?search=<merchant_name>&start=Y&end=Z`
- [x]All charts render empty state when no data available
- [x]All charts are responsive on mobile

### Phase 5: Polish

- [x]Chart colors use CSS theme variables (`--color-chart-1` through `--color-chart-5`) instead of hardcoded hex, respecting dark/light mode
- [x]Hover states on all clickable chart elements (opacity change or highlight)
- [x]Mobile-friendly chart layouts (stacked on narrow screens)
- [x]Loading skeletons for charts while data is fetching

## Dependencies & Risks

**Dependencies:**
- Recharts v3.8 — already installed, supports click handlers on all chart types
- React Router v7 — already installed, `useSearchParams` available
- shadcn/ui Command component — already installed, used by CategoryPicker

**Risks:**
- **Heatmap not in Recharts**: Recharts does not have a native calendar heatmap component. Options: (a) build custom with `<svg>` grid + Recharts tooltip, (b) use a lightweight third-party like `react-calendar-heatmap`, or (c) build with plain HTML/CSS grid. Recommend option (c) — a simple CSS grid with Tailwind styling, keeping the dependency count low.
- **Chart click target sizes on mobile**: Pie slices and small bars can be hard to tap. Mitigate with tooltips that include a "View transactions" link as an alternative click target.
- **Naming confusion**: `/analytics` vs `/analysis` — ensure sidebar labels are distinct ("Analytics" with a chart icon vs "Reports" or "AI Analysis" with a document icon). Consider renaming `/analysis` to `/reports` in a follow-up.

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-24-interactive-charts-brainstorm.md](docs/brainstorms/2026-03-24-interactive-charts-brainstorm.md) — Key decisions carried forward: URL-driven filtering, dedicated Analytics page, all charts clickable, navigate to Transactions (not inline).

### Internal References

- Dashboard charts: `web/src/pages/Dashboard.tsx:79-116`
- Transaction filtering: `web/src/pages/Transactions.tsx` (full file)
- Transaction store/filter: `internal/store/transactions.go` (TransactionFilter struct, List method, CountByCategory)
- Dashboard API handler: `internal/api/dashboard.go` (TrendPoint struct, trend data generation)
- Category summary endpoint: `internal/api/categories.go:Summary`
- React Query hooks: `web/src/api/queries.ts`
- Route definitions: `web/src/App.tsx`
- Sidebar navigation: `web/src/components/layout/Sidebar.tsx`
- CSS chart variables: `web/src/index.css` (--color-chart-1 through --color-chart-5)

### Key Files to Create

- `internal/api/analytics.go` — new analytics handler with 3 endpoints
- `web/src/pages/Analytics.tsx` — new analytics page component
- `web/src/components/CategoryFilter.tsx` — category filter dropdown (distinct from CategoryPicker)
- `web/src/components/charts/CategoryTrend.tsx` — category trend line chart
- `web/src/components/charts/SpendingHeatmap.tsx` — daily spending heatmap
- `web/src/components/charts/TopMerchants.tsx` — top merchants bar chart

### Key Files to Modify

- `internal/store/transactions.go` — extend TransactionFilter, add analytics queries, fix Uncategorized handling
- `internal/api/dashboard.go` — extend TrendPoint with Start/End
- `internal/server/server.go` — register new analytics routes
- `web/src/pages/Dashboard.tsx` — add click handlers to charts
- `web/src/pages/Transactions.tsx` — refactor to useSearchParams, add category filter
- `web/src/api/queries.ts` — add analytics query hooks, update transaction query
- `web/src/App.tsx` — add /analytics route
- `web/src/components/layout/Sidebar.tsx` — add Analytics nav link
- `web/src/index.css` — potentially extend chart color variables if 5 isn't enough
