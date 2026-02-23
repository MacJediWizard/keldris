-- Docker container log backup tables

-- Docker log backup records
CREATE TABLE docker_log_backups (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    container_name TEXT NOT NULL,
    image_name TEXT,
    log_path TEXT,
    original_size BIGINT DEFAULT 0,
    compressed_size BIGINT DEFAULT 0,
    compressed BOOLEAN DEFAULT false,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    line_count BIGINT DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    backup_schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for querying backups by agent
CREATE INDEX idx_docker_log_backups_agent_id ON docker_log_backups(agent_id);

-- Index for querying backups by container
CREATE INDEX idx_docker_log_backups_container_id ON docker_log_backups(agent_id, container_id);

-- Index for querying backups by status
CREATE INDEX idx_docker_log_backups_status ON docker_log_backups(status);

-- Index for querying backups by time range
CREATE INDEX idx_docker_log_backups_time_range ON docker_log_backups(start_time, end_time);

-- Docker log settings per agent
CREATE TABLE docker_log_settings (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL UNIQUE REFERENCES agents(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT false,
    cron_expression TEXT NOT NULL DEFAULT '0 * * * *',
    -- Retention policy as JSONB
    retention_policy JSONB NOT NULL DEFAULT '{"max_age_days": 30, "max_size_bytes": 1073741824, "max_files_per_day": 24, "compress_enabled": true, "compress_level": 6}'::JSONB,
    -- Container filtering
    include_containers JSONB DEFAULT '[]'::JSONB,
    exclude_containers JSONB DEFAULT '[]'::JSONB,
    include_labels JSONB DEFAULT '{}'::JSONB,
    exclude_labels JSONB DEFAULT '{}'::JSONB,
    -- Log options
    timestamps BOOLEAN DEFAULT true,
    tail INTEGER DEFAULT 0,
    since TEXT,
    until TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for querying settings by agent
CREATE INDEX idx_docker_log_settings_agent_id ON docker_log_settings(agent_id);

-- Index for enabled settings (for scheduler)
CREATE INDEX idx_docker_log_settings_enabled ON docker_log_settings(enabled) WHERE enabled = true;
