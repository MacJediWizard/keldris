-- Migration: Add snapshot_comments table for notes/comments on snapshots

CREATE TABLE snapshot_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_snapshot_comments_org ON snapshot_comments(org_id);
CREATE INDEX idx_snapshot_comments_snapshot ON snapshot_comments(snapshot_id);
CREATE INDEX idx_snapshot_comments_user ON snapshot_comments(user_id);
CREATE INDEX idx_snapshot_comments_created_at ON snapshot_comments(created_at DESC);
