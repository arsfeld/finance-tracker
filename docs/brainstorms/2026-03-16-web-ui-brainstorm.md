# Brainstorm: Web UI for Finance Tracker

**Date:** 2026-03-16
**Status:** Complete

## What We're Building

A full-featured web UI that **replaces the CLI** as the primary interface for the finance tracker. The web application will serve as a visual dashboard for browsing transactions and spending trends, a reader for LLM analysis results, and a control panel for managing filters, triggering analyses, and adjusting settings.

**Target user:** Single user running on a local machine or homelab. No authentication required.

## Why This Approach

The CLI works well for scheduled batch runs, but a web UI provides:

- **At-a-glance financial insights** via charts and dashboards instead of reading email summaries
- **Interactive exploration** of transactions, categories, and trends across billing periods
- **On-demand control** to trigger analyses, adjust filters, and manage configuration without editing `.env` files or YAML
- **Persistent, browsable history** of past analyses and spending patterns

The web server absorbs the CLI's responsibilities (fetching, analyzing, notifying) and adds a visual layer on top.

## Key Decisions

### 1. Architecture: Web Replaces CLI

The web server becomes the single entry point. It will:
- Run as a long-lived process (vs. the current batch CLI)
- Handle scheduled transaction fetching and analysis (replacing cron + CLI)
- Serve the React frontend and REST API
- Continue sending email/ntfy notifications on schedule

The CLI binary goes away. All orchestration moves into the web server.

### 2. Backend: Go stdlib HTTP (net/http with Go 1.22+ routing)

- Zero new dependencies for the HTTP layer
- Go 1.22+ provides method-based routing (`GET /api/transactions`, `POST /api/analyze`)
- Sufficient for a single-user personal tool
- Middleware for logging, CORS, etc. written by hand (minimal)

### 3. Data Storage: SQLite with Pure Go Driver

- Replace flat JSON files (`transaction_cache.json`, `categories.json`) with a single SQLite database
- Embedded, zero-config, single-file — easy to back up
- Enables proper querying: date ranges, category aggregations, search
- Go driver: `modernc.org/sqlite` (pure Go, no CGO) — simpler builds, no gcc dependency
- Migration system needed for schema evolution
- Start fresh (no migration from existing JSON files) — app will re-fetch from SimpleFin on first run

### 4. Frontend: React + Vite + TypeScript

- React 19 with Vite for fast development and optimized builds
- TypeScript for type safety
- Deployed via Docker — Go server serves the built frontend as static files from disk

### 5. UI Components: Tailwind CSS + shadcn/ui

- Tailwind for utility-first styling
- shadcn/ui for accessible, copy-paste components (tables, cards, dialogs, forms)
- Dark mode support via Tailwind's dark variant

### 6. Charts: Recharts

- React-native, declarative charting built on D3
- Covers the needed chart types: line (trends), bar (period comparisons), pie/donut (category breakdown), area (spending over time)
- Good defaults, customizable

### 7. Notifications: Kept as scheduled feature

- The web server runs `robfig/cron` for configurable scheduled fetching, analysis, and notification dispatch
- Email and ntfy channels remain as-is
- The web UI provides a settings page to configure notification preferences and schedule
- On-demand analysis can also be triggered from the web UI

## High-Level Feature Map

### Dashboard (Home)
- Current period spending summary (total, daily average, burn rate)
- Spending by category (pie/donut chart)
- Spending trend (line chart across recent billing periods)
- Recent transactions list
- Latest LLM analysis summary

### Transactions
- Searchable, filterable, sortable transaction table
- Filter by date range, category, account, amount range
- Category assignment/override
- Export capability

### Analysis
- View current and past LLM analysis reports
- Trigger on-demand analysis
- Compare periods side-by-side

### Settings / Configuration
- Manage notification channels (email, ntfy)
- Configure billing day
- Manage transaction filters (replace YAML editing)
- Set OpenRouter model preferences
- View/manage accounts (which to include/exclude)

## Project Structure (Proposed)

```
finance_tracker/
  cmd/
    server/
      main.go              # Entry point, starts HTTP server + scheduler
  internal/
    server/
      server.go            # HTTP server setup, routes
      middleware.go         # Logging, CORS
    api/
      transactions.go      # Transaction API handlers
      analysis.go          # Analysis API handlers
      settings.go          # Settings API handlers
      accounts.go          # Account API handlers
    store/
      sqlite.go            # SQLite connection, migrations
      transactions.go      # Transaction queries
      accounts.go          # Account queries
      categories.go        # Category queries
      analysis.go          # Analysis result storage
    simplefin/
      client.go            # SimpleFin API client (extracted from current simplefin.go)
    llm/
      client.go            # OpenRouter client (extracted from current llm.go)
      categorize.go        # Merchant categorization
      analyze.go           # Spending analysis
    scheduler/
      scheduler.go         # Periodic fetch + analyze + notify
    notify/
      email.go             # Email notifications
      ntfy.go              # Ntfy notifications
    config/
      config.go            # Configuration loading
    models/
      models.go            # Shared data types
  web/                     # React frontend
    src/
      components/
      pages/
      api/                 # API client hooks
      App.tsx
      main.tsx
    package.json
    vite.config.ts
    tailwind.config.ts
    tsconfig.json
  migrations/              # SQLite schema migrations
  Dockerfile
  docker-compose.yml
  justfile
```

## Resolved Questions

1. **SQLite driver:** `modernc.org/sqlite` (pure Go). No CGO simplifies Docker builds. Performance difference negligible for single-user.

2. **Frontend serving:** Serve from filesystem within Docker container. No need for `go:embed` since deployment is Docker-based.

3. **Data migration:** Start fresh. No import from existing JSON files. App re-fetches from SimpleFin on first run.

4. **Scheduling:** `robfig/cron` for flexible, configurable cron-expression-based scheduling via the web UI.
