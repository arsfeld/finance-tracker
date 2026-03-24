---
title: "feat: Add transaction drill-down to budget categories"
type: feat
status: completed
date: 2026-03-24
origin: docs/brainstorms/2026-03-24-budget-transaction-drilldown-brainstorm.md
---

# feat: Add transaction drill-down to budget categories

## Overview

Make transactions visible and accessible directly from the budget screen. Each budget category row (budgeted and unbudgeted) becomes expandable via a chevron toggle, revealing the top 5 transactions by amount for that category within the current billing period. A "View all" link navigates to the Transactions page with pre-applied filters.

This is a **frontend-only feature** — no backend changes required. The existing `GET /api/transactions` endpoint already supports all needed filters.

## Problem Statement / Motivation

The budget screen shows aggregate spending per category (spent/budget/remaining/percent) but provides no visibility into *which transactions* contribute to that spending. Users must manually navigate to the Transactions page and set filters to investigate. This friction makes budgets feel disconnected from the underlying transaction data.

(see brainstorm: `docs/brainstorms/2026-03-24-budget-transaction-drilldown-brainstorm.md`)

## Proposed Solution

Add an accordion-style expand/collapse to each category row on the Budgets page:

1. **Chevron toggle** on each row (budgeted + unbudgeted) — clicking expands a section below showing the top 5 transactions by absolute amount
2. **Lazy data fetch** — transactions are fetched via the existing `/api/transactions` endpoint only when a row is expanded
3. **"View all" link** — navigates to `/transactions?category=X&start=...&end=...&included_only=true`, following the existing Dashboard navigation pattern
4. **Consistent treatment** — both budgeted and unbudgeted categories (including in the EmptyState) get the same drill-down behavior

## Technical Considerations

### Click Target Design (Critical)

Both `BudgetedRow` and `UnbudgetedRow` already contain interactive elements (inline editor, delete button, "Fix with AI", "Ask AI", "Accept"/"Custom" buttons). Making the entire row clickable would cause accidental accordion toggles.

**Solution:** Add a dedicated **chevron icon button** as the toggle target, placed at the left of each row. This:
- Avoids event propagation conflicts with existing controls
- Provides clear visual affordance (chevron rotates on expand)
- Gets keyboard accessibility for free via Radix `CollapsibleTrigger`

### Transaction Query Parameters

The accordion's API call must match how budgets calculate "spent":

```
GET /api/transactions?category=X&start={period.start}&end={period.end}&sort_by=amount&sort_dir=asc&limit=5&included_only=true
```

- `sort_by=amount&sort_dir=asc` — expenses are negative, so ascending sort surfaces the largest expenses first
- Do **NOT** pass `include_positive=true` — budget spending only counts negative amounts, so the drill-down should match
- `included_only=true` — matches budget status which only counts included accounts

### "View all" Navigation

Follow the existing Dashboard pattern (`Dashboard.tsx:40-48`):

```typescript
const params = new URLSearchParams({
  category: item.category,
  start: String(period.start),
  end: String(period.end),
  included_only: "true",
});
navigate(`/transactions?${params}`);
```

Note: The Transactions page defaults `include_positive` to `true`, so the full view will include credits/refunds. This is an existing behavior (Dashboard → Transactions has the same discrepancy) and is acceptable — the full Transactions page is meant to show the complete picture.

### Radix Collapsible (No New UI Wrapper Needed)

The `radix-ui` package (v1.4.3) is already installed and provides `Collapsible`, `CollapsibleTrigger`, and `CollapsibleContent`. These can be imported directly:

```typescript
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "radix-ui"
```

No need to create a new shadcn wrapper component — use the primitives directly in `Budgets.tsx`, keeping all budget components in one file (existing pattern).

### Caching Strategy

- Use TanStack Query's `enabled` option so the fetch only fires when the accordion is open
- TanStack Query will cache results per query key, so re-expanding a previously opened row won't re-fetch (within staleTime)
- No special invalidation needed — standard React Query behavior handles this

## Acceptance Criteria

### Functional

- [x] Each budgeted category row has a chevron toggle that expands to show top 5 transactions by amount
- [x] Each unbudgeted category row has the same expandable behavior
- [x] Unbudgeted rows in the EmptyState component are also expandable
- [x] Expanded section shows: date (short format), description (truncated with ellipsis), and amount (absolute value) for each transaction
- [x] Expanded section shows a "View all" link that navigates to `/transactions?category=X&start=...&end=...&included_only=true`
- [x] "View all" link is always visible (even when <= 5 transactions) since it provides access to search, sort, and export features
- [x] Transaction data is fetched lazily only when a row is expanded
- [x] Collapsing and re-expanding uses cached data (no unnecessary re-fetch)
- [x] Multiple rows can be expanded simultaneously across both cards

