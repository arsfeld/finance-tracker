---
title: "feat: Replace CLI with Full-Featured Web UI"
type: feat
status: in_progress
date: 2026-03-16
origin: docs/brainstorms/2026-03-16-web-ui-brainstorm.md
---

# feat: Replace CLI with Full-Featured Web UI

## Overview

Replace the current CLI-based finance tracker with a long-lived web server that provides a visual dashboard, transaction browser, analysis viewer, and settings management UI. The web server absorbs all CLI responsibilities (fetching, analyzing, notifying) and adds interactive data visualization on top.

Single-user, local/homelab deployment. No authentication. Docker-based.

## Problem Statement

The current CLI runs as a batch process, producing analysis summaries sent via email/ntfy. This works for scheduled reports but provides no way to:

- Interactively browse transactions or drill into spending patterns
- View historical analyses or compare periods visually
- Adjust configuration without editing `.env` files or YAML
- Trigger on-demand analysis from a browser
- See real-time spending charts and category breakdowns

## Proposed Solution

A Go web server serving a React SPA, backed by SQLite for persistence and `robfig/cron` for scheduled jobs.

**Key decisions carried forward from brainstorm** (see brainstorm: `docs/brainstorms/2026-03-16-web-ui-brainstorm.md`):

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Architecture | Web replaces CLI | Single entry point, long-lived process |
| Go HTTP | `net/http` stdlib (Go 1.22+) | Zero deps, method-based routing sufficient |
| Database | SQLite via `modernc.org/sqlite` | Pure Go, no CGO, single-file backup |
| Frontend | React 19 + Vite + TypeScript | Largest ecosystem, best chart/component libraries |
| UI framework | Tailwind CSS v4 + shadcn/ui | Utility-first + accessible copy-paste components |
| Charts | Recharts 3.x | React-native, D3-based, good defaults |
| Scheduler | `robfig/cron` v3 | Cron expressions, overlap prevention, graceful shutdown |
| Notifications | Kept (email + ntfy) | Scheduled dispatch continues, configurable via web UI |
| Data migration | Start fresh | No JSON import; re-fetch from SimpleFin on first run |
| Deployment | Docker | Multi-stage build (Node + Go + Alpine) |

## Technical Approach

### Architecture

```
                    +------------------+
                    |   React SPA      |
                    | (Vite + TS)      |
                    +--------+---------+
                             |
                       HTTP / SSE
                             |
                    +--------+---------+
                    |  Go HTTP Server  |
                    |  (net/http)      |
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
        +-----+----+  +-----+----+  +------+-----+
        | REST API |  | Scheduler|  |  SSE Hub   |
        | Handlers |  | (cron)   |  | (events)   |
        +-----+----+  +-----+----+  +------+-----+
              |              |              |
              +--------------+--------------+
                             |
                    +--------+---------+
                    |    Store Layer   |
                    |    (SQLite)      |
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
        +-----+----+  +-----+----+  +------+-----+
        | SimpleFin|  | OpenRouter|  | Notifications|
        | Client   |  | Client   |  | (email/ntfy) |
        +----------+  +----------+  +--------------+
```

### Key Architectural Decisions (Resolved from SpecFlow Analysis)

**Settings storage:** SQLite `settings` table with key-value pairs. Environment variables serve as initial defaults on first boot. Once the user modifies settings via the web UI, DB values take precedence. Secrets (API keys) stored in DB since this is a single-user local tool.

**Concurrency (scheduler vs on-demand):** A `sync.Mutex`-based job runner. If a scheduled job is in progress and the user clicks "Sync Now" or "Analyze Now", the API returns HTTP 409 with a message indicating the operation is already running. The frontend shows this status.

**Real-time updates:** Server-Sent Events (SSE) on `GET /api/events`. The backend emits events like `sync_started`, `sync_complete`, `analysis_started`, `analysis_complete`, `error`. The frontend uses an `EventSource` to receive these and invalidate TanStack Query caches. Lightweight, no WebSocket dependency.

**Long-running operations:** Async job pattern. `POST /api/sync` and `POST /api/analysis/run` return `202 Accepted` with a job status. Completion is communicated via SSE. The frontend shows a progress indicator.

**Transaction storage:** Store ALL transactions (including positive/income). Apply expense-only filter at analysis time, not at storage time. The Transactions page shows all with a filter toggle.

**Billing period display:** Dashboard shows the current billing period by default. If within the 5-day auto-rollback window, show a banner: "Previous period may be more complete" with a link to switch.

