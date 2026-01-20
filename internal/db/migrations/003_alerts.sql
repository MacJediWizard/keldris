-- 002_alerts.sql
-- Migration: Add monitoring alerts and alert rules

-- Alert rules define conditions that trigger alerts
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- agent_offline, backup_sla, storage_usage
    enabled BOOLEAN DEFAULT true,
    config JSONB NOT NULL, -- type-specific configuration (thresholds, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Alerts represent triggered alert instances
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    rule_id UUID REFERENCES alert_rules(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL, -- agent_offline, backup_sla, storage_usage
    severity VARCHAR(20) NOT NULL, -- info, warning, critical
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, acknowledged, resolved
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    resource_type VARCHAR(50), -- agent, schedule, repository
    resource_id UUID,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    metadata JSONB, -- additional context data
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_alert_rules_org ON alert_rules(org_id);
CREATE INDEX idx_alert_rules_type ON alert_rules(type);
CREATE INDEX idx_alerts_org ON alerts(org_id);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_type ON alerts(type);
CREATE INDEX idx_alerts_resource ON alerts(resource_type, resource_id);
CREATE INDEX idx_alerts_created ON alerts(created_at DESC);
