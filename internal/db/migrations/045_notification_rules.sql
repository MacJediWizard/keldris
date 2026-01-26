-- 045_notification_rules.sql
-- Migration: Add notification rules for escalation and automated responses

-- Notification rules for conditional escalation (e.g., if backup fails 3x, escalate to PagerDuty)
CREATE TABLE notification_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_type VARCHAR(100) NOT NULL, -- backup_failed, agent_offline, alert_created, etc.
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0, -- Lower number = higher priority
    conditions JSONB NOT NULL DEFAULT '{}', -- count, time_window, severity, etc.
    actions JSONB NOT NULL DEFAULT '[]', -- notify_channel, escalate, suppress, etc.
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notification_rules_org ON notification_rules(org_id);
CREATE INDEX idx_notification_rules_trigger ON notification_rules(org_id, trigger_type);
CREATE INDEX idx_notification_rules_enabled ON notification_rules(org_id, enabled) WHERE enabled = true;
CREATE INDEX idx_notification_rules_priority ON notification_rules(org_id, priority);

-- Rule event tracking for counting occurrences within time windows
CREATE TABLE notification_rule_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES notification_rules(id) ON DELETE CASCADE,
    trigger_type VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100), -- agent, schedule, backup, etc.
    resource_id UUID,
    event_data JSONB,
    occurred_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rule_events_org ON notification_rule_events(org_id);
CREATE INDEX idx_rule_events_rule ON notification_rule_events(rule_id);
CREATE INDEX idx_rule_events_occurred ON notification_rule_events(occurred_at);
CREATE INDEX idx_rule_events_resource ON notification_rule_events(resource_type, resource_id);
-- Index for time-windowed queries
CREATE INDEX idx_rule_events_time_window ON notification_rule_events(org_id, trigger_type, resource_id, occurred_at);

-- Rule execution history for audit trail
CREATE TABLE notification_rule_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES notification_rules(id) ON DELETE CASCADE,
    triggered_by_event_id UUID REFERENCES notification_rule_events(id) ON DELETE SET NULL,
    actions_taken JSONB NOT NULL DEFAULT '[]',
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    executed_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rule_executions_org ON notification_rule_executions(org_id);
CREATE INDEX idx_rule_executions_rule ON notification_rule_executions(rule_id);
CREATE INDEX idx_rule_executions_executed ON notification_rule_executions(executed_at);
