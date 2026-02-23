-- Migration: Add missing schedule columns for backup_type, docker_options, pihole_config

ALTER TABLE schedules ADD COLUMN IF NOT EXISTS backup_type VARCHAR(50) DEFAULT 'filesystem';
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS docker_options JSONB;
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS pihole_config JSONB;
