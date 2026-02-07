-- Migration: Add password reset tokens for non-OIDC users

-- Create password_reset_tokens table
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL, -- SHA-256 hash of the token
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient lookups
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at);

-- Rate limiting table for password reset requests
CREATE TABLE password_reset_rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier VARCHAR(255) NOT NULL, -- email or IP address
    identifier_type VARCHAR(20) NOT NULL, -- 'email' or 'ip'
    request_count INT NOT NULL DEFAULT 1,
    window_start TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Unique constraint for rate limit lookups
CREATE UNIQUE INDEX idx_password_reset_rate_limits_identifier ON password_reset_rate_limits(identifier, identifier_type);
CREATE INDEX idx_password_reset_rate_limits_window ON password_reset_rate_limits(window_start);

-- Cleanup function for expired tokens and old rate limit entries
-- Tokens older than 24 hours and rate limit entries older than 1 hour should be cleaned up
