-- Migration: Add agent logs table for centralized log collection
-- This migration adds tables for storing and querying agent logs

-- Agent logs table
CREATE TABLE agent_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    level VARCHAR(10) NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error')),
    message TEXT NOT NULL,
    component VARCHAR(100),
    metadata JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_agent_logs_agent_id ON agent_logs(agent_id);
CREATE INDEX idx_agent_logs_org_id ON agent_logs(org_id);
CREATE INDEX idx_agent_logs_level ON agent_logs(level);
CREATE INDEX idx_agent_logs_timestamp ON agent_logs(timestamp DESC);
CREATE INDEX idx_agent_logs_agent_timestamp ON agent_logs(agent_id, timestamp DESC);
CREATE INDEX idx_agent_logs_component ON agent_logs(component) WHERE component IS NOT NULL;

-- Full-text search index for log messages
CREATE INDEX idx_agent_logs_message_search ON agent_logs USING gin(to_tsvector('english', message));

-- Function to clean up old log records (retention: 7 days by default)
CREATE OR REPLACE FUNCTION cleanup_agent_logs(retention_days INTEGER DEFAULT 7) RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM agent_logs
    WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create a partial index for recent logs (last 24 hours) for faster real-time queries
CREATE INDEX idx_agent_logs_recent ON agent_logs(agent_id, timestamp DESC)
WHERE timestamp > NOW() - INTERVAL '24 hours';
