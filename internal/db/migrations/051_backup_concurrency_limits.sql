-- Migration: Add backup concurrency limits for organizations and agents
-- This migration adds fields to limit simultaneous backups per org/agent

-- Add max_concurrent_backups to organizations
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS max_concurrent_backups INTEGER;

-- Add max_concurrent_backups to agents
ALTER TABLE agents ADD COLUMN IF NOT EXISTS max_concurrent_backups INTEGER;

-- Create backup queue table to track queued backups when limits are reached
CREATE TABLE IF NOT EXISTS backup_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    priority INTEGER DEFAULT 0, -- Higher values = higher priority
    queued_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'queued', -- queued, started, canceled
    queue_position INTEGER, -- Calculated position in queue
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient queue queries
CREATE INDEX IF NOT EXISTS idx_backup_queue_org_status ON backup_queue(org_id, status) WHERE status = 'queued';
CREATE INDEX IF NOT EXISTS idx_backup_queue_agent_status ON backup_queue(agent_id, status) WHERE status = 'queued';
CREATE INDEX IF NOT EXISTS idx_backup_queue_queued_at ON backup_queue(queued_at) WHERE status = 'queued';

-- Function to get queue position for an org
CREATE OR REPLACE FUNCTION get_backup_queue_position(p_org_id UUID, p_queue_id UUID) RETURNS INTEGER AS $$
DECLARE
    position INTEGER;
BEGIN
    SELECT COUNT(*) + 1 INTO position
    FROM backup_queue
    WHERE org_id = p_org_id
      AND status = 'queued'
      AND (priority > (SELECT priority FROM backup_queue WHERE id = p_queue_id)
           OR (priority = (SELECT priority FROM backup_queue WHERE id = p_queue_id)
               AND queued_at < (SELECT queued_at FROM backup_queue WHERE id = p_queue_id)));
    RETURN position;
END;
$$ LANGUAGE plpgsql;

-- Comment for documentation
COMMENT ON COLUMN organizations.max_concurrent_backups IS 'Maximum number of concurrent backups allowed for this organization. NULL means unlimited.';
COMMENT ON COLUMN agents.max_concurrent_backups IS 'Maximum number of concurrent backups allowed for this agent. NULL means use organization default.';
COMMENT ON TABLE backup_queue IS 'Queue for backups waiting when concurrency limits are reached.';
