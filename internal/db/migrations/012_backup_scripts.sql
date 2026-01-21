-- 012_backup_scripts.sql
-- Migration: Add backup scripts for pre/post backup hooks

-- Create backup_scripts table for storing scripts associated with schedules
CREATE TABLE backup_scripts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('pre_backup', 'post_success', 'post_failure', 'post_always')),
    script TEXT NOT NULL,
    timeout_seconds INTEGER DEFAULT 300,
    fail_on_error BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (schedule_id, type)
);

CREATE INDEX idx_backup_scripts_schedule ON backup_scripts(schedule_id);

COMMENT ON TABLE backup_scripts IS 'Scripts to run before or after backup operations';
COMMENT ON COLUMN backup_scripts.type IS 'Script type: pre_backup, post_success, post_failure, post_always';
COMMENT ON COLUMN backup_scripts.script IS 'The shell script content to execute';
COMMENT ON COLUMN backup_scripts.timeout_seconds IS 'Maximum execution time in seconds (default 300)';
COMMENT ON COLUMN backup_scripts.fail_on_error IS 'If true and pre_backup fails, abort the backup';
COMMENT ON COLUMN backup_scripts.enabled IS 'Whether this script is active';

-- Add script output columns to backups table
ALTER TABLE backups ADD COLUMN pre_script_output TEXT;
ALTER TABLE backups ADD COLUMN pre_script_error TEXT;
ALTER TABLE backups ADD COLUMN post_script_output TEXT;
ALTER TABLE backups ADD COLUMN post_script_error TEXT;

COMMENT ON COLUMN backups.pre_script_output IS 'Combined stdout/stderr from pre-backup script';
COMMENT ON COLUMN backups.pre_script_error IS 'Error message if pre-backup script failed';
COMMENT ON COLUMN backups.post_script_output IS 'Combined stdout/stderr from post-backup script';
COMMENT ON COLUMN backups.post_script_error IS 'Error message if post-backup script failed';
