-- Config Templates table for export/import and template library
CREATE TABLE IF NOT EXISTS config_templates (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('schedule', 'agent', 'repository', 'bundle')),
    visibility VARCHAR(50) NOT NULL DEFAULT 'organization' CHECK (visibility IN ('private', 'organization', 'public')),
    tags JSONB DEFAULT '[]',
    config JSONB NOT NULL,
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Indexes for common queries
CREATE INDEX idx_config_templates_org_id ON config_templates(org_id);
CREATE INDEX idx_config_templates_type ON config_templates(type);
CREATE INDEX idx_config_templates_visibility ON config_templates(visibility);
CREATE INDEX idx_config_templates_created_by_id ON config_templates(created_by_id);
CREATE INDEX idx_config_templates_usage_count ON config_templates(usage_count DESC);

-- Index for public templates query
CREATE INDEX idx_config_templates_public ON config_templates(visibility, usage_count DESC) WHERE visibility = 'public';

-- Index for tags search using GIN
CREATE INDEX idx_config_templates_tags ON config_templates USING GIN (tags);