**New accounts:** Auto-classify using existing `isCreditCard` heuristic. Show a notification badge on Settings when new accounts are detected for user review.

**Pagination:** Offset-based, 50 items per page. SQLite handles `LIMIT/OFFSET` well for expected volumes.

**Category overrides:** Per-transaction override stored in a `category_overrides` table. When overriding, offer a checkbox: "Apply to all transactions from this merchant" which updates the merchant-level category store.

**Export:** CSV download of the current filtered transaction view.

**Dark mode:** Implemented from day one using Tailwind's `dark:` classes with a toggle in the header. Low incremental cost.

### SQLite Schema

```sql
-- accounts: synced from SimpleFin
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,          -- SimpleFin account ID
    name TEXT NOT NULL,
    balance REAL NOT NULL DEFAULT 0,
    balance_date INTEGER,         -- Unix timestamp
    currency TEXT,
    org_name TEXT,
    org_domain TEXT,
    is_included INTEGER NOT NULL DEFAULT 1,  -- user can include/exclude
    first_seen_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- transactions: synced from SimpleFin
CREATE TABLE transactions (
    id TEXT PRIMARY KEY,          -- SimpleFin transaction ID
    account_id TEXT NOT NULL REFERENCES accounts(id),
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    posted INTEGER NOT NULL,      -- Unix timestamp
    transacted_at INTEGER,        -- Unix timestamp, nullable
    pending INTEGER NOT NULL DEFAULT 0,
    cached_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_transactions_account ON transactions(account_id);
CREATE INDEX idx_transactions_posted ON transactions(posted);
CREATE INDEX idx_transactions_description ON transactions(description);

-- categories: merchant-level categorization (LLM-assigned)
CREATE TABLE categories (
    merchant_description TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'llm',  -- 'llm' or 'user'
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- category_overrides: per-transaction user overrides
CREATE TABLE category_overrides (
    transaction_id TEXT PRIMARY KEY REFERENCES transactions(id),
    category TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- analyses: stored LLM analysis results
CREATE TABLE analyses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    period_start INTEGER NOT NULL,     -- Unix timestamp
    period_end INTEGER NOT NULL,       -- Unix timestamp
    billing_day INTEGER NOT NULL,
    date_range_type TEXT NOT NULL,
    response_text TEXT NOT NULL,
    model_used TEXT,
    is_multi_period INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'success'  -- 'success', 'error'
);
CREATE INDEX idx_analyses_created ON analyses(created_at);

-- filter_rules: replaces YAML filter config
CREATE TABLE filter_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'substring',  -- 'substring', 'prefix', 'suffix'
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- specific_exclusions: date-specific filters
CREATE TABLE specific_exclusions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,
    pattern TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'substring',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- settings: key-value configuration
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- sync_log: track fetch history
CREATE TABLE sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT,
    status TEXT NOT NULL DEFAULT 'running',  -- 'running', 'success', 'error', 'partial'
    transactions_added INTEGER DEFAULT 0,
    transactions_updated INTEGER DEFAULT 0,
    error_message TEXT,
    api_errors TEXT  -- JSON array of SimpleFin API error strings
);
```

### REST API Contract

```
# Transactions
GET    /api/transactions              # List (query: page, limit, account_id, category, start, end, search, include_positive)
GET    /api/transactions/{id}         # Detail
PATCH  /api/transactions/{id}/category  # Override category (body: {category, apply_to_merchant?})
GET    /api/transactions/export       # CSV download (same filters as list)

# Accounts
GET    /api/accounts                  # List all accounts
PATCH  /api/accounts/{id}             # Update (body: {is_included})

# Categories
GET    /api/categories                # List all merchant categories
GET    /api/categories/summary        # Category totals for a period (query: start, end)

# Analysis
GET    /api/analyses                  # List past analyses
GET    /api/analyses/{id}             # Get specific analysis
GET    /api/analyses/latest           # Get most recent analysis
POST   /api/analyses/run              # Trigger on-demand analysis (returns 202)

# Sync
POST   /api/sync                      # Trigger on-demand SimpleFin fetch (returns 202)
GET    /api/sync/status               # Current sync status
GET    /api/sync/log                  # Recent sync history

# Dashboard
GET    /api/dashboard                 # Aggregated dashboard data (current period summary, category breakdown, trend)

# Settings
GET    /api/settings                  # Get all settings
PATCH  /api/settings                  # Update settings (body: {key: value, ...})
POST   /api/settings/test-notification  # Send test notification

# Filter Rules
GET    /api/filters                   # List filter rules
POST   /api/filters                   # Create filter rule
PATCH  /api/filters/{id}              # Update filter rule
DELETE /api/filters/{id}              # Delete filter rule

# Events
GET    /api/events                    # SSE stream (sync/analysis status, errors)

# Health
GET    /api/health                    # Server health check

# Static
GET    /                              # SPA fallback (serves index.html)
GET    /assets/*                      # Static assets (JS, CSS, images)
```

