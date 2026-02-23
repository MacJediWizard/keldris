-- 012_agent_registration_codes.sql
-- Add agent registration codes for 2FA registration flow

CREATE TABLE agent_registration_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(8) NOT NULL,
    hostname VARCHAR(255),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    used_by_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_registration_codes_org ON agent_registration_codes(org_id);
CREATE INDEX idx_registration_codes_code ON agent_registration_codes(code);
CREATE INDEX idx_registration_codes_expires ON agent_registration_codes(expires_at);
CREATE INDEX idx_registration_codes_created_by ON agent_registration_codes(created_by);

-- Add unique constraint on unused codes per org
-- This prevents duplicate codes while allowing reuse after they are consumed
CREATE UNIQUE INDEX idx_registration_codes_active_code ON agent_registration_codes(org_id, code)
    WHERE used_at IS NULL;
