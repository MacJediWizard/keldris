package models

import (
	"time"

	"github.com/google/uuid"
)

// AgentGroup represents a logical grouping of agents by environment or purpose.
type AgentGroup struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Color       string    `json:"color,omitempty"` // Hex color code like #FF5733
	AgentCount  int       `json:"agent_count"`     // Computed field, not stored
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewAgentGroup creates a new AgentGroup with the given details.
func NewAgentGroup(orgID uuid.UUID, name, description, color string) *AgentGroup {
	now := time.Now()
	return &AgentGroup{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        name,
		Description: description,
		Color:       color,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AgentGroupMember represents a membership of an agent in a group.
type AgentGroupMember struct {
	ID        uuid.UUID `json:"id"`
	AgentID   uuid.UUID `json:"agent_id"`
	GroupID   uuid.UUID `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}

// NewAgentGroupMember creates a new AgentGroupMember.
func NewAgentGroupMember(agentID, groupID uuid.UUID) *AgentGroupMember {
	return &AgentGroupMember{
		ID:        uuid.New(),
		AgentID:   agentID,
		GroupID:   groupID,
		CreatedAt: time.Now(),
	}
}

// AgentWithGroups extends Agent with its group memberships.
type AgentWithGroups struct {
	Agent
	Groups []AgentGroup `json:"groups,omitempty"`
}
