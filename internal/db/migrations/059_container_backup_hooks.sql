-- 059_container_backup_hooks.sql
-- Add support for Docker container backup hooks

-- Create container_backup_hooks table for storing hooks associated with schedules
CREATE TABLE container_backup_hooks (
    id UUID PRIMARY KEY,
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    container_name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'pre_backup' or 'post_backup'
    template VARCHAR(50) DEFAULT 'none', -- 'none', 'postgres', 'mysql', 'mongodb', 'redis', 'elasticsearch'
    command TEXT NOT NULL,
    working_dir VARCHAR(512),
    user_name VARCHAR(255),
    timeout_seconds INTEGER DEFAULT 300,
    fail_on_error BOOLEAN DEFAULT FALSE,
    enabled BOOLEAN DEFAULT TRUE,
    description TEXT,
    template_vars JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for looking up hooks by schedule
CREATE INDEX idx_container_backup_hooks_schedule ON container_backup_hooks(schedule_id);

-- Index for looking up hooks by container
CREATE INDEX idx_container_backup_hooks_container ON container_backup_hooks(container_name);

-- Index for enabled hooks
CREATE INDEX idx_container_backup_hooks_enabled ON container_backup_hooks(schedule_id, enabled) WHERE enabled = TRUE;

-- Create container_hook_executions table for storing execution logs
CREATE TABLE container_hook_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hook_id UUID NOT NULL REFERENCES container_backup_hooks(id) ON DELETE CASCADE,
    backup_id UUID NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    container_name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    command TEXT NOT NULL,
    output TEXT,
    exit_code INTEGER,
    error TEXT,
    duration_ms INTEGER,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for looking up executions by backup
CREATE INDEX idx_container_hook_executions_backup ON container_hook_executions(backup_id);

-- Index for looking up executions by hook
CREATE INDEX idx_container_hook_executions_hook ON container_hook_executions(hook_id);

-- Add columns to backups table for container hook output
ALTER TABLE backups
ADD COLUMN container_pre_hook_output TEXT,
ADD COLUMN container_pre_hook_error TEXT,
ADD COLUMN container_post_hook_output TEXT,
ADD COLUMN container_post_hook_error TEXT;

COMMENT ON TABLE container_backup_hooks IS 'Docker container hooks to run before or after backup operations';
COMMENT ON COLUMN container_backup_hooks.type IS 'Hook type: pre_backup, post_backup';
COMMENT ON COLUMN container_backup_hooks.template IS 'Pre-defined template: none, postgres, mysql, mongodb, redis, elasticsearch';
COMMENT ON COLUMN container_backup_hooks.command IS 'The shell command to execute inside the container';
COMMENT ON COLUMN container_backup_hooks.working_dir IS 'Working directory inside the container';
COMMENT ON COLUMN container_backup_hooks.user_name IS 'User to run the command as inside the container';
COMMENT ON COLUMN container_backup_hooks.timeout_seconds IS 'Maximum execution time in seconds (default 300)';
COMMENT ON COLUMN container_backup_hooks.fail_on_error IS 'If true and pre_backup fails, abort the backup';
COMMENT ON COLUMN container_backup_hooks.template_vars IS 'Template-specific variables as JSON';
COMMENT ON TABLE container_hook_executions IS 'Execution logs for container backup hooks';
