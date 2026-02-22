-- Migration: Add database connections for MySQL/MariaDB backup

-- Create database_connections table
CREATE TABLE database_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 3306,
    username VARCHAR(255) NOT NULL,
    credentials_encrypted BYTEA NOT NULL,
    ssl_mode VARCHAR(50),
    enabled BOOLEAN NOT NULL DEFAULT true,
    health_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    last_health_check TIMESTAMPTZ,
    last_health_error TEXT,
    version VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_database_type CHECK (type IN ('mysql', 'mariadb')),
    CONSTRAINT valid_health_status CHECK (health_status IN ('healthy', 'unhealthy', 'unknown')),
    CONSTRAINT unique_name_per_org UNIQUE (org_id, name)
);

-- Create indexes for efficient queries
CREATE INDEX idx_database_connections_org ON database_connections(org_id);
CREATE INDEX idx_database_connections_agent ON database_connections(agent_id) WHERE agent_id IS NOT NULL;
CREATE INDEX idx_database_connections_org_enabled ON database_connections(org_id, enabled) WHERE enabled = true;
CREATE INDEX idx_database_connections_health ON database_connections(org_id, health_status);
CREATE INDEX idx_database_connections_type ON database_connections(org_id, type);

-- Add mysql_config column to schedules for MySQL backup configuration
ALTER TABLE schedules ADD COLUMN mysql_config JSONB;
