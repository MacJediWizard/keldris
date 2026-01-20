-- Migration: Add tags and backup_tags tables for backup organization

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) NOT NULL DEFAULT '#6366f1', -- Hex color (e.g., #6366f1)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, name)
);

CREATE INDEX idx_tags_org ON tags(org_id);
CREATE INDEX idx_tags_name ON tags(name);

CREATE TABLE backup_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backup_id UUID NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(backup_id, tag_id)
);

CREATE INDEX idx_backup_tags_backup ON backup_tags(backup_id);
CREATE INDEX idx_backup_tags_tag ON backup_tags(tag_id);

-- Also add tags to snapshots for more flexibility
CREATE TABLE snapshot_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id VARCHAR(255) NOT NULL,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(snapshot_id, tag_id)
);

CREATE INDEX idx_snapshot_tags_snapshot ON snapshot_tags(snapshot_id);
CREATE INDEX idx_snapshot_tags_tag ON snapshot_tags(tag_id);
