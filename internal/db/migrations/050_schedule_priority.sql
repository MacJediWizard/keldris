-- Migration: Add priority and preemption support to schedules

ALTER TABLE schedules
ADD COLUMN priority INT DEFAULT 2,
ADD COLUMN preemptible BOOLEAN DEFAULT false;

-- Add constraint to validate priority values (1=high, 2=medium, 3=low)
ALTER TABLE schedules
ADD CONSTRAINT schedules_priority_check CHECK (priority >= 1 AND priority <= 3);

-- Index for finding schedules by priority (useful for queue ordering)
CREATE INDEX idx_schedules_priority ON schedules(agent_id, priority, enabled)
WHERE enabled = true;

-- Create backup queue table for managing pending backups
CREATE TABLE backup_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    priority INT NOT NULL DEFAULT 2,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    preempted_by UUID REFERENCES backup_queue(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT backup_queue_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'preempted', 'canceled'))
);

-- Index for finding pending backups by priority
CREATE INDEX idx_backup_queue_pending ON backup_queue(agent_id, priority, queued_at)
WHERE status = 'pending';

-- Index for finding running backups per agent
CREATE INDEX idx_backup_queue_running ON backup_queue(agent_id, status)
WHERE status = 'running';

-- Comment on columns
COMMENT ON COLUMN schedules.priority IS 'Backup priority: 1=high, 2=medium, 3=low';
COMMENT ON COLUMN schedules.preemptible IS 'If true, backup can be preempted by higher priority backups';
COMMENT ON TABLE backup_queue IS 'Queue for managing pending and running backups with priority ordering';
