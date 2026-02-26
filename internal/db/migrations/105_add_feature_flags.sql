-- Migration: Add feature_flags column to organizations
-- Used by branding handler to check per-org feature flags

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS feature_flags JSONB DEFAULT '{}';
