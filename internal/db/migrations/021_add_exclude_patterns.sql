-- Migration: Add exclude patterns library table
-- This table stores both built-in and custom exclude patterns for organizations

CREATE TABLE IF NOT EXISTS exclude_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    patterns JSONB NOT NULL DEFAULT '[]',
    category VARCHAR(50) NOT NULL,
    is_builtin BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for org-specific queries (custom patterns)
CREATE INDEX idx_exclude_patterns_org ON exclude_patterns(org_id) WHERE org_id IS NOT NULL;

-- Index for category filtering
CREATE INDEX idx_exclude_patterns_category ON exclude_patterns(category);

-- Index for built-in patterns lookup
CREATE INDEX idx_exclude_patterns_builtin ON exclude_patterns(is_builtin) WHERE is_builtin = true;

-- Unique constraint: org-specific patterns must have unique names within org
CREATE UNIQUE INDEX idx_exclude_patterns_org_name ON exclude_patterns(org_id, name) WHERE org_id IS NOT NULL;

-- Unique constraint: built-in patterns must have unique names globally
CREATE UNIQUE INDEX idx_exclude_patterns_builtin_name ON exclude_patterns(name) WHERE is_builtin = true;
