-- Migration: Add job queue table for managing backups, restores, and verifications

-- Job queue table stores all jobs with priority ordering and retry support
CREATE TABLE job_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    job_type VARCHAR(50) NOT NULL, -- backup, restore, verification
    priority INT NOT NULL DEFAULT 0, -- Higher values = higher priority
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, dead_letter

    -- Payload contains job-specific data (schedule_id, repo_id, snapshot_id, etc.)
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Retry configuration
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ,

    -- Error tracking
    error_message TEXT,
    last_error_at TIMESTAMPTZ,

    -- Timing
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Reference to related entity (optional, for quick lookups)
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    repository_id UUID REFERENCES repositories(id) ON DELETE SET NULL,
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL
);

-- Indexes for efficient queue operations
CREATE INDEX idx_job_queue_org ON job_queue(org_id);
CREATE INDEX idx_job_queue_status ON job_queue(status);
CREATE INDEX idx_job_queue_type ON job_queue(job_type);
CREATE INDEX idx_job_queue_priority ON job_queue(priority DESC, created_at ASC);

-- Index for fetching pending jobs by priority
CREATE INDEX idx_job_queue_pending ON job_queue(org_id, status, priority DESC, created_at ASC)
    WHERE status = 'pending';

-- Index for fetching jobs ready for retry
CREATE INDEX idx_job_queue_retry ON job_queue(next_retry_at)
    WHERE status = 'failed' AND next_retry_at IS NOT NULL;

-- Index for dead letter queue
CREATE INDEX idx_job_queue_dead_letter ON job_queue(org_id, created_at DESC)
    WHERE status = 'dead_letter';

-- Index for running jobs
CREATE INDEX idx_job_queue_running ON job_queue(org_id, started_at DESC)
    WHERE status = 'running';

-- Indexes for related entity lookups
CREATE INDEX idx_job_queue_agent ON job_queue(agent_id) WHERE agent_id IS NOT NULL;
CREATE INDEX idx_job_queue_repository ON job_queue(repository_id) WHERE repository_id IS NOT NULL;
CREATE INDEX idx_job_queue_schedule ON job_queue(schedule_id) WHERE schedule_id IS NOT NULL;

-- Index for cleanup queries
CREATE INDEX idx_job_queue_completed_at ON job_queue(completed_at)
    WHERE status IN ('completed', 'dead_letter');
