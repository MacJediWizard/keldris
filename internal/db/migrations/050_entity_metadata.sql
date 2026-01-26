-- 045_entity_metadata.sql
-- Migration: Add user-defined metadata fields to agents, repositories, and schedules

-- Metadata schemas define the available fields per organization
CREATE TABLE metadata_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL, -- agent, repository, schedule
    name VARCHAR(255) NOT NULL,
    field_key VARCHAR(100) NOT NULL, -- unique key for the field within entity_type
    field_type VARCHAR(50) NOT NULL, -- text, number, date, select, boolean
    description TEXT,
    required BOOLEAN DEFAULT false,
    default_value JSONB, -- default value (type-appropriate)
    options JSONB, -- for select type: array of allowed values
    validation JSONB, -- validation rules (min, max, pattern, etc.)
    display_order INTEGER DEFAULT 0, -- for UI ordering
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, entity_type, field_key)
);

-- Add metadata column to agents
ALTER TABLE agents ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Add metadata column to repositories
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Add metadata column to schedules
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Indexes for metadata schemas
CREATE INDEX idx_metadata_schemas_org ON metadata_schemas(org_id);
CREATE INDEX idx_metadata_schemas_entity ON metadata_schemas(org_id, entity_type);
CREATE INDEX idx_metadata_schemas_key ON metadata_schemas(org_id, entity_type, field_key);

-- Indexes for metadata search on entities (using GIN for JSONB)
CREATE INDEX idx_agents_metadata ON agents USING GIN(metadata);
CREATE INDEX idx_repositories_metadata ON repositories USING GIN(metadata);
CREATE INDEX idx_schedules_metadata ON schedules USING GIN(metadata);
