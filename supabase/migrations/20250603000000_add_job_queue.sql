-- Add job queue system for background sync workers
-- Created: 2025-06-03

-- Job status enum
CREATE TYPE job_status AS ENUM (
    'pending',
    'running', 
    'completed',
    'failed',
    'cancelled',
    'paused'
);

-- Job priority enum
CREATE TYPE job_priority AS ENUM (
    'low',
    'normal', 
    'high',
    'urgent'
);

-- Job type enum
CREATE TYPE job_type AS ENUM (
    'sync_transactions',
    'sync_accounts',
    'full_sync',
    'test_connection'
);

-- Jobs table for queue management
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type job_type NOT NULL,
    status job_status NOT NULL DEFAULT 'pending',
    priority job_priority NOT NULL DEFAULT 'normal',
    
    -- Job metadata
    title TEXT NOT NULL,
    description TEXT,
    
    -- Target data (what to sync)
    provider_connection_id UUID REFERENCES provider_connections(id) ON DELETE CASCADE,
    bank_account_id UUID REFERENCES bank_accounts(id) ON DELETE SET NULL,
    
    -- Job parameters (JSON payload)
    parameters JSONB DEFAULT '{}',
    
    -- Progress tracking
    progress_current INTEGER DEFAULT 0,
    progress_total INTEGER DEFAULT 100,
    progress_message TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Worker info
    worker_id TEXT,
    
    -- Results and errors
    result JSONB,
    error_message TEXT,
    error_details JSONB,
    
    -- Retry logic
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    retry_delay_seconds INTEGER DEFAULT 60,
    
    CONSTRAINT valid_progress CHECK (
        progress_current >= 0 AND 
        progress_total > 0 AND 
        progress_current <= progress_total
    )
);

-- Worker registry for tracking active workers
CREATE TABLE job_workers (
    id TEXT PRIMARY KEY, -- worker instance ID
    hostname TEXT NOT NULL,
    pid INTEGER NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'active', -- active, paused, stopping
    
    -- Worker configuration
    max_concurrent_jobs INTEGER DEFAULT 1,
    current_job_count INTEGER DEFAULT 0,
    
    -- Worker metadata
    version TEXT,
    worker_type TEXT DEFAULT 'sync',
    
    CONSTRAINT valid_job_count CHECK (current_job_count >= 0)
);

-- Job queue configuration per organization
CREATE TABLE job_queue_settings (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    
    -- Concurrency limits
    max_concurrent_jobs INTEGER DEFAULT 2,
    max_retries INTEGER DEFAULT 3,
    retry_delay_seconds INTEGER DEFAULT 60,
    
    -- Auto-sync settings
    auto_sync_enabled BOOLEAN DEFAULT true,
    auto_sync_interval_minutes INTEGER DEFAULT 60,
    auto_sync_quiet_hours_start TIME,
    auto_sync_quiet_hours_end TIME,
    
    -- Rate limiting
    max_jobs_per_hour INTEGER DEFAULT 10,
    
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job execution history for audit/analytics
CREATE TABLE job_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    worker_id TEXT REFERENCES job_workers(id) ON DELETE SET NULL,
    
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    status job_status NOT NULL,
    
    duration_seconds INTEGER,
    error_message TEXT,
    
    -- Metrics
    items_processed INTEGER DEFAULT 0,
    items_total INTEGER DEFAULT 0
);

-- Add indexes for performance
CREATE INDEX idx_jobs_organization_status ON jobs(organization_id, status);
CREATE INDEX idx_jobs_status_priority_scheduled ON jobs(status, priority DESC, scheduled_at ASC);
CREATE INDEX idx_jobs_provider_connection ON jobs(provider_connection_id);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_job_workers_heartbeat ON job_workers(last_heartbeat DESC);
CREATE INDEX idx_job_executions_job_id ON job_executions(job_id);

-- RLS policies
ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_workers ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_queue_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_executions ENABLE ROW LEVEL SECURITY;

-- Jobs policies - users can only see jobs for their organizations
CREATE POLICY "Users can view jobs for their organizations" ON jobs
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Users can create jobs for their organizations" ON jobs
    FOR INSERT WITH CHECK (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid()
            AND role IN ('owner', 'admin', 'member')
        )
    );

CREATE POLICY "Users can update jobs for their organizations" ON jobs
    FOR UPDATE USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid()
            AND role IN ('owner', 'admin')
        )
    );

