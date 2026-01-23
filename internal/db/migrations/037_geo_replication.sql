-- Migration: Add geo-replication configuration and tracking

-- Create geo_replication_configs table
CREATE TABLE geo_replication_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    source_repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    target_repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    source_region VARCHAR(64) NOT NULL,
    target_region VARCHAR(64) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    status VARCHAR(32) DEFAULT 'pending',
    last_snapshot_id VARCHAR(255),
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,
    max_lag_snapshots INTEGER DEFAULT 5,
    max_lag_duration_hours INTEGER DEFAULT 24,
    alert_on_lag BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(source_repository_id, target_repository_id)
);

-- Create replication_events table to track individual replication operations
CREATE TABLE replication_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES geo_replication_configs(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    bytes_copied BIGINT,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX idx_geo_replication_configs_org ON geo_replication_configs(org_id);
CREATE INDEX idx_geo_replication_configs_source_repo ON geo_replication_configs(source_repository_id);
CREATE INDEX idx_geo_replication_configs_target_repo ON geo_replication_configs(target_repository_id);
CREATE INDEX idx_geo_replication_configs_enabled ON geo_replication_configs(enabled) WHERE enabled = true;
CREATE INDEX idx_geo_replication_configs_status ON geo_replication_configs(status);

CREATE INDEX idx_replication_events_config ON replication_events(config_id);
CREATE INDEX idx_replication_events_status ON replication_events(status);
CREATE INDEX idx_replication_events_started_at ON replication_events(started_at DESC);
CREATE INDEX idx_replication_events_snapshot ON replication_events(config_id, snapshot_id);

-- Add region columns to repositories table for geo-location tracking
ALTER TABLE repositories ADD COLUMN region VARCHAR(64);

-- Create index for region queries
CREATE INDEX idx_repositories_region ON repositories(region) WHERE region IS NOT NULL;
