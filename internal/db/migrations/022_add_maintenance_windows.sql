-- Migration: Add maintenance windows for scheduled backup pauses

CREATE TABLE maintenance_windows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    notify_before_minutes INT DEFAULT 60,
    notification_sent BOOLEAN DEFAULT false,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT chk_maintenance_window_valid CHECK (ends_at > starts_at)
);

-- Index for org lookup
CREATE INDEX idx_maintenance_windows_org ON maintenance_windows(org_id);

-- Composite index for time range queries
CREATE INDEX idx_maintenance_windows_time ON maintenance_windows(starts_at, ends_at);