**Standard response format:**
```json
{
  "data": { ... },
  "meta": { "page": 1, "limit": 50, "total": 234 }
}
```

**Standard error format:**
```json
{
  "error": { "code": "SYNC_IN_PROGRESS", "message": "A sync operation is already running" }
}
```

### Project Structure

```
finance_tracker/
  cmd/
    server/
      main.go                    # Entry point: wiring, graceful shutdown
  internal/
    config/
      config.go                  # Load from env + DB, settings struct
    database/
      database.go                # SQLite open (read/write pools), migrate
      migrations/
        001_initial_schema.sql   # Full schema from above
    models/
      models.go                  # Shared domain types
    store/
      transactions.go            # Transaction CRUD + queries
      accounts.go                # Account CRUD + queries
      categories.go              # Category + override queries
      analyses.go                # Analysis CRUD + queries
      settings.go                # Settings key-value CRUD
      filters.go                 # Filter rule CRUD
      sync_log.go                # Sync log CRUD
    simplefin/
      client.go                  # SimpleFin API client
    llm/
      client.go                  # OpenRouter HTTP client
      analyze.go                 # Prompt generation, analysis logic
      categorize.go              # Merchant categorization
    notify/
      email.go                   # Email (SMTP) sender + HTML template
      ntfy.go                    # Ntfy sender
      dispatcher.go              # Multi-channel dispatch
    scheduler/
      scheduler.go               # Cron wrapper, job runner with mutex
    server/
      server.go                  # HTTP server, route registration
      middleware.go              # Logging, CORS, recovery
    api/
      transactions.go            # Transaction handlers
      accounts.go                # Account handlers
      categories.go              # Category handlers
      analyses.go                # Analysis handlers
      sync.go                    # Sync handlers
      dashboard.go               # Dashboard aggregation handler
      settings.go                # Settings handlers
      filters.go                 # Filter rule handlers
      events.go                  # SSE hub
      helpers.go                 # JSON response/error helpers
    billing/
      periods.go                 # Billing period calculation (from date.go)
      trends.go                  # Category trend analysis (from llm.go)
  web/                           # React SPA
    src/
      api/
        client.ts                # Fetch wrapper with types
        queries.ts               # TanStack Query hooks
        types.ts                 # API response types
      components/
        layout/
          AppLayout.tsx          # Sidebar nav + header + outlet
          Sidebar.tsx            # Navigation links
          Header.tsx             # Dark mode toggle, sync status
        dashboard/
          SpendingSummary.tsx     # Cards: total, daily avg, burn rate
          CategoryChart.tsx      # Pie/donut chart
          TrendChart.tsx         # Line chart across periods
          RecentTransactions.tsx  # Mini transaction list
          AnalysisSummary.tsx    # Latest LLM summary excerpt
        transactions/
          TransactionTable.tsx   # DataTable with sorting/filtering
          TransactionFilters.tsx # Filter controls
          CategoryOverride.tsx   # Category edit dialog
        analysis/
          AnalysisList.tsx       # Past analyses list
          AnalysisView.tsx       # Markdown-rendered analysis
          PeriodComparison.tsx   # Side-by-side comparison
        settings/
          NotificationSettings.tsx
          BillingSettings.tsx
          FilterRuleManager.tsx
          ModelSettings.tsx
          AccountManager.tsx
          SchedulerSettings.tsx
        ui/                      # shadcn/ui components
      hooks/
        useSSE.ts                # EventSource hook for real-time updates
        useTheme.ts              # Dark mode toggle
      pages/
        Dashboard.tsx
        Transactions.tsx
        Analysis.tsx
        Settings.tsx
      App.tsx
      main.tsx
    index.html
    package.json
    vite.config.ts
    tsconfig.json
    tsconfig.app.json
    tsconfig.node.json
    components.json              # shadcn/ui config
  migrations/                    # (symlink or copy into internal/database/migrations/)
  Dockerfile
  docker-compose.yml
  justfile
  go.mod
  go.sum
```

