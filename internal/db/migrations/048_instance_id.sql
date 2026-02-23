-- Migration: Add server_settings table for instance ID and other key/value settings

CREATE TABLE IF NOT EXISTS server_settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
