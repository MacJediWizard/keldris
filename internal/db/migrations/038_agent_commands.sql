-- Migration: Add agent commands queue

-- Create agent_commands table for push commands to agents
CREATE TABLE agent_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    payload JSONB,
    result JSONB,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    timeout_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fetching pending commands for an agent (used by agent polling)
CREATE INDEX idx_agent_commands_agent_pending ON agent_commands(agent_id, status)
    WHERE status = 'pending';

-- Index for fetching commands by agent for history
CREATE INDEX idx_agent_commands_agent_created ON agent_commands(agent_id, created_at DESC);

-- Index for org-level command management
CREATE INDEX idx_agent_commands_org ON agent_commands(org_id);

-- Index for finding timed out commands
CREATE INDEX idx_agent_commands_timeout ON agent_commands(timeout_at)
    WHERE status IN ('pending', 'acknowledged', 'running');

-- Index for command status filtering
CREATE INDEX idx_agent_commands_status ON agent_commands(status);

-- Comment on table
COMMENT ON TABLE agent_commands IS 'Queue of commands pushed from server to agents';

-- Add check constraint for valid command types
ALTER TABLE agent_commands ADD CONSTRAINT agent_commands_type_check
    CHECK (type IN ('backup_now', 'update', 'restart', 'diagnostics'));

-- Add check constraint for valid status values
ALTER TABLE agent_commands ADD CONSTRAINT agent_commands_status_check
    CHECK (status IN ('pending', 'acknowledged', 'running', 'completed', 'failed', 'timed_out', 'canceled'));
