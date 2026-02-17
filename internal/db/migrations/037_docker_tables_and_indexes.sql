-- Migration: Add Docker state tables, docker_backups table, schedule Docker columns, and missing indexes

-- Docker container state reported by agents
CREATE TABLE docker_containers (
    container_id VARCHAR(64) NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(512) NOT NULL,
    status VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL,
    created VARCHAR(100),
    ports JSONB,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (org_id, agent_id, container_id)
);

CREATE INDEX idx_docker_containers_agent ON docker_containers(org_id, agent_id);

-- Docker volume state reported by agents
CREATE TABLE docker_volumes (
    name VARCHAR(255) NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    driver VARCHAR(100) NOT NULL DEFAULT 'local',
    mountpoint TEXT,
    size_bytes BIGINT DEFAULT 0,
    created VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (org_id, agent_id, name)
);

CREATE INDEX idx_docker_volumes_agent ON docker_volumes(org_id, agent_id);

-- Docker daemon status per agent
CREATE TABLE docker_daemon_status (
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    available BOOLEAN NOT NULL DEFAULT false,
    version VARCHAR(50),
    container_count INTEGER DEFAULT 0,
    volume_count INTEGER DEFAULT 0,
    server_os VARCHAR(100),
    docker_root_dir TEXT,
    storage_driver VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (org_id, agent_id)
);

-- Docker backup jobs
CREATE TABLE docker_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    container_ids JSONB,
    volume_names JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_docker_backups_org ON docker_backups(org_id);
CREATE INDEX idx_docker_backups_agent ON docker_backups(org_id, agent_id);

-- Add Docker columns to schedules
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS docker_volumes JSONB;
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS docker_pause_containers BOOLEAN DEFAULT false;

-- Add missing performance indexes
CREATE INDEX IF NOT EXISTS idx_backups_agent_id ON backups(agent_id);
CREATE INDEX IF NOT EXISTS idx_backups_created_at ON backups(created_at);
CREATE INDEX IF NOT EXISTS idx_users_org_id ON users(org_id);
