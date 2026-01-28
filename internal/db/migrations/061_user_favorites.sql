-- Migration: Add user_favorites table for starring frequently used items

CREATE TABLE user_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_user_favorites_user_org ON user_favorites(user_id, org_id);
CREATE INDEX idx_user_favorites_entity_type ON user_favorites(user_id, org_id, entity_type);
CREATE INDEX idx_user_favorites_entity ON user_favorites(entity_type, entity_id);

-- Unique constraint: one favorite per user per entity
CREATE UNIQUE INDEX idx_user_favorites_unique ON user_favorites(user_id, entity_type, entity_id);
