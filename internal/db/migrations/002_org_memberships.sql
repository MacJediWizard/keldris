-- 002_org_memberships.sql
-- Migration: Add multi-organization support with role-based access control

-- Create org_memberships table to support users belonging to multiple orgs
CREATE TABLE org_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, org_id)
);

-- Roles: owner, admin, member, readonly
-- owner: Full control, can delete org, manage all members
-- admin: Can manage members (except owner), manage all resources
-- member: Can create and manage own resources
-- readonly: View-only access

CREATE INDEX idx_org_memberships_user ON org_memberships(user_id);
CREATE INDEX idx_org_memberships_org ON org_memberships(org_id);
CREATE INDEX idx_org_memberships_role ON org_memberships(role);

-- Create org_invitations table for pending invites
CREATE TABLE org_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    token VARCHAR(255) UNIQUE NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_org_invitations_org ON org_invitations(org_id);
CREATE INDEX idx_org_invitations_email ON org_invitations(email);
CREATE INDEX idx_org_invitations_token ON org_invitations(token);

-- Migrate existing users to org_memberships
-- Users with 'admin' role become 'owner', others become 'member'
INSERT INTO org_memberships (user_id, org_id, role, created_at, updated_at)
SELECT
    id as user_id,
    org_id,
    CASE
        WHEN role = 'admin' THEN 'owner'
        WHEN role = 'viewer' THEN 'readonly'
        ELSE 'member'
    END as role,
    created_at,
    updated_at
FROM users
ON CONFLICT (user_id, org_id) DO NOTHING;

-- Add settings column to organizations
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';
