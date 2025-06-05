-- River job queue tables
-- This migration adds the necessary tables for River job queue system

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- River jobs table
CREATE TABLE river_job (
    id bigserial PRIMARY KEY,
    args bytea NOT NULL,
    attempt smallint NOT NULL DEFAULT 0,
    attempted_at timestamptz,
    attempted_by text[],
    created_at timestamptz NOT NULL DEFAULT NOW(),
    errors text[],
    finalized_at timestamptz,
    kind text NOT NULL,
    max_attempts smallint NOT NULL,
    metadata jsonb,
    priority smallint NOT NULL DEFAULT 1,
    queue text NOT NULL DEFAULT 'default',
    state text NOT NULL DEFAULT 'available',
    scheduled_at timestamptz NOT NULL DEFAULT NOW(),
    tags text[],
    unique_key bytea,
    CONSTRAINT river_job_unique_key UNIQUE (unique_key) DEFERRABLE INITIALLY DEFERRED
);

-- Indexes for job queries
CREATE INDEX river_job_state_and_finalized_at_index ON river_job (state, finalized_at) WHERE finalized_at IS NOT NULL;
CREATE INDEX river_job_priority_and_scheduled_at_index ON river_job (priority, scheduled_at, id);
CREATE INDEX river_job_state_and_scheduled_at_index ON river_job (state, scheduled_at) WHERE state IN ('available', 'retryable');
CREATE INDEX river_job_args_index ON river_job USING hash (args);
CREATE INDEX river_job_metadata_index ON river_job USING gin (metadata);
CREATE INDEX river_job_kind_index ON river_job (kind);
CREATE INDEX river_job_queue_index ON river_job (queue);
CREATE INDEX river_job_created_at_index ON river_job (created_at);

-- River leader table for leader election
CREATE TABLE river_leader (
    elected_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    leader_id text NOT NULL,
    name text PRIMARY KEY
);

-- River migration table
CREATE TABLE river_migration (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    version bigint NOT NULL UNIQUE
);

-- Insert initial migration version
INSERT INTO river_migration (version) VALUES (1);

-- River client table for tracking connected clients
CREATE TABLE river_client (
    id text PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    metadata jsonb
);

-- Add organization-specific job tracking
CREATE TABLE river_job_organization (
    job_id bigint REFERENCES river_job(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, organization_id)
);

CREATE INDEX river_job_organization_org_index ON river_job_organization (organization_id, created_at DESC);

-- Add functions for job management

-- Function to get jobs for an organization
CREATE OR REPLACE FUNCTION get_jobs_for_organization(
    org_id uuid,
    job_limit int DEFAULT 50,
    job_offset int DEFAULT 0,
    job_queue text DEFAULT NULL,
    job_state text DEFAULT NULL,
    job_kind text DEFAULT NULL
)
RETURNS TABLE (
    id bigint,
    args bytea,
    attempt smallint,
    attempted_at timestamptz,
    attempted_by text[],
    created_at timestamptz,
    errors text[],
    finalized_at timestamptz,
    kind text,
    max_attempts smallint,
    metadata jsonb,
    priority smallint,
    queue text,
    state text,
    scheduled_at timestamptz,
    tags text[],
    unique_key bytea
) 
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY 
    SELECT 
        rj.id, rj.args, rj.attempt, rj.attempted_at, rj.attempted_by,
        rj.created_at, rj.errors, rj.finalized_at, rj.kind, rj.max_attempts,
        rj.metadata, rj.priority, rj.queue, rj.state, rj.scheduled_at,
        rj.tags, rj.unique_key
    FROM river_job rj
    INNER JOIN river_job_organization rjo ON rj.id = rjo.job_id
    WHERE rjo.organization_id = org_id
        AND (job_queue IS NULL OR rj.queue = job_queue)
        AND (job_state IS NULL OR rj.state = job_state)  
        AND (job_kind IS NULL OR rj.kind = job_kind)
    ORDER BY rj.created_at DESC
    LIMIT job_limit OFFSET job_offset;
