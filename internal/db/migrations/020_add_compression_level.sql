-- 020_add_compression_level.sql
-- 012_add_compression_level.sql
-- Migration: Add compression level setting for schedules

ALTER TABLE schedules ADD COLUMN compression_level VARCHAR(50);

COMMENT ON COLUMN schedules.compression_level IS 'Compression level for backups: off, auto, max (null = auto, restic default)';
