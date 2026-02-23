-- Migration: Add lifecycle_policies table for automated snapshot retention management

CREATE TABLE lifecycle_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- active, draft, disabled
    rules JSONB NOT NULL DEFAULT '[]',
    repository_ids JSONB, -- Array of repository UUIDs (null = all)
    schedule_ids JSONB, -- Array of schedule UUIDs (null = all)
    last_evaluated_at TIMESTAMPTZ,
    last_deletion_at TIMESTAMPTZ,
    deletion_count BIGINT NOT NULL DEFAULT 0,
    bytes_reclaimed BIGINT NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient org-based queries
CREATE INDEX idx_lifecycle_policies_org ON lifecycle_policies(org_id);

-- Index for status queries
CREATE INDEX idx_lifecycle_policies_status ON lifecycle_policies(status);

-- Index for finding active policies
CREATE INDEX idx_lifecycle_policies_active ON lifecycle_policies(org_id, status) WHERE status = 'active';

-- Unique constraint: one policy name per org
CREATE UNIQUE INDEX idx_lifecycle_policies_org_name ON lifecycle_policies(org_id, name);

-- Table for audit logging of lifecycle deletions
CREATE TABLE lifecycle_deletion_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES lifecycle_policies(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(255) NOT NULL,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    deleted_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deleted_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for finding deletions by policy
CREATE INDEX idx_lifecycle_deletion_events_policy ON lifecycle_deletion_events(policy_id);

-- Index for finding deletions by org and time
CREATE INDEX idx_lifecycle_deletion_events_org_time ON lifecycle_deletion_events(org_id, deleted_at DESC);

-- Index for finding deletions by repository
CREATE INDEX idx_lifecycle_deletion_events_repo ON lifecycle_deletion_events(repository_id);

-- Index for finding deletions by snapshot
CREATE INDEX idx_lifecycle_deletion_events_snapshot ON lifecycle_deletion_events(snapshot_id);
