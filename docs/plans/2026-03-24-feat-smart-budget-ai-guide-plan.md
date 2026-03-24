---
title: "feat: Add smart budget AI guide"
type: feat
status: active
date: 2026-03-24
origin: docs/brainstorms/2026-03-24-smart-budget-ai-guide-brainstorm.md
---

# feat: Add Smart Budget AI Guide

## Overview

Enhance the existing budget system with AI guidance. Add budget tools to the chat system, a chat drawer on the budgets page triggered by contextual "Fix with AI" buttons, smart budget suggestions for unbudgeted categories, and budget data in the LLM analysis prompt. (see brainstorm: `docs/brainstorms/2026-03-24-smart-budget-ai-guide-brainstorm.md`)

## Problem Statement / Motivation

The budget page shows numbers but provides no guidance. Over-budget categories display red bars, but the user must figure out what to do on their own. Unbudgeted categories show a generic "Set Budget" button with no suggested amount. The AI analysis reports don't mention budgets at all.

## Proposed Solution

Four enhancements layered onto the existing budget and chat systems:

1. **Budget chat tools** — 3 new tools (`get_budget_status`, `set_budget`, `delete_budget`) added to the existing `POST /api/chat` endpoint
2. **Chat drawer on /budgets** — slide-out panel with "Fix with AI" buttons on over-budget and unbudgeted rows
3. **Smart suggestions** — 3-month average spending shown as suggested budget amounts for unbudgeted categories
4. **LLM analysis integration** — budget data injected into the scheduled analysis prompt

## Technical Approach

### Phase 1: Backend — Budget Chat Tools

**Add `BudgetStore` to `ChatHandler`** (`internal/api/chat.go`):

The `ChatHandler` struct currently has `txnStore`, `acctStore`, `catStore`, `analysisStore`, and `events`. Add `budgetStore *store.BudgetStore`. Update `NewChatHandler` signature and the wire-up in `internal/server/server.go:109`.

**3 new tool definitions** added to the `chatTools` slice:

```go
// get_budget_status — no parameters
// Returns: current period budgets with spending, unbudgeted categories

// set_budget — params: category (string, required), amount (number, required)
// Returns: confirmation with new budget state

// delete_budget — params: category (string, required)
// Returns: confirmation of deletion
```

**Tool execution** in the `executeTool` switch statement:

- `get_budget_status`: calls `billing.CurrentBillingPeriod`, then `budgetStore.GetAll` + `txnStore.CountByCategory`, formats as readable text (matches the BudgetHandler.Status logic)
- `set_budget`: validates amount > 0, category not "Uncategorized" (same rules as REST endpoint), calls `budgetStore.Upsert`, broadcasts `budgets_updated` SSE event, returns confirmation
- `delete_budget`: calls `budgetStore.Delete`, broadcasts `budgets_updated` SSE event, returns confirmation

**Validation parity:** The chat tool must apply the same validation as `BudgetHandler.Upsert` — amount > 0, <= 1M, not NaN/Inf, category trimmed, not "Uncategorized", max 100 chars. Return validation errors as tool result text so the LLM can explain the issue to the user.

**SSE event for budget mutations:** Broadcast `budgets_updated` (with `events.Broadcast("budgets_updated", ...)`) after successful `set_budget` or `delete_budget` calls, matching the existing `categories_updated` pattern in chat.go lines 422/429/463.

**Update `chatSystemPrompt`** to include:

```
You can also view and manage the user's budgets:
- Use get_budget_status to see current budgets and spending progress
- Use set_budget to create or update a budget for a category
- Use delete_budget to remove a budget
Always describe what you plan to do and ask for confirmation before modifying or deleting budgets.
```

**Files:**
- `internal/api/chat.go` (modify — add budgetStore field, 3 tool defs, 3 tool implementations, update system prompt)
- `internal/server/server.go` (modify — pass budgetStore to NewChatHandler)

### Phase 2: Backend — Smart Suggestions + Analysis Integration

**Smart suggestions in status endpoint** (`internal/api/budgets.go`):

Extend `UnbudgetedCategory` with a `SuggestedAmount` field:

```go
type UnbudgetedCategory struct {
    Category        string   `json:"category"`
    Spent           float64  `json:"spent"`
    SuggestedAmount *float64 `json:"suggested_amount"` // nil if insufficient data
}
```

In `BudgetHandler.Status`, after computing current spending, also compute 3-month historical averages:
1. Get the 3 prior billing periods via `billing.CalculateBillingPeriods`
2. Call `txnStore.CountByCategory` for each period
3. Average each category's spending across available periods (min 1 period required)
4. Round up to the nearest $10 (e.g., $487.33 → $490)
5. Set `SuggestedAmount` on each unbudgeted category

