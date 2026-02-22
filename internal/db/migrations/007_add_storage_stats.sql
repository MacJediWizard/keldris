-- 007_add_storage_stats.sql
-- 002_add_storage_stats.sql
-- Migration: Add storage statistics table for tracking repository storage efficiency

CREATE TABLE storage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    total_size BIGINT NOT NULL DEFAULT 0,
    total_file_count INTEGER NOT NULL DEFAULT 0,
    raw_data_size BIGINT NOT NULL DEFAULT 0,
    restore_size BIGINT NOT NULL DEFAULT 0,
    dedup_ratio DOUBLE PRECISION NOT NULL DEFAULT 0,
    space_saved BIGINT NOT NULL DEFAULT 0,
    space_saved_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
    snapshot_count INTEGER NOT NULL DEFAULT 0,
    collected_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast lookups by repository
CREATE INDEX idx_storage_stats_repository ON storage_stats(repository_id);

-- Index for time-based queries (storage growth over time)
CREATE INDEX idx_storage_stats_collected ON storage_stats(collected_at);

-- Composite index for efficient queries by repository and time
CREATE INDEX idx_storage_stats_repo_collected ON storage_stats(repository_id, collected_at DESC);
