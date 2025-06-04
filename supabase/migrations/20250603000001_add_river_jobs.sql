-- Add River job queue tables
-- River requires specific table structures for job management

-- River jobs table (River will create this, but we define it for our customizations)
-- Note: River will handle the main jobs table creation through its migrations
-- We just need to add any custom columns or indexes we need

-- River job states table for tracking custom state
CREATE TABLE IF NOT EXISTS river_job_states (
    id BIGSERIAL PRIMARY KEY,
    river_job_id BIGINT NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    connection_id UUID REFERENCES provider_connections(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(river_job_id)
);

-- Index for performance
CREATE INDEX idx_river_job_states_org_id ON river_job_states(organization_id);
CREATE INDEX idx_river_job_states_connection_id ON river_job_states(connection_id);
CREATE INDEX idx_river_job_states_river_job_id ON river_job_states(river_job_id);

-- RLS policies for river_job_states
ALTER TABLE river_job_states ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can view job states for their organizations" ON river_job_states
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Service role can manage job states" ON river_job_states
    FOR ALL USING (auth.role() = 'service_role');

-- Function to get jobs with organization context
CREATE OR REPLACE FUNCTION get_organization_jobs(org_id UUID, limit_count INT DEFAULT 50, offset_count INT DEFAULT 0)
RETURNS TABLE(
    job_id BIGINT,
    kind TEXT,
    state TEXT,
    attempt INT,
    max_attempts INT,
    created_at TIMESTAMPTZ,
    scheduled_at TIMESTAMPTZ,
    attempted_at TIMESTAMPTZ,
    finalized_at TIMESTAMPTZ,
    errors JSONB,
    metadata JSONB
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        rj.id,
        rj.kind,
        rj.state::TEXT,
        rj.attempt,
        rj.max_attempts,
        rj.created_at,
        rj.scheduled_at,
        rj.attempted_at,
        rj.finalized_at,
        rj.errors,
        rj.metadata
    FROM river_job rj
    INNER JOIN river_job_states rjs ON rj.id = rjs.river_job_id
    WHERE rjs.organization_id = org_id
    ORDER BY rj.created_at DESC
    LIMIT limit_count
    OFFSET offset_count;
END;
$$;

-- Function to get job statistics for organization
CREATE OR REPLACE FUNCTION get_organization_job_stats(org_id UUID, since_date TIMESTAMPTZ DEFAULT NOW() - INTERVAL '7 days')
RETURNS TABLE(
    state TEXT,
    count BIGINT,
    avg_duration_seconds NUMERIC
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        rj.state::TEXT,
        COUNT(*) as count,
        AVG(EXTRACT(EPOCH FROM (COALESCE(rj.finalized_at, NOW()) - rj.created_at)))::NUMERIC as avg_duration_seconds
    FROM river_job rj
    INNER JOIN river_job_states rjs ON rj.id = rjs.river_job_id
    WHERE rjs.organization_id = org_id
        AND rj.created_at >= since_date
    GROUP BY rj.state
    ORDER BY count DESC;
END;
$$;