If fewer than 3 months of data exist, use whatever is available (1-2 months). The frontend can note "Based on N months of data."

**Budget handler needs TransactionStore for historical queries:** The `BudgetHandler` already has `txnStore` — no new dependency needed.

**LLM analysis integration** (`internal/llm/analyze.go`, `internal/api/analysis_run.go`):

Add `budgetStore *store.BudgetStore` to `AnalysisRunHandler`. In `runAnalysis`, fetch budgets and pass to `GeneratePrompt`:

```go
// New signature
func GeneratePrompt(txns, accounts, startDate, endDate, billingDay, isMultiPeriod, budgets []models.Budget) string
```

Add a "Budget Status" section to the prompt after category breakdown:

```
Budget Status:
- Groceries: $620 spent / $500 budget (124% — OVER by $120)
- Dining: $180 spent / $200 budget (90%)
- Shopping: $450 spent (no budget set)

Comment on budget adherence where budgets are set. Note any categories that are significantly over budget.
```

If no budgets exist, omit the section entirely (don't waste tokens).

**Files:**
- `internal/api/budgets.go` (modify — add suggested_amount calculation)
- `internal/llm/analyze.go` (modify — add budgets parameter, add budget section to prompt)
- `internal/api/analysis_run.go` (modify — add budgetStore, fetch budgets, pass to GeneratePrompt)
- `internal/server/server.go` (modify — pass budgetStore to NewAnalysisRunHandler)

### Phase 3: Frontend — Chat Drawer + Smart Suggestions

**Install Sheet component:**

```bash
cd web && npx shadcn@latest add sheet
```

This creates `web/src/components/ui/sheet.tsx` based on Radix Dialog with slide-in animation.

**Extract reusable chat component** from `web/src/pages/Chat.tsx`:

Create `web/src/components/ChatPanel.tsx` — a headless chat component that accepts:
```typescript
interface ChatPanelProps {
  initialMessages?: Message[];  // pre-seeded context
  storageKey?: string;          // localStorage key (null = no persistence)
  className?: string;
}
```

The existing `/chat` page becomes a thin wrapper: `<ChatPanel storageKey="finance-chat-messages" />`.

**Budget chat drawer** on `/budgets` page:

```typescript
// State in Budgets.tsx
const [drawerOpen, setDrawerOpen] = useState(false);
const [drawerContext, setDrawerContext] = useState<{category: string; ...} | null>(null);

// "Fix with AI" click handler
const handleFixWithAI = (category: string, spent: number, amount?: number) => {
  const contextMsg = amount
    ? `Help me with my ${category} budget. Current status: ${formatCurrency(spent)} spent of ${formatCurrency(amount)} limit (${Math.round((spent/amount)*100)}%). This billing period: ${period.label}.`
    : `I'm spending ${formatCurrency(spent)} on ${category} this period but have no budget set. Help me decide on a budget.`;
  setDrawerContext({ category, initialMessage: contextMsg });
  setDrawerOpen(true);
};
```

The drawer renders:
```tsx
<Sheet open={drawerOpen} onOpenChange={setDrawerOpen}>
  <SheetContent side="right" className="w-[420px] sm:w-[420px] p-0">
    <SheetHeader className="p-4 border-b">
      <SheetTitle>Budget Assistant</SheetTitle>
    </SheetHeader>
    <ChatPanel
      initialMessages={drawerContext ? [{ role: "user", content: drawerContext.initialMessage }] : []}
      storageKey={null} // ephemeral, no persistence
    />
  </SheetContent>