-- Job workers policies - service role access for workers
CREATE POLICY "Service role can manage workers" ON job_workers
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Users can view workers" ON job_workers
    FOR SELECT USING (true);

-- Job queue settings policies
CREATE POLICY "Users can view settings for their organizations" ON job_queue_settings
    FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid()
        )
    );

CREATE POLICY "Admins can update settings for their organizations" ON job_queue_settings
    FOR ALL USING (
        organization_id IN (
            SELECT organization_id FROM organization_members 
            WHERE user_id = auth.uid()
            AND role IN ('owner', 'admin')
        )
    );

-- Job executions policies
CREATE POLICY "Users can view executions for their organization jobs" ON job_executions
    FOR SELECT USING (
        job_id IN (
            SELECT id FROM jobs 
            WHERE organization_id IN (
                SELECT organization_id FROM organization_members 
                WHERE user_id = auth.uid()
            )
        )
    );

-- Functions for job management

-- Function to get next job for worker
CREATE OR REPLACE FUNCTION get_next_job(worker_id_param TEXT)
RETURNS TABLE(
    job_id UUID,
    job_type job_type,
    parameters JSONB,
    provider_connection_id UUID,
    bank_account_id UUID
) 
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    selected_job_id UUID;
BEGIN
    -- Get next available job with proper locking
    SELECT id INTO selected_job_id
    FROM jobs
    WHERE status = 'pending'
        AND scheduled_at <= NOW()
    ORDER BY priority DESC, scheduled_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1;
    
    IF selected_job_id IS NOT NULL THEN
        -- Update job status and worker assignment
        UPDATE jobs 
        SET 
            status = 'running',
            started_at = NOW(),
            worker_id = worker_id_param
        WHERE id = selected_job_id;
        
        -- Return job details
        RETURN QUERY
        SELECT 
            j.id,
            j.type,
            j.parameters,
            j.provider_connection_id,
            j.bank_account_id
        FROM jobs j
        WHERE j.id = selected_job_id;
    END IF;
END;
$$;

-- Function to update job progress
CREATE OR REPLACE FUNCTION update_job_progress(
    job_id_param UUID,
    current_param INTEGER,
    total_param INTEGER,
    message_param TEXT DEFAULT NULL
)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    UPDATE jobs
    SET 
        progress_current = current_param,
        progress_total = total_param,
        progress_message = COALESCE(message_param, progress_message)
    WHERE id = job_id_param
        AND status = 'running';
    
    RETURN FOUND;
END;
$$;

-- Function to complete job
CREATE OR REPLACE FUNCTION complete_job(
    job_id_param UUID,
    status_param job_status,
    result_param JSONB DEFAULT NULL,
    error_message_param TEXT DEFAULT NULL
)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    UPDATE jobs
    SET 
        status = status_param,
        completed_at = NOW(),
        result = result_param,
        error_message = error_message_param,
        progress_current = progress_total
    WHERE id = job_id_param
        AND status = 'running';
    
    -- Record execution history
    INSERT INTO job_executions (
        job_id,
        worker_id,
        started_at,
        completed_at,
        status,
        duration_seconds,
        error_message
    )
    SELECT 
        id,
        worker_id,
        started_at,
        completed_at,
        status,
        EXTRACT(EPOCH FROM (completed_at - started_at))::INTEGER,
        error_message
    FROM jobs
    WHERE id = job_id_param;
    
    RETURN FOUND;
END;
$$;

-- Function to register/update worker heartbeat
CREATE OR REPLACE FUNCTION update_worker_heartbeat(
    worker_id_param TEXT,
    hostname_param TEXT,
    pid_param INTEGER,
    max_concurrent_param INTEGER DEFAULT 1
)
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    INSERT INTO job_workers (
        id,
        hostname,
        pid,
        max_concurrent_jobs,
        last_heartbeat
    )
    VALUES (
        worker_id_param,
        hostname_param,
        pid_param,
        max_concurrent_param,
        NOW()
    )
    ON CONFLICT (id)
    DO UPDATE SET
        hostname = EXCLUDED.hostname,
        pid = EXCLUDED.pid,
        max_concurrent_jobs = EXCLUDED.max_concurrent_jobs,
        last_heartbeat = NOW();
END;
$$;

-- Create default settings for existing organizations
INSERT INTO job_queue_settings (organization_id)
SELECT id FROM organizations
ON CONFLICT DO NOTHING;