-- Migration: Add SSO group mappings for OIDC group sync

-- Table to store OIDC group to Keldris org/role mappings
CREATE TABLE sso_group_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    oidc_group_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    auto_create_org BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_org_group UNIQUE (org_id, oidc_group_name)
);

CREATE INDEX idx_sso_group_mappings_org ON sso_group_mappings(org_id);
CREATE INDEX idx_sso_group_mappings_group ON sso_group_mappings(oidc_group_name);

-- Table to track user's SSO groups from their last login
CREATE TABLE user_sso_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    oidc_groups TEXT[] NOT NULL DEFAULT '{}',
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_user_sso_groups UNIQUE (user_id)
);

CREATE INDEX idx_user_sso_groups_user ON user_sso_groups(user_id);

-- Add default role for unmapped groups (per org setting)
ALTER TABLE organizations ADD COLUMN sso_default_role VARCHAR(50);
ALTER TABLE organizations ADD COLUMN sso_auto_create_orgs BOOLEAN DEFAULT false;
