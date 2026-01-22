-- Migration: Add snapshot mounts for FUSE filesystem access

-- Create table to track mounted snapshots
CREATE TABLE snapshot_mounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    mount_path VARCHAR(1024) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    mounted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    unmounted_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_snapshot_mounts_org ON snapshot_mounts(org_id);
CREATE INDEX idx_snapshot_mounts_agent ON snapshot_mounts(agent_id);
CREATE INDEX idx_snapshot_mounts_status ON snapshot_mounts(status);
CREATE INDEX idx_snapshot_mounts_snapshot ON snapshot_mounts(snapshot_id);
CREATE INDEX idx_snapshot_mounts_expires ON snapshot_mounts(expires_at) WHERE status = 'mounted';

-- Ensure only one active mount per snapshot per agent
CREATE UNIQUE INDEX idx_snapshot_mounts_active ON snapshot_mounts(agent_id, snapshot_id)
    WHERE status IN ('pending', 'mounting', 'mounted');
