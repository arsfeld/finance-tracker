-- Fix infinite recursion in organization_members RLS policies
-- The issue is that the policies are querying the same table they're protecting

-- Drop all existing organization_members policies
DROP POLICY IF EXISTS "Users can view organization members" ON organization_members;
DROP POLICY IF EXISTS "Users can insert themselves as members" ON organization_members;
DROP POLICY IF EXISTS "Admins can insert other members" ON organization_members;
DROP POLICY IF EXISTS "Admins can update member roles" ON organization_members;
DROP POLICY IF EXISTS "Admins can delete members" ON organization_members;
DROP POLICY IF EXISTS "Members can view organization members" ON organization_members;
DROP POLICY IF EXISTS "Admins can manage members" ON organization_members;

-- Create a simple function to check if user is admin of an organization
-- This bypasses the recursion issue by using a direct query
CREATE OR REPLACE FUNCTION is_organization_admin(org_id uuid, user_uuid uuid DEFAULT auth.uid())
RETURNS boolean AS $$
BEGIN
  -- Use a simple EXISTS query that doesn't trigger RLS
  RETURN EXISTS (
    SELECT 1 
    FROM organization_members 
    WHERE organization_id = org_id 
    AND user_id = user_uuid 
    AND role IN ('owner', 'admin')
  );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create a function to check if user is a member of an organization
CREATE OR REPLACE FUNCTION is_organization_member(org_id uuid, user_uuid uuid DEFAULT auth.uid())
RETURNS boolean AS $$
BEGIN
  RETURN EXISTS (
    SELECT 1 
    FROM organization_members 
    WHERE organization_id = org_id 
    AND user_id = user_uuid
  );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create new policies that don't cause recursion
-- These policies use the security definer functions to bypass RLS

-- Allow users to see all members in organizations they belong to
CREATE POLICY "Members can view organization members" ON organization_members
  FOR SELECT USING (
    -- Users can see members of organizations they belong to
    is_organization_member(organization_id, auth.uid())
  );

-- Allow users to join organizations (insert themselves)
CREATE POLICY "Users can join organizations" ON organization_members
  FOR INSERT WITH CHECK (
    user_id = auth.uid()
  );

-- Allow admins to add other users to their organizations
CREATE POLICY "Admins can add members" ON organization_members
  FOR INSERT WITH CHECK (
    is_organization_admin(organization_id, auth.uid())
    OR user_id = auth.uid() -- Allow self-insertion
  );

-- Allow admins to update member roles
CREATE POLICY "Admins can update member roles" ON organization_members
  FOR UPDATE USING (
    is_organization_admin(organization_id, auth.uid())
  );

-- Allow admins to remove members, and users to remove themselves
CREATE POLICY "Admins can remove members" ON organization_members
  FOR DELETE USING (
    is_organization_admin(organization_id, auth.uid())
    OR user_id = auth.uid() -- Allow users to remove themselves
  );

-- Grant execute permission on the helper functions
GRANT EXECUTE ON FUNCTION is_organization_admin(uuid, uuid) TO authenticated;
GRANT EXECUTE ON FUNCTION is_organization_member(uuid, uuid) TO authenticated;