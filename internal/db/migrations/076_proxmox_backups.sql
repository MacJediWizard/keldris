-- 076_proxmox_backups.sql
-- Add support for Proxmox VM and container backups

-- Add proxmox_options column to schedules for Proxmox-specific backup configuration
ALTER TABLE schedules ADD COLUMN proxmox_options JSONB;

-- Add proxmox_info column to agents for storing detected Proxmox VMs/containers
ALTER TABLE agents ADD COLUMN proxmox_info JSONB;

-- Create proxmox_connections table for storing Proxmox API connection configurations
CREATE TABLE proxmox_connections (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER DEFAULT 8006,
    node VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL,
    token_id VARCHAR(255),
    token_secret_encrypted BYTEA,
    verify_ssl BOOLEAN DEFAULT TRUE,
    enabled BOOLEAN DEFAULT TRUE,
    last_connected_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for looking up connections by organization
CREATE INDEX idx_proxmox_connections_org ON proxmox_connections(org_id);

-- Index for looking up enabled connections
CREATE INDEX idx_proxmox_connections_enabled ON proxmox_connections(org_id, enabled) WHERE enabled = TRUE;

-- Add unique constraint on name within organization
CREATE UNIQUE INDEX idx_proxmox_connections_org_name ON proxmox_connections(org_id, name);

COMMENT ON TABLE proxmox_connections IS 'Proxmox VE API connection configurations for VM/container backup';
COMMENT ON COLUMN proxmox_connections.host IS 'Proxmox VE server hostname or IP address';
COMMENT ON COLUMN proxmox_connections.port IS 'Proxmox VE API port (default 8006)';
COMMENT ON COLUMN proxmox_connections.node IS 'Proxmox node name for backup operations';
COMMENT ON COLUMN proxmox_connections.username IS 'API username (e.g., root@pam or user@pve)';
COMMENT ON COLUMN proxmox_connections.token_id IS 'API token ID for token-based authentication';
COMMENT ON COLUMN proxmox_connections.token_secret_encrypted IS 'Encrypted API token secret';
COMMENT ON COLUMN proxmox_connections.verify_ssl IS 'Whether to verify SSL certificates';
COMMENT ON COLUMN schedules.proxmox_options IS 'Proxmox-specific backup options as JSON';
COMMENT ON COLUMN agents.proxmox_info IS 'Detected Proxmox VMs and containers as JSON';
