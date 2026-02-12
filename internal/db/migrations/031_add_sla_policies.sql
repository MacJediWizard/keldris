-- Migration: Add SLA policy tracking tables

CREATE TABLE sla_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    target_rpo_hours DOUBLE PRECISION NOT NULL,
    target_rto_hours DOUBLE PRECISION NOT NULL,
    target_success_rate DOUBLE PRECISION NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sla_policies_org ON sla_policies(org_id);

CREATE TABLE sla_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES sla_policies(id) ON DELETE CASCADE,
    rpo_hours DOUBLE PRECISION NOT NULL,
    rto_hours DOUBLE PRECISION NOT NULL,
    success_rate DOUBLE PRECISION NOT NULL,
    compliant BOOLEAN NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sla_status_history_policy ON sla_status_history(policy_id);
CREATE INDEX idx_sla_status_history_calculated ON sla_status_history(policy_id, calculated_at DESC);
