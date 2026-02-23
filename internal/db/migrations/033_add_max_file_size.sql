-- 031_add_max_file_size.sql
-- Add max file size limit to schedules for auto-excluding large files

-- Add max_file_size_mb column to schedules table (in MB, 0 or NULL = disabled)
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS max_file_size_mb INTEGER;

-- Add excluded_large_files column to backups table to track files excluded due to size
ALTER TABLE backups ADD COLUMN IF NOT EXISTS excluded_large_files JSONB;