## System-Wide Impact

### Interaction Graph

1. **Scheduled sync**: Cron fires -> `scheduler.RunSync()` acquires mutex -> `simplefin.Client.Fetch()` -> `store.Transactions.Upsert()` + `store.Accounts.Upsert()` -> `store.SyncLog.Create()` -> SSE hub broadcasts `sync_complete` -> releases mutex
2. **Scheduled analysis**: Cron fires -> `scheduler.RunAnalysis()` acquires mutex -> `store.Transactions.GetForPeriod()` -> `llm.Analyze()` (with retry/fallback) -> `store.Analyses.Create()` -> `notify.Dispatch()` -> SSE hub broadcasts `analysis_complete` -> releases mutex
3. **On-demand sync/analysis**: API handler checks mutex -> if locked, returns 409 -> if free, runs same pipeline as scheduled, SSE broadcasts progress
4. **Category override**: API handler -> `store.CategoryOverrides.Set()` -> optionally `store.Categories.Update()` (if apply_to_merchant) -> SSE broadcasts `categories_updated` -> frontend invalidates queries

### Error Propagation

- **SimpleFin API errors**: Collected as `[]string`, stored in `sync_log.api_errors`. Warning notifications sent for each. SSE broadcasts `sync_partial` with error details. Dashboard shows warning badge.
- **LLM errors**: Retry with exponential backoff (5 retries, 2s initial). On total failure, analysis record stored with `status='error'`. SSE broadcasts `analysis_error`. No notification sent for failed analyses.
- **SQLite write errors**: Propagated up to API handler, returned as 500. Logged with zerolog. No notification.
- **Notification failures**: Logged but do not block the main pipeline. Non-fatal.

### State Lifecycle Risks

- **Partial sync**: If the server crashes mid-upsert, SQLite's transaction guarantees prevent partial writes. Each sync runs in a single write transaction.
- **Partial analysis**: If LLM call succeeds but DB write fails, the analysis is lost. Acceptable for a personal tool.
- **Settings change during scheduled run**: Settings are read once at the start of each job run. Changes during a run take effect on the next run.

### Integration Test Scenarios

1. Scheduler fires sync while user is browsing transactions -> reads continue, SSE notifies when done, UI refreshes
2. User triggers analysis while scheduled sync is running -> returns 409 with clear message
3. SimpleFin returns partial errors (some accounts fail) -> successful accounts stored, errors logged and notified, dashboard shows partial data
4. LLM rate limit hit during categorization -> categories from previous runs still apply, uncategorized shown as "Uncategorized"
5. User changes billing day -> historical analyses retain their original billing day, new analyses use updated day

## Acceptance Criteria

### Functional Requirements

- [x] Go web server starts and serves React SPA on configurable port
- [x] SQLite database created with full schema on first run
- [x] SimpleFin transactions fetched and stored in SQLite (upsert by ID)
- [x] Account include/exclude configurable via Settings, with auto-detection heuristic for new accounts
- [x] Transactions displayed in searchable, filterable, sortable, paginated table
- [x] Transaction categories displayed (LLM-assigned) with user override capability
- [x] Category override option to apply to all matching merchants
- [x] Dashboard shows: spending summary, category pie chart, trend line chart, recent transactions, latest analysis
- [ ] LLM analysis runs on schedule and on-demand, results stored and viewable
- [x] Past analyses browsable as a list, individual analysis rendered as markdown
- [ ] Period comparison view (side-by-side) for two selected analyses
- [x] Notifications (email + ntfy) dispatched on scheduled analysis completion
- [x] Warning notifications sent to separate ntfy topic for SimpleFin API errors
- [x] Settings page allows configuration of: billing day, notification channels, OpenRouter models, cron schedule, account inclusion
- [x] Filter rules manageable via Settings (create, edit, delete) replacing YAML config
- [x] Test notification button in Settings
- [x] CSV export of filtered transaction view
- [x] SSE provides real-time status updates for sync and analysis operations
- [x] Mutual exclusion prevents concurrent sync/analysis operations
- [x] Dark mode toggle in header, persisted in settings
- [x] Graceful shutdown (stop scheduler, drain HTTP connections)

### Non-Functional Requirements

