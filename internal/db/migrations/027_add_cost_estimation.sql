-- Migration: Add cost estimation tables
-- This migration adds tables for custom storage pricing and cost alerts.

-- Custom storage pricing configuration per organization
CREATE TABLE storage_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_type VARCHAR(50) NOT NULL,
    storage_per_gb_month DECIMAL(10, 6) NOT NULL DEFAULT 0,
    egress_per_gb DECIMAL(10, 6) NOT NULL DEFAULT 0,
    operations_per_k DECIMAL(10, 6) NOT NULL DEFAULT 0,
    provider_name VARCHAR(100),
    provider_description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, repository_type)
);

CREATE INDEX idx_storage_pricing_org ON storage_pricing(org_id);

-- Cost estimation snapshots for historical tracking
CREATE TABLE cost_estimates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    storage_size_bytes BIGINT NOT NULL DEFAULT 0,
    monthly_cost DECIMAL(12, 4) NOT NULL DEFAULT 0,
    yearly_cost DECIMAL(12, 4) NOT NULL DEFAULT 0,
    cost_per_gb DECIMAL(10, 6) NOT NULL DEFAULT 0,
    estimated_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cost_estimates_org ON cost_estimates(org_id);
CREATE INDEX idx_cost_estimates_repo ON cost_estimates(repository_id);
CREATE INDEX idx_cost_estimates_repo_estimated ON cost_estimates(repository_id, estimated_at DESC);

-- Cost alerts configuration
CREATE TABLE cost_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    monthly_threshold DECIMAL(12, 4) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    notify_on_exceed BOOLEAN DEFAULT true,
    notify_on_forecast BOOLEAN DEFAULT false,
    forecast_months INT DEFAULT 3,
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cost_alerts_org ON cost_alerts(org_id);
CREATE INDEX idx_cost_alerts_org_enabled ON cost_alerts(org_id, enabled);
