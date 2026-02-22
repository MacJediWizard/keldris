// Package sla provides SLA tracking and compliance calculation.
package sla

import (
	"context"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// BackupStore defines the interface for backup data needed for compliance.
type BackupStore interface {
	GetLastBackupTimeByAgent(ctx context.Context, agentID uuid.UUID) (*time.Time, error)
	GetLastRestoreDurationByAgent(ctx context.Context, agentID uuid.UUID) (*int, error)
	GetAgentUptimePercentage(ctx context.Context, agentID uuid.UUID, periodStart, periodEnd time.Time) (float64, int, error)
}

// Tracker calculates SLA compliance and tracks breaches.
type Tracker struct {
	store       Store
	backupStore BackupStore
	logger      zerolog.Logger
}

// NewTracker creates a new SLA tracker.
func NewTracker(store Store, backupStore BackupStore, logger zerolog.Logger) *Tracker {
	return &Tracker{
		store:       store,
		backupStore: backupStore,
		logger:      logger.With().Str("component", "sla_tracker").Logger(),
	}
}

// RPOTarget represents an RPO target configuration.
type RPOTarget struct {
	MaxMinutes int
}

// RTOTarget represents an RTO target configuration.
type RTOTarget struct {
	MaxMinutes int
}

// UptimeTarget represents an uptime target configuration.
type UptimeTarget struct {
	Percentage float64
}

// ComplianceResult holds the result of a compliance check.
type ComplianceResult struct {
	IsCompliant       bool
	RPOCompliant      *bool
	RPOActualMinutes  *int
	RPOBreaches       int
	RTOCompliant      *bool
	RTOActualMinutes  *int
	RTOBreaches       int
	UptimeCompliant   *bool
	UptimeActual      *float64
	UptimeDowntimeMin int
	Notes             string
}

// CheckRPOCompliance checks if an agent is compliant with RPO.
func (t *Tracker) CheckRPOCompliance(ctx context.Context, agentID uuid.UUID, target RPOTarget) (*bool, *int, error) {
	lastBackup, err := t.backupStore.GetLastBackupTimeByAgent(ctx, agentID)
	if err != nil {
		return nil, nil, err
	}

	if lastBackup == nil {
		// No backups yet - not compliant
		compliant := false
		return &compliant, nil, nil
	}

	minutesSinceBackup := int(time.Since(*lastBackup).Minutes())
	compliant := minutesSinceBackup <= target.MaxMinutes

	return &compliant, &minutesSinceBackup, nil
}

// CheckRTOCompliance checks if an agent is compliant with RTO.
func (t *Tracker) CheckRTOCompliance(ctx context.Context, agentID uuid.UUID, target RTOTarget) (*bool, *int, error) {
	lastRestoreDuration, err := t.backupStore.GetLastRestoreDurationByAgent(ctx, agentID)
	if err != nil {
		return nil, nil, err
	}

	if lastRestoreDuration == nil {
		// No restores yet - can't determine compliance, assume compliant
		compliant := true
		return &compliant, nil, nil
	}

	compliant := *lastRestoreDuration <= target.MaxMinutes
	return &compliant, lastRestoreDuration, nil
}

// CheckUptimeCompliance checks if an agent is compliant with uptime target.
func (t *Tracker) CheckUptimeCompliance(ctx context.Context, agentID uuid.UUID, target UptimeTarget, periodStart, periodEnd time.Time) (*bool, *float64, *int, error) {
	uptimePct, downtimeMin, err := t.backupStore.GetAgentUptimePercentage(ctx, agentID, periodStart, periodEnd)
	if err != nil {
		return nil, nil, nil, err
	}

	compliant := uptimePct >= target.Percentage
	return &compliant, &uptimePct, &downtimeMin, nil
}

// CalculateCompliance calculates compliance for an SLA assignment.
func (t *Tracker) CalculateCompliance(ctx context.Context, sla *models.SLADefinition, assignment *models.SLAAssignment, periodStart, periodEnd time.Time) (*models.SLACompliance, error) {
	compliance := models.NewSLACompliance(sla.OrgID, sla.ID, periodStart, periodEnd)
	compliance.AgentID = assignment.AgentID
	compliance.RepositoryID = assignment.RepositoryID

	allCompliant := true

	// Check RPO if defined
	if sla.HasRPO() && assignment.AgentID != nil {
		compliant, actual, err := t.CheckRPOCompliance(ctx, *assignment.AgentID, RPOTarget{MaxMinutes: *sla.RPOMinutes})
		if err != nil {
			t.logger.Warn().Err(err).Str("agent_id", assignment.AgentID.String()).Msg("failed to check RPO compliance")
		} else {
			compliance.RPOCompliant = compliant
			compliance.RPOActualMinutes = actual
			if compliant != nil && !*compliant {
				allCompliant = false
				compliance.RPOBreaches = 1
			}
		}
	}

	// Check RTO if defined
	if sla.HasRTO() && assignment.AgentID != nil {
		compliant, actual, err := t.CheckRTOCompliance(ctx, *assignment.AgentID, RTOTarget{MaxMinutes: *sla.RTOMinutes})
		if err != nil {
			t.logger.Warn().Err(err).Str("agent_id", assignment.AgentID.String()).Msg("failed to check RTO compliance")
		} else {
			compliance.RTOCompliant = compliant
			compliance.RTOActualMinutes = actual
			if compliant != nil && !*compliant {
				allCompliant = false
				compliance.RTOBreaches = 1
			}
		}
	}

	// Check Uptime if defined
	if sla.HasUptime() && assignment.AgentID != nil {
		compliant, actual, downtime, err := t.CheckUptimeCompliance(ctx, *assignment.AgentID, UptimeTarget{Percentage: *sla.UptimePercentage}, periodStart, periodEnd)
		if err != nil {
			t.logger.Warn().Err(err).Str("agent_id", assignment.AgentID.String()).Msg("failed to check uptime compliance")
		} else {
			compliance.UptimeCompliant = compliant
			compliance.UptimeActualPercentage = actual
			if downtime != nil {
				compliance.UptimeDowntimeMinutes = *downtime
			}
			if compliant != nil && !*compliant {
				allCompliant = false
			}
		}
	}

	compliance.IsCompliant = allCompliant
	return compliance, nil
}

// RecordBreach creates a new breach record.
func (t *Tracker) RecordBreach(ctx context.Context, sla *models.SLADefinition, assignment *models.SLAAssignment, breachType models.BreachType, expected, actual float64) error {
	breach := models.NewSLABreach(sla.OrgID, sla.ID, breachType, time.Now())
	breach.AgentID = assignment.AgentID
	breach.RepositoryID = assignment.RepositoryID
	breach.ExpectedValue = &expected
	breach.ActualValue = &actual

	switch breachType {
	case models.BreachTypeRPO:
		breach.Description = "RPO target exceeded"
	case models.BreachTypeRTO:
		breach.Description = "RTO target exceeded"
	case models.BreachTypeUptime:
		breach.Description = "Uptime target not met"
	}

	if err := t.store.CreateSLABreach(ctx, breach); err != nil {
		return err
	}

	t.logger.Info().
		Str("sla_id", sla.ID.String()).
		Str("breach_type", string(breachType)).
		Float64("expected", expected).
		Float64("actual", actual).
		Msg("SLA breach recorded")

	return nil
}

// CalculateOrgCompliance calculates compliance for all SLAs in an organization.
func (t *Tracker) CalculateOrgCompliance(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) ([]*models.SLACompliance, error) {
	slas, err := t.store.ListActiveSLADefinitionsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var results []*models.SLACompliance

	for _, sla := range slas {
		assignments, err := t.store.ListSLAAssignmentsBySLA(ctx, sla.ID)
		if err != nil {
			t.logger.Warn().Err(err).Str("sla_id", sla.ID.String()).Msg("failed to list assignments")
			continue
		}

		for _, assignment := range assignments {
			compliance, err := t.CalculateCompliance(ctx, sla, assignment, periodStart, periodEnd)
			if err != nil {
				t.logger.Warn().Err(err).Str("sla_id", sla.ID.String()).Msg("failed to calculate compliance")
				continue
			}

			if err := t.store.CreateSLACompliance(ctx, compliance); err != nil {
				t.logger.Warn().Err(err).Str("sla_id", sla.ID.String()).Msg("failed to store compliance")
				continue
			}

			results = append(results, compliance)

			// Record breaches for non-compliant items
			if !compliance.IsCompliant {
				if compliance.RPOCompliant != nil && !*compliance.RPOCompliant && compliance.RPOActualMinutes != nil {
					expected := float64(*sla.RPOMinutes)
					actual := float64(*compliance.RPOActualMinutes)
					_ = t.RecordBreach(ctx, sla, assignment, models.BreachTypeRPO, expected, actual)
				}
				if compliance.RTOCompliant != nil && !*compliance.RTOCompliant && compliance.RTOActualMinutes != nil {
					expected := float64(*sla.RTOMinutes)
					actual := float64(*compliance.RTOActualMinutes)
					_ = t.RecordBreach(ctx, sla, assignment, models.BreachTypeRTO, expected, actual)
				}
				if compliance.UptimeCompliant != nil && !*compliance.UptimeCompliant && compliance.UptimeActualPercentage != nil {
					_ = t.RecordBreach(ctx, sla, assignment, models.BreachTypeUptime, *sla.UptimePercentage, *compliance.UptimeActualPercentage)
				}
			}
		}
	}

	return results, nil
}

// GenerateMonthlyReport generates a monthly SLA report.
func (t *Tracker) GenerateMonthlyReport(ctx context.Context, orgID uuid.UUID, reportMonth time.Time) (*models.SLAReport, error) {
	// Get the first and last day of the month
	year, month, _ := reportMonth.Date()
	periodStart := time.Date(year, month, 1, 0, 0, 0, 0, reportMonth.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	report := &models.SLAReport{
		OrgID:       orgID,
		ReportMonth: periodStart,
		GeneratedAt: time.Now(),
	}

	slas, err := t.store.ListActiveSLADefinitionsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, sla := range slas {
		assignments, err := t.store.ListSLAAssignmentsBySLA(ctx, sla.ID)
		if err != nil {
			continue
		}

		summary := models.SLAComplianceSummary{
			SLAID:       sla.ID,
			SLAName:     sla.Name,
			TotalTargets: len(assignments),
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
		}

		// For each assignment, calculate compliance
		for _, assignment := range assignments {
			compliance, err := t.CalculateCompliance(ctx, sla, assignment, periodStart, periodEnd)
			if err != nil {
				continue
			}
			if compliance.IsCompliant {
				summary.CompliantTargets++
			}
			summary.TotalBreaches += compliance.RPOBreaches + compliance.RTOBreaches
			if compliance.UptimeCompliant != nil && !*compliance.UptimeCompliant {
				summary.TotalBreaches++
			}
		}

		if summary.TotalTargets > 0 {
			summary.ComplianceRate = float64(summary.CompliantTargets) / float64(summary.TotalTargets) * 100
		}

		// Count active breaches
		breaches, err := t.store.ListActiveSLABreachesByOrg(ctx, orgID)
		if err == nil {
			for _, b := range breaches {
				if b.SLAID == sla.ID {
					summary.ActiveBreaches++
				}
			}
		}

		report.SLASummaries = append(report.SLASummaries, summary)
		report.TotalBreaches += summary.TotalBreaches
	}

	return report, nil
}
