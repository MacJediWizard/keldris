-- 031_ransomware_detection.sql
-- Migration: Add ransomware detection tables

-- Ransomware detection settings per schedule
CREATE TABLE ransomware_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT true,
    change_threshold_percent INTEGER DEFAULT 30, -- Alert if > X% files changed
    extensions_to_detect JSONB, -- Custom ransomware extensions (null = use defaults)
    entropy_detection_enabled BOOLEAN DEFAULT true,
    entropy_threshold DECIMAL(3,1) DEFAULT 7.5, -- 0-8 scale
    auto_pause_on_alert BOOLEAN DEFAULT false, -- Auto-pause backups on detection
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_ransomware_settings_schedule UNIQUE (schedule_id)
);

-- Ransomware alerts
CREATE TABLE ransomware_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    backup_id UUID NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    schedule_name VARCHAR(255) NOT NULL,
    agent_hostname VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, investigating, false_positive, confirmed, resolved
    risk_score INTEGER NOT NULL, -- 0-100
    indicators JSONB, -- Detailed indicators that triggered the alert
    files_changed INTEGER DEFAULT 0,
    files_new INTEGER DEFAULT 0,
    total_files INTEGER DEFAULT 0,
    backups_paused BOOLEAN DEFAULT false,
    paused_at TIMESTAMPTZ,
    resumed_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    resolution TEXT, -- Description of how the alert was resolved
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ransomware_settings
CREATE INDEX idx_ransomware_settings_schedule ON ransomware_settings(schedule_id);

-- Indexes for ransomware_alerts
CREATE INDEX idx_ransomware_alerts_org ON ransomware_alerts(org_id);
CREATE INDEX idx_ransomware_alerts_schedule ON ransomware_alerts(schedule_id);
CREATE INDEX idx_ransomware_alerts_status ON ransomware_alerts(status);
CREATE INDEX idx_ransomware_alerts_risk_score ON ransomware_alerts(risk_score DESC);
CREATE INDEX idx_ransomware_alerts_created ON ransomware_alerts(created_at DESC);
CREATE INDEX idx_ransomware_alerts_active ON ransomware_alerts(org_id, status)
    WHERE status IN ('active', 'investigating');
