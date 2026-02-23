-- 076_invite_enhancements.sql
-- Migration: Add invite enhancement columns for resend tracking

-- Add resend tracking columns to org_invitations
ALTER TABLE org_invitations ADD COLUMN IF NOT EXISTS resent_at TIMESTAMPTZ;
ALTER TABLE org_invitations ADD COLUMN IF NOT EXISTS resent_count INTEGER DEFAULT 0;

-- Index for finding invitations by ID quickly
CREATE INDEX IF NOT EXISTS idx_org_invitations_id ON org_invitations(id);

-- Add partial index for pending invitations (not accepted)
CREATE INDEX IF NOT EXISTS idx_org_invitations_pending ON org_invitations(org_id, email)
    WHERE accepted_at IS NULL;
