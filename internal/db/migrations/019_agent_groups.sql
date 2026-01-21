-- Migration: Add agent groups for organizing agents by environment/purpose

CREATE TABLE agent_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(7), -- Hex color code like #FF5733
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agent_groups_org ON agent_groups(org_id);
CREATE UNIQUE INDEX idx_agent_groups_org_name ON agent_groups(org_id, name);

-- Junction table for many-to-many relationship between agents and groups
CREATE TABLE agent_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agent_group_members_agent ON agent_group_members(agent_id);
CREATE INDEX idx_agent_group_members_group ON agent_group_members(group_id);
CREATE UNIQUE INDEX idx_agent_group_members_unique ON agent_group_members(agent_id, group_id);

-- Add optional agent_group_id to schedules for group-level scheduling
ALTER TABLE schedules ADD COLUMN agent_group_id UUID REFERENCES agent_groups(id) ON DELETE SET NULL;
CREATE INDEX idx_schedules_agent_group ON schedules(agent_group_id);