### Edge Cases

- [x] Category with 0 transactions shows "No transactions this period" message
- [x] Category with < 5 transactions shows all available transactions
- [x] Loading state shows "Loading..." in muted text while transactions fetch
- [x] Error state shows brief error message in destructive color

### UX / Accessibility

- [x] Chevron icon rotates smoothly (CSS transition) to indicate expanded/collapsed state
- [x] Chevron toggle is keyboard accessible (Enter/Space via Radix CollapsibleTrigger)
- [x] Long transaction descriptions are truncated with ellipsis
- [x] Expand/collapse has a smooth height animation
- [x] Existing row controls (edit, delete, AI buttons) continue to work without triggering accordion

## Implementation Approach

### Files to Modify

| File | Change |
|------|--------|
| `web/src/pages/Budgets.tsx` | Add Collapsible wrapper to `BudgetedRow`, `UnbudgetedRow`, and EmptyState rows; add chevron toggle; add transaction preview section |
| `web/src/api/queries.ts` | Add `useCategoryTransactions(category, start, end, enabled)` hook |

### New Hook: `useCategoryTransactions`

```typescript
// web/src/api/queries.ts
export function useCategoryTransactions(
  category: string,
  start: number,
  end: number,
  enabled: boolean
) {
  const params: Record<string, string> = {
    category,
    start: String(start),
    end: String(end),
    sort_by: "amount",
    sort_dir: "asc",
    limit: "5",
    included_only: "true",
  };
  return useTransactions(enabled ? params : undefined);
  // Note: useTransactions with undefined params should be
  // handled via the `enabled` option on useQuery instead.
  // Implementation may need a slight refactor to support this.
}
```

Alternatively, create a dedicated query that uses `enabled`:

```typescript
export function useCategoryTransactions(
  category: string,
  start: number,
  end: number,
  enabled: boolean
) {
  return useQuery({
    queryKey: ["transactions", "category-preview", category, start, end],
    queryFn: async () => {
      const params = new URLSearchParams({
        category,
        start: String(start),
        end: String(end),
        sort_by: "amount",
        sort_dir: "asc",
        limit: "5",
        included_only: "true",
      });
      const res = await fetch(`/api/transactions?${params}`);
      const json = await res.json();
      return json.data as DBTransaction[];
    },
    enabled,
  });
}
```

### Component Structure (BudgetedRow example)

```tsx
function BudgetedRow({ item, period, onEdit, onDelete, onFixWithAI }) {
  const [isOpen, setIsOpen] = useState(false);
  const { data: transactions, isLoading, isError } = useCategoryTransactions(
    item.category, period.start, period.end, isOpen
  );

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className="py-3 space-y-2">
        {/* Existing row content */}
        <div className="flex items-center gap-2">
          <CollapsibleTrigger asChild>
            <button className="...">
              <ChevronRight className={cn("h-4 w-4 transition-transform", isOpen && "rotate-90")} />
            </button>
          </CollapsibleTrigger>
          {/* ... existing category name, amounts, controls ... */}
        </div>
        {/* ... existing progress bar ... */}
      </div>

      <CollapsibleContent>
        <TransactionPreview
          transactions={transactions}
          isLoading={isLoading}
          isError={isError}
          category={item.category}
          period={period}
        />
      </CollapsibleContent>
    </Collapsible>
  );
}
```

### TransactionPreview Sub-component

A small component (defined in `Budgets.tsx`) that renders:
- Loading state: "Loading..." in muted text
- Error state: "Failed to load transactions" in destructive color
- Empty state: "No transactions this period" in muted text
- Transaction list: compact table/list with date, description (truncated), amount
- "View all" link at the bottom

## Dependencies & Risks

**Dependencies:**
- None — all required packages (`radix-ui`, `lucide-react` for ChevronRight icon) are already installed

**Risks:**
- **Low:** Layout shift when expanding rows in a `divide-y` container — may need minor CSS adjustments
- **Low:** If many rows are expanded simultaneously, multiple concurrent API calls — TanStack Query handles this gracefully

## Sources & References

- **Origin brainstorm:** [docs/brainstorms/2026-03-24-budget-transaction-drilldown-brainstorm.md](docs/brainstorms/2026-03-24-budget-transaction-drilldown-brainstorm.md) — key decisions: accordion UI, top 5 by amount, no backend changes, lazy loading, same treatment for budgeted/unbudgeted
- Dashboard category navigation pattern: `web/src/pages/Dashboard.tsx:40-48`
- Budget status API handler: `internal/api/budgets.go:131`
- Transaction list API handler: `internal/api/transactions.go:23-58`
- Existing Budgets page: `web/src/pages/Budgets.tsx`
- TanStack Query hooks: `web/src/api/queries.ts`
