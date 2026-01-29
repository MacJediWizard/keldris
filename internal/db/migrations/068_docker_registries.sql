-- Migration: Add Docker registry credentials management

-- Create docker_registries table
CREATE TABLE docker_registries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    url VARCHAR(512) NOT NULL,
    credentials_encrypted BYTEA NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    enabled BOOLEAN NOT NULL DEFAULT true,
    health_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    last_health_check TIMESTAMPTZ,
    last_health_error TEXT,
    credentials_rotated_at TIMESTAMPTZ,
    credentials_expires_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_registry_type CHECK (type IN ('dockerhub', 'gcr', 'ecr', 'acr', 'ghcr', 'private')),
    CONSTRAINT valid_health_status CHECK (health_status IN ('healthy', 'unhealthy', 'unknown')),
    CONSTRAINT unique_name_per_org UNIQUE (org_id, name)
);

-- Create indexes for efficient queries
CREATE INDEX idx_docker_registries_org ON docker_registries(org_id);
CREATE INDEX idx_docker_registries_org_enabled ON docker_registries(org_id, enabled) WHERE enabled = true;
CREATE INDEX idx_docker_registries_default ON docker_registries(org_id, is_default) WHERE is_default = true;
CREATE INDEX idx_docker_registries_health ON docker_registries(org_id, health_status);
CREATE INDEX idx_docker_registries_credentials_expiry ON docker_registries(credentials_expires_at) WHERE credentials_expires_at IS NOT NULL;

-- Create docker_registry_audit_log table for credential access auditing
CREATE TABLE docker_registry_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    registry_id UUID NOT NULL REFERENCES docker_registries(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB DEFAULT '{}',
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_audit_action CHECK (action IN ('login', 'create', 'update', 'delete', 'rotate_credentials', 'health_check', 'view_credentials'))
);

-- Create index for audit log queries
CREATE INDEX idx_docker_registry_audit_org ON docker_registry_audit_log(org_id);
CREATE INDEX idx_docker_registry_audit_registry ON docker_registry_audit_log(registry_id);
CREATE INDEX idx_docker_registry_audit_time ON docker_registry_audit_log(created_at DESC);

-- Ensure only one default registry per org
CREATE UNIQUE INDEX idx_docker_registries_single_default ON docker_registries(org_id) WHERE is_default = true;
