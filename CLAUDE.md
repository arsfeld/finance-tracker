# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WalletMind is now a multi-tenant web application using Supabase for the backend. It allows multiple users to track their finances collaboratively within organizations.

## Memories

- User devenv shell to run commands

## Common Development Commands

### Build Commands
```bash
# Build the project (default)
just build
# or
go build -o bin/finance_tracker ./src

# Build with version information
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o bin/finance_tracker ./src
```

### Run Commands

#### CLI Mode (Financial Analysis)
```bash
# Run with default settings
just run

# Run with verbose logging
just run-verbose

# Run with specific notifications (e.g., email or ntfy)
just run-notify email
just run-notify ntfy

# Run with custom date range
./bin/finance_tracker --date-range last_month
./bin/finance_tracker --date-range custom --start-date 2024-01-01 --end-date 2024-03-31

# Force fresh analysis (bypass cache)
./bin/finance_tracker --force
```

#### Web Server Mode
```bash
# Run web server (default port 8080)
just web

# Run web server with verbose logging
just web-verbose

# Run web server on specific port
just web-port 3000

# Run web server in production mode
just web-prod

# Run web server with hot-reload (development)
just web-dev

# Run web server with hot-reload on specific port
just web-dev-port 3000

# Watch and rebuild on changes (without running)
just watch
```

#### River Job Worker Mode
```bash
# Start River job worker (processes background jobs)
go run src/main.go worker

# Start worker with custom settings
go run src/main.go worker --queues=sync,analysis --concurrency=5

# Create jobs via CLI
go run src/main.go job create sync-transactions --org-id=<uuid> --conn-id=<uuid>
go run src/main.go job create full-sync --org-id=<uuid> --conn-id=<uuid> --force
go run src/main.go job create analyze-spending --org-id=<uuid>
go run src/main.go job create cleanup --org-id=<uuid> --dry-run

# List and manage jobs
go run src/main.go job list --org-id=<uuid> --limit=10
go run src/main.go job list --queue=sync --state=running
go run src/main.go job cancel <job-id>
```

### Frontend Development Commands
```bash
# Install frontend dependencies
npm install

# Run frontend dev server (Vite with HMR and TypeScript)
npm run dev

# Build frontend for production (TypeScript compilation)
npm run build

# Preview production build
npm run preview

# Type check TypeScript files
npx tsc --noEmit
```

### Clean Build Artifacts
```bash
just clean
```

## Technology Stack

- **Backend**: Go with Chi router and Gonertia (Inertia.js adapter)
- **Database**: PostgreSQL via Supabase
- **Authentication**: Supabase Auth (JWT-based)
- **Real-time**: Supabase Realtime
- **Job Queue**: River (PostgreSQL-based background jobs)
- **Frontend**: Inertia.js + React + TypeScript + Tailwind CSS
- **Build Tool**: Vite for fast development and optimized production builds
- **File Storage**: Supabase Storage

## High-Level Architecture

### Core Application Flow
1. **CLI Entry** (`main.go`): Cobra-based CLI that processes flags and orchestrates the entire workflow
2. **Date Calculation** (`date.go`): Calculates analysis period based on billing cycle (default: 15th of month)
3. **SimpleFin Integration** (`simplefin.go`): Fetches financial data from SimpleFin Bridge API
4. **Database Cache** (`db.go`): BadgerDB for notification throttling and account state tracking
5. **LLM Analysis** (`llm.go`): Sends transactions to OpenRouter API for AI-powered insights
6. **Notifications** (`notifications.go`): Distributes analysis via email and/or ntfy push notifications

### Key Architectural Decisions

#### Transaction Processing Pipeline
- Fetches all transactions within date range from SimpleFin
- Filters out positive transactions (income) to focus on expenses
- Checks database cache to prevent duplicate notifications (2-day cooldown)
- Sends expense data to LLM for pattern analysis
- Distributes insights through configured notification channels

#### Billing Cycle Awareness
The application understands billing cycles (configurable, default 15th):
- Automatically switches to "last month" analysis if within 5 days of billing date
- Ensures complete billing period analysis rather than calendar months
- Handles edge cases for months with fewer days than billing date

#### Notification System
- **Email**: Rich HTML with styled tables, logo support, and responsive design
- **Ntfy**: Plain text push notifications with markdown stripped
- **API Errors**: Separate notification handling for SimpleFin API failures
- Implements cooldown period to prevent notification spam

