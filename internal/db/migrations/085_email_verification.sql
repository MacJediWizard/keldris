-- Migration: Add email verification for non-OIDC users

-- Add email verification fields to users table
ALTER TABLE users
ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

-- OIDC users are automatically verified (they authenticated via identity provider)
UPDATE users SET email_verified = true WHERE oidc_subject IS NOT NULL;

-- Create email verification tokens table
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(token_hash)
);

-- Indexes for efficient token lookup and cleanup
CREATE INDEX idx_email_verification_tokens_user ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verification_tokens_hash ON email_verification_tokens(token_hash) WHERE used_at IS NULL;
CREATE INDEX idx_email_verification_tokens_expires ON email_verification_tokens(expires_at) WHERE used_at IS NULL;
CREATE INDEX idx_users_email_verified ON users(email_verified) WHERE email_verified = false;

-- Add admin bypass flag to system_settings for email verification requirement
-- This will be stored as part of security settings JSON
COMMENT ON TABLE email_verification_tokens IS 'Stores email verification tokens for non-OIDC users. Tokens expire after a configurable period.';
