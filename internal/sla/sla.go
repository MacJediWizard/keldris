package sla

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Store defines the database operations needed for SLA calculations and tracking.
type Store interface {
	// Calculator methods
// Store defines the database operations needed for SLA calculations.
type Store interface {
	GetSLAPolicyByID(ctx context.Context, id uuid.UUID) (*models.SLAPolicy, error)
	GetBackupSuccessRateForOrg(ctx context.Context, orgID uuid.UUID, hours int) (float64, error)
	GetMaxRPOHoursForOrg(ctx context.Context, orgID uuid.UUID) (float64, error)
	CreateSLAStatusSnapshot(ctx context.Context, snapshot *models.SLAStatusSnapshot) error

	// Tracker methods
	GetSLADefinitionByID(ctx context.Context, id uuid.UUID) (*models.SLADefinition, error)
	ListActiveSLADefinitionsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLADefinition, error)
	ListSLAAssignmentsBySLA(ctx context.Context, slaID uuid.UUID) ([]*models.SLAAssignment, error)
	ListSLAAssignmentsByAgent(ctx context.Context, agentID uuid.UUID) ([]*models.SLAAssignment, error)
	CreateSLACompliance(ctx context.Context, c *models.SLACompliance) error
	CreateSLABreach(ctx context.Context, b *models.SLABreach) error
	ListActiveSLABreachesByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLABreach, error)
	UpdateSLABreach(ctx context.Context, b *models.SLABreach) error
}

// Calculator performs SLA compliance calculations.
type Calculator struct {
	store Store
}

// NewCalculator creates a new SLA Calculator.
func NewCalculator(store Store) *Calculator {
	return &Calculator{store: store}
}

// CalculateSLAStatus computes the current SLA status for a policy.
func (c *Calculator) CalculateSLAStatus(ctx context.Context, orgID uuid.UUID, policyID uuid.UUID) (*models.SLAStatus, error) {
	policy, err := c.store.GetSLAPolicyByID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("get SLA policy: %w", err)
	}

	if policy.OrgID != orgID {
		return nil, fmt.Errorf("SLA policy not found")
	}

	// Calculate current RPO (max hours since last backup across all agents)
	currentRPO, err := c.store.GetMaxRPOHoursForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("calculate RPO: %w", err)
	}

	// Calculate backup success rate over 24 hours
	successRate, err := c.store.GetBackupSuccessRateForOrg(ctx, orgID, 24)
	if err != nil {
		return nil, fmt.Errorf("calculate success rate: %w", err)
	}

	// RTO is estimated as current RPO (time to restore = time since last backup in simplest model)
	currentRTO := currentRPO

	status := &models.SLAStatus{
		PolicyID:     policyID,
		CurrentRPO:   currentRPO,
		CurrentRTO:   currentRTO,
		SuccessRate:  successRate,
		CalculatedAt: time.Now(),
	}
	status.Compliant = CheckCompliance(policy, status)

	return status, nil
}

// CheckCompliance determines if the current status meets the policy targets.
func CheckCompliance(policy *models.SLAPolicy, status *models.SLAStatus) bool {
	if status.CurrentRPO > policy.TargetRPOHours {
		return false
	}
	if status.CurrentRTO > policy.TargetRTOHours {
		return false
	}
	if status.SuccessRate < policy.TargetSuccessRate {
		return false
	}
	return true
}
