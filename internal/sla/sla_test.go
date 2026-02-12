package sla

import (
	"context"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

type mockStore struct {
	policy      *models.SLAPolicy
	successRate float64
	maxRPO      float64
	policyErr   error
	rateErr     error
	rpoErr      error
}

func (m *mockStore) GetSLAPolicyByID(_ context.Context, _ uuid.UUID) (*models.SLAPolicy, error) {
	if m.policyErr != nil {
		return nil, m.policyErr
	}
	return m.policy, nil
}

func (m *mockStore) GetBackupSuccessRateForOrg(_ context.Context, _ uuid.UUID, _ int) (float64, error) {
	if m.rateErr != nil {
		return 0, m.rateErr
	}
	return m.successRate, nil
}

func (m *mockStore) GetMaxRPOHoursForOrg(_ context.Context, _ uuid.UUID) (float64, error) {
	if m.rpoErr != nil {
		return 0, m.rpoErr
	}
	return m.maxRPO, nil
}

func (m *mockStore) CreateSLAStatusSnapshot(_ context.Context, _ *models.SLAStatusSnapshot) error {
	return nil
}

func TestCheckCompliance(t *testing.T) {
	policy := &models.SLAPolicy{
		TargetRPOHours:    24,
		TargetRTOHours:    4,
		TargetSuccessRate: 99.0,
	}

	t.Run("compliant", func(t *testing.T) {
		status := &models.SLAStatus{
			CurrentRPO:  12,
			CurrentRTO:  2,
			SuccessRate: 99.5,
		}
		if !CheckCompliance(policy, status) {
			t.Fatal("expected compliant")
		}
	})

	t.Run("rpo violation", func(t *testing.T) {
		status := &models.SLAStatus{
			CurrentRPO:  30,
			CurrentRTO:  2,
			SuccessRate: 99.5,
		}
		if CheckCompliance(policy, status) {
			t.Fatal("expected non-compliant due to RPO violation")
		}
	})

	t.Run("rto violation", func(t *testing.T) {
		status := &models.SLAStatus{
			CurrentRPO:  12,
			CurrentRTO:  6,
			SuccessRate: 99.5,
		}
		if CheckCompliance(policy, status) {
			t.Fatal("expected non-compliant due to RTO violation")
		}
	})

	t.Run("success rate violation", func(t *testing.T) {
		status := &models.SLAStatus{
			CurrentRPO:  12,
			CurrentRTO:  2,
			SuccessRate: 95.0,
		}
		if CheckCompliance(policy, status) {
			t.Fatal("expected non-compliant due to success rate violation")
		}
	})

	t.Run("exactly at target is compliant", func(t *testing.T) {
		status := &models.SLAStatus{
			CurrentRPO:  24,
			CurrentRTO:  4,
			SuccessRate: 99.0,
		}
		if !CheckCompliance(policy, status) {
			t.Fatal("expected compliant when exactly at targets")
		}
	})
}

func TestCalculateSLAStatus(t *testing.T) {
	orgID := uuid.New()
	policyID := uuid.New()

	t.Run("compliant status", func(t *testing.T) {
		store := &mockStore{
			policy: &models.SLAPolicy{
				ID:                policyID,
				OrgID:             orgID,
				TargetRPOHours:    24,
				TargetRTOHours:    4,
				TargetSuccessRate: 99.0,
			},
			successRate: 99.5,
			maxRPO:      2.5,
		}
		calc := NewCalculator(store)

		status, err := calc.CalculateSLAStatus(context.Background(), orgID, policyID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !status.Compliant {
			t.Fatal("expected compliant status")
		}
		if status.CurrentRPO != 2.5 {
			t.Fatalf("expected RPO 2.5, got %f", status.CurrentRPO)
		}
		if status.SuccessRate != 99.5 {
			t.Fatalf("expected success rate 99.5, got %f", status.SuccessRate)
		}
	})

	t.Run("non-compliant status", func(t *testing.T) {
		store := &mockStore{
			policy: &models.SLAPolicy{
				ID:                policyID,
				OrgID:             orgID,
				TargetRPOHours:    24,
				TargetRTOHours:    4,
				TargetSuccessRate: 99.0,
			},
			successRate: 95.0,
			maxRPO:      30.0,
		}
		calc := NewCalculator(store)

		status, err := calc.CalculateSLAStatus(context.Background(), orgID, policyID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Compliant {
			t.Fatal("expected non-compliant status")
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		store := &mockStore{
			policy: &models.SLAPolicy{
				ID:    policyID,
				OrgID: uuid.New(), // different org
			},
		}
		calc := NewCalculator(store)

		_, err := calc.CalculateSLAStatus(context.Background(), orgID, policyID)
		if err == nil {
			t.Fatal("expected error for wrong org")
		}
	})
}
