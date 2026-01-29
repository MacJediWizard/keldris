-- Docker Health Monitoring Tables
-- Migration: 059_docker_health.sql

-- Agent Docker Health table stores Docker health state for each agent
CREATE TABLE IF NOT EXISTS agent_docker_health (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    docker_health JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(agent_id)
);

-- Container Restart Events table tracks container restart history
CREATE TABLE IF NOT EXISTS container_restart_events (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    container_name TEXT NOT NULL,
    restart_count INTEGER NOT NULL DEFAULT 0,
    exit_code INTEGER,
    reason TEXT,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for agent_docker_health
CREATE INDEX IF NOT EXISTS idx_agent_docker_health_org_id ON agent_docker_health(org_id);
CREATE INDEX IF NOT EXISTS idx_agent_docker_health_agent_id ON agent_docker_health(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_docker_health_updated_at ON agent_docker_health(updated_at);

-- Indexes for container_restart_events
CREATE INDEX IF NOT EXISTS idx_container_restart_events_org_id ON container_restart_events(org_id);
CREATE INDEX IF NOT EXISTS idx_container_restart_events_agent_id ON container_restart_events(agent_id);
CREATE INDEX IF NOT EXISTS idx_container_restart_events_container_id ON container_restart_events(container_id);
CREATE INDEX IF NOT EXISTS idx_container_restart_events_occurred_at ON container_restart_events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_container_restart_events_agent_container ON container_restart_events(agent_id, container_id);

-- Comments
COMMENT ON TABLE agent_docker_health IS 'Stores Docker health state for each agent';
COMMENT ON COLUMN agent_docker_health.docker_health IS 'JSON containing Docker daemon info, container list, and volume info';
COMMENT ON TABLE container_restart_events IS 'Tracks container restart events for restart loop detection';
COMMENT ON COLUMN container_restart_events.exit_code IS 'Exit code of the container when it restarted';
COMMENT ON COLUMN container_restart_events.reason IS 'Reason for the restart if available';
