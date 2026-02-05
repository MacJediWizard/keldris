-- +goose Up
-- License tiers for organizations

CREATE TABLE IF NOT EXISTS organization_licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tier VARCHAR(50) NOT NULL DEFAULT 'free',
    activated_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT org_licenses_tier_check CHECK (tier IN ('free', 'pro', 'enterprise')),
    CONSTRAINT org_licenses_org_unique UNIQUE (org_id)
);

-- Index for looking up license by org
CREATE INDEX IF NOT EXISTS idx_org_licenses_org_id ON organization_licenses(org_id);

-- Index for finding expired licenses
CREATE INDEX IF NOT EXISTS idx_org_licenses_expires_at ON organization_licenses(expires_at) WHERE expires_at IS NOT NULL;

-- Audit log for license changes
CREATE TABLE IF NOT EXISTS license_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    old_tier VARCHAR(50),
    new_tier VARCHAR(50),
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for audit logs by org
CREATE INDEX IF NOT EXISTS idx_license_audit_logs_org_id ON license_audit_logs(org_id);

-- Index for audit logs by time
CREATE INDEX IF NOT EXISTS idx_license_audit_logs_created_at ON license_audit_logs(created_at);

-- +goose Down
DROP TABLE IF EXISTS license_audit_logs;
DROP TABLE IF EXISTS organization_licenses;
