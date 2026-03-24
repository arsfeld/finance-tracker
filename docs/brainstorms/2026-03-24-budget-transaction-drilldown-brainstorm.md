# Brainstorm: Budget-to-Transaction Drill-Down

**Date:** 2026-03-24
**Status:** Complete

## What We're Building

Make transactions visible and accessible from the budget screen. Each budget category (budgeted and unbudgeted) becomes expandable, showing the top 5 transactions by amount inline, with a "View all" link that navigates to the Transactions page pre-filtered by category and billing period.

**Core user story:** "I see I've spent $450 of my $500 Groceries budget. I click the row and immediately see the 5 biggest charges — Costco $120, Whole Foods $85... — and can click 'View all' to see every Groceries transaction this period."

## Why This Approach

### Problem
The budget screen shows aggregate spending per category (spent/budget/remaining/percent) but provides no visibility into *which transactions* make up that spending. Users have to manually navigate to the Transactions page and set filters to investigate. This friction makes budgets feel disconnected from the underlying data.

### Chosen Direction
- **Accordion expansion** on budget category rows — click to expand and see top 5 transactions by amount
- **"View all" link** navigates to `/transactions?category=X&start=...&end=...` with pre-applied filters
- **Both budgeted and unbudgeted categories** get the same treatment
- **No backend changes** — reuse existing `GET /api/transactions` endpoint with category/date/sort/limit params
- **Lazy loading** — transaction data fetched only when a row is expanded

### Why Not Alternatives
- **Side panel / Sheet:** Heavier interaction for a quick peek; covers other budget rows
- **Hover card / Popover:** Poor mobile/touch support; positioning issues with many rows
- **Embed in budget status response:** Over-fetches data for categories the user never expands
- **New dedicated endpoint:** Unnecessary — existing transactions endpoint already supports all needed filters

## Key Decisions

1. **Accordion UI pattern** — Clicking a budget category row expands a section below it showing transactions. Familiar, mobile-friendly, minimal layout disruption. Uses shadcn Collapsible or similar.

2. **Top 5 by amount** — The inline preview shows the 5 largest transactions (by absolute amount) in the category for the current billing period. This highlights the biggest spending drivers, which is what users care about when checking budget progress.

3. **Lazy fetch on expand** — Transaction data is fetched via `GET /api/transactions?category=X&start=...&end=...&sort_by=amount&sort_dir=asc&limit=5&included_only=true` only when the user expands a row. Keeps the initial budget page load fast.

4. **Same treatment for unbudgeted categories** — Unbudgeted categories also expand to show top transactions. Helps users understand unbudgeted spending and decide whether to create a budget.

5. **No backend changes** — The existing transactions endpoint with its filtering, sorting, and pagination capabilities covers everything needed. Frontend-only feature.

## What Exists to Build On

- **Transactions endpoint** — `GET /api/transactions` supports `category`, `start_date`, `end_date`, `sort_by`, `sort_dir`, `limit`, and `included_only` filters
- **Budget status response** — Already provides `period.start` and `period.end` Unix timestamps for constructing transaction queries
- **URL-driven Transactions page** — Already reads `?category=X&start=...&end=...` from URL params, so deep-linking works out of the box
- **TanStack Query** — Existing `useTransactions()` hook can be parameterized per-category
- **shadcn/ui** — Collapsible, Table, and other components available for the accordion UI
- **Dashboard precedent** — The category pie chart already links to `/transactions?category=X&start=...&end=...`, establishing the navigation pattern

## Resolved Questions

1. **What data to show in the inline preview?** — Top 5 transactions by amount. Shows date, description, and amount for each.

2. **Should unbudgeted categories have drill-down?** — Yes, same treatment as budgeted categories.

3. **Where does transaction data come from?** — Reuse existing `/api/transactions` endpoint. No new backend endpoints.

4. **How to handle the "View all" link?** — Navigate to `/transactions?category=X&start=...&end=...` which the Transactions page already supports via URL params.

## Out of Scope (for now)

- Inline category editing from the transaction preview
- Transaction search within the budget accordion
- Historical period comparison within the accordion
- Aggregate stats in the accordion (average transaction, transaction count)
