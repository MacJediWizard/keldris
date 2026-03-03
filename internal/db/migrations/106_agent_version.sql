-- Migration: Add agent_version column and update_restic command type

ALTER TABLE agents ADD COLUMN IF NOT EXISTS agent_version VARCHAR(50);

ALTER TABLE agent_commands DROP CONSTRAINT IF EXISTS agent_commands_type_check;
ALTER TABLE agent_commands ADD CONSTRAINT agent_commands_type_check
    CHECK (type IN ('backup_now', 'update', 'restart', 'diagnostics', 'update_restic'));
