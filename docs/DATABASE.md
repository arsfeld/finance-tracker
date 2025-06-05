# Database Schema Documentation

## Overview

Finaro uses SQLite as its primary database. The schema is designed for multi-tenancy, data integrity, and query performance.

## Design Principles

1. **Multi-tenancy**: Every table includes organization_id for data isolation
2. **Soft Deletes**: Use is_active flags instead of hard deletes where appropriate
3. **Audit Trail**: created_at/updated_at on all tables
4. **UUIDs**: For better distribution and avoiding ID conflicts
5. **JSON Flexibility**: Metadata fields for extensibility without schema changes

## Core Tables

### organizations
Primary tenant boundary. All users belong to one or more organizations.

```sql
CREATE TABLE organizations (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
    settings JSON DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Settings JSON structure:
```json
{
  "billing_cycle_day": 15,
  "default_currency": "USD",
  "fiscal_year_start": "01-01",
  "timezone": "America/New_York"
}
```

### users
System users who can belong to multiple organizations.

```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    settings JSON DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Settings JSON structure:
```json
{
  "theme": "light",
  "language": "en",
  "date_format": "MM/DD/YYYY",
  "currency_display": "symbol"
}
```

### organization_members
Junction table for user-organization relationships with roles.

```sql
CREATE TABLE organization_members (
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, user_id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

Role permissions:
- **owner**: Full access, can delete organization
- **admin**: Full access except organization deletion
- **member**: Read/write access to financial data
- **viewer**: Read-only access

### provider_connections
Stores encrypted credentials for financial data providers.

```sql
CREATE TABLE provider_connections (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    organization_id TEXT NOT NULL,
    provider_type TEXT NOT NULL,
    name TEXT NOT NULL,
    credentials_encrypted TEXT NOT NULL,
    settings JSON DEFAULT '{}',
    last_sync TIMESTAMP,
    sync_status TEXT DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);
```

Provider types: 'simplefin', 'plaid', 'manual'

### bank_accounts
Represents actual bank accounts from providers.

```sql
CREATE TABLE bank_accounts (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    organization_id TEXT NOT NULL,
    connection_id TEXT NOT NULL,
    provider_account_id TEXT NOT NULL,
    name TEXT NOT NULL,
    institution TEXT,
    account_type TEXT,
    balance REAL,
    currency TEXT DEFAULT 'USD',
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSON DEFAULT '{}',
    last_sync TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (connection_id) REFERENCES provider_connections(id) ON DELETE CASCADE,
    UNIQUE(connection_id, provider_account_id)
);
```

Account types: 'checking', 'savings', 'credit', 'investment', 'loan'

### transactions
Financial transactions from all accounts.

```sql
CREATE TABLE transactions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    organization_id TEXT NOT NULL,
    bank_account_id TEXT NOT NULL,
    provider_transaction_id TEXT,
    amount REAL NOT NULL,
    description TEXT,
    merchant_name TEXT,
    category_id INTEGER,
    date DATE NOT NULL,
    posted_date DATE,
    pending BOOLEAN DEFAULT FALSE,
    transaction_type TEXT,
    metadata JSON DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (bank_account_id) REFERENCES bank_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id),
    UNIQUE(bank_account_id, provider_transaction_id)
);
```

Transaction types: 'debit', 'credit', 'transfer'

### categories
Hierarchical transaction categories per organization.

```sql
CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    parent_id INTEGER,
    color TEXT,
    icon TEXT,
    rules JSON DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES categories(id),
    UNIQUE(organization_id, name)
);
```

Rules JSON structure:
```json
[
  {
    "field": "merchant_name",
    "operator": "contains",
    "value": "grocery",
    "case_sensitive": false
  }
]
```

### ai_analyses
Stores AI-generated financial analyses.

```sql
CREATE TABLE ai_analyses (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    organization_id TEXT NOT NULL,
    date_start DATE NOT NULL,
    date_end DATE NOT NULL,
    analysis_type TEXT NOT NULL,
    content TEXT NOT NULL,
    model TEXT,
    metadata JSON DEFAULT '{}',
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);
```

Analysis types: 'monthly', 'quarterly', 'yearly', 'custom'

### chat_sessions
AI chat sessions for financial Q&A.

```sql
CREATE TABLE chat_sessions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    title TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### chat_messages
Individual messages within chat sessions.

```sql
CREATE TABLE chat_messages (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    metadata JSON DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
);
```

Metadata JSON structure:
```json
{
  "referenced_transactions": ["tx_id1", "tx_id2"],
  "date_range": {
    "start": "2024-01-01",
    "end": "2024-01-31"
  },
  "categories": ["groceries", "dining"]
}
```

### notification_settings
Per-user, per-organization notification preferences.

```sql
CREATE TABLE notification_settings (
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    settings JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, user_id, channel),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

Channels: 'email', 'ntfy', 'webhook'

### sessions
Web session storage.

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    data JSON NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### audit_logs
Comprehensive audit trail for compliance and debugging.

```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    metadata JSON DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## Indexes

```sql
-- Performance indexes
CREATE INDEX idx_transactions_date ON transactions(organization_id, date);
CREATE INDEX idx_transactions_account ON transactions(bank_account_id, date);
CREATE INDEX idx_transactions_category ON transactions(category_id);
CREATE INDEX idx_bank_accounts_org ON bank_accounts(organization_id);
CREATE INDEX idx_org_members_user ON organization_members(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_audit_logs_org_date ON audit_logs(organization_id, created_at);

-- Full-text search
CREATE VIRTUAL TABLE transactions_fts USING fts5(
    transaction_id,
    description,
    merchant_name,
    content=transactions
);
```

## Migration Strategy

1. Migrations stored in `src/store/sqlite/migrations/`
2. Sequential numbering: `001_initial_schema.sql`, `002_add_categories.sql`
3. Up and down migrations for rollback support
4. Migration table tracks applied migrations
5. Automatic migration on startup

## Backup Strategy

1. Daily automated SQLite backups
2. Point-in-time recovery support
3. Export to SQL for portability
4. Encrypted backup storage

## Performance Considerations

1. **Indexes**: Strategic indexes on foreign keys and common query patterns
2. **Pagination**: Limit/offset for large result sets
3. **Aggregations**: Materialized views for complex analytics
4. **Vacuum**: Regular VACUUM for space reclamation
5. **WAL Mode**: Write-Ahead Logging for concurrency