#### Database Strategy
Uses BadgerDB (embedded key-value store) for:
- Tracking last notification timestamp per account
- Storing account balance update dates
- XDG-compliant cache directory (`~/.cache/finance-tracker`)

#### LLM Integration
- Supports multiple models via OpenRouter with random selection
- Retry logic with exponential backoff for reliability
- Structured prompt engineering for consistent financial reports
- Model attribution in generated summaries

### Multi-Tenant Architecture
- **Organizations**: Primary tenant boundary
- **Users**: Can belong to multiple organizations
- **Roles**: Owner, Admin, Member, Viewer
- **Row Level Security**: Automatic data isolation
- **Provider Abstraction**: Support for multiple financial data sources

### Background Job System (River)
- **Job Queue**: PostgreSQL-based River job queue system
- **Job Types**: 
  - Sync jobs (transactions, accounts, full sync, test connection)
  - Analysis jobs (spending analysis, trend analysis, insights)
  - Maintenance jobs (cleanup, backup)
- **Queue Management**: Multiple queues with different priorities and worker counts
  - `default`: General purpose (10 workers)
  - `sync`: Financial data synchronization (5 workers)
  - `analysis`: LLM-powered analysis (3 workers)
  - `maintenance`: Cleanup and backup tasks (2 workers)
  - `high_priority`: Urgent jobs (8 workers)
- **Features**:
  - Automatic retries with exponential backoff
  - Job scheduling and recurring jobs
  - Progress tracking and metadata
  - Organization-based access control
  - Graceful shutdown and worker management

### Development Environment
- Uses devenv.sh for consistent development setup
- Requires Go, GCC, Git, and Just
- Supabase project (local or cloud)
- Environment variables loaded from `.env` file
- See `.env.example` for required variables

## Supabase Setup

1. Create a Supabase project
2. Run migrations from `docs/SUPABASE_SETUP.md`
3. Configure environment variables
4. Set up authentication providers

## Project Structure

```
src/
├── internal/          # Internal packages
│   ├── auth/         # Authentication middleware
│   ├── config/       # Configuration including Supabase client
│   ├── models/       # Data models
│   └── services/     # Business logic
├── providers/        # Financial provider integrations
│   ├── interface.go  # Provider interface
│   └── simplefin/    # SimpleFin implementation
├── web/             # Web server
│   ├── handlers/    # HTTP handlers
│   ├── middleware/  # HTTP middleware
│   └── templates/   # HTML templates
└── main.go          # Application entry point
```

## Key Development Commands

```bash
# Set up Supabase locally (optional)
supabase start

# Run web server
go run src/main.go web

# Run River job worker (replaces old sync worker)
go run src/main.go worker

# Run migrations (in Supabase dashboard)
# Copy SQL from docs/SUPABASE_SETUP.md and River migration from supabase/migrations/
```

## Web API Endpoints

### Authentication
- `POST /auth/register` - Register new user with organization
- `POST /auth/login` - Login user  
- `POST /auth/logout` - Logout user

### Organizations
- `GET /api/v1/organizations` - List user's organizations
- `POST /api/v1/organizations` - Create new organization
- `GET /api/v1/organizations/{id}` - Get organization details
- `POST /api/v1/organizations/{id}/switch` - Switch current organization

### Members
- `GET /api/v1/organizations/{id}/members` - List organization members
- `POST /api/v1/organizations/{id}/members` - Invite member
- `PUT /api/v1/organizations/{id}/members/{userId}` - Update member role
- `DELETE /api/v1/organizations/{id}/members/{userId}` - Remove member

### Jobs (River Background Jobs)
- `POST /api/v1/connections/{id}/sync` - Create sync job for connection
- `POST /api/v1/analysis/jobs` - Create analysis job
- `POST /api/v1/maintenance/jobs` - Create maintenance job (cleanup/backup)
- `GET /api/v1/jobs` - List jobs with filtering
- `GET /api/v1/jobs/{id}` - Get job details
- `POST /api/v1/jobs/{id}/cancel` - Cancel job
- `GET /api/v1/jobs/stats` - Get job statistics
- `GET /api/v1/workers` - List workers
- `GET /api/v1/workers/stats` - Get worker statistics
- `GET /api/v1/queues` - List queues

### Health Check
- `GET /health` - API health status
- `GET /api/v1/jobs/health` - Job system health check
```