-- Migration: Add report schedules and history tables

-- Report schedules configuration per organization
CREATE TABLE report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    frequency VARCHAR(20) NOT NULL, -- 'daily', 'weekly', 'monthly'
    recipients JSONB NOT NULL DEFAULT '[]'::jsonb,
    channel_id UUID REFERENCES notification_channels(id) ON DELETE SET NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    enabled BOOLEAN DEFAULT true,
    last_sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, name)
);

CREATE INDEX idx_report_schedules_org ON report_schedules(org_id);
CREATE INDEX idx_report_schedules_enabled ON report_schedules(enabled) WHERE enabled = true;

-- Report history/log for tracking sent reports
CREATE TABLE report_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES report_schedules(id) ON DELETE SET NULL,
    report_type VARCHAR(20) NOT NULL, -- 'daily', 'weekly', 'monthly', 'manual'
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    recipients JSONB NOT NULL,
    status VARCHAR(50) NOT NULL, -- 'sent', 'failed', 'preview'
    error_message TEXT,
    report_data JSONB,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_report_history_org ON report_history(org_id);
CREATE INDEX idx_report_history_schedule ON report_history(schedule_id);
CREATE INDEX idx_report_history_created ON report_history(created_at DESC);
CREATE INDEX idx_report_history_status ON report_history(status);
