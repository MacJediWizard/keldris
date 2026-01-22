-- Migration: Add backup checkpoint tracking for resumable backups

-- Create backup_checkpoints table to track interrupted backups
CREATE TABLE backup_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    backup_id UUID REFERENCES backups(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    files_processed BIGINT NOT NULL DEFAULT 0,
    bytes_processed BIGINT NOT NULL DEFAULT 0,
    total_files BIGINT,
    total_bytes BIGINT,
    last_processed_path TEXT,
    restic_state BYTEA,
    error_message TEXT,
    resume_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_checkpoint_status CHECK (status IN ('active', 'completed', 'canceled', 'expired'))
);

-- Add indexes for efficient queries
CREATE INDEX idx_backup_checkpoints_schedule ON backup_checkpoints(schedule_id);
CREATE INDEX idx_backup_checkpoints_agent ON backup_checkpoints(agent_id);
CREATE INDEX idx_backup_checkpoints_repository ON backup_checkpoints(repository_id);
CREATE INDEX idx_backup_checkpoints_backup ON backup_checkpoints(backup_id);
CREATE INDEX idx_backup_checkpoints_status ON backup_checkpoints(status) WHERE status = 'active';
CREATE INDEX idx_backup_checkpoints_expires ON backup_checkpoints(expires_at) WHERE status = 'active';

-- Add resumed tracking columns to backups table
ALTER TABLE backups ADD COLUMN resumed BOOLEAN DEFAULT false;
ALTER TABLE backups ADD COLUMN checkpoint_id UUID REFERENCES backup_checkpoints(id) ON DELETE SET NULL;
ALTER TABLE backups ADD COLUMN original_backup_id UUID REFERENCES backups(id) ON DELETE SET NULL;

-- Index for finding resumed backups
CREATE INDEX idx_backups_resumed ON backups(resumed) WHERE resumed = true;
CREATE INDEX idx_backups_checkpoint ON backups(checkpoint_id) WHERE checkpoint_id IS NOT NULL;
