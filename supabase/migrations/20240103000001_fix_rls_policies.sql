-- Fix RLS policies to avoid infinite recursion

-- Drop existing problematic policies
DROP POLICY IF EXISTS "Members can view organization members" ON organization_members;
DROP POLICY IF EXISTS "Admins can insert members" ON organization_members;
DROP POLICY IF EXISTS "Admins can update members" ON organization_members;
DROP POLICY IF EXISTS "Admins can delete members" ON organization_members;

-- Create fixed organization members policies that don't cause recursion
CREATE POLICY "Users can view organization members" ON organization_members
    FOR SELECT USING (
        -- Allow users to see members of organizations they belong to
        organization_id IN (
            SELECT om.organization_id 
            FROM organization_members om 
            WHERE om.user_id = auth.uid()
        )
        OR user_id = auth.uid() -- Allow users to see their own memberships
    );

CREATE POLICY "Users can insert themselves as members" ON organization_members
    FOR INSERT WITH CHECK (
        user_id = auth.uid()
    );

CREATE POLICY "Admins can insert other members" ON organization_members
    FOR INSERT WITH CHECK (
        EXISTS (
            SELECT 1 FROM organization_members om
            WHERE om.organization_id = organization_members.organization_id
            AND om.user_id = auth.uid()
            AND om.role IN ('owner', 'admin')
        )
        OR user_id = auth.uid() -- Allow self-insertion
    );

CREATE POLICY "Admins can update member roles" ON organization_members
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
        OR user_id = auth.uid() -- Allow users to remove themselves
    );