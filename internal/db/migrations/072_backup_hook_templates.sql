-- Backup Hook Templates
-- Stores custom backup hook templates created by users

CREATE TABLE IF NOT EXISTS backup_hook_templates (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    service_type VARCHAR(100) NOT NULL,
    icon VARCHAR(50),
    tags JSONB DEFAULT '[]',
    variables JSONB DEFAULT '[]',
    scripts JSONB NOT NULL,
    visibility VARCHAR(20) NOT NULL DEFAULT 'private' CHECK (visibility IN ('built_in', 'private', 'organization')),
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_backup_hook_templates_org_id ON backup_hook_templates(org_id);
CREATE INDEX IF NOT EXISTS idx_backup_hook_templates_created_by_id ON backup_hook_templates(created_by_id);
CREATE INDEX IF NOT EXISTS idx_backup_hook_templates_service_type ON backup_hook_templates(service_type);
CREATE INDEX IF NOT EXISTS idx_backup_hook_templates_visibility ON backup_hook_templates(visibility);
CREATE INDEX IF NOT EXISTS idx_backup_hook_templates_tags ON backup_hook_templates USING GIN(tags);

-- Unique constraint for user templates
CREATE UNIQUE INDEX IF NOT EXISTS idx_backup_hook_templates_org_name
    ON backup_hook_templates(org_id, name)
    WHERE visibility != 'built_in';

-- Comments
COMMENT ON TABLE backup_hook_templates IS 'Stores custom backup hook templates for pre/post backup scripts';
COMMENT ON COLUMN backup_hook_templates.service_type IS 'Type of service this template is for (e.g., postgresql, mysql, mongodb)';
COMMENT ON COLUMN backup_hook_templates.variables IS 'JSON array of customizable variables for the template';
COMMENT ON COLUMN backup_hook_templates.scripts IS 'JSON object containing pre_backup, post_success, post_failure, post_always scripts';
COMMENT ON COLUMN backup_hook_templates.visibility IS 'Who can see this template: built_in (system), private (creator only), organization (org members)';
