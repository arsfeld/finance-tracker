-- Create trigger to automatically create organization when user signs up
-- Based on Supabase docs: https://supabase.com/docs/guides/auth/managing-user-data

-- First, create a function that will be called by the trigger
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER SET search_path = public
AS $$
DECLARE
    new_org_id UUID;
    org_name TEXT;
BEGIN
    -- Generate organization name from email
    org_name := NEW.email || '''s Organization';
    
    -- Create organization
    INSERT INTO public.organizations (name)
    VALUES (org_name)
    RETURNING id INTO new_org_id;
    
    -- Add user as owner
    INSERT INTO public.organization_members (organization_id, user_id, role)
    VALUES (new_org_id, NEW.id, 'owner');
    
    -- Create default categories
    INSERT INTO public.categories (organization_id, name, color, icon) VALUES
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
EXCEPTION
    WHEN OTHERS THEN
        -- Log the error but don't fail the user creation
        RAISE LOG 'Error creating organization for user %: %', NEW.id, SQLERRM;
        RETURN NEW;
END;
$$;

-- Create the trigger on auth.users
DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();

-- Grant necessary permissions
GRANT USAGE ON SCHEMA public TO postgres, anon, authenticated, service_role;
GRANT ALL ON ALL TABLES IN SCHEMA public TO postgres, anon, authenticated, service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO postgres, anon, authenticated, service_role;

-- Test message
SELECT 'User trigger created successfully - organizations will be auto-created on signup' as message;