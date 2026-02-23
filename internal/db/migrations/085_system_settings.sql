-- Migration: Add system settings tables for SMTP, OIDC, storage, and security configuration

-- Replace global system_settings (from 084) with org-scoped version
DROP TABLE IF EXISTS system_settings CASCADE;
CREATE TABLE system_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    setting_key VARCHAR(64) NOT NULL,
    setting_value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, setting_key)
);

-- Index for efficient org-based lookups
CREATE INDEX idx_system_settings_org ON system_settings(org_id);
CREATE INDEX idx_system_settings_key ON system_settings(org_id, setting_key);

-- Audit log for tracking settings changes (compliance/security)
CREATE TABLE settings_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    setting_key VARCHAR(64) NOT NULL,
    old_value JSONB,
    new_value JSONB NOT NULL,
    changed_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET
);

-- Indexes for audit log queries
CREATE INDEX idx_settings_audit_org ON settings_audit_log(org_id);
CREATE INDEX idx_settings_audit_key ON settings_audit_log(org_id, setting_key);
CREATE INDEX idx_settings_audit_changed_at ON settings_audit_log(changed_at DESC);
CREATE INDEX idx_settings_audit_user ON settings_audit_log(changed_by);

-- Insert default settings for existing organizations
-- SMTP defaults (disabled by default)
INSERT INTO system_settings (org_id, setting_key, setting_value, description)
SELECT id, 'smtp',
    '{"host": "", "port": 587, "username": "", "password": "", "from_email": "", "from_name": "", "encryption": "starttls", "enabled": false, "skip_tls_verify": false, "connection_timeout_seconds": 30}'::jsonb,
    'SMTP email server configuration'
FROM organizations;

-- OIDC defaults (disabled by default)
INSERT INTO system_settings (org_id, setting_key, setting_value, description)
SELECT id, 'oidc',
    '{"enabled": false, "issuer": "", "client_id": "", "client_secret": "", "redirect_url": "", "scopes": ["openid", "profile", "email"], "auto_create_users": false, "default_role": "member", "allowed_domains": [], "require_email_verification": true}'::jsonb,
    'OIDC single sign-on configuration'
FROM organizations;

-- Storage defaults
INSERT INTO system_settings (org_id, setting_key, setting_value, description)
SELECT id, 'storage_defaults',
    '{"default_retention_days": 30, "max_retention_days": 365, "default_storage_backend": "local", "max_backup_size_gb": 100, "enable_compression": true, "compression_level": 6, "default_encryption_method": "aes256", "prune_schedule": "0 2 * * *", "auto_prune_enabled": true}'::jsonb,
    'Default storage and retention policies'
FROM organizations;

-- Security settings
INSERT INTO system_settings (org_id, setting_key, setting_value, description)
SELECT id, 'security',
    '{"session_timeout_minutes": 480, "max_concurrent_sessions": 5, "require_mfa": false, "mfa_grace_period_days": 7, "allowed_ip_ranges": [], "blocked_ip_ranges": [], "failed_login_lockout_attempts": 5, "failed_login_lockout_minutes": 30, "api_key_expiration_days": 0, "enable_audit_logging": true, "audit_log_retention_days": 90, "force_https": true, "allow_password_login": true}'::jsonb,
    'Security and access control settings'
FROM organizations;

-- Create trigger to update updated_at on system_settings
CREATE OR REPLACE FUNCTION update_system_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_system_settings_updated_at();
