-- Migration: Add downtime tracking and historical outage events

-- Create downtime_events table to track historical outages
CREATE TABLE downtime_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    component_type VARCHAR(50) NOT NULL, -- 'agent', 'server', 'repository', 'service'
    component_id UUID, -- Optional reference to specific component
    component_name VARCHAR(255) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER, -- Computed on end
    severity VARCHAR(20) NOT NULL DEFAULT 'warning', -- 'info', 'warning', 'critical'
    cause VARCHAR(255),
    notes TEXT,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    auto_detected BOOLEAN NOT NULL DEFAULT true,
    alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL, -- Link to alert if applicable
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create uptime_stats table for aggregated uptime metrics
CREATE TABLE uptime_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    component_type VARCHAR(50) NOT NULL,
    component_id UUID,
    component_name VARCHAR(255) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    total_seconds INTEGER NOT NULL,
    downtime_seconds INTEGER NOT NULL DEFAULT 0,
    uptime_percent DECIMAL(5, 2) NOT NULL DEFAULT 100.00,
    incident_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, component_type, component_id, period_start, period_end)
);

-- Create uptime_badges table for displaying uptime badges
CREATE TABLE uptime_badges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    component_type VARCHAR(50), -- NULL for overall system
    component_id UUID,
    component_name VARCHAR(255),
    badge_type VARCHAR(20) NOT NULL DEFAULT '30d', -- '7d', '30d', '90d', '365d'
    uptime_percent DECIMAL(5, 2) NOT NULL,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, component_type, component_id, badge_type)
);

-- Create downtime_alerts table for configuring downtime notifications
CREATE TABLE downtime_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    uptime_threshold DECIMAL(5, 2) NOT NULL DEFAULT 99.90, -- Alert if below this
    evaluation_period VARCHAR(20) NOT NULL DEFAULT '30d', -- '7d', '30d', '90d'
    component_type VARCHAR(50), -- NULL for all components
    notify_on_breach BOOLEAN NOT NULL DEFAULT true,
    notify_on_recovery BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX idx_downtime_events_org ON downtime_events(org_id);
CREATE INDEX idx_downtime_events_component ON downtime_events(org_id, component_type, component_id);
CREATE INDEX idx_downtime_events_time ON downtime_events(org_id, started_at DESC);
CREATE INDEX idx_downtime_events_active ON downtime_events(org_id, ended_at) WHERE ended_at IS NULL;
CREATE INDEX idx_uptime_stats_org ON uptime_stats(org_id);
CREATE INDEX idx_uptime_stats_component ON uptime_stats(org_id, component_type, component_id);
CREATE INDEX idx_uptime_stats_period ON uptime_stats(org_id, period_start, period_end);
CREATE INDEX idx_uptime_badges_org ON uptime_badges(org_id);
CREATE INDEX idx_uptime_badges_component ON uptime_badges(org_id, component_type, component_id);
CREATE INDEX idx_downtime_alerts_org ON downtime_alerts(org_id);
