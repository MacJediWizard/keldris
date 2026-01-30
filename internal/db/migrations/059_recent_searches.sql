-- Migration: Add recent_searches table for global search history

CREATE TABLE recent_searches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    query VARCHAR(500) NOT NULL,
    types TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_recent_searches_user_org ON recent_searches(user_id, org_id);
CREATE INDEX idx_recent_searches_created_at ON recent_searches(created_at DESC);

-- Unique constraint to prevent duplicate queries
CREATE UNIQUE INDEX idx_recent_searches_unique_query ON recent_searches(user_id, org_id, query);
