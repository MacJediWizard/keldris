-- 059_superuser.sql
-- Add global superuser support for system-wide administration

-- Add is_superuser column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_superuser BOOLEAN DEFAULT false NOT NULL;

-- Create index for superuser lookups
CREATE INDEX IF NOT EXISTS idx_users_is_superuser ON users(is_superuser) WHERE is_superuser = true;

-- Create superuser_audit_logs table for tracking superuser-specific actions
-- These are separate from org-scoped audit logs for security isolation
CREATE TABLE IF NOT EXISTS superuser_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    superuser_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(100) NOT NULL,
    target_id UUID,
    target_org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    impersonated_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_superuser_audit_logs_superuser ON superuser_audit_logs(superuser_id);
CREATE INDEX idx_superuser_audit_logs_action ON superuser_audit_logs(action);
CREATE INDEX idx_superuser_audit_logs_target_org ON superuser_audit_logs(target_org_id);
CREATE INDEX idx_superuser_audit_logs_created ON superuser_audit_logs(created_at DESC);

-- Create system_settings table for global system configuration
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default system settings
INSERT INTO system_settings (key, value, description) VALUES
    ('allow_new_registrations', 'true', 'Allow new users to register'),
    ('default_org_settings', '{}', 'Default settings applied to new organizations'),
    ('maintenance_mode', 'false', 'Enable system-wide maintenance mode')
ON CONFLICT (key) DO NOTHING;
