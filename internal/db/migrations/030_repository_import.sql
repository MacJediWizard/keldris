-- Migration: Add repository import tracking

-- Add columns to repositories table to track imported repositories
ALTER TABLE repositories ADD COLUMN imported BOOLEAN DEFAULT false;
ALTER TABLE repositories ADD COLUMN imported_at TIMESTAMPTZ;
ALTER TABLE repositories ADD COLUMN original_snapshot_count INTEGER;

-- Create table to track imported snapshots metadata
-- This allows us to track snapshots that existed before import
CREATE TABLE imported_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    restic_snapshot_id VARCHAR(255) NOT NULL,
    short_id VARCHAR(64) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    username VARCHAR(255),
    snapshot_time TIMESTAMPTZ NOT NULL,
    paths JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags JSONB DEFAULT '[]'::jsonb,
    imported_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(repository_id, restic_snapshot_id)
);

CREATE INDEX idx_imported_snapshots_repository ON imported_snapshots(repository_id);
CREATE INDEX idx_imported_snapshots_agent ON imported_snapshots(agent_id);
CREATE INDEX idx_imported_snapshots_hostname ON imported_snapshots(hostname);
CREATE INDEX idx_imported_snapshots_time ON imported_snapshots(snapshot_time DESC);
CREATE INDEX idx_repositories_imported ON repositories(imported) WHERE imported = true;