- [x] Pure Go build (no CGO) — `modernc.org/sqlite`
- [x] Multi-stage Docker build (Node + Go + Alpine)
- [x] Single Docker image serves both API and frontend
- [x] SQLite WAL mode enabled for concurrent read performance
- [x] API responses under 200ms for cached data queries
- [ ] Frontend loads in under 3 seconds on local network

### Quality Gates

- [x] Go code formatted with `gofmt`
- [x] Go code passes `go vet`
- [x] Frontend builds without TypeScript errors
- [ ] Frontend passes ESLint
- [ ] Docker image builds successfully

## Implementation Phases

### Phase 1: Foundation — Go Project Restructure + SQLite

**Goal:** New project structure with working Go server, SQLite database, and health endpoint.

**Tasks:**
1. Create `cmd/server/main.go` entry point
2. Create `internal/database/` with SQLite connection setup (WAL mode, pragmas, read/write pools using `modernc.org/sqlite`)
3. Create `internal/database/migrations/001_initial_schema.sql` with full schema
4. Implement migration runner using `pressly/goose` with `embed.FS`
5. Create `internal/config/config.go` — loads env vars via `godotenv`, provides defaults
6. Create `internal/server/server.go` — basic `net/http` server with route registration
7. Create `internal/server/middleware.go` — logging (zerolog), CORS, recovery
8. Create `internal/api/helpers.go` — JSON response/error helpers
9. Implement `GET /api/health` endpoint
10. Update `go.mod` with new dependencies (`modernc.org/sqlite`, `pressly/goose`, `robfig/cron`)
11. Update `justfile` with new build commands

**Success criteria:** `go build ./cmd/server` compiles. Server starts, creates SQLite DB, runs migrations, responds to `/api/health`.

**Key files:**
- `cmd/server/main.go`
- `internal/database/database.go`
- `internal/database/migrations/001_initial_schema.sql`
- `internal/config/config.go`
- `internal/server/server.go`
- `internal/server/middleware.go`
- `internal/api/helpers.go`

---

### Phase 2: Store Layer + Data Models

**Goal:** Complete SQLite store layer with all CRUD operations.

**Tasks:**
1. Create `internal/models/models.go` — domain types adapted from current `src/models.go` for SQLite (use `time.Time` instead of raw int64 where appropriate, keep `Balance` type)
2. Implement `internal/store/accounts.go` — Upsert, List, GetByID, UpdateInclusion
3. Implement `internal/store/transactions.go` — Upsert (batch), List (with filters, pagination, sorting), GetByID, GetForPeriod, CountByCategory
4. Implement `internal/store/categories.go` — Get, Set, ListAll, BulkUpsert
5. Implement `internal/store/category_overrides.go` — Set, Get, ApplyToMerchant
6. Implement `internal/store/analyses.go` — Create, List, GetByID, GetLatest
7. Implement `internal/store/settings.go` — Get, Set, GetAll, SetMultiple
8. Implement `internal/store/filters.go` — Create, Update, Delete, List, ListActive
9. Implement `internal/store/sync_log.go` — Create, Complete, List

**Success criteria:** All store functions compile and are tested with in-memory SQLite.

**Key files:**
- `internal/models/models.go`
- `internal/store/*.go` (8 files)

---

### Phase 3: Extract Business Logic from CLI

**Goal:** Extract SimpleFin client, LLM client, notification system, and billing logic into internal packages.

**Tasks:**
1. Extract `internal/simplefin/client.go` from `src/simplefin.go` — adapt to use store layer for upserts instead of returning raw slices
2. Extract `internal/llm/client.go` from `src/llm.go` lines 76-227 — OpenRouter HTTP client (`callOpenRouter`, `getLLMResponse`, `getLLMResponseJSON`)
3. Extract `internal/llm/analyze.go` from `src/llm.go` lines 229-636, 803-1121 — prompt generation, billing period calculation, category breakdowns, trend highlights
4. Extract `internal/llm/categorize.go` from `src/llm.go` lines 691-801 — categorization orchestration, adapted to use store layer
5. Extract `internal/notify/email.go` from `src/notifications.go` lines 76-319 — email sender + HTML template
6. Extract `internal/notify/ntfy.go` from `src/notifications.go` lines 24-74 — ntfy sender
7. Create `internal/notify/dispatcher.go` — multi-channel dispatch (from `sendNotification`)
8. Extract `internal/billing/periods.go` from `src/date.go` — billing period calculation, date range types
9. Move `retryWithBackoff` generic from `src/helpers.go` to a shared utility or keep in `internal/llm/`
10. Move markdown helpers from `src/helpers.go` to `internal/notify/`

