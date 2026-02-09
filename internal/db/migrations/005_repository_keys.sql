-- 005_repository_keys.sql
-- Migration: Add repository encryption keys table

CREATE TABLE repository_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    encrypted_key BYTEA NOT NULL,
    escrow_enabled BOOLEAN DEFAULT false,
    escrow_encrypted_key BYTEA,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(repository_id)
);

CREATE INDEX idx_repository_keys_repository ON repository_keys(repository_id);