</Sheet>
```

**"Fix with AI" button placement:**
- On `BudgetedRow` — show only when `percent >= 100` (over budget)
- On `UnbudgetedRow` — show alongside "Set Budget" button
- Not shown on under-budget rows (they don't need "fixing")

**Smart suggestion display** on unbudgeted rows:

When `suggested_amount` is present, show it as a pre-filled value in the `SetBudgetInput` component:
```
Shopping  $145 spent  [Suggested: $490] [Accept] [Set Budget]
```

Clicking "Accept" calls `POST /api/budgets` with the suggested amount. "Set Budget" opens the inline editor as before for manual entry.

**SSE listener for budget updates:**

In `web/src/hooks/useSSE.ts`, add `budgets_updated` to the event listener. On receive, invalidate `queryKey: ["budgets"]`.

**Files:**
- `web/src/components/ui/sheet.tsx` (new — via shadcn CLI)
- `web/src/components/ChatPanel.tsx` (new — extracted from Chat.tsx)
- `web/src/pages/Chat.tsx` (modify — use ChatPanel wrapper)
- `web/src/pages/Budgets.tsx` (modify — add drawer, "Fix with AI" buttons, smart suggestions)
- `web/src/api/types.ts` (modify — add suggested_amount to UnbudgetedCategory)
- `web/src/hooks/useSSE.ts` (modify — listen for budgets_updated)

## Technical Considerations

- **Drawer is ephemeral** — no localStorage persistence. Each "Fix with AI" click starts fresh. The `/chat` page keeps its own persistent history.
- **Drawer width:** 420px on desktop, full-width on mobile (< 640px). The Sheet component from shadcn handles responsive behavior.
- **Drawer auto-sends first message** — when opened with context, the initial user message is sent immediately so the AI responds right away. The user sees the context message and the AI's response.
- **AI must confirm before mutating** — the system prompt instructs the AI to describe planned changes and ask for confirmation before calling `set_budget` or `delete_budget`. This is a prompt-level guardrail, not a code-level one.
- **3-month average rounds to nearest $10** — `math.Ceil(avg / 10) * 10`. Shows "Based on N months" when fewer than 3 months available.
- **Tool call results not visible in UI** — the existing chat frontend only renders `user` and `assistant` messages. Tool calls happen invisibly, matching current behavior. The AI describes what it did in its response text.

## System-Wide Impact

- **API surface parity:** Budget tools added to `POST /api/chat` — both the drawer and the /chat page can manage budgets. No new endpoints.
- **Error propagation:** Tool validation errors return text to the LLM which explains the issue to the user. LLM API failures show an error message in the drawer.
- **State lifecycle:** Budget mutations via chat tools write to SQLite, broadcast SSE event, frontend invalidates React Query cache. No race conditions — SQLite serializes writes.
- **Cross-store dependency:** `ChatHandler` gains `budgetStore`, `AnalysisRunHandler` gains `budgetStore`. Both are already multi-store handlers.

## Acceptance Criteria

- [x] `get_budget_status` chat tool returns current budgets with spending data
- [x] `set_budget` chat tool creates/updates budgets with same validation as REST endpoint
- [x] `delete_budget` chat tool removes budgets
- [x] Chat system prompt updated to describe budget tools and require confirmation before mutations
- [x] Budget mutations via chat broadcast `budgets_updated` SSE event
- [x] Unbudgeted categories in `GET /api/budgets/status` include `suggested_amount` (3-month average, rounded to $10)
- [x] LLM analysis prompt includes budget adherence section when budgets exist
- [x] Sheet/Drawer component installed and working on /budgets page
- [x] "Fix with AI" button appears on over-budget rows and unbudgeted rows
- [x] Clicking "Fix with AI" opens drawer with context-aware first message
- [x] Chat panel extracted as reusable component; /chat page still works
- [x] Drawer is ephemeral (no persistence, fresh conversation each time)
- [x] "Accept" button on suggested amounts creates the budget with one click
- [x] Frontend listens for `budgets_updated` SSE and invalidates budget queries

## Dependencies & Risks

- **OpenRouter availability** — "Fix with AI" button should be hidden if `OPENROUTER_URL` is not configured (check via settings or a config endpoint). Graceful degradation.
- **LLM tool-calling quality** — the AI might call tools with wrong arguments. Validation in tool execution prevents bad data; the LLM receives error text and can retry.
- **3-month average accuracy** — new users with < 3 months of data get less reliable suggestions. Show "Based on N months" label.

## Success Metrics

- User can fix an over-budget category via AI in under 60 seconds
- Chat tool calls succeed on first attempt > 90% of the time
- Scheduled analysis mentions budget adherence when budgets are configured

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-24-smart-budget-ai-guide-brainstorm.md](docs/brainstorms/2026-03-24-smart-budget-ai-guide-brainstorm.md) — Key decisions: extend existing chat with budget tools, chat drawer on budgets page, smart suggestions, LLM analysis integration

### Internal References

- Chat system + tools: `internal/api/chat.go` (tool definitions at line 80, dispatch at line 309)
- Chat system prompt: `internal/api/chat.go:191`
- Budget handler + status: `internal/api/budgets.go`
- Budget store: `internal/store/budgets.go`
- CountByCategory: `internal/store/transactions.go:242`
- Billing periods: `internal/billing/periods.go`
- LLM analysis prompt: `internal/llm/analyze.go:21` (GeneratePrompt), line 263 (buildCategoryBreakdown)
- Analysis runner: `internal/api/analysis_run.go:120` (GeneratePrompt call site)
- SSE events: `internal/api/events.go`, `web/src/hooks/useSSE.ts`
- Chat frontend: `web/src/pages/Chat.tsx`
- Budget frontend: `web/src/pages/Budgets.tsx`
- Server wiring: `internal/server/server.go:24-109`
