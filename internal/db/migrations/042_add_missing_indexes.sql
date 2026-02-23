-- Migration: Add missing indexes for performance

CREATE INDEX IF NOT EXISTS idx_repositories_org_id ON repositories(org_id);
CREATE INDEX IF NOT EXISTS idx_schedules_agent_id ON schedules(agent_id);
