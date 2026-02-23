-- Migration: Add recent items tracking for quick access to recently viewed pages

-- Create recent_items table
CREATE TABLE recent_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Track what was viewed
    item_type VARCHAR(50) NOT NULL,
    item_id UUID NOT NULL,
    item_name VARCHAR(255) NOT NULL,

    -- Navigation metadata
    page_path VARCHAR(500) NOT NULL,

    -- Timestamps
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT valid_item_type CHECK (item_type IN ('agent', 'repository', 'schedule', 'backup', 'policy', 'snapshot')),
    -- Only one entry per user+type+item combination (update viewed_at on revisit)
    UNIQUE(org_id, user_id, item_type, item_id)
);

-- Indexes for efficient queries
CREATE INDEX idx_recent_items_user ON recent_items(org_id, user_id, viewed_at DESC);
CREATE INDEX idx_recent_items_user_type ON recent_items(org_id, user_id, item_type, viewed_at DESC);
CREATE INDEX idx_recent_items_org ON recent_items(org_id);
