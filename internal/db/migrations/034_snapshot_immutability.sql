-- Migration: Add snapshot immutability for compliance

-- Add immutability settings to repositories table
ALTER TABLE repositories ADD COLUMN immutability_enabled BOOLEAN DEFAULT false;
ALTER TABLE repositories ADD COLUMN default_immutability_days INTEGER;

-- Create table to track snapshot immutability locks
-- This allows us to prevent deletion of snapshots until the lock expires
CREATE TABLE snapshot_immutability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    short_id VARCHAR(64) NOT NULL,
    locked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until TIMESTAMPTZ NOT NULL,
    locked_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reason VARCHAR(500),
    s3_object_lock_enabled BOOLEAN DEFAULT false,
    s3_object_lock_mode VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(repository_id, snapshot_id)
);

-- Create indexes for efficient querying
CREATE INDEX idx_snapshot_immutability_repo ON snapshot_immutability(repository_id);
CREATE INDEX idx_snapshot_immutability_org ON snapshot_immutability(org_id);
CREATE INDEX idx_snapshot_immutability_snapshot ON snapshot_immutability(snapshot_id);
CREATE INDEX idx_snapshot_immutability_locked_until ON snapshot_immutability(locked_until);
CREATE INDEX idx_snapshot_immutability_active ON snapshot_immutability(locked_until) WHERE locked_until > NOW();
