-- Migration: Add agent debug mode for verbose logging
-- This migration adds fields to enable debug/verbose logging on agents remotely

-- Add debug mode columns to agents table
ALTER TABLE agents ADD COLUMN IF NOT EXISTS debug_mode BOOLEAN DEFAULT false;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS debug_mode_expires_at TIMESTAMPTZ;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS debug_mode_enabled_at TIMESTAMPTZ;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS debug_mode_enabled_by UUID REFERENCES users(id);

-- Index for efficient querying of agents with debug mode enabled
CREATE INDEX IF NOT EXISTS idx_agents_debug_mode ON agents(debug_mode) WHERE debug_mode = true;

-- Function to auto-disable expired debug modes (called by scheduled job)
CREATE OR REPLACE FUNCTION disable_expired_debug_modes() RETURNS void AS $$
BEGIN
    UPDATE agents
    SET debug_mode = false,
        debug_mode_expires_at = NULL,
        updated_at = NOW()
    WHERE debug_mode = true
      AND debug_mode_expires_at IS NOT NULL
      AND debug_mode_expires_at < NOW();
END;
$$ LANGUAGE plpgsql;
