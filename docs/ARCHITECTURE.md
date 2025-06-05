# Finaro Architecture

## Overview

Finaro is a multi-tenant web application that helps users track and analyze their financial transactions. It supports multiple financial data providers, AI-powered insights, and collaborative features for organizations.

## Technology Stack

### Backend
- **Language**: Go
- **Web Framework**: Chi router (lightweight, idiomatic)
- **Database**: PostgreSQL (via Supabase)
- **Authentication**: Supabase Auth (JWT-based)
- **Real-time**: Supabase Realtime (WebSockets)
- **File Storage**: Supabase Storage
- **Background Jobs**: Supabase Edge Functions or self-hosted workers

### Frontend
- **UI Framework**: Inertia.js (modern monolith approach)
- **JavaScript Framework**: React 18
- **Styling**: Tailwind CSS
- **Build Tool**: Vite (fast HMR and optimized builds)
- **Charts**: Chart.js or Recharts
- **State Management**: Inertia's built-in props and React hooks

## Core Concepts

### Multi-Tenancy Model

1. **Organizations**: The primary tenant boundary. All financial data belongs to an organization.
2. **Users**: Can belong to multiple organizations with different roles.
3. **Roles**: Owner, Admin, Member, Viewer - each with specific permissions.
4. **Provider Connections**: Each organization can have multiple financial data sources.
5. **Bank Accounts**: Linked to provider connections, represent actual bank accounts.

### Provider Abstraction

The system is designed to support multiple financial data providers through a common interface:

```go
type FinancialProvider interface {
    GetProviderType() string
    ValidateCredentials(credentials map[string]string) error
    ListAccounts(ctx context.Context, credentials map[string]string) ([]ProviderAccount, error)
    GetTransactions(ctx context.Context, credentials map[string]string, accountID string, startDate, endDate time.Time) ([]ProviderTransaction, error)
}
```

Current providers:
- SimpleFin (implemented)
- Manual Entry (planned)
- Plaid (future)

### Security Model

1. **Authentication**: Supabase Auth with JWT tokens
2. **Authorization**: Row Level Security (RLS) policies in PostgreSQL
3. **Data Isolation**: RLS ensures users only see their organization's data
4. **Encryption**: Supabase handles encryption at rest and in transit
5. **API Keys**: Separate anon and service keys for different access levels
6. **Audit Trail**: All significant actions logged with PostgreSQL triggers

## Data Flow

### Transaction Sync Flow

1. User adds provider connection with encrypted credentials
2. Background worker periodically syncs transactions
3. New transactions are categorized automatically
4. AI analysis runs on new data periods
5. Notifications sent based on user preferences

### Request Flow

1. User request → Chi router or Supabase client
2. Supabase validates JWT token
3. RLS policies automatically filter data by organization
4. Handler processes request (if backend) or direct DB query (if frontend)
5. Real-time updates via WebSocket subscriptions
6. Response rendered (HTML partial, JSON, or real-time event)

## Architecture Layers

The application follows a clean, layered architecture pattern with clear separation of concerns:

```
┌─────────────────────────────────┐
│          HTTP Layer             │  ← Web handlers, routing, middleware
├─────────────────────────────────┤
│        Use Case Layer           │  ← Business logic, orchestration
├─────────────────────────────────┤
│        Service Layer            │  ← Domain services, external APIs
├─────────────────────────────────┤
│         Data Layer              │  ← Database access, Supabase client
└─────────────────────────────────┘
```

### HTTP Layer (`src/web/`)

**Server Setup (`src/web/server.go`)**
- Centralized server configuration
- Route setup and middleware registration
- Graceful shutdown handling
- Health check endpoints

**Handler Packages (`src/web/handlers/`)**
- **Authentication Handlers** (`auth.go`): Login, register, logout
- **Organization Handlers** (`organization.go`): Org management, member operations
- **API Handlers** (`api.go`): Financial data endpoints (transactions, accounts)
- **Page Handlers** (`pages.go`): Server-side rendered pages
- **Template Renderer** (`renderer.go`): HTML template processing

**Key Features:**
- Clean separation of HTTP concerns
- Use case dependency injection
- Consistent error handling
- HTMX integration for reactive UI

### Use Case Layer (`src/internal/usecases/`)

Contains pure business logic with no HTTP or database dependencies:

**Authentication Use Case (`auth.go`)**
- User registration with organization creation
- Login with organization context
- Token validation and user context

**Organization Use Case (`organization.go`)**
- Organization CRUD operations
- Member management (invite, update roles, remove)
- Access control validation
- Organization switching logic

**Benefits:**
- Easy to unit test (no external dependencies)
- Reusable across different interfaces (HTTP, CLI, etc.)
- Clear business logic separation
- Simplified handler logic

### Service Layer (`src/internal/services/`)

Domain services that handle complex operations:
- **Organization Service**: Database operations for orgs and members
- **Sync Service**: Transaction synchronization workflows
- **Analytics Service**: Financial calculation and insights
- **Categorizer Service**: AI-powered transaction categorization
- **Notifier Service**: Multi-channel notification delivery

