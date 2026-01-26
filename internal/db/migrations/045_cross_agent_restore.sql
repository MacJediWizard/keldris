-- Migration: Add cross-agent restore support

-- Add source_agent_id to track the original agent for cross-agent restores
ALTER TABLE restores ADD COLUMN source_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL;

-- Add path_mappings to handle path remapping between agents
ALTER TABLE restores ADD COLUMN path_mappings JSONB DEFAULT '[]'::jsonb;

-- Add progress tracking fields
ALTER TABLE restores ADD COLUMN files_restored BIGINT DEFAULT 0;
ALTER TABLE restores ADD COLUMN bytes_restored BIGINT DEFAULT 0;
ALTER TABLE restores ADD COLUMN total_files BIGINT;
ALTER TABLE restores ADD COLUMN total_bytes BIGINT;
ALTER TABLE restores ADD COLUMN current_file VARCHAR(2048);

-- Create index for source_agent_id to efficiently query cross-agent restores
CREATE INDEX idx_restores_source_agent ON restores(source_agent_id) WHERE source_agent_id IS NOT NULL;

-- Create index for finding restores by either agent (source or target)
CREATE INDEX idx_restores_any_agent ON restores(agent_id, source_agent_id);

-- Add comment to clarify agent_id is the target agent for cross-agent restores
COMMENT ON COLUMN restores.agent_id IS 'Target agent ID where the restore will be executed';
COMMENT ON COLUMN restores.source_agent_id IS 'Source agent ID for cross-agent restores (NULL for same-agent restores)';
COMMENT ON COLUMN restores.path_mappings IS 'JSON array of path mappings for cross-agent restores: [{source_path, target_path}]';