END;
$$;

-- Function to get job statistics for an organization
CREATE OR REPLACE FUNCTION get_job_stats_for_organization(
    org_id uuid,
    since_date timestamptz DEFAULT NOW() - INTERVAL '7 days'
)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    result jsonb;
BEGIN
    SELECT jsonb_build_object(
        'total', COUNT(*),
        'by_state', jsonb_object_agg(
            state, 
            jsonb_build_object('count', state_count)
        ),
        'by_queue', jsonb_object_agg(
            queue,
            jsonb_build_object('count', queue_count)  
        ),
        'by_kind', jsonb_object_agg(
            kind,
            jsonb_build_object('count', kind_count)
        )
    ) INTO result
    FROM (
        SELECT 
            rj.state,
            COUNT(*) as state_count,
            rj.queue,
            COUNT(*) as queue_count,
            rj.kind,
            COUNT(*) as kind_count
        FROM river_job rj
        INNER JOIN river_job_organization rjo ON rj.id = rjo.job_id  
        WHERE rjo.organization_id = org_id
            AND rj.created_at >= since_date
        GROUP BY GROUPING SETS ((rj.state), (rj.queue), (rj.kind))
    ) stats;
    
    RETURN COALESCE(result, '{}'::jsonb);
END;
$$;

-- Function to automatically link jobs to organizations based on job args
CREATE OR REPLACE FUNCTION link_job_to_organization()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    org_id uuid;
    args_json jsonb;
BEGIN
    -- Try to extract organization_id from job args
    BEGIN
        -- Convert bytea args to text and then to jsonb
        args_json := convert_from(NEW.args, 'UTF8')::jsonb;
        
        -- Extract organization_id
        org_id := (args_json->>'organization_id')::uuid;
        
        -- Insert into linking table if we found an org_id
        IF org_id IS NOT NULL THEN
            INSERT INTO river_job_organization (job_id, organization_id)
            VALUES (NEW.id, org_id)
            ON CONFLICT DO NOTHING;
        END IF;
    EXCEPTION 
        WHEN OTHERS THEN
            -- If we can't parse the args, just continue
            NULL;
    END;
    
    RETURN NEW;
END;
$$;

-- Trigger to automatically link jobs to organizations
CREATE TRIGGER link_job_to_organization_trigger
    AFTER INSERT ON river_job
    FOR EACH ROW
    EXECUTE FUNCTION link_job_to_organization();

-- Row Level Security policies
ALTER TABLE river_job ENABLE ROW LEVEL SECURITY;
ALTER TABLE river_job_organization ENABLE ROW LEVEL SECURITY;

-- Policy for river_job table - only allow access through organization linking
CREATE POLICY "Jobs are accessible through organization membership" ON river_job
    USING (
        id IN (
            SELECT job_id 
            FROM river_job_organization rjo
            INNER JOIN organization_members om ON rjo.organization_id = om.organization_id
            WHERE om.user_id = auth.uid()
        )
    );

-- Policy for river_job_organization table
CREATE POLICY "Job organization links are accessible by organization members" ON river_job_organization
    USING (
        organization_id IN (
            SELECT organization_id 
            FROM organization_members 
            WHERE user_id = auth.uid()
        )
    );

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON river_job TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON river_job_organization TO authenticated;
GRANT USAGE ON SEQUENCE river_job_id_seq TO authenticated;

-- Comments for documentation
COMMENT ON TABLE river_job IS 'River job queue table for background job processing';
COMMENT ON TABLE river_job_organization IS 'Links jobs to organizations for access control';
COMMENT ON FUNCTION get_jobs_for_organization IS 'Retrieves jobs for a specific organization with filtering';
COMMENT ON FUNCTION get_job_stats_for_organization IS 'Gets job statistics for an organization';
COMMENT ON FUNCTION link_job_to_organization IS 'Automatically links jobs to organizations based on job arguments';