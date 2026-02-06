-- Database backups table for tracking Keldris PostgreSQL backups
CREATE TABLE IF NOT EXISTS database_backups (
    id UUID PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    file_path TEXT,
    size_bytes BIGINT,
    checksum VARCHAR(128),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    error_message TEXT,
    triggered_by UUID REFERENCES users(id) ON DELETE SET NULL,
    is_scheduled BOOLEAN NOT NULL DEFAULT TRUE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for querying backups by status
CREATE INDEX IF NOT EXISTS idx_database_backups_status ON database_backups(status);

-- Index for querying backups by creation time (for retention cleanup)
CREATE INDEX IF NOT EXISTS idx_database_backups_created_at ON database_backups(created_at DESC);

-- Index for finding latest backup quickly
CREATE INDEX IF NOT EXISTS idx_database_backups_completed_at ON database_backups(completed_at DESC NULLS LAST);

-- Add comment for documentation
COMMENT ON TABLE database_backups IS 'Tracks Keldris PostgreSQL database backups including scheduled and manual backups';
COMMENT ON COLUMN database_backups.status IS 'Backup status: pending, running, completed, failed';
COMMENT ON COLUMN database_backups.file_path IS 'Path to the encrypted backup file';
COMMENT ON COLUMN database_backups.size_bytes IS 'Size of the backup file in bytes';
COMMENT ON COLUMN database_backups.checksum IS 'SHA256 checksum of the encrypted backup file';
COMMENT ON COLUMN database_backups.triggered_by IS 'User who triggered a manual backup (null for scheduled)';
COMMENT ON COLUMN database_backups.is_scheduled IS 'True if backup was created by scheduler, false if manual';
COMMENT ON COLUMN database_backups.verified IS 'True if backup integrity has been verified';