**Success criteria:** All extracted packages compile. Business logic is callable from the server context. No more `src/` package main dependencies needed for the web server.

**Key files:**
- `internal/simplefin/client.go`
- `internal/llm/client.go`, `analyze.go`, `categorize.go`
- `internal/notify/email.go`, `ntfy.go`, `dispatcher.go`
- `internal/billing/periods.go`

---

### Phase 4: REST API Handlers

**Goal:** All API endpoints implemented and returning real data.

**Tasks:**
1. Implement transaction handlers (`internal/api/transactions.go`): List (with all filters), GetByID, OverrideCategory, ExportCSV
2. Implement account handlers (`internal/api/accounts.go`): List, UpdateInclusion
3. Implement category handlers (`internal/api/categories.go`): List, Summary (aggregated totals)
4. Implement analysis handlers (`internal/api/analyses.go`): List, GetByID, GetLatest, RunAnalysis (async, returns 202)
5. Implement sync handlers (`internal/api/sync.go`): TriggerSync (async, returns 202), GetStatus, GetLog
6. Implement dashboard handler (`internal/api/dashboard.go`): Aggregated endpoint (current period summary, category breakdown, trend data, recent transactions, latest analysis excerpt)
7. Implement settings handlers (`internal/api/settings.go`): Get, Update, TestNotification
8. Implement filter handlers (`internal/api/filters.go`): CRUD
9. Implement SSE hub (`internal/api/events.go`): EventSource endpoint, broadcast mechanism
10. Create `internal/scheduler/scheduler.go` — `robfig/cron` wrapper with `SkipIfStillRunning`, mutex-based job runner, graceful shutdown
11. Wire scheduler into `cmd/server/main.go` with configurable cron expression from settings
12. Register all routes in `internal/server/server.go`

**Success criteria:** All API endpoints respond correctly (test with `curl`). Scheduler runs sync + analysis on schedule. SSE broadcasts events.

**Key files:**
- `internal/api/*.go` (9 files)
- `internal/scheduler/scheduler.go`
- `internal/server/server.go` (route registration)

---

### Phase 5: React Frontend Foundation

**Goal:** React SPA scaffolded with routing, layout, theme, and API client — connected to Go backend.

**Tasks:**
1. Scaffold React project: `npm create vite@latest web -- --template react-ts`
2. Install and configure Tailwind CSS v4 (`tailwindcss`, `@tailwindcss/vite`)
3. Initialize shadcn/ui (`npx shadcn@latest init -t vite`)
4. Add shadcn components: button, card, table, tabs, dialog, form, select, input, label, badge, separator, dropdown-menu, toast
5. Install React Router 7: `npm install react-router`
6. Install TanStack Query: `npm install @tanstack/react-query @tanstack/react-query-devtools`
7. Install TanStack Table: `npm install @tanstack/react-table`
8. Install Recharts: `npm install recharts`
9. Install react-markdown: `npm install react-markdown`
10. Configure Vite proxy (`/api` -> `http://localhost:8080`)
11. Create `src/api/types.ts` — TypeScript interfaces matching API response types
12. Create `src/api/client.ts` — fetch wrapper with error handling
13. Create `src/api/queries.ts` — TanStack Query hooks for all endpoints
14. Create `src/hooks/useSSE.ts` — EventSource hook that invalidates queries on events
15. Create `src/hooks/useTheme.ts` — dark mode toggle, persists to localStorage + API
16. Create `src/components/layout/AppLayout.tsx` — sidebar nav + header + `<Outlet />`
17. Create `src/components/layout/Sidebar.tsx` — nav links with active state
18. Create `src/components/layout/Header.tsx` — dark mode toggle, sync status indicator
19. Create page stubs: `src/pages/Dashboard.tsx`, `Transactions.tsx`, `Analysis.tsx`, `Settings.tsx`
20. Wire routing in `src/App.tsx` (declarative mode)
21. Wire providers in `src/main.tsx` (QueryClientProvider, BrowserRouter)

**Success criteria:** Frontend builds, serves via Vite dev server, navigates between 4 pages, dark mode toggles, API client connects to Go backend via proxy.

