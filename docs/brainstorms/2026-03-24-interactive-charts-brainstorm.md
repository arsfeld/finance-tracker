# Brainstorm: Interactive Charts & Graph-to-Transaction Navigation

**Date:** 2026-03-24
**Status:** Ready for planning

## What We're Building

Add richer charting to the finance tracker web UI and make all charts interactive — clicking a data point (category, month, merchant, day) navigates to the Transactions page with the appropriate filters pre-applied. This creates a drill-down workflow from visual summaries to detailed transaction data.

### Scope

1. **Make existing Dashboard charts clickable**
   - Pie chart (spending by category): click a slice → `/transactions?category=X&start=Y&end=Z`
   - Bar chart (spending trend): click a bar → `/transactions?start=Y&end=Z` for that billing period

2. **New dedicated Analytics page** (`/analytics`)
   - Category trend over time: line/area chart showing per-category spending across billing periods
   - Daily spending heatmap: calendar-style grid showing spending intensity by day
   - Top merchants bar chart: horizontal bar chart of top N merchants by total spending
   - All three charts are clickable, navigating to filtered transactions

3. **Add category filter to Transactions page**
   - Category dropdown/combobox filter alongside existing billing period and search filters
   - Syncs with URL query params (so chart click-throughs work and URLs are shareable)

4. **URL-driven filtering**
   - All filters (category, date range, search, merchant) expressed as URL search params
   - Transactions page reads params on mount and applies them
   - Browser back/forward navigation works naturally

## Why This Approach

- **URL-driven filtering** is the simplest approach — no global state management needed, URLs are shareable and debuggable, browser history works for free
- **Dedicated Analytics page** keeps the Dashboard clean (summary-focused) while giving charts room to breathe
- **All charts clickable** provides consistent UX — every visual element is a gateway to the underlying data
- **Category filter on Transactions** completes the filtering story — users can also manually filter without needing to click a chart

## Key Decisions

1. **Navigate to Transactions** (not inline expansion) when clicking chart elements — keeps pages focused, reuses existing table
2. **Dedicated /analytics page** for new charts — Dashboard keeps existing pie + bar + summary cards
3. **All charts are interactive** — consistent drill-down behavior across every visualization
4. **URL params for filter state** — simple, shareable, works with browser history
5. **Add visible category filter** to Transactions page — not just implicit from URL, users get a dropdown they can use directly

## New API Requirements

- **Category trend data**: Need an endpoint (or extend existing) returning per-category totals across multiple billing periods (for the line/area chart)
- **Daily spending data**: Need an endpoint returning per-day spending totals for the heatmap (within a date range)
- **Top merchants**: Need an endpoint returning top N merchants by total spending (within a date range)
- The existing `/api/categories/summary` endpoint may be extendable for some of this

## Technical Notes

- Frontend uses **Recharts** — all new charts should use Recharts for consistency
- **React Router** `useSearchParams` for reading/writing URL filter state
- **React Query** for data fetching — new query hooks for new endpoints
- Existing `TransactionFilter` struct in Go already supports `category` and `account_id` — just not exposed in UI
- shadcn/ui component library available for the category filter dropdown (Combobox/Command pattern)

## Open Questions

None — all key decisions resolved during brainstorming.
