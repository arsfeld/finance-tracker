# Supabase Setup Guide

This guide walks through setting up Supabase for the WalletMind application.

## Prerequisites

1. Create a Supabase account at https://supabase.com
2. Create a new project
3. Save your project URL and keys

## Database Schema

Run these SQL commands in the Supabase SQL editor:

### 1. Create Tables

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Organization members (extends Supabase auth.users)
CREATE TABLE organization_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (organization_id, user_id)
);

-- Financial provider connections
CREATE TABLE provider_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL,
    name TEXT NOT NULL,
    credentials_encrypted TEXT NOT NULL,
    settings JSONB DEFAULT '{}',
    last_sync TIMESTAMPTZ,
    sync_status TEXT DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Bank accounts
CREATE TABLE bank_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES provider_connections(id) ON DELETE CASCADE,
    provider_account_id TEXT NOT NULL,
    name TEXT NOT NULL,
    institution TEXT,
    account_type TEXT,
    balance DECIMAL(19,4),
    currency TEXT DEFAULT 'USD',
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    last_sync TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(connection_id, provider_account_id)
);

-- Transactions
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    bank_account_id UUID NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
    provider_transaction_id TEXT,
    amount DECIMAL(19,4) NOT NULL,
    description TEXT,
    merchant_name TEXT,
    category_id INTEGER,
    date DATE NOT NULL,
    posted_date DATE,
    pending BOOLEAN DEFAULT FALSE,
    transaction_type TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(bank_account_id, provider_transaction_id)
);

-- Categories
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    parent_id INTEGER REFERENCES categories(id),
    color TEXT,
    icon TEXT,
    rules JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(organization_id, name)
);

-- AI analyses
CREATE TABLE ai_analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    date_start DATE NOT NULL,
    date_end DATE NOT NULL,
    analysis_type TEXT NOT NULL,
    content TEXT NOT NULL,
    model TEXT,
    metadata JSONB DEFAULT '{}',
    created_by UUID REFERENCES auth.users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chat sessions
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chat messages
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification settings
CREATE TABLE notification_settings (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    settings JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (organization_id, user_id, channel)
);

-- Audit logs
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id),
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 2. Create Indexes

```sql
-- Performance indexes
CREATE INDEX idx_transactions_date ON transactions(organization_id, date);
CREATE INDEX idx_transactions_account ON transactions(bank_account_id, date);
CREATE INDEX idx_transactions_category ON transactions(category_id);
CREATE INDEX idx_bank_accounts_org ON bank_accounts(organization_id);
CREATE INDEX idx_org_members_user ON organization_members(user_id);
CREATE INDEX idx_audit_logs_org_date ON audit_logs(organization_id, created_at);

-- Full-text search on transactions
CREATE INDEX idx_transactions_search ON transactions 
    USING gin(to_tsvector('english', COALESCE(description, '') || ' ' || COALESCE(merchant_name, '')));
```

### 3. Create Updated At Triggers

```sql
-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to tables
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_provider_connections_updated_at BEFORE UPDATE ON provider_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bank_accounts_updated_at BEFORE UPDATE ON bank_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chat_sessions_updated_at BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_settings_updated_at BEFORE UPDATE ON notification_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 4. Row Level Security (RLS)

```sql
-- Enable RLS on all tables
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE organization_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE provider_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE bank_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_analyses ENABLE ROW LEVEL SECURITY;
ALTER TABLE chat_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE chat_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- Organizations policies
CREATE POLICY "Users can view their organizations" ON organizations
    FOR SELECT USING (
        id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Owners can update organizations" ON organizations
    FOR UPDATE USING (
        id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid() AND role = 'owner'
        )
    );

-- Organization members policies
CREATE POLICY "Members can view organization members" ON organization_members
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Admins can manage members" ON organization_members
    FOR ALL USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid() AND role IN ('owner', 'admin')
        )
    );

-- Provider connections policies
CREATE POLICY "Members can view connections" ON provider_connections
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Admins can manage connections" ON provider_connections
    FOR ALL USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid() AND role IN ('owner', 'admin')
        )
    );

-- Bank accounts policies
CREATE POLICY "Members can view accounts" ON bank_accounts
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );

-- Transactions policies
CREATE POLICY "Members can view transactions" ON transactions
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Members can update transactions" ON transactions
    FOR UPDATE USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid() AND role IN ('owner', 'admin', 'member')
        )
    );

-- Categories policies
CREATE POLICY "Members can view categories" ON categories
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Members can manage categories" ON categories
    FOR ALL USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid() AND role IN ('owner', 'admin', 'member')
        )
    );

