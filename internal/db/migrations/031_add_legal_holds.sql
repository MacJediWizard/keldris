-- Migration: Add legal_holds table for legal discovery hold functionality

CREATE TABLE legal_holds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    reason TEXT NOT NULL,
    placed_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient org-based queries
CREATE INDEX idx_legal_holds_org ON legal_holds(org_id);

-- Index for efficient snapshot lookups
CREATE INDEX idx_legal_holds_snapshot ON legal_holds(snapshot_id);

-- Index for user lookup
CREATE INDEX idx_legal_holds_placed_by ON legal_holds(placed_by);

-- Index for sorting by creation time
CREATE INDEX idx_legal_holds_created_at ON legal_holds(created_at DESC);

-- Unique constraint: one hold per snapshot per org
CREATE UNIQUE INDEX idx_legal_holds_org_snapshot ON legal_holds(org_id, snapshot_id);
