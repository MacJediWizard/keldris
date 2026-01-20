-- 008_add_retention_logging.sql
-- Migration: Add retention action logging to backups table

ALTER TABLE backups ADD COLUMN retention_applied BOOLEAN DEFAULT false;
ALTER TABLE backups ADD COLUMN snapshots_removed INTEGER;
ALTER TABLE backups ADD COLUMN snapshots_kept INTEGER;
ALTER TABLE backups ADD COLUMN retention_error TEXT;
