-- Migration: Add password policies for non-OIDC login

-- Create password_policies table for organization-level password requirements
CREATE TABLE password_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    min_length INT NOT NULL DEFAULT 8,
    require_uppercase BOOLEAN NOT NULL DEFAULT true,
    require_lowercase BOOLEAN NOT NULL DEFAULT true,
    require_number BOOLEAN NOT NULL DEFAULT true,
    require_special BOOLEAN NOT NULL DEFAULT false,
    max_age_days INT DEFAULT NULL, -- NULL means no expiration
    history_count INT NOT NULL DEFAULT 0, -- Number of previous passwords to remember (0 = disabled)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id)
);

-- Add password fields to users table for non-OIDC authentication
ALTER TABLE users
ADD COLUMN password_hash VARCHAR(255),
ADD COLUMN password_changed_at TIMESTAMPTZ,
ADD COLUMN password_expires_at TIMESTAMPTZ,
ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT false;

-- Make oidc_subject nullable for password-only users
ALTER TABLE users
ALTER COLUMN oidc_subject DROP NOT NULL;

-- Create unique index that allows multiple NULLs but unique non-NULL values
DROP INDEX IF EXISTS users_oidc_subject_key;
CREATE UNIQUE INDEX users_oidc_subject_unique ON users(oidc_subject) WHERE oidc_subject IS NOT NULL;

-- Create password history table for tracking previous passwords
CREATE TABLE password_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_password_policies_org ON password_policies(org_id);
CREATE INDEX idx_password_history_user ON password_history(user_id);
CREATE INDEX idx_password_history_created ON password_history(user_id, created_at DESC);
CREATE INDEX idx_users_password_expires ON users(password_expires_at) WHERE password_expires_at IS NOT NULL;
CREATE INDEX idx_users_email_org ON users(email, org_id);
