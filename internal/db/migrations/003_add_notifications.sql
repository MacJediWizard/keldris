-- 002_add_notifications.sql
-- Migration: Add notification channels and preferences

-- Notification channels (SMTP, Slack, Webhook, etc.)
CREATE TABLE notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- email, slack, webhook, pagerduty
    config_encrypted BYTEA NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notification_channels_org ON notification_channels(org_id);
CREATE INDEX idx_notification_channels_type ON notification_channels(org_id, type);

-- Notification preferences per organization
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES notification_channels(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL, -- backup_success, backup_failed, agent_offline
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, channel_id, event_type)
);

CREATE INDEX idx_notification_preferences_org ON notification_preferences(org_id);
CREATE INDEX idx_notification_preferences_event ON notification_preferences(org_id, event_type);

-- Notification history/log
CREATE TABLE notification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES notification_channels(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    subject VARCHAR(500),
    status VARCHAR(50) NOT NULL, -- sent, failed, queued
    error_message TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notification_logs_org ON notification_logs(org_id);
CREATE INDEX idx_notification_logs_status ON notification_logs(org_id, status);
CREATE INDEX idx_notification_logs_created ON notification_logs(created_at);