-- AI analyses policies
CREATE POLICY "Members can view analyses" ON ai_analyses
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );

-- Chat policies
CREATE POLICY "Users can view their chat sessions" ON chat_sessions
    FOR SELECT USING (user_id = auth.uid());

CREATE POLICY "Users can create chat sessions" ON chat_sessions
    FOR INSERT WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can view messages in their sessions" ON chat_messages
    FOR SELECT USING (
        session_id IN (
            SELECT id FROM chat_sessions WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Users can create messages in their sessions" ON chat_messages
    FOR INSERT WITH CHECK (
        session_id IN (
            SELECT id FROM chat_sessions WHERE user_id = auth.uid()
        )
    );

-- Notification settings policies
CREATE POLICY "Users can view their notification settings" ON notification_settings
    FOR SELECT USING (user_id = auth.uid());

CREATE POLICY "Users can update their notification settings" ON notification_settings
    FOR ALL USING (user_id = auth.uid());

-- Audit logs policies
CREATE POLICY "Members can view audit logs" ON audit_logs
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members WHERE user_id = auth.uid()
        )
    );
```

### 5. Create Views

```sql
-- Account summary view
CREATE VIEW account_summary AS
SELECT 
    ba.id,
    ba.organization_id,
    ba.name,
    ba.institution,
    ba.account_type,
    ba.balance,
    ba.currency,
    ba.last_sync,
    COUNT(t.id) as transaction_count,
    MAX(t.date) as last_transaction_date
FROM bank_accounts ba
LEFT JOIN transactions t ON ba.id = t.bank_account_id
GROUP BY ba.id;

-- Monthly spending view
CREATE MATERIALIZED VIEW monthly_spending AS
SELECT 
    organization_id,
    date_trunc('month', date) as month,
    category_id,
    SUM(CASE WHEN amount < 0 THEN amount ELSE 0 END) as expenses,
    SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END) as income,
    COUNT(*) as transaction_count
FROM transactions
GROUP BY organization_id, month, category_id;

-- Create index on materialized view
CREATE INDEX idx_monthly_spending ON monthly_spending(organization_id, month);
```

### 6. Create Functions

```sql
-- Function to create default categories for new organizations
CREATE OR REPLACE FUNCTION create_default_categories(org_id UUID)
RETURNS void AS $$
BEGIN
    INSERT INTO categories (organization_id, name, color, icon) VALUES
    (org_id, 'Food & Dining', '#FF6B6B', 'utensils'),
    (org_id, 'Transportation', '#4ECDC4', 'car'),
    (org_id, 'Shopping', '#45B7D1', 'shopping-bag'),
    (org_id, 'Entertainment', '#96CEB4', 'film'),
    (org_id, 'Bills & Utilities', '#FECA57', 'file-text'),
    (org_id, 'Healthcare', '#FF9FF3', 'heart'),
    (org_id, 'Education', '#54A0FF', 'book'),
    (org_id, 'Travel', '#48DBFB', 'plane'),
    (org_id, 'Other', '#A0A0A0', 'dots-horizontal');
END;
$$ LANGUAGE plpgsql;

-- Function to get user's role in organization
CREATE OR REPLACE FUNCTION get_user_role(org_id UUID, user_uuid UUID)
RETURNS TEXT AS $$
BEGIN
    RETURN (
        SELECT role FROM organization_members 
        WHERE organization_id = org_id AND user_id = user_uuid
    );
END;
$$ LANGUAGE plpgsql;
```

## Supabase Configuration

### 1. Authentication Settings

In Supabase Dashboard > Authentication > Providers:

1. Enable Email provider
2. Configure email templates
3. Set redirect URLs for your application
4. Enable any OAuth providers you want (Google, GitHub, etc.)

### 2. Storage Buckets

Create storage buckets for file uploads:

```sql
-- In SQL editor
INSERT INTO storage.buckets (id, name, public) VALUES
    ('exports', 'exports', false),
    ('receipts', 'receipts', false);
```

### 3. Edge Functions (Optional)

For background jobs and webhooks, create Edge Functions:

```bash
# Create sync function
supabase functions new sync-transactions

# Create webhook handler
supabase functions new webhook-handler
```

## Environment Variables

Add to your `.env` file:

```bash
# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_KEY=your-service-key

# Keep existing vars for LLM, notifications, etc.
```

## Next Steps

1. Set up Supabase client in Go
2. Implement provider abstraction
3. Create web interface
4. Set up real-time subscriptions