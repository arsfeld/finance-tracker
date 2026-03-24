# Brainstorm: Per-Category Budgets

**Date:** 2026-03-24
**Status:** Complete

## What We're Building

A per-category budget tracking system that lets users set spending limits for each transaction category and see how actual spending compares against those limits within each billing cycle.

**Core user story:** "I want to set a $500/month budget for Groceries and see at a glance that I've spent $320 so far this billing period — 64% of my limit."

## Why This Approach

### Problem
The finance tracker shows what you spent but not whether that spending is on track. Users have no way to set targets or see progress against personal spending goals.

### Chosen Direction
- **Per-category limits** — one budget amount per category (e.g., Groceries: $500, Dining: $200)
- **Billing cycle alignment** — budgets reset on the billing day (currently 15th), matching existing transaction grouping
- **Dedicated Budgets page** — new page in the sidebar for full budget management and progress visualization
- **Web UI management** — budgets are set and edited through the web frontend, stored in the database

### Why Not Alternatives
- **Settings key-value store:** No proper schema, string parsing for amounts, hard to extend
- **Column on categories table:** Categories are keyed by `merchant_description` (many-to-one with category names), no clean place to attach a per-category-name budget amount
- **Config file:** User wants web UI management, not file editing

## Key Decisions

1. **New `budgets` database table** — migration 003, keyed by category name with an amount column. Clean relational design that's easy to query and extend.

2. **Billing cycle periods** — budgets align with existing billing periods (resets on billing day), not calendar months. Reuses the mature `billing/periods.go` logic.

3. **Per-category only** — no overall spending cap or sub-category budgets. Keep it simple. Can extend later if needed.

4. **Dedicated page** — a new `/budgets` route in the sidebar, showing all categories with progress bars (spent vs. limit), and controls to set/edit budget amounts.

5. **Backend: new API endpoints** — `GET/POST/PUT/DELETE /api/budgets` for CRUD, plus `GET /api/budgets/status` that joins budget limits with actual spending (`CountByCategory`) for the current billing period.

## What Exists to Build On

- **Category system** — mature, with `CountByCategory(ctx, start, end)` already computing per-category spending totals for arbitrary date ranges
- **Billing periods** — well-established in `internal/billing/periods.go`, easy to get current period boundaries
- **Dashboard** — already shows category pie chart and spending summary
- **Migration system** — existing pattern with numbered SQL files in `internal/database/migrations/`
- **REST API patterns** — consistent `{"data": ...}` response format, React Query hooks on the frontend
- **shadcn/ui + Tailwind** — component library already in use for consistent UI

## Resolved Questions

1. **Uncategorized transactions** — No. Only named categories can have budgets. This encourages categorizing transactions.

2. **Historical view** — Multi-period history. Show current period progress plus a trend of budget performance over the last 3-6 billing periods (over/under by category).

3. **LLM integration** — Yes. Include budget limits in the analysis prompt so the AI can comment on adherence (e.g., "You're 90% through your Dining budget with 2 weeks left").

## Out of Scope (for now)

- Alert notifications when exceeding a budget (email/ntfy)
- Overall total spending cap
- Sub-category budgets
- Rollover of unused budget between periods
- Shared/collaborative budgets
