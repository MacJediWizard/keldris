-- Migration: Add backup policies for reusable schedule templates

CREATE TABLE backup_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    paths JSONB,
    excludes JSONB,
    retention_policy JSONB,
    bandwidth_limit_kbps INTEGER,
    backup_window_start TIME,
    backup_window_end TIME,
    excluded_hours JSONB,
    cron_expression VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, name)
);

CREATE INDEX idx_backup_policies_org ON backup_policies(org_id);

-- Add policy reference to schedules
ALTER TABLE schedules ADD COLUMN policy_id UUID REFERENCES backup_policies(id) ON DELETE SET NULL;
CREATE INDEX idx_schedules_policy ON schedules(policy_id);

COMMENT ON TABLE backup_policies IS 'Reusable backup configuration templates';
COMMENT ON COLUMN backup_policies.paths IS 'Default backup paths for schedules using this policy';
COMMENT ON COLUMN backup_policies.cron_expression IS 'Default schedule cron expression';
COMMENT ON COLUMN schedules.policy_id IS 'Policy this schedule was created from (null if manually created)';
