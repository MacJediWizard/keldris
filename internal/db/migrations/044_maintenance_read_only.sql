-- Migration: Add read-only mode and countdown to maintenance windows

ALTER TABLE maintenance_windows
ADD COLUMN read_only BOOLEAN DEFAULT false,
ADD COLUMN countdown_start_minutes INT DEFAULT 30,
ADD COLUMN emergency_override BOOLEAN DEFAULT false,
ADD COLUMN overridden_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN overridden_at TIMESTAMPTZ;

-- Index for finding active read-only windows
CREATE INDEX idx_maintenance_windows_read_only ON maintenance_windows(org_id, read_only, starts_at, ends_at)
WHERE read_only = true AND emergency_override = false;
