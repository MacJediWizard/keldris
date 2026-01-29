-- Migration: Add Docker container and volume restore support

-- Create docker_restores table to track Docker restore operations
CREATE TABLE docker_restores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    container_name VARCHAR(255),
    volume_name VARCHAR(255),
    new_container_name VARCHAR(255),
    new_volume_name VARCHAR(255),
    target JSONB, -- Docker host target configuration
    overwrite_existing BOOLEAN NOT NULL DEFAULT false,
    start_after_restore BOOLEAN NOT NULL DEFAULT true,
    verify_start BOOLEAN NOT NULL DEFAULT true,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, preparing, restoring_volumes, creating_container, starting, verifying, completed, failed, canceled
    progress JSONB, -- Restore progress tracking
    restored_container_id VARCHAR(255),
    restored_volumes JSONB, -- Array of restored volume names
    start_verified BOOLEAN NOT NULL DEFAULT false,
    warnings JSONB, -- Array of warning messages
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX idx_docker_restores_org ON docker_restores(org_id);
CREATE INDEX idx_docker_restores_agent ON docker_restores(agent_id);
CREATE INDEX idx_docker_restores_repository ON docker_restores(repository_id);
CREATE INDEX idx_docker_restores_status ON docker_restores(org_id, status);
CREATE INDEX idx_docker_restores_created ON docker_restores(org_id, created_at DESC);
CREATE INDEX idx_docker_restores_active ON docker_restores(org_id, status) WHERE status NOT IN ('completed', 'failed', 'canceled');

-- Add comment explaining the table
COMMENT ON TABLE docker_restores IS 'Tracks Docker container and volume restore operations from backup snapshots';
COMMENT ON COLUMN docker_restores.target IS 'Docker host target configuration (local or remote with TLS settings)';
COMMENT ON COLUMN docker_restores.progress IS 'Real-time progress tracking during restore operation';
COMMENT ON COLUMN docker_restores.restored_volumes IS 'Array of volume names that were restored';
COMMENT ON COLUMN docker_restores.warnings IS 'Array of warning messages generated during restore';
