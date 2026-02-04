-- 076_server_setup.sql
-- First-time server setup wizard state and license management

-- Singleton table for tracking server-wide setup state
-- Only one row exists (id=1), enforced by constraint
CREATE TABLE IF NOT EXISTS server_setup (
    id SERIAL PRIMARY KEY,
    setup_completed BOOLEAN NOT NULL DEFAULT false,
    setup_completed_at TIMESTAMPTZ,
    setup_completed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    current_step VARCHAR(50) NOT NULL DEFAULT 'database',
    completed_steps TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT server_setup_single_row CHECK (id = 1)
);

-- Insert initial setup record
INSERT INTO server_setup (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- License keys table for license management
CREATE TABLE IF NOT EXISTS license_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_key VARCHAR(255) NOT NULL UNIQUE,
    license_type VARCHAR(50) NOT NULL, -- trial, standard, professional, enterprise
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, expired, revoked
    max_agents INTEGER,
    max_repositories INTEGER,
    max_storage_gb INTEGER,
    features JSONB DEFAULT '{}',
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    activated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    company_name VARCHAR(255),
    contact_email VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_license_keys_status ON license_keys(status);
CREATE INDEX IF NOT EXISTS idx_license_keys_type ON license_keys(license_type);
CREATE INDEX IF NOT EXISTS idx_license_keys_expires ON license_keys(expires_at) WHERE expires_at IS NOT NULL;

-- Audit log for server setup actions
CREATE TABLE IF NOT EXISTS server_setup_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action VARCHAR(100) NOT NULL,
    step VARCHAR(50),
    performed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_server_setup_audit_created ON server_setup_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_server_setup_audit_action ON server_setup_audit_log(action);

-- Add password_hash column to users for password-based authentication during setup
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

-- Trigger for updated_at on server_setup
CREATE OR REPLACE FUNCTION update_server_setup_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_server_setup_updated ON server_setup;
CREATE TRIGGER trigger_server_setup_updated
    BEFORE UPDATE ON server_setup
    FOR EACH ROW
    EXECUTE FUNCTION update_server_setup_timestamp();

-- Trigger for updated_at on license_keys
DROP TRIGGER IF EXISTS trigger_license_keys_updated ON license_keys;
CREATE TRIGGER trigger_license_keys_updated
    BEFORE UPDATE ON license_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_server_setup_timestamp();

COMMENT ON TABLE server_setup IS 'Tracks first-time server setup wizard state (singleton table with id=1)';
COMMENT ON TABLE license_keys IS 'License key storage and validation';
COMMENT ON TABLE server_setup_audit_log IS 'Audit log for server setup actions';
COMMENT ON COLUMN server_setup.current_step IS 'Valid steps: database, superuser, smtp, oidc, license, organization, complete';
COMMENT ON COLUMN license_keys.license_type IS 'Valid types: trial, standard, professional, enterprise';
COMMENT ON COLUMN license_keys.status IS 'Valid statuses: active, expired, revoked';
