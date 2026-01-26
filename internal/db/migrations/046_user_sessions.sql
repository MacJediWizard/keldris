-- Migration: Add user sessions table for session management

-- Create user_sessions table
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash VARCHAR(64) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMPTZ
);

-- Index for finding sessions by user
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);

-- Index for finding sessions by token hash (for validation)
CREATE INDEX idx_user_sessions_token_hash ON user_sessions(session_token_hash) WHERE revoked = false;

-- Index for finding active sessions
CREATE INDEX idx_user_sessions_active ON user_sessions(user_id, revoked, expires_at)
WHERE revoked = false;

-- Index for cleanup of expired sessions
CREATE INDEX idx_user_sessions_expires ON user_sessions(expires_at)
WHERE revoked = false AND expires_at IS NOT NULL;
