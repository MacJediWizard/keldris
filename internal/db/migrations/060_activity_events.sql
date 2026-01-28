-- Migration: Add activity events table for real-time activity feed

-- Create activity_events table
CREATE TABLE activity_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    category VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    -- User who triggered the event (if applicable)
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_name VARCHAR(255),
    -- Agent related to the event (if applicable)
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    agent_name VARCHAR(255),
    -- Generic resource reference
    resource_type VARCHAR(50),
    resource_id UUID,
    resource_name VARCHAR(255),
    -- Additional metadata as JSON
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX idx_activity_events_org ON activity_events(org_id);
CREATE INDEX idx_activity_events_org_created ON activity_events(org_id, created_at DESC);
CREATE INDEX idx_activity_events_category ON activity_events(org_id, category);
CREATE INDEX idx_activity_events_type ON activity_events(org_id, type);
CREATE INDEX idx_activity_events_user ON activity_events(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_activity_events_agent ON activity_events(agent_id) WHERE agent_id IS NOT NULL;

-- Add comment for table documentation
COMMENT ON TABLE activity_events IS 'Stores system activity events for the real-time activity feed';
COMMENT ON COLUMN activity_events.type IS 'Event type (e.g., backup_started, user_login, alert_triggered)';
COMMENT ON COLUMN activity_events.category IS 'Event category for filtering (e.g., backup, user, alert)';
