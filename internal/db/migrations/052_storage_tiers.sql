-- Migration: Add storage tiering for hot/warm/cold/archive data lifecycle

-- Create storage_tier_configs table for organization tier settings
CREATE TABLE storage_tier_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tier_type VARCHAR(32) NOT NULL CHECK (tier_type IN ('hot', 'warm', 'cold', 'archive')),
    name VARCHAR(128) NOT NULL,
    description TEXT,
    cost_per_gb_month DECIMAL(10, 6) DEFAULT 0,
    retrieval_cost DECIMAL(10, 6) DEFAULT 0,
    retrieval_time VARCHAR(64) DEFAULT 'immediate',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, tier_type)
);

-- Create tier_rules table for automatic tier transitions
CREATE TABLE tier_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES schedules(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    from_tier VARCHAR(32) NOT NULL CHECK (from_tier IN ('hot', 'warm', 'cold', 'archive')),
    to_tier VARCHAR(32) NOT NULL CHECK (to_tier IN ('hot', 'warm', 'cold', 'archive')),
    age_threshold_days INTEGER NOT NULL DEFAULT 30,
    min_copies INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 100,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (from_tier != to_tier)
);

-- Create snapshot_tiers table to track current tier of each snapshot
CREATE TABLE snapshot_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id VARCHAR(255) NOT NULL,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    current_tier VARCHAR(32) NOT NULL DEFAULT 'hot' CHECK (current_tier IN ('hot', 'warm', 'cold', 'archive')),
    size_bytes BIGINT DEFAULT 0,
    snapshot_time TIMESTAMPTZ NOT NULL,
    tiered_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(snapshot_id, repository_id)
);

-- Create tier_transitions table for audit trail
CREATE TABLE tier_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_tier_id UUID NOT NULL REFERENCES snapshot_tiers(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    from_tier VARCHAR(32) NOT NULL CHECK (from_tier IN ('hot', 'warm', 'cold', 'archive')),
    to_tier VARCHAR(32) NOT NULL CHECK (to_tier IN ('hot', 'warm', 'cold', 'archive')),
    trigger_rule_id UUID REFERENCES tier_rules(id) ON DELETE SET NULL,
    trigger_reason TEXT,
    size_bytes BIGINT DEFAULT 0,
    estimated_saving DECIMAL(10, 4) DEFAULT 0,
    status VARCHAR(32) DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create tier_cost_reports table for cost optimization reports
CREATE TABLE tier_cost_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    report_date DATE NOT NULL,
    total_size_bytes BIGINT DEFAULT 0,
    current_monthly_cost DECIMAL(12, 4) DEFAULT 0,
    optimized_monthly_cost DECIMAL(12, 4) DEFAULT 0,
    potential_monthly_savings DECIMAL(12, 4) DEFAULT 0,
    tier_breakdown JSONB DEFAULT '[]'::jsonb,
    suggestions JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, report_date)
);

-- Create cold_restore_requests table for tracking cold/archive restores
CREATE TABLE cold_restore_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_tier VARCHAR(32) NOT NULL CHECK (from_tier IN ('cold', 'archive')),
    target_path TEXT,
    priority VARCHAR(32) DEFAULT 'standard' CHECK (priority IN ('standard', 'expedited', 'bulk')),
    status VARCHAR(32) DEFAULT 'pending' CHECK (status IN ('pending', 'warming', 'ready', 'restoring', 'completed', 'failed', 'expired')),
    estimated_ready_at TIMESTAMPTZ,
    ready_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    error_message TEXT,
    retrieval_cost DECIMAL(10, 4) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX idx_storage_tier_configs_org ON storage_tier_configs(org_id);
CREATE INDEX idx_storage_tier_configs_enabled ON storage_tier_configs(org_id, enabled) WHERE enabled = true;

CREATE INDEX idx_tier_rules_org ON tier_rules(org_id);
CREATE INDEX idx_tier_rules_repo ON tier_rules(repository_id) WHERE repository_id IS NOT NULL;
CREATE INDEX idx_tier_rules_schedule ON tier_rules(schedule_id) WHERE schedule_id IS NOT NULL;
CREATE INDEX idx_tier_rules_enabled ON tier_rules(org_id, enabled, priority) WHERE enabled = true;
CREATE INDEX idx_tier_rules_transitions ON tier_rules(from_tier, to_tier);

CREATE INDEX idx_snapshot_tiers_repo ON snapshot_tiers(repository_id);
CREATE INDEX idx_snapshot_tiers_org ON snapshot_tiers(org_id);
CREATE INDEX idx_snapshot_tiers_current ON snapshot_tiers(current_tier);
CREATE INDEX idx_snapshot_tiers_snapshot ON snapshot_tiers(snapshot_id);
CREATE INDEX idx_snapshot_tiers_time ON snapshot_tiers(snapshot_time DESC);
CREATE INDEX idx_snapshot_tiers_tiered ON snapshot_tiers(tiered_at);

CREATE INDEX idx_tier_transitions_snapshot ON tier_transitions(snapshot_tier_id);
CREATE INDEX idx_tier_transitions_org ON tier_transitions(org_id);
CREATE INDEX idx_tier_transitions_status ON tier_transitions(status);
CREATE INDEX idx_tier_transitions_created ON tier_transitions(created_at DESC);
CREATE INDEX idx_tier_transitions_rule ON tier_transitions(trigger_rule_id) WHERE trigger_rule_id IS NOT NULL;

CREATE INDEX idx_tier_cost_reports_org ON tier_cost_reports(org_id);
CREATE INDEX idx_tier_cost_reports_date ON tier_cost_reports(org_id, report_date DESC);

CREATE INDEX idx_cold_restore_requests_org ON cold_restore_requests(org_id);
CREATE INDEX idx_cold_restore_requests_status ON cold_restore_requests(status);
CREATE INDEX idx_cold_restore_requests_snapshot ON cold_restore_requests(snapshot_id, repository_id);
CREATE INDEX idx_cold_restore_requests_user ON cold_restore_requests(requested_by);
CREATE INDEX idx_cold_restore_requests_expires ON cold_restore_requests(expires_at) WHERE expires_at IS NOT NULL AND status = 'ready';

-- Add tiering_enabled column to repositories
ALTER TABLE repositories ADD COLUMN tiering_enabled BOOLEAN DEFAULT false;
ALTER TABLE repositories ADD COLUMN default_tier VARCHAR(32) DEFAULT 'hot' CHECK (default_tier IN ('hot', 'warm', 'cold', 'archive'));

-- Create index for tiering-enabled repositories
CREATE INDEX idx_repositories_tiering ON repositories(tiering_enabled) WHERE tiering_enabled = true;
