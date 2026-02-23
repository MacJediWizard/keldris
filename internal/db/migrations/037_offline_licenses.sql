-- Migration: Add offline license storage for air-gap deployments

CREATE TABLE offline_licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id VARCHAR(255) NOT NULL,
    tier VARCHAR(50) NOT NULL,
    license_data BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_offline_licenses_org ON offline_licenses(org_id);
CREATE INDEX idx_offline_licenses_expires ON offline_licenses(expires_at);
