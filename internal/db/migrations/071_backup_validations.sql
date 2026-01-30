-- Migration 059: Backup Validations
-- Adds automated backup validation tracking

-- Create backup_validations table
CREATE TABLE IF NOT EXISTS backup_validations (
    id UUID PRIMARY KEY,
    backup_id UUID NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    snapshot_id TEXT NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    status TEXT NOT NULL DEFAULT 'running',
    duration_ms BIGINT,
    error_message TEXT,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_backup_validations_backup_id ON backup_validations(backup_id);
CREATE INDEX IF NOT EXISTS idx_backup_validations_repository_id ON backup_validations(repository_id);
CREATE INDEX IF NOT EXISTS idx_backup_validations_status ON backup_validations(status);
CREATE INDEX IF NOT EXISTS idx_backup_validations_created_at ON backup_validations(created_at DESC);

-- Add validation fields to backups table
ALTER TABLE backups ADD COLUMN IF NOT EXISTS validation_id UUID REFERENCES backup_validations(id) ON DELETE SET NULL;
ALTER TABLE backups ADD COLUMN IF NOT EXISTS validation_status TEXT;
ALTER TABLE backups ADD COLUMN IF NOT EXISTS validation_error TEXT;

-- Create index for validation lookup
CREATE INDEX IF NOT EXISTS idx_backups_validation_id ON backups(validation_id);
CREATE INDEX IF NOT EXISTS idx_backups_validation_status ON backups(validation_status);
