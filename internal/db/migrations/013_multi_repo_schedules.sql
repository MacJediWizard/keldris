-- 013_multi_repo_schedules.sql
-- 002_multi_repo_schedules.sql
-- Migration: Add multi-repository support for schedules with failover and replication

-- Create junction table for schedule-repository many-to-many relationship
CREATE TABLE schedule_repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    priority INTEGER NOT NULL DEFAULT 0,  -- 0 = primary, 1+ = secondary by priority order
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(schedule_id, repository_id)
);

CREATE INDEX idx_schedule_repos_schedule ON schedule_repositories(schedule_id);
CREATE INDEX idx_schedule_repos_repo ON schedule_repositories(repository_id);
CREATE INDEX idx_schedule_repos_priority ON schedule_repositories(schedule_id, priority);

-- Create replication status tracking table
CREATE TABLE replication_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    source_repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    target_repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    last_snapshot_id VARCHAR(255),
    last_sync_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, syncing, synced, failed
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(schedule_id, source_repository_id, target_repository_id)
);

CREATE INDEX idx_replication_schedule ON replication_status(schedule_id);
CREATE INDEX idx_replication_status ON replication_status(status);

-- Add repository_id to backups table to track which repo the backup was made to
ALTER TABLE backups ADD COLUMN repository_id UUID REFERENCES repositories(id) ON DELETE SET NULL;
CREATE INDEX idx_backups_repo ON backups(repository_id);

-- Migrate existing schedule-repository relationships to junction table
INSERT INTO schedule_repositories (schedule_id, repository_id, priority, enabled)
SELECT id, repository_id, 0, enabled
FROM schedules
WHERE repository_id IS NOT NULL;

-- Update existing backups with repository_id from their schedule
UPDATE backups b
SET repository_id = s.repository_id
FROM schedules s
WHERE b.schedule_id = s.id;

-- Drop the old repository_id column from schedules
ALTER TABLE schedules DROP COLUMN repository_id;
