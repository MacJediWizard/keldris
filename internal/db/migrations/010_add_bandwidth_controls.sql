-- 010_add_bandwidth_controls.sql
-- Migration: Add bandwidth and time window controls for schedules

ALTER TABLE schedules ADD COLUMN bandwidth_limit_kbps INTEGER;
ALTER TABLE schedules ADD COLUMN backup_window_start TIME;
ALTER TABLE schedules ADD COLUMN backup_window_end TIME;
ALTER TABLE schedules ADD COLUMN excluded_hours JSONB;

COMMENT ON COLUMN schedules.bandwidth_limit_kbps IS 'Upload bandwidth limit in KB/s (null = unlimited)';
COMMENT ON COLUMN schedules.backup_window_start IS 'Start of allowed backup window (null = any time)';
COMMENT ON COLUMN schedules.backup_window_end IS 'End of allowed backup window (null = any time)';
COMMENT ON COLUMN schedules.excluded_hours IS 'Array of hours (0-23) when backups should not run';
