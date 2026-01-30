-- Migration: Add user management features for admin control

-- Add user status for enable/disable functionality
ALTER TABLE users
ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active',
ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS last_login_ip VARCHAR(45),
ADD COLUMN IF NOT EXISTS failed_login_attempts INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS invited_at TIMESTAMPTZ;

-- User status values: active, disabled, pending, locked
-- active: Normal user account
-- disabled: Administratively disabled
-- pending: Invited but not yet accepted
-- locked: Locked due to failed login attempts

CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_org_status ON users(org_id, status);
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login_at DESC);

-- Create user activity log table for tracking user actions
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    ip_address VARCHAR(45),
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_activity_logs_user ON user_activity_logs(user_id);
CREATE INDEX idx_user_activity_logs_org ON user_activity_logs(org_id);
CREATE INDEX idx_user_activity_logs_action ON user_activity_logs(action);
CREATE INDEX idx_user_activity_logs_created ON user_activity_logs(created_at DESC);
CREATE INDEX idx_user_activity_logs_user_created ON user_activity_logs(user_id, created_at DESC);

-- Create user impersonation log table for audit trail
CREATE TABLE IF NOT EXISTS user_impersonation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    reason TEXT,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    ip_address VARCHAR(45),
    user_agent TEXT
);

CREATE INDEX idx_impersonation_logs_admin ON user_impersonation_logs(admin_user_id);
CREATE INDEX idx_impersonation_logs_target ON user_impersonation_logs(target_user_id);
CREATE INDEX idx_impersonation_logs_org ON user_impersonation_logs(org_id);
CREATE INDEX idx_impersonation_logs_started ON user_impersonation_logs(started_at DESC);

-- Add superuser flag to users table for system-wide admin capabilities
ALTER TABLE users
ADD COLUMN IF NOT EXISTS is_superuser BOOLEAN NOT NULL DEFAULT false;

-- Create index for superuser lookup
CREATE INDEX IF NOT EXISTS idx_users_superuser ON users(is_superuser) WHERE is_superuser = true;
