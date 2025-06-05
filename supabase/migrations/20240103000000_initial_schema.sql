-- Finaro Initial Schema Migration

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create custom types
CREATE TYPE organization_role AS ENUM ('owner', 'admin', 'member', 'viewer');
CREATE TYPE provider_type AS ENUM ('simplefin', 'plaid', 'manual');
CREATE TYPE sync_status AS ENUM ('idle', 'syncing', 'success', 'error');

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    settings JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organization members junction table
CREATE TABLE organization_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role organization_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, user_id)
);

-- Financial provider connections
CREATE TABLE provider_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_type provider_type NOT NULL,
    name TEXT NOT NULL,
    credentials_encrypted TEXT NOT NULL,
    settings JSONB DEFAULT '{}'::jsonb,
    last_sync TIMESTAMPTZ,
    sync_status sync_status DEFAULT 'idle',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
    metadata JSONB DEFAULT '{}'::jsonb,
    last_sync TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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
    rules JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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
    metadata JSONB DEFAULT '{}'::jsonb,
    created_by UUID REFERENCES auth.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notification settings
CREATE TABLE notification_settings (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    settings JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, user_id, channel)
);

-- Create indexes for performance
CREATE INDEX idx_organization_members_user_id ON organization_members(user_id);
CREATE INDEX idx_provider_connections_organization_id ON provider_connections(organization_id);
CREATE INDEX idx_bank_accounts_organization_id ON bank_accounts(organization_id);
CREATE INDEX idx_bank_accounts_connection_id ON bank_accounts(connection_id);
CREATE INDEX idx_transactions_organization_id ON transactions(organization_id);
CREATE INDEX idx_transactions_bank_account_id ON transactions(bank_account_id);
CREATE INDEX idx_transactions_date ON transactions(organization_id, date DESC);
CREATE INDEX idx_transactions_amount ON transactions(amount);
CREATE INDEX idx_transactions_category ON transactions(category_id);
CREATE INDEX idx_categories_organization_id ON categories(organization_id);
CREATE INDEX idx_ai_analyses_organization_id ON ai_analyses(organization_id);

-- Full-text search on transactions
CREATE INDEX idx_transactions_search ON transactions 
    USING gin(to_tsvector('english', COALESCE(description, '') || ' ' || COALESCE(merchant_name, '')));

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update triggers
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_provider_connections_updated_at BEFORE UPDATE ON provider_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bank_accounts_updated_at BEFORE UPDATE ON bank_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_settings_updated_at BEFORE UPDATE ON notification_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Enable Row Level Security (RLS) on all tables
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE organization_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE provider_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE bank_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_analyses ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_settings ENABLE ROW LEVEL SECURITY;

-- Organizations policies
CREATE POLICY "Users can view their organizations" ON organizations
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = organizations.id
            AND organization_members.user_id = auth.uid()
        )
    );

CREATE POLICY "Owners can update organizations" ON organizations
    FOR UPDATE USING (
        EXISTS (
            SELECT 1 FROM organization_members 
            WHERE organization_members.organization_id = organizations.id
            AND organization_members.user_id = auth.uid()
            AND organization_members.role = 'owner'
        )
    );

-- Organization members policies
CREATE POLICY "Members can view organization members" ON organization_members
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members om
            WHERE om.organization_id = organization_members.organization_id
            AND om.user_id = auth.uid()
        )
    );

CREATE POLICY "Admins can insert members" ON organization_members
    FOR INSERT WITH CHECK (
        EXISTS (
            SELECT 1 FROM organization_members om
            WHERE om.organization_id = organization_members.organization_id
            AND om.user_id = auth.uid()
            AND om.role IN ('owner', 'admin')
        )
    );

CREATE POLICY "Admins can update members" ON organization_members
    FOR UPDATE USING (
        EXISTS (
            SELECT 1 FROM organization_members om
            WHERE om.organization_id = organization_members.organization_id
            AND om.user_id = auth.uid()
            AND om.role IN ('owner', 'admin')
        )
    );

CREATE POLICY "Admins can delete members" ON organization_members
    FOR DELETE USING (
        EXISTS (
            SELECT 1 FROM organization_members om
            WHERE om.organization_id = organization_members.organization_id
            AND om.user_id = auth.uid()
            AND om.role IN ('owner', 'admin')
        )
    );

-- Provider connections policies
CREATE POLICY "Members can view connections" ON provider_connections
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = provider_connections.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

CREATE POLICY "Admins can manage connections" ON provider_connections
    FOR ALL USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = provider_connections.organization_id
            AND organization_members.user_id = auth.uid()
            AND organization_members.role IN ('owner', 'admin')
        )
    );

-- Bank accounts policies
CREATE POLICY "Members can view accounts" ON bank_accounts
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = bank_accounts.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

-- Transactions policies
CREATE POLICY "Members can view transactions" ON transactions
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = transactions.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

CREATE POLICY "Members can update transactions" ON transactions
    FOR UPDATE USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = transactions.organization_id
            AND organization_members.user_id = auth.uid()
            AND organization_members.role IN ('owner', 'admin', 'member')
        )
    );

-- Categories policies
CREATE POLICY "Members can view categories" ON categories
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = categories.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

CREATE POLICY "Members can manage categories" ON categories
    FOR ALL USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = categories.organization_id
            AND organization_members.user_id = auth.uid()
            AND organization_members.role IN ('owner', 'admin', 'member')
        )
    );

-- AI analyses policies
CREATE POLICY "Members can view analyses" ON ai_analyses
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = ai_analyses.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

-- Notification settings policies
CREATE POLICY "Users can view their notification settings" ON notification_settings
    FOR SELECT USING (user_id = auth.uid());

CREATE POLICY "Users can update their notification settings" ON notification_settings
    FOR ALL USING (user_id = auth.uid());

-- Create helper function to create organization with owner
CREATE OR REPLACE FUNCTION create_organization_with_owner(
    org_name TEXT,
    owner_id UUID
) RETURNS organizations AS $$
DECLARE
    new_org organizations;
BEGIN
    -- Create the organization
    INSERT INTO organizations (name)
    VALUES (org_name)
    RETURNING * INTO new_org;
    
    -- Add the owner as a member
    INSERT INTO organization_members (organization_id, user_id, role)
    VALUES (new_org.id, owner_id, 'owner');
    
    -- Create default categories
    PERFORM create_default_categories(new_org.id);
    
    RETURN new_org;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

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
RETURNS organization_role AS $$
BEGIN
    RETURN (
        SELECT role FROM organization_members 
        WHERE organization_id = org_id AND user_id = user_uuid
    );
END;
$$ LANGUAGE plpgsql;

-- Grant execute permissions
GRANT EXECUTE ON FUNCTION create_organization_with_owner TO authenticated;
GRANT EXECUTE ON FUNCTION create_default_categories TO authenticated;
GRANT EXECUTE ON FUNCTION get_user_role TO authenticated;

-- Create useful views
CREATE VIEW user_organizations AS
SELECT 
    o.*,
    om.role,
    om.joined_at
FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = auth.uid();

-- Grant permissions on views
GRANT SELECT ON user_organizations TO authenticated;