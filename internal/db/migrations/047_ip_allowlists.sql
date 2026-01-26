-- Migration: Add IP allowlists for access control

-- Create ip_allowlists table
CREATE TABLE ip_allowlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    cidr VARCHAR(50) NOT NULL,
    description VARCHAR(255),
    type VARCHAR(20) NOT NULL DEFAULT 'both' CHECK (type IN ('ui', 'agent', 'both')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create ip_allowlist_settings table for org-level settings
CREATE TABLE ip_allowlist_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    enforce_for_ui BOOLEAN NOT NULL DEFAULT true,
    enforce_for_agent BOOLEAN NOT NULL DEFAULT true,
    allow_admin_bypass BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create ip_blocked_attempts table for audit logging blocked attempts
CREATE TABLE ip_blocked_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    ip_address VARCHAR(50) NOT NULL,
    request_type VARCHAR(20) NOT NULL,
    path VARCHAR(255),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ip_allowlists_org ON ip_allowlists(org_id);
CREATE INDEX idx_ip_allowlists_org_enabled ON ip_allowlists(org_id, enabled) WHERE enabled = true;
CREATE INDEX idx_ip_allowlists_type ON ip_allowlists(org_id, type) WHERE enabled = true;
CREATE INDEX idx_ip_blocked_attempts_org ON ip_blocked_attempts(org_id);
CREATE INDEX idx_ip_blocked_attempts_created ON ip_blocked_attempts(org_id, created_at);
CREATE INDEX idx_ip_blocked_attempts_ip ON ip_blocked_attempts(org_id, ip_address);
