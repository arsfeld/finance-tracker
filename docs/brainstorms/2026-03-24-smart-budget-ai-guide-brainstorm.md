# Brainstorm: Smart Budget AI Guide

**Date:** 2026-03-24
**Status:** Complete

## What We're Building

An AI-guided enhancement to the existing budget system. The budget page remains the primary interface, but gains contextual AI assistance:

1. **"Fix with AI" buttons** on over-budget categories that open a chat drawer pre-loaded with context
2. **Budget chat tools** added to the existing tool-calling chat system, allowing the AI to create, adjust, and delete budgets
3. **Smart suggestions for unbudgeted categories** based on historical spending averages
4. **Budget data in LLM analysis prompt** so the scheduled analysis naturally comments on budget adherence

**Core user story:** "When I see Dining is over budget, I click 'Fix with AI' and a chat drawer opens. The AI already knows the context — it might suggest lowering the budget, point out a spike in restaurant spending, or recommend splitting the category. It can directly adjust the budget for me."

## Why This Approach

### Problem
The manual budget system requires users to interpret numbers and make adjustments themselves. Over-budget categories just show red — there's no guidance on *what to do about it*. Unbudgeted categories show a generic "Set Budget" button with no suggested amount.

### Chosen Direction
- **AI as guide, not manager** — the user stays in control via the budget page, but can invoke AI help at any point
- **Extend existing chat, don't build new AI infrastructure** — the chat system already supports tool-calling with 6 tools. Adding 3 budget tools is minimal effort.
- **Contextual pre-seeding** — when the user clicks "Fix with AI," the chat receives the category context so the AI immediately understands the problem
- **Chat drawer on budget page** — keeps the user in context rather than navigating away to /chat
- **Smart unbudgeted suggestions** — use 3-month spending average to suggest budget amounts, shown inline on the budget page with one-click accept

### Why Not Alternatives
- **Dedicated budget advisor endpoint:** Duplicates LLM infrastructure, less flexible, harder to have a conversation
- **Fully autonomous manager:** Too risky — AI could set bad budgets. Doesn't match "AI as guide" vision. Users lose control.

## Key Decisions

1. **Add 3 budget tools to existing `POST /api/chat` endpoint** — `get_budget_status` (read current budgets + spending), `set_budget` (create/update a budget), `delete_budget` (remove a budget). These use the existing `BudgetStore` methods. No new backend endpoint — the budget drawer and the /chat page both call the same `POST /api/chat`. This means the /chat page also gains budget tool access, which is a feature.

2. **Chat drawer component on /budgets page** — a slide-out panel (not a modal, not navigation). Minimal chat interface: message list, text input, send button. Reuses the same `POST /api/chat` backend. Each "Fix with AI" click starts a fresh ephemeral conversation (no persistence, no shared history with /chat page).

3. **Smart suggestions for unbudgeted categories** — extend `GET /api/budgets/status` to include a `suggested_amount` field on unbudgeted categories, calculated from 3-month average spending (via existing `CountByCategory` across billing periods). Shown inline with a one-click "Accept" button.

4. **Budget data injected into LLM analysis prompt** — the existing scheduled analysis (every 6 hours) includes budget limits and adherence data so the AI naturally comments on budget issues in its reports. This is the v2 LLM integration from the budget plan, now prioritized.

5. **Context pre-seeding** — clicking "Fix with AI" on a category sends the first user message automatically: "Help me with my Dining budget. Current status: $240 spent of $200 limit (120%). This billing period: Mar 15 - Apr 14, 2026." The AI responds with awareness of the situation.

## Resolved Questions

1. **Should the chat drawer share history with /chat page?** — No. The budget chat drawer is ephemeral — each "Fix with AI" click starts a fresh conversation with budget context. The /chat page keeps its own separate history.

2. **What model should the budget chat use?** — Same as the existing chat system (configured via `OPENROUTER_MODEL`). No separate model config needed.

3. **Should smart suggestions appear automatically or on-demand?** — Automatically. The status endpoint includes suggested amounts for unbudgeted categories so they appear when the page loads.

## Out of Scope (for now)

- Proactive ntfy/email notifications for budget alerts (can add later as a scheduled job)
- AI automatically adjusting budgets without user interaction
- Budget forecasting ("at this rate, you'll exceed Dining by the 25th")
- Multi-period budget trend analysis by the AI
- Chat drawer on pages other than /budgets