**Key files:**
- `web/vite.config.ts`
- `web/src/main.tsx`, `App.tsx`
- `web/src/api/types.ts`, `client.ts`, `queries.ts`
- `web/src/hooks/useSSE.ts`, `useTheme.ts`
- `web/src/components/layout/*.tsx`
- `web/src/pages/*.tsx`

---

### Phase 6: Dashboard Page

**Goal:** Fully functional dashboard with charts and real-time data.

**Tasks:**
1. Implement `SpendingSummary.tsx` — cards showing total spending, daily average, burn rate, days remaining
2. Implement `CategoryChart.tsx` — Recharts PieChart with category breakdown, custom tooltips
3. Implement `TrendChart.tsx` — Recharts LineChart showing spending across last 5 billing periods
4. Implement `RecentTransactions.tsx` — mini table of last 10 transactions with description, amount, date
5. Implement `AnalysisSummary.tsx` — latest LLM analysis rendered as markdown (truncated with "View full" link)
6. Wire `Dashboard.tsx` page with all components, loading/error states, billing period selector
7. Add SSE-driven auto-refresh when sync/analysis completes
8. Add "Sync Now" and "Analyze Now" buttons with loading states
9. Add 5-day auto-rollback banner when applicable

**Success criteria:** Dashboard shows real financial data with charts. Auto-refreshes on background events.

---

### Phase 7: Transactions Page

**Goal:** Full transaction browsing with filtering, sorting, pagination, and category management.

**Tasks:**
1. Implement `TransactionTable.tsx` using shadcn DataTable pattern (TanStack Table + shadcn Table)
   - Columns: Date, Description, Amount, Category, Account, Status (pending/posted)
   - Sortable by all columns
   - Server-side pagination (50 per page)
2. Implement `TransactionFilters.tsx` — date range picker, category select, account select, amount range, search input, positive/negative toggle
3. Implement `CategoryOverride.tsx` — dialog for changing category with "apply to all" checkbox
4. Implement CSV export button (triggers `/api/transactions/export` download)
5. Wire everything in `Transactions.tsx` page with URL-based filter state (useSearchParams)

**Success criteria:** Transaction table renders with all data, filters work, categories can be overridden, CSV exports.

---

### Phase 8: Analysis Page

**Goal:** Analysis history, on-demand trigger, and period comparison.

**Tasks:**
1. Implement `AnalysisList.tsx` — list of past analyses with date, period, model, status
2. Implement `AnalysisView.tsx` — full markdown-rendered analysis with metadata header
3. Implement `PeriodComparison.tsx` — side-by-side view of two analyses selected by the user
4. Add "Run Analysis" button with period selector and loading state
5. Wire in `Analysis.tsx` page

**Success criteria:** All past analyses viewable, new analysis triggerable, two periods comparable side-by-side.

---

### Phase 9: Settings Page

**Goal:** Full configuration management via web UI.

**Tasks:**
1. Implement `BillingSettings.tsx` — billing day selector (1-28)
2. Implement `NotificationSettings.tsx` — email SMTP config, ntfy topic/server config, enable/disable toggles, test notification button
3. Implement `ModelSettings.tsx` — OpenRouter URL, API key (masked), model list (comma-separated)
4. Implement `AccountManager.tsx` — list of accounts with include/exclude toggles, "new accounts detected" badge
5. Implement `FilterRuleManager.tsx` — CRUD for filter rules (pattern, match type, active toggle)
6. Implement `SchedulerSettings.tsx` — cron expression input with human-readable preview
7. Wire all sections in `Settings.tsx` page with tabs or sections
8. Implement first-run setup flow: if no SimpleFin URL configured, show setup wizard on first visit

**Success criteria:** All settings configurable via UI, saved to SQLite, take effect on next scheduled run.

---

### Phase 10: Docker + SPA Serving + Polish

**Goal:** Production-ready Docker image, Go serves React SPA, final polish.

**Tasks:**
1. Update `Dockerfile` to multi-stage: Node 22 (build frontend), Go 1.24 (build backend), Alpine (runtime)
2. Configure Go server to serve `web/dist/` as static files with SPA fallback (non-API routes serve `index.html`)
3. Update `docker-compose.yml` — expose port, mount volume for SQLite data
4. Update `justfile` with new commands: `dev` (run Go + Vite concurrently), `build` (Docker build), `up` (docker compose up)
5. Add CORS middleware for development mode (Vite on :5173, Go on :8080)
6. Final UI polish: loading skeletons, error toasts, empty states, responsive layout
7. Remove old `src/` CLI code (or move to `src-legacy/` if keeping as reference)
8. Update CLAUDE.md with new architecture documentation
9. Update `.github/workflows/tests.yaml` for new project structure

