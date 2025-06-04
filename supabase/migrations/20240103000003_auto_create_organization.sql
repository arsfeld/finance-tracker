-- Auto-create organization when user signs up

-- Function to create organization for new user
CREATE OR REPLACE FUNCTION auto_create_user_organization()
RETURNS TRIGGER AS $$
DECLARE
    new_org_id UUID;
    org_name TEXT;
BEGIN
    -- Generate organization name from email
    org_name := NEW.email || '''s Organization';
    
    -- Create organization
    INSERT INTO organizations (name)
    VALUES (org_name)
    RETURNING id INTO new_org_id;
    
    -- Add user as owner
    INSERT INTO organization_members (organization_id, user_id, role)
    VALUES (new_org_id, NEW.id, 'owner');
    
    -- Create default categories
    INSERT INTO categories (organization_id, name, color, icon) VALUES
    (new_org_id, 'Food & Dining', '#FF6B6B', 'utensils'),
    (new_org_id, 'Transportation', '#4ECDC4', 'car'),
    (new_org_id, 'Shopping', '#45B7D1', 'shopping-bag'),
    (new_org_id, 'Entertainment', '#96CEB4', 'film'),
    (new_org_id, 'Bills & Utilities', '#FECA57', 'file-text'),
    (new_org_id, 'Healthcare', '#FF9FF3', 'heart'),
    (new_org_id, 'Education', '#54A0FF', 'book'),
    (new_org_id, 'Travel', '#48DBFB', 'plane'),
    (new_org_id, 'Other', '#A0A0A0', 'dots-horizontal');
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create trigger on auth.users insert
-- Note: This might not work on managed Supabase due to auth schema restrictions
-- Alternative: Use a webhook or Edge Function

-- Instead, let's create a function that can be called manually
-- and hook it into the registration process
DROP FUNCTION IF EXISTS create_organization_with_owner;

CREATE OR REPLACE FUNCTION create_organization_with_owner(
    user_email TEXT,
    owner_id UUID
) RETURNS organizations AS $$
DECLARE
    new_org organizations;
    org_name TEXT;
BEGIN
    -- Generate organization name from email
    org_name := user_email || '''s Organization';
    
    -- Create organization
    INSERT INTO organizations (name)
    VALUES (org_name)
    RETURNING * INTO new_org;
    
    -- Add user as owner
    INSERT INTO organization_members (organization_id, user_id, role)
    VALUES (new_org.id, owner_id, 'owner');
    
    -- Create default categories
    INSERT INTO categories (organization_id, name, color, icon) VALUES
    (new_org.id, 'Food & Dining', '#FF6B6B', 'utensils'),
    (new_org.id, 'Transportation', '#4ECDC4', 'car'),
    (new_org.id, 'Shopping', '#45B7D1', 'shopping-bag'),
    (new_org.id, 'Entertainment', '#96CEB4', 'film'),
    (new_org.id, 'Bills & Utilities', '#FECA57', 'file-text'),
    (new_org.id, 'Healthcare', '#FF9FF3', 'heart'),
    (new_org.id, 'Education', '#54A0FF', 'book'),
    (new_org.id, 'Travel', '#48DBFB', 'plane'),
    (new_org.id, 'Other', '#A0A0A0', 'dots-horizontal');
    
    RETURN new_org;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Grant permissions
GRANT EXECUTE ON FUNCTION create_organization_with_owner TO authenticated;

-- Test the function
SELECT 'Organization auto-creation function created successfully' as message;