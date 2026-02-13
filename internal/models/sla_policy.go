package models

import (
	"time"

	"github.com/google/uuid"
)

// SLAPolicy defines an SLA policy with target metrics.
type SLAPolicy struct {
	ID                uuid.UUID `json:"id"`
	OrgID             uuid.UUID `json:"org_id"`
	Name              string    `json:"name"`
	Description       string    `json:"description,omitempty"`
	TargetRPOHours    float64   `json:"target_rpo_hours"`
	TargetRTOHours    float64   `json:"target_rto_hours"`
	TargetSuccessRate float64   `json:"target_success_rate"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NewSLAPolicy creates a new SLAPolicy with the given details.
func NewSLAPolicy(orgID uuid.UUID, name string, targetRPO, targetRTO, targetSuccessRate float64) *SLAPolicy {
	now := time.Now()
	return &SLAPolicy{
		ID:                uuid.New(),
		OrgID:             orgID,
		Name:              name,
		TargetRPOHours:    targetRPO,
		TargetRTOHours:    targetRTO,
		TargetSuccessRate: targetSuccessRate,
		Enabled:           true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// SLAStatus holds the current SLA compliance status for a policy.
type SLAStatus struct {
	PolicyID    uuid.UUID `json:"policy_id"`
	CurrentRPO  float64   `json:"current_rpo_hours"`
	CurrentRTO  float64   `json:"current_rto_hours"`
	SuccessRate float64   `json:"success_rate"`
	Compliant   bool      `json:"compliant"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// SLAStatusSnapshot represents a historical SLA status record.
type SLAStatusSnapshot struct {
	ID           uuid.UUID `json:"id"`
	PolicyID     uuid.UUID `json:"policy_id"`
	RPOHours     float64   `json:"rpo_hours"`
	RTOHours     float64   `json:"rto_hours"`
	SuccessRate  float64   `json:"success_rate"`
	Compliant    bool      `json:"compliant"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// CreateSLAPolicyRequest is the request body for creating an SLA policy.
type CreateSLAPolicyRequest struct {
	Name              string  `json:"name" binding:"required,min=1,max=255"`
	Description       string  `json:"description,omitempty"`
	TargetRPOHours    float64 `json:"target_rpo_hours" binding:"required,gt=0"`
	TargetRTOHours    float64 `json:"target_rto_hours" binding:"required,gt=0"`
	TargetSuccessRate float64 `json:"target_success_rate" binding:"required,gt=0,lte=100"`
}

// UpdateSLAPolicyRequest is the request body for updating an SLA policy.
type UpdateSLAPolicyRequest struct {
	Name              *string  `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	TargetRPOHours    *float64 `json:"target_rpo_hours,omitempty"`
	TargetRTOHours    *float64 `json:"target_rto_hours,omitempty"`
	TargetSuccessRate *float64 `json:"target_success_rate,omitempty"`
	Enabled           *bool    `json:"enabled,omitempty"`
}