**Success criteria:** `docker compose up` builds and starts the full app. Browser opens to working dashboard. All features functional end-to-end.

---

## Alternative Approaches Considered

| Alternative | Why Rejected |
|-------------|-------------|
| **Phoenix/Elixir rewrite** | Previous backlog had Phoenix tasks, but Go + React keeps the existing Go investment and avoids a full language switch (see brainstorm) |
| **Go templates + HTMX** | Simpler stack but limited charting, less interactive data tables, harder to build a rich dashboard |
| **Keep CLI + add web** | More complex architecture maintaining two entry points; web should be the single interface |
| **PostgreSQL** | Overkill for single-user; SQLite is simpler to deploy and back up |
| **CGO SQLite (mattn)** | Requires gcc in build, complicates Docker builds; pure Go driver sufficient for this scale |
| **Embed frontend in binary** | Unnecessary since deployment is Docker-based; serving from filesystem is simpler for this use case |

## Dependencies & Prerequisites

| Dependency | Version | Purpose |
|------------|---------|---------|
| `modernc.org/sqlite` | latest | Pure Go SQLite driver |
| `github.com/pressly/goose/v3` | latest | Database migrations |
| `github.com/robfig/cron/v3` | latest | Scheduled jobs |
| `github.com/rs/zerolog` | 1.34+ | Structured logging (existing) |
| `github.com/joho/godotenv` | 1.5+ | Env file loading (existing) |
| `github.com/gomarkdown/markdown` | latest | Markdown to HTML for email (existing) |
| React | 19.x | Frontend framework |
| Vite | 7.x or 8.x | Frontend build tool |
| TypeScript | 5.4+ | Type safety |
| Tailwind CSS | 4.x | Utility CSS |
| shadcn/ui | latest | UI components |
| Recharts | 3.x | Charts |
| React Router | 7.x | Client-side routing |
| TanStack Query | 5.x | Data fetching |
| TanStack Table | 8.x | Data table |
| react-markdown | latest | Render LLM analysis |

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| SQLite locking under concurrent access | Low | Medium | WAL mode + busy_timeout + single write connection pool |
| LLM rate limits on free tier (50/day) | Medium | Medium | Track usage, warn user, allow manual-only mode |
| SimpleFin API changes | Low | High | Isolate client code, log raw responses |
| Large transaction volumes (1000+ rows) | Low | Low | SQLite handles this easily; pagination prevents frontend issues |
| Vite 8 instability (released days ago) | Medium | Low | Use Vite 7.x if issues arise; easy to switch |
| React 19 + Recharts 3 compatibility | Low | Medium | Both claim compatibility; fall back to Recharts 2 if needed |

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-16-web-ui-brainstorm.md](docs/brainstorms/2026-03-16-web-ui-brainstorm.md) — Key decisions carried forward: Go stdlib HTTP, SQLite with pure Go driver, React+Vite+TypeScript, Tailwind+shadcn/ui, Recharts, robfig/cron, web replaces CLI

### Internal References

- Current CLI orchestration: `src/main.go:254-532` (run function)
- SimpleFin client: `src/simplefin.go`
- LLM integration: `src/llm.go` (1121 lines, largest file)
- Notification system: `src/notifications.go`
- Cache system: `src/cache.go`
- Data models: `src/models.go`
- Date/billing logic: `src/date.go`
- Configuration: `src/settings.go`
- Category store: `src/categories.go`
- Account filtering heuristic: `src/main.go:107-134`

### External References

- [Go 1.22 Routing Enhancements](https://go.dev/blog/routing-enhancements)
- [modernc.org/sqlite docs](https://pkg.go.dev/modernc.org/sqlite)
- [pressly/goose migration library](https://github.com/pressly/goose)
- [robfig/cron v3](https://github.com/robfig/cron)
- [shadcn/ui Vite installation](https://ui.shadcn.com/docs/installation/vite)
- [Tailwind CSS v4](https://tailwindcss.com/blog/tailwindcss-v4)
- [React Router v7](https://reactrouter.com/start/modes)
- [TanStack Query v5](https://tanstack.com/query/latest)
- [Recharts 3.x](https://github.com/recharts/recharts)
- [SQLite WAL mode best practices](https://www.sqlite.org/wal.html)