### Data Layer (`src/internal/config/`)

- **Supabase Client**: Database and auth integration
- **Configuration Management**: Environment-based settings
- **Connection Pooling**: Optimized database connections

## Component Interactions

### Request Flow Example (Authentication)

1. **HTTP Request** → `auth.HandleLogin()` in HTTP layer
2. **Business Logic** → `authUseCase.Login()` in Use Case layer  
3. **External API** → Supabase Auth via Service layer
4. **Database Query** → Organization lookup via Data layer
5. **Response** → JSON/HTML response back through layers

### Benefits of This Architecture

**Testability:**
- Use cases can be unit tested without HTTP or database
- Handlers can be tested with mocked use cases
- Clear dependency boundaries

**Maintainability:**
- Single responsibility per layer
- Easy to locate and modify specific functionality
- Reduced file sizes (cmd_web.go: 926 → 50 lines!)

**Scalability:**
- Easy to add new features following established patterns
- Clear separation allows for independent scaling
- Simple dependency injection

## Database Design

See [DATABASE.md](./DATABASE.md) and [SUPABASE_SETUP.md](./SUPABASE_SETUP.md) for detailed schema documentation.

Key design decisions:
- PostgreSQL with Supabase
- UUID primary keys (using uuid-ossp extension)
- JSONB columns for flexible metadata
- Row Level Security for multi-tenancy
- Materialized views for analytics
- Full-text search indexes
- Automatic updated_at triggers

## API Design

RESTful API with consistent patterns:
- Versioned endpoints (`/api/v1`)
- Resource-based URLs
- Standard HTTP methods
- JSON request/response
- Consistent error format

See [API.md](./API.md) for complete API documentation.

## Frontend Architecture

### HTMX Patterns

- Partial page updates for responsiveness
- Progressive enhancement
- Server-side rendering
- Minimal JavaScript

### Component Structure

Templates organized by feature:
- Layouts provide consistent structure
- Partials for reusable components
- Feature-specific templates
- HTMX attributes for interactivity

## Development Workflow

### Local Development

1. Set up Supabase project (local or cloud)
2. Run database migrations via Supabase dashboard
3. Configure environment variables (see `.env.example`)
4. Start web server with hot reload: `just web-dev`
5. Use devenv for consistent environment
6. Supabase Studio for database management

### Adding New Features

**Following the Layered Architecture:**

1. **Define Use Case** (`src/internal/usecases/`)
   - Create pure business logic functions
   - Define request/response types
   - Handle validation and orchestration

2. **Create HTTP Handlers** (`src/web/handlers/`)
   - Parse HTTP requests
   - Call appropriate use case
   - Return HTTP responses

3. **Update Routes** (`src/web/server.go`)
   - Add new endpoints to router
   - Apply appropriate middleware

4. **Add Services** (if needed) (`src/internal/services/`)
   - Create domain-specific operations
   - Handle external API integrations

**Example: Adding Budget Feature**
```go
// 1. Use case
func (uc *BudgetUseCase) CreateBudget(ctx context.Context, req CreateBudgetRequest) (*Budget, error)

// 2. HTTP handler  
func (h *BudgetHandlers) HandleCreateBudget(w http.ResponseWriter, r *http.Request)

// 3. Route registration
r.Post("/budgets", budgetHandlers.HandleCreateBudget)
```

### Testing Strategy

**Unit Testing:**
- **Use Cases**: Pure business logic, easily mockable dependencies
- **Services**: Domain operations with mocked external dependencies
- **Utilities**: Helper functions and data transformations

**Integration Testing:**
- **HTTP Handlers**: Test complete request/response cycles
- **Database Operations**: Test against real/test database
- **External APIs**: Test provider integrations

**Testing Structure:**
```
tests/
├── unit/
│   ├── usecases/     # Business logic tests
│   └── services/     # Service layer tests
├── integration/
│   ├── handlers/     # HTTP handler tests  
│   └── providers/    # External API tests
└── fixtures/         # Test data and mocks
```

**Testing Benefits from Architecture:**
- Use cases are pure functions → easy unit testing
- HTTP layer can be tested with mocked use cases
- Clear dependency injection makes mocking simple
- Each layer can be tested independently

### Deployment

- Single binary deployment (Go backend)
- Embedded static files
- Supabase handles database, auth, and storage
- Environment-based configuration
- Health check endpoints
- Edge Functions for serverless compute (optional)

## Future Considerations

### Scalability

- PostgreSQL auto-scaling with Supabase
- Built-in connection pooling
- Read replicas available
- Horizontal scaling for Go backend
- Edge Functions for distributed compute
- Real-time subscriptions scale automatically

### Features

- Budget management
- Bill tracking
- Investment tracking
- Tax reporting
- Mobile app

### Integrations

- More financial providers
- Accounting software
- Export formats
- Webhooks