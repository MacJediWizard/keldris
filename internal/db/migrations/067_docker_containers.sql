-- Migration: Add Docker container backup configuration
-- Supports label-based backup configuration for Docker containers

-- Create docker_containers table
CREATE TABLE docker_containers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    container_id VARCHAR(64) NOT NULL,
    container_name VARCHAR(255) NOT NULL,
    image_name VARCHAR(512) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    schedule VARCHAR(50) NOT NULL DEFAULT 'daily',
    cron_expression VARCHAR(100),
    excludes JSONB,
    pre_hook TEXT,
    post_hook TEXT,
    stop_on_backup BOOLEAN NOT NULL DEFAULT false,
    backup_volumes BOOLEAN NOT NULL DEFAULT true,
    backup_bind_mounts BOOLEAN NOT NULL DEFAULT false,
    labels JSONB,
    overrides JSONB,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_backup_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_schedule CHECK (schedule IN ('hourly', 'daily', 'weekly', 'monthly', 'custom')),
    UNIQUE(agent_id, container_id)
);

-- Create indexes for efficient queries
CREATE INDEX idx_docker_containers_agent ON docker_containers(agent_id);
CREATE INDEX idx_docker_containers_enabled ON docker_containers(agent_id, enabled) WHERE enabled = true;
CREATE INDEX idx_docker_containers_container_id ON docker_containers(agent_id, container_id);
CREATE INDEX idx_docker_containers_schedule ON docker_containers(schedule);
CREATE INDEX idx_docker_containers_discovered ON docker_containers(discovered_at);

-- Add comment explaining the table
COMMENT ON TABLE docker_containers IS 'Stores backup configuration for Docker containers discovered via labels';
COMMENT ON COLUMN docker_containers.container_id IS 'Docker container ID (short or long format)';
COMMENT ON COLUMN docker_containers.schedule IS 'Backup schedule: hourly, daily, weekly, monthly, or custom';
COMMENT ON COLUMN docker_containers.cron_expression IS 'Custom cron expression when schedule is "custom"';
COMMENT ON COLUMN docker_containers.excludes IS 'JSON array of paths to exclude from backup';
COMMENT ON COLUMN docker_containers.pre_hook IS 'Command to run before backup (e.g., pg_dump)';
COMMENT ON COLUMN docker_containers.post_hook IS 'Command to run after backup completes';
COMMENT ON COLUMN docker_containers.stop_on_backup IS 'Stop container during backup for consistency';
COMMENT ON COLUMN docker_containers.backup_volumes IS 'Include Docker named volumes in backup';
COMMENT ON COLUMN docker_containers.backup_bind_mounts IS 'Include bind mounts in backup';
COMMENT ON COLUMN docker_containers.labels IS 'Original Docker labels (keldris.backup.* labels)';
COMMENT ON COLUMN docker_containers.overrides IS 'UI-configured overrides that take precedence over labels';
