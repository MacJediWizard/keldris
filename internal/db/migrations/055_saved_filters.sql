-- Migration: Add saved_filters table for dashboard filter persistence

CREATE TABLE saved_filters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    filters JSONB NOT NULL DEFAULT '{}',
    shared BOOLEAN NOT NULL DEFAULT FALSE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_saved_filters_user_org ON saved_filters(user_id, org_id);
CREATE INDEX idx_saved_filters_entity_type ON saved_filters(entity_type);
CREATE INDEX idx_saved_filters_shared ON saved_filters(org_id, shared) WHERE shared = TRUE;

-- Unique constraint: one default filter per user per entity type
CREATE UNIQUE INDEX idx_saved_filters_default ON saved_filters(user_id, org_id, entity_type) WHERE is_default = TRUE;
