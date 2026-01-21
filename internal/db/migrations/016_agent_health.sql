-- Migration: Add agent health monitoring tables
-- This migration adds tables for tracking agent health metrics and history

-- Add health status and metrics columns to agents table
ALTER TABLE agents ADD COLUMN IF NOT EXISTS health_status VARCHAR(20) DEFAULT 'unknown';
ALTER TABLE agents ADD COLUMN IF NOT EXISTS health_metrics JSONB;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS health_checked_at TIMESTAMPTZ;

-- Agent health history table for tracking metrics over time
CREATE TABLE agent_health_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    health_status VARCHAR(20) NOT NULL, -- healthy, warning, critical, unknown
    cpu_usage DECIMAL(5, 2),
    memory_usage DECIMAL(5, 2),
    disk_usage DECIMAL(5, 2),
    disk_free_bytes BIGINT,
    disk_total_bytes BIGINT,
    network_up BOOLEAN DEFAULT true,
    restic_version VARCHAR(50),
    restic_available BOOLEAN DEFAULT false,
    issues JSONB, -- Array of health issues
    recorded_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_agent_health_history_agent ON agent_health_history(agent_id);
CREATE INDEX idx_agent_health_history_org ON agent_health_history(org_id);
CREATE INDEX idx_agent_health_history_recorded_at ON agent_health_history(recorded_at);
CREATE INDEX idx_agent_health_history_agent_recorded ON agent_health_history(agent_id, recorded_at DESC);
CREATE INDEX idx_agent_health_history_status ON agent_health_history(health_status);

-- Fleet health summary view
CREATE OR REPLACE VIEW fleet_health_summary AS
SELECT
    org_id,
    COUNT(*) as total_agents,
    COUNT(*) FILTER (WHERE health_status = 'healthy') as healthy_count,
    COUNT(*) FILTER (WHERE health_status = 'warning') as warning_count,
    COUNT(*) FILTER (WHERE health_status = 'critical') as critical_count,
    COUNT(*) FILTER (WHERE health_status = 'unknown' OR health_status IS NULL) as unknown_count,
    COUNT(*) FILTER (WHERE status = 'active') as active_count,
    COUNT(*) FILTER (WHERE status = 'offline') as offline_count,
    AVG(CASE WHEN health_metrics->>'cpu_usage' IS NOT NULL
        THEN (health_metrics->>'cpu_usage')::DECIMAL ELSE NULL END) as avg_cpu_usage,
    AVG(CASE WHEN health_metrics->>'memory_usage' IS NOT NULL
        THEN (health_metrics->>'memory_usage')::DECIMAL ELSE NULL END) as avg_memory_usage,
    AVG(CASE WHEN health_metrics->>'disk_usage' IS NOT NULL
        THEN (health_metrics->>'disk_usage')::DECIMAL ELSE NULL END) as avg_disk_usage
FROM agents
GROUP BY org_id;

-- Add new alert type for health status changes
INSERT INTO alert_rules (id, org_id, name, type, enabled, config, created_at, updated_at)
SELECT
    gen_random_uuid(),
    o.id,
    'Agent Health Critical',
    'agent_health',
    true,
    '{"health_status": "critical"}'::jsonb,
    NOW(),
    NOW()
FROM organizations o
WHERE NOT EXISTS (
    SELECT 1 FROM alert_rules ar
    WHERE ar.org_id = o.id AND ar.type = 'agent_health'
);

-- Function to clean up old health history records (retention: 30 days)
CREATE OR REPLACE FUNCTION cleanup_agent_health_history() RETURNS void AS $$
BEGIN
    DELETE FROM agent_health_history
    WHERE recorded_at < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- Add notification event type for agent health
DO $$
BEGIN
    -- Add agent_health_critical event type if notification_preferences exists
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'notification_preferences') THEN
        -- This is handled by the application, no schema changes needed
        NULL;
    END IF;
END $$;
