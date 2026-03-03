-- Migration: Add dry_run and uninstall command types to agent_commands

ALTER TABLE agent_commands DROP CONSTRAINT IF EXISTS agent_commands_type_check;
ALTER TABLE agent_commands ADD CONSTRAINT agent_commands_type_check
    CHECK (type IN ('backup_now', 'update', 'restart', 'diagnostics', 'update_restic', 'dry_run', 'uninstall'));
