-- 012_metrics_history.sql
-- Migration: Add metrics history table for time-series dashboard data

-- Metrics history table for storing time-series backup and system metrics
CREATE TABLE metrics_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Backup metrics
    backup_count INTEGER NOT NULL DEFAULT 0,
    backup_success_count INTEGER NOT NULL DEFAULT 0,
    backup_failed_count INTEGER NOT NULL DEFAULT 0,
    backup_total_size BIGINT NOT NULL DEFAULT 0,
    backup_total_duration_ms BIGINT NOT NULL DEFAULT 0,

    -- Agent metrics
    agent_total_count INTEGER NOT NULL DEFAULT 0,
    agent_online_count INTEGER NOT NULL DEFAULT 0,
    agent_offline_count INTEGER NOT NULL DEFAULT 0,

    -- Storage metrics (aggregated from storage_stats)
    storage_used_bytes BIGINT NOT NULL DEFAULT 0,
    storage_raw_bytes BIGINT NOT NULL DEFAULT 0,
    storage_space_saved BIGINT NOT NULL DEFAULT 0,

    -- Repository metrics
    repository_count INTEGER NOT NULL DEFAULT 0,
    total_snapshots INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    collected_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast lookups by organization
CREATE INDEX idx_metrics_history_org ON metrics_history(org_id);

-- Index for time-based queries (trends over time)
CREATE INDEX idx_metrics_history_collected ON metrics_history(collected_at);

-- Composite index for efficient queries by org and time
CREATE INDEX idx_metrics_history_org_collected ON metrics_history(org_id, collected_at DESC);

-- Daily aggregated metrics view for efficient dashboard queries
CREATE TABLE metrics_daily_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    date DATE NOT NULL,

    -- Daily backup stats
    backups_total INTEGER NOT NULL DEFAULT 0,
    backups_successful INTEGER NOT NULL DEFAULT 0,
    backups_failed INTEGER NOT NULL DEFAULT 0,
    total_backup_size BIGINT NOT NULL DEFAULT 0,
    avg_backup_duration_ms BIGINT NOT NULL DEFAULT 0,

    -- Daily storage change
    storage_delta BIGINT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id, date)
);

CREATE INDEX idx_metrics_daily_org ON metrics_daily_summary(org_id);
CREATE INDEX idx_metrics_daily_date ON metrics_daily_summary(date);
CREATE INDEX idx_metrics_daily_org_date ON metrics_daily_summary(org_id, date DESC);
