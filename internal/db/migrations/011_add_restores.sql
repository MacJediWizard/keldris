-- Migration: Add restores table for tracking restore operations

CREATE TABLE restores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    target_path VARCHAR(1024) NOT NULL,
    include_paths JSONB DEFAULT '[]'::jsonb,
    exclude_paths JSONB DEFAULT '[]'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_restores_agent ON restores(agent_id);
CREATE INDEX idx_restores_repository ON restores(repository_id);
CREATE INDEX idx_restores_snapshot ON restores(snapshot_id);
CREATE INDEX idx_restores_status ON restores(status);
CREATE INDEX idx_restores_created_at ON restores(created_at DESC);

-- Add index for snapshot_id lookups on backups table
CREATE INDEX idx_backups_snapshot_id ON backups(snapshot_id) WHERE snapshot_id IS NOT NULL;
