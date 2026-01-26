package models

import (
	"time"

	"github.com/google/uuid"
)

// SLAScope defines what an SLA applies to.
type SLAScope string

const (
	SLAScopeAgent        SLAScope = "agent"
	SLAScopeRepository   SLAScope = "repository"
	SLAScopeOrganization SLAScope = "organization"
)

// BreachType defines the type of SLA breach.
type BreachType string

const (
	BreachTypeRPO    BreachType = "rpo"
	BreachTypeRTO    BreachType = "rto"
	BreachTypeUptime BreachType = "uptime"
)

// SLADefinition represents an SLA configuration.
type SLADefinition struct {
	ID               uuid.UUID  `json:"id"`
	OrgID            uuid.UUID  `json:"org_id"`
	Name             string     `json:"name"`
	Description      string     `json:"description,omitempty"`
	RPOMinutes       *int       `json:"rpo_minutes,omitempty"`
	RTOMinutes       *int       `json:"rto_minutes,omitempty"`
	UptimePercentage *float64   `json:"uptime_percentage,omitempty"`
	Scope            SLAScope   `json:"scope"`
	Active           bool       `json:"active"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// NewSLADefinition creates a new SLA definition.
func NewSLADefinition(orgID uuid.UUID, name string, scope SLAScope) *SLADefinition {
	now := time.Now()
	return &SLADefinition{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Scope:     scope,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// HasRPO returns true if the SLA defines an RPO target.
func (s *SLADefinition) HasRPO() bool {
	return s.RPOMinutes != nil
}

// HasRTO returns true if the SLA defines an RTO target.
func (s *SLADefinition) HasRTO() bool {
	return s.RTOMinutes != nil
}

// HasUptime returns true if the SLA defines an uptime target.
func (s *SLADefinition) HasUptime() bool {
	return s.UptimePercentage != nil
}

// SLAAssignment links an SLA to an agent or repository.
type SLAAssignment struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	SLAID        uuid.UUID  `json:"sla_id"`
	AgentID      *uuid.UUID `json:"agent_id,omitempty"`
	RepositoryID *uuid.UUID `json:"repository_id,omitempty"`
	AssignedBy   *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedAt   time.Time  `json:"assigned_at"`
}

// NewSLAAssignment creates a new SLA assignment.
func NewSLAAssignment(orgID, slaID uuid.UUID) *SLAAssignment {
	return &SLAAssignment{
		ID:         uuid.New(),
		OrgID:      orgID,
		SLAID:      slaID,
		AssignedAt: time.Now(),
	}
}

// SLACompliance records compliance status for a period.
type SLACompliance struct {
	ID                     uuid.UUID  `json:"id"`
	OrgID                  uuid.UUID  `json:"org_id"`
	SLAID                  uuid.UUID  `json:"sla_id"`
	AgentID                *uuid.UUID `json:"agent_id,omitempty"`
	RepositoryID           *uuid.UUID `json:"repository_id,omitempty"`
	PeriodStart            time.Time  `json:"period_start"`
	PeriodEnd              time.Time  `json:"period_end"`
	RPOCompliant           *bool      `json:"rpo_compliant,omitempty"`
	RPOActualMinutes       *int       `json:"rpo_actual_minutes,omitempty"`
	RPOBreaches            int        `json:"rpo_breaches"`
	RTOCompliant           *bool      `json:"rto_compliant,omitempty"`
	RTOActualMinutes       *int       `json:"rto_actual_minutes,omitempty"`
	RTOBreaches            int        `json:"rto_breaches"`
	UptimeCompliant        *bool      `json:"uptime_compliant,omitempty"`
	UptimeActualPercentage *float64   `json:"uptime_actual_percentage,omitempty"`
	UptimeDowntimeMinutes  int        `json:"uptime_downtime_minutes"`
	IsCompliant            bool       `json:"is_compliant"`
	Notes                  string     `json:"notes,omitempty"`
	CalculatedAt           time.Time  `json:"calculated_at"`
}

// NewSLACompliance creates a new compliance record.
func NewSLACompliance(orgID, slaID uuid.UUID, periodStart, periodEnd time.Time) *SLACompliance {
	return &SLACompliance{
		ID:           uuid.New(),
		OrgID:        orgID,
		SLAID:        slaID,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		IsCompliant:  true,
		CalculatedAt: time.Now(),
	}
}

// SLABreach records an SLA breach event.
type SLABreach struct {
	ID              uuid.UUID  `json:"id"`
	OrgID           uuid.UUID  `json:"org_id"`
	SLAID           uuid.UUID  `json:"sla_id"`
	AgentID         *uuid.UUID `json:"agent_id,omitempty"`
	RepositoryID    *uuid.UUID `json:"repository_id,omitempty"`
	BreachType      BreachType `json:"breach_type"`
	ExpectedValue   *float64   `json:"expected_value,omitempty"`
	ActualValue     *float64   `json:"actual_value,omitempty"`
	BreachStart     time.Time  `json:"breach_start"`
	BreachEnd       *time.Time `json:"breach_end,omitempty"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	Acknowledged    bool       `json:"acknowledged"`
	AcknowledgedBy  *uuid.UUID `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`
	Resolved        bool       `json:"resolved"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	Description     string     `json:"description,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// NewSLABreach creates a new breach record.
func NewSLABreach(orgID, slaID uuid.UUID, breachType BreachType, breachStart time.Time) *SLABreach {
	return &SLABreach{
		ID:          uuid.New(),
		OrgID:       orgID,
		SLAID:       slaID,
		BreachType:  breachType,
		BreachStart: breachStart,
		CreatedAt:   time.Now(),
	}
}

// Acknowledge marks the breach as acknowledged.
func (b *SLABreach) Acknowledge(userID uuid.UUID) {
	b.Acknowledged = true
	b.AcknowledgedBy = &userID
	now := time.Now()
	b.AcknowledgedAt = &now
}

// Resolve marks the breach as resolved.
func (b *SLABreach) Resolve() {
	b.Resolved = true
	now := time.Now()
	b.ResolvedAt = &now
	b.BreachEnd = &now
	if b.BreachStart.Before(now) {
		duration := int(now.Sub(b.BreachStart).Minutes())
		b.DurationMinutes = &duration
	}
}

// SLAWithAssignments includes the SLA definition with its assignments.
type SLAWithAssignments struct {
	SLADefinition
	AgentCount      int `json:"agent_count"`
	RepositoryCount int `json:"repository_count"`
}

// SLAComplianceSummary summarizes compliance across multiple targets.
type SLAComplianceSummary struct {
	SLAID            uuid.UUID `json:"sla_id"`
	SLAName          string    `json:"sla_name"`
	TotalTargets     int       `json:"total_targets"`
	CompliantTargets int       `json:"compliant_targets"`
	ComplianceRate   float64   `json:"compliance_rate"`
	ActiveBreaches   int       `json:"active_breaches"`
	TotalBreaches    int       `json:"total_breaches"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
}

// SLADashboardStats provides overall SLA statistics for the dashboard.
type SLADashboardStats struct {
	TotalSLAs           int                    `json:"total_slas"`
	ActiveSLAs          int                    `json:"active_slas"`
	OverallCompliance   float64                `json:"overall_compliance"`
	ActiveBreaches      int                    `json:"active_breaches"`
	UnacknowledgedCount int                    `json:"unacknowledged_count"`
	ComplianceTrend     []SLAComplianceSummary `json:"compliance_trend,omitempty"`
}

// SLAReport contains a monthly SLA report.
type SLAReport struct {
	OrgID           uuid.UUID              `json:"org_id"`
	ReportMonth     time.Time              `json:"report_month"`
	GeneratedAt     time.Time              `json:"generated_at"`
	SLASummaries    []SLAComplianceSummary `json:"sla_summaries"`
	TotalBreaches   int                    `json:"total_breaches"`
	ResolvedBreaches int                    `json:"resolved_breaches"`
	MeanTimeToResolve *int                  `json:"mean_time_to_resolve_minutes,omitempty"`
}

// CreateSLADefinitionRequest is the request body for creating an SLA.
type CreateSLADefinitionRequest struct {
	Name             string   `json:"name" binding:"required,min=1,max=255"`
	Description      string   `json:"description,omitempty"`
	RPOMinutes       *int     `json:"rpo_minutes,omitempty"`
	RTOMinutes       *int     `json:"rto_minutes,omitempty"`
	UptimePercentage *float64 `json:"uptime_percentage,omitempty"`
	Scope            SLAScope `json:"scope" binding:"required,oneof=agent repository organization"`
	Active           *bool    `json:"active,omitempty"`
}

// UpdateSLADefinitionRequest is the request body for updating an SLA.
type UpdateSLADefinitionRequest struct {
	Name             *string   `json:"name,omitempty"`
	Description      *string   `json:"description,omitempty"`
	RPOMinutes       *int      `json:"rpo_minutes,omitempty"`
	RTOMinutes       *int      `json:"rto_minutes,omitempty"`
	UptimePercentage *float64  `json:"uptime_percentage,omitempty"`
	Scope            *SLAScope `json:"scope,omitempty"`
	Active           *bool     `json:"active,omitempty"`
}

// AssignSLARequest is the request body for assigning an SLA.
type AssignSLARequest struct {
	AgentID      *uuid.UUID `json:"agent_id,omitempty"`
	RepositoryID *uuid.UUID `json:"repository_id,omitempty"`
}

// AcknowledgeBreachRequest is the request body for acknowledging a breach.
type AcknowledgeBreachRequest struct {
	Notes string `json:"notes,omitempty"`
}

// SLADefinitionsResponse is the response for listing SLA definitions.
type SLADefinitionsResponse struct {
	SLAs []SLAWithAssignments `json:"slas"`
}

// SLAAssignmentsResponse is the response for listing SLA assignments.
type SLAAssignmentsResponse struct {
	Assignments []SLAAssignment `json:"assignments"`
}

// SLAComplianceResponse is the response for compliance records.
type SLAComplianceResponse struct {
	Compliance []SLACompliance `json:"compliance"`
}

// SLABreachesResponse is the response for listing breaches.
type SLABreachesResponse struct {
	Breaches []SLABreach `json:"breaches"`
}

// SLADashboardResponse is the response for the SLA dashboard.
type SLADashboardResponse struct {
	Stats SLADashboardStats `json:"stats"`
}

// SLAReportResponse is the response for a monthly report.
type SLAReportResponse struct {
	Report SLAReport `json:"report"`
}
