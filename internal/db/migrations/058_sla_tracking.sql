-- Migration: Add SLA definitions and compliance tracking

-- Create sla_definitions table
CREATE TABLE sla_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    -- RPO: Recovery Point Objective (max time between backups in minutes)
    rpo_minutes INTEGER,
    -- RTO: Recovery Time Objective (max restore time in minutes)
    rto_minutes INTEGER,
    -- Uptime: percentage target (e.g., 99.9)
    uptime_percentage DECIMAL(5,2),
    -- Scope: can apply to agents, repositories, or both
    scope VARCHAR(50) NOT NULL DEFAULT 'agent',
    -- Whether this SLA is actively enforced
    active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_scope CHECK (scope IN ('agent', 'repository', 'organization')),
    CONSTRAINT valid_uptime CHECK (uptime_percentage IS NULL OR (uptime_percentage >= 0 AND uptime_percentage <= 100)),
    CONSTRAINT has_at_least_one_target CHECK (rpo_minutes IS NOT NULL OR rto_minutes IS NOT NULL OR uptime_percentage IS NOT NULL)
);

-- Create sla_assignments table to assign SLAs to agents/repos
CREATE TABLE sla_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sla_id UUID NOT NULL REFERENCES sla_definitions(id) ON DELETE CASCADE,
    -- Either agent_id or repository_id should be set, but not both
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT one_target CHECK (
        (agent_id IS NOT NULL AND repository_id IS NULL) OR
        (agent_id IS NULL AND repository_id IS NOT NULL)
    ),
    UNIQUE(sla_id, agent_id),
    UNIQUE(sla_id, repository_id)
);

-- Create sla_compliance table to track compliance records
CREATE TABLE sla_compliance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sla_id UUID NOT NULL REFERENCES sla_definitions(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    -- Period being evaluated
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    -- Compliance metrics
    rpo_compliant BOOLEAN,
    rpo_actual_minutes INTEGER,
    rpo_breaches INTEGER DEFAULT 0,
    rto_compliant BOOLEAN,
    rto_actual_minutes INTEGER,
    rto_breaches INTEGER DEFAULT 0,
    uptime_compliant BOOLEAN,
    uptime_actual_percentage DECIMAL(5,2),
    uptime_downtime_minutes INTEGER DEFAULT 0,
    -- Overall compliance for this period
    is_compliant BOOLEAN NOT NULL,
    -- Additional context
    notes TEXT,
    calculated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create sla_breaches table to track individual breach events
CREATE TABLE sla_breaches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sla_id UUID NOT NULL REFERENCES sla_definitions(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    -- Type of breach
    breach_type VARCHAR(50) NOT NULL,
    -- Expected vs actual values
    expected_value DECIMAL(10,2),
    actual_value DECIMAL(10,2),
    -- When the breach occurred
    breach_start TIMESTAMPTZ NOT NULL,
    breach_end TIMESTAMPTZ,
    -- Breach duration in minutes (for resolved breaches)
    duration_minutes INTEGER,
    -- Whether the breach has been acknowledged/resolved
    acknowledged BOOLEAN NOT NULL DEFAULT false,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_at TIMESTAMPTZ,
    -- Additional context
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_breach_type CHECK (breach_type IN ('rpo', 'rto', 'uptime'))
);

-- Create indexes for efficient queries
CREATE INDEX idx_sla_definitions_org ON sla_definitions(org_id);
CREATE INDEX idx_sla_definitions_active ON sla_definitions(org_id, active) WHERE active = true;

CREATE INDEX idx_sla_assignments_org ON sla_assignments(org_id);
CREATE INDEX idx_sla_assignments_sla ON sla_assignments(sla_id);
CREATE INDEX idx_sla_assignments_agent ON sla_assignments(agent_id) WHERE agent_id IS NOT NULL;
CREATE INDEX idx_sla_assignments_repo ON sla_assignments(repository_id) WHERE repository_id IS NOT NULL;

CREATE INDEX idx_sla_compliance_org ON sla_compliance(org_id);
CREATE INDEX idx_sla_compliance_sla ON sla_compliance(sla_id);
CREATE INDEX idx_sla_compliance_period ON sla_compliance(org_id, period_start, period_end);
CREATE INDEX idx_sla_compliance_agent ON sla_compliance(agent_id) WHERE agent_id IS NOT NULL;
CREATE INDEX idx_sla_compliance_repo ON sla_compliance(repository_id) WHERE repository_id IS NOT NULL;

CREATE INDEX idx_sla_breaches_org ON sla_breaches(org_id);
CREATE INDEX idx_sla_breaches_sla ON sla_breaches(sla_id);
CREATE INDEX idx_sla_breaches_unresolved ON sla_breaches(org_id, resolved) WHERE resolved = false;
CREATE INDEX idx_sla_breaches_agent ON sla_breaches(agent_id) WHERE agent_id IS NOT NULL;
CREATE INDEX idx_sla_breaches_repo ON sla_breaches(repository_id) WHERE repository_id IS NOT NULL;
CREATE INDEX idx_sla_breaches_time ON sla_breaches(breach_start);
