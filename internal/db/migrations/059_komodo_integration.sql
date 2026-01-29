-- 059_komodo_integration.sql
-- Migration: Add Komodo integration tables

-- Komodo integrations (connection to Komodo instances)
CREATE TABLE komodo_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(1024) NOT NULL,
    config_encrypted BYTEA NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_komodo_integrations_org ON komodo_integrations(org_id);
CREATE INDEX idx_komodo_integrations_status ON komodo_integrations(org_id, status);

-- Komodo stacks discovered from Komodo
CREATE TABLE komodo_stacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES komodo_integrations(id) ON DELETE CASCADE,
    komodo_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    server_id VARCHAR(255),
    server_name VARCHAR(255),
    container_count INTEGER DEFAULT 0,
    running_count INTEGER DEFAULT 0,
    last_discovered_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(integration_id, komodo_id)
);

CREATE INDEX idx_komodo_stacks_org ON komodo_stacks(org_id);
CREATE INDEX idx_komodo_stacks_integration ON komodo_stacks(integration_id);
CREATE INDEX idx_komodo_stacks_komodo_id ON komodo_stacks(integration_id, komodo_id);

-- Komodo containers discovered from Komodo
CREATE TABLE komodo_containers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES komodo_integrations(id) ON DELETE CASCADE,
    komodo_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(1024),
    stack_name VARCHAR(255),
    stack_id VARCHAR(255),
    status VARCHAR(50) DEFAULT 'unknown',
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    volumes TEXT[], -- Array of volume mount strings
    labels JSONB DEFAULT '{}',
    backup_enabled BOOLEAN DEFAULT false,
    last_discovered_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(integration_id, komodo_id)
);

CREATE INDEX idx_komodo_containers_org ON komodo_containers(org_id);
CREATE INDEX idx_komodo_containers_integration ON komodo_containers(integration_id);
CREATE INDEX idx_komodo_containers_komodo_id ON komodo_containers(integration_id, komodo_id);
CREATE INDEX idx_komodo_containers_agent ON komodo_containers(agent_id);
CREATE INDEX idx_komodo_containers_stack ON komodo_containers(integration_id, stack_id);
CREATE INDEX idx_komodo_containers_backup ON komodo_containers(org_id, backup_enabled);

-- Komodo webhook events received from Komodo
CREATE TABLE komodo_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES komodo_integrations(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload BYTEA,
    status VARCHAR(50) NOT NULL DEFAULT 'received',
    error_message TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_komodo_webhook_events_org ON komodo_webhook_events(org_id);
CREATE INDEX idx_komodo_webhook_events_integration ON komodo_webhook_events(integration_id);
CREATE INDEX idx_komodo_webhook_events_status ON komodo_webhook_events(org_id, status);
CREATE INDEX idx_komodo_webhook_events_created ON komodo_webhook_events(created_at);
