-- 077_usage_metrics.sql
-- Add usage metering for limits and billing tracking

-- Daily usage snapshots for organizations
CREATE TABLE IF NOT EXISTS usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,

    -- Agent counts
    agent_count INTEGER NOT NULL DEFAULT 0,
    active_agent_count INTEGER NOT NULL DEFAULT 0,

    -- User counts
    user_count INTEGER NOT NULL DEFAULT 0,
    active_user_count INTEGER NOT NULL DEFAULT 0,

    -- Storage metrics (in bytes)
    total_storage_bytes BIGINT NOT NULL DEFAULT 0,
    backup_storage_bytes BIGINT NOT NULL DEFAULT 0,

    -- Backup counts for the period
    backups_completed INTEGER NOT NULL DEFAULT 0,
    backups_failed INTEGER NOT NULL DEFAULT 0,
    backups_total INTEGER NOT NULL DEFAULT 0,

    -- Repository counts
    repository_count INTEGER NOT NULL DEFAULT 0,

    -- Schedule counts
    schedule_count INTEGER NOT NULL DEFAULT 0,

    -- Snapshot counts
    snapshot_count INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id, snapshot_date)
);

CREATE INDEX idx_usage_metrics_org_id ON usage_metrics(org_id);
CREATE INDEX idx_usage_metrics_snapshot_date ON usage_metrics(snapshot_date);
CREATE INDEX idx_usage_metrics_org_date ON usage_metrics(org_id, snapshot_date DESC);

-- Organization usage limits for billing tiers
CREATE TABLE IF NOT EXISTS org_usage_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE UNIQUE,

    -- Agent limits
    max_agents INTEGER,  -- NULL means unlimited

    -- User limits
    max_users INTEGER,   -- NULL means unlimited

    -- Storage limits (in bytes)
    max_storage_bytes BIGINT,  -- NULL means unlimited

    -- Backup limits
    max_backups_per_month INTEGER,  -- NULL means unlimited

    -- Repository limits
    max_repositories INTEGER,  -- NULL means unlimited

    -- Alert thresholds (percentage 0-100)
    warning_threshold INTEGER DEFAULT 80,
    critical_threshold INTEGER DEFAULT 95,

    -- Billing tier info
    billing_tier VARCHAR(50) DEFAULT 'free',
    billing_period_start DATE,
    billing_period_end DATE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_org_usage_limits_org_id ON org_usage_limits(org_id);

-- Usage alerts for tracking limit warnings
CREATE TABLE IF NOT EXISTS usage_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,  -- 'agents', 'users', 'storage', 'backups', 'repositories'
    severity VARCHAR(20) NOT NULL,     -- 'warning', 'critical', 'exceeded'
    current_value BIGINT NOT NULL,
    limit_value BIGINT NOT NULL,
    percentage_used DECIMAL(5,2) NOT NULL,
    message TEXT NOT NULL,
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_usage_alerts_org_id ON usage_alerts(org_id);
CREATE INDEX idx_usage_alerts_type ON usage_alerts(alert_type);
CREATE INDEX idx_usage_alerts_severity ON usage_alerts(severity);
CREATE INDEX idx_usage_alerts_active ON usage_alerts(org_id, resolved, created_at DESC);
CREATE INDEX idx_usage_alerts_unresolved ON usage_alerts(resolved, acknowledged) WHERE resolved = false;

-- Monthly usage aggregates for billing
CREATE TABLE IF NOT EXISTS monthly_usage_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    year_month VARCHAR(7) NOT NULL,  -- Format: YYYY-MM

    -- Peak values during the month
    peak_agent_count INTEGER NOT NULL DEFAULT 0,
    peak_user_count INTEGER NOT NULL DEFAULT 0,
    peak_storage_bytes BIGINT NOT NULL DEFAULT 0,

    -- Totals for the month
    total_backups_completed INTEGER NOT NULL DEFAULT 0,
    total_backups_failed INTEGER NOT NULL DEFAULT 0,
    total_data_backed_up_bytes BIGINT NOT NULL DEFAULT 0,

    -- Average values
    avg_agent_count DECIMAL(10,2) DEFAULT 0,
    avg_storage_bytes BIGINT DEFAULT 0,

    -- For billing calculations
    billable_agent_hours INTEGER DEFAULT 0,
    billable_storage_gb_hours BIGINT DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id, year_month)
);

CREATE INDEX idx_monthly_usage_summary_org_id ON monthly_usage_summary(org_id);
CREATE INDEX idx_monthly_usage_summary_period ON monthly_usage_summary(year_month);
CREATE INDEX idx_monthly_usage_summary_org_period ON monthly_usage_summary(org_id, year_month DESC);
