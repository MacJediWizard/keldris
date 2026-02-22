-- 024_add_network_mounts.sql
-- 012_add_network_mounts.sql
-- Migration: Add network mount tracking for agents and schedule behavior

-- Add network mounts tracking to agents (stores array of detected mounts)
ALTER TABLE agents ADD COLUMN network_mounts JSONB DEFAULT '[]'::jsonb;

-- Add mount unavailable behavior to schedules
ALTER TABLE schedules ADD COLUMN on_mount_unavailable VARCHAR(20) DEFAULT 'fail';

-- Add constraint for valid values
ALTER TABLE schedules ADD CONSTRAINT chk_mount_behavior
    CHECK (on_mount_unavailable IN ('skip', 'fail'));

COMMENT ON COLUMN agents.network_mounts IS 'Array of detected network mounts on the agent';
COMMENT ON COLUMN schedules.on_mount_unavailable IS 'Behavior when network mount unavailable: skip or fail';
