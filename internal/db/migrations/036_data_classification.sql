-- Migration: Add data classification tables for sensitive data tagging

-- Classification levels: public, internal, confidential, restricted
-- Data types: pii, phi, pci, proprietary, general

-- Table for organization-level classification rules based on path patterns
CREATE TABLE path_classification_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    pattern VARCHAR(1024) NOT NULL,
    level VARCHAR(32) NOT NULL CHECK (level IN ('public', 'internal', 'confidential', 'restricted')),
    data_types JSONB NOT NULL DEFAULT '["general"]'::jsonb,
    description TEXT,
    is_builtin BOOLEAN DEFAULT false,
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_path_classification_rules_org ON path_classification_rules(org_id);
CREATE INDEX idx_path_classification_rules_enabled ON path_classification_rules(org_id, enabled) WHERE enabled = true;

-- Table for schedule-level classifications (aggregated from paths)
-- Stores the computed classification for each schedule
CREATE TABLE schedule_classifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    level VARCHAR(32) NOT NULL CHECK (level IN ('public', 'internal', 'confidential', 'restricted')),
    data_types JSONB NOT NULL DEFAULT '["general"]'::jsonb,
    auto_classified BOOLEAN DEFAULT true,
    classified_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(schedule_id)
);

CREATE INDEX idx_schedule_classifications_level ON schedule_classifications(level);
CREATE INDEX idx_schedule_classifications_schedule ON schedule_classifications(schedule_id);

-- Table for backup-level classifications (inherited from schedule or paths at backup time)
CREATE TABLE backup_classifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backup_id UUID NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    level VARCHAR(32) NOT NULL CHECK (level IN ('public', 'internal', 'confidential', 'restricted')),
    data_types JSONB NOT NULL DEFAULT '["general"]'::jsonb,
    paths_classified JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(backup_id)
);

CREATE INDEX idx_backup_classifications_level ON backup_classifications(level);
CREATE INDEX idx_backup_classifications_backup ON backup_classifications(backup_id);
CREATE INDEX idx_backup_classifications_schedule ON backup_classifications(schedule_id);

-- Add classification_level to schedules for quick filtering
ALTER TABLE schedules ADD COLUMN classification_level VARCHAR(32) DEFAULT 'public';
ALTER TABLE schedules ADD COLUMN classification_data_types JSONB DEFAULT '["general"]'::jsonb;

CREATE INDEX idx_schedules_classification_level ON schedules(classification_level);

-- Add classification_level to backups for quick filtering
ALTER TABLE backups ADD COLUMN classification_level VARCHAR(32) DEFAULT 'public';
ALTER TABLE backups ADD COLUMN classification_data_types JSONB DEFAULT '["general"]'::jsonb;

CREATE INDEX idx_backups_classification_level ON backups(classification_level);
