-- Migration: Add rate limiting configuration and tracking tables

-- Rate limit configurations per endpoint
CREATE TABLE rate_limit_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    requests_per_period INT NOT NULL DEFAULT 100,
    period_seconds INT NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, endpoint)
);

-- Track blocked requests for stats
CREATE TABLE blocked_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    ip_address TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    user_agent TEXT,
    blocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT NOT NULL DEFAULT 'rate_limit'
);

-- IP bans for repeat offenders
CREATE TABLE ip_bans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    ip_address TEXT NOT NULL,
    reason TEXT NOT NULL,
    ban_count INT NOT NULL DEFAULT 0,
    banned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    banned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_rate_limit_configs_org ON rate_limit_configs(org_id);
CREATE INDEX idx_blocked_requests_org_time ON blocked_requests(org_id, blocked_at DESC);
CREATE INDEX idx_blocked_requests_ip ON blocked_requests(ip_address, blocked_at DESC);
CREATE INDEX idx_blocked_requests_endpoint ON blocked_requests(endpoint, blocked_at DESC);
CREATE INDEX idx_ip_bans_org ON ip_bans(org_id);
CREATE INDEX idx_ip_bans_ip ON ip_bans(ip_address);
CREATE INDEX idx_ip_bans_expires ON ip_bans(expires_at) WHERE expires_at IS NOT NULL;
