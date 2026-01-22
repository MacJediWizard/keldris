package models

import (
	"time"

	"github.com/google/uuid"
)

// OnboardingStep represents a step in the onboarding process.
type OnboardingStep string

const (
	OnboardingStepWelcome      OnboardingStep = "welcome"
	OnboardingStepOrganization OnboardingStep = "organization"
	OnboardingStepSMTP         OnboardingStep = "smtp"
	OnboardingStepRepository   OnboardingStep = "repository"
	OnboardingStepAgent        OnboardingStep = "agent"
	OnboardingStepSchedule     OnboardingStep = "schedule"
	OnboardingStepVerify       OnboardingStep = "verify"
	OnboardingStepComplete     OnboardingStep = "complete"
)

// OnboardingSteps defines the ordered list of onboarding steps.
var OnboardingSteps = []OnboardingStep{
	OnboardingStepWelcome,
	OnboardingStepOrganization,
	OnboardingStepSMTP,
	OnboardingStepRepository,
	OnboardingStepAgent,
	OnboardingStepSchedule,
	OnboardingStepVerify,
	OnboardingStepComplete,
}

// OnboardingProgress tracks an organization's progress through setup.
type OnboardingProgress struct {
	ID             uuid.UUID        `json:"id"`
	OrgID          uuid.UUID        `json:"org_id"`
	CurrentStep    OnboardingStep   `json:"current_step"`
	CompletedSteps []OnboardingStep `json:"completed_steps"`
	Skipped        bool             `json:"skipped"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// NewOnboardingProgress creates a new onboarding progress record.
func NewOnboardingProgress(orgID uuid.UUID) *OnboardingProgress {
	now := time.Now()
	return &OnboardingProgress{
		ID:             uuid.New(),
		OrgID:          orgID,
		CurrentStep:    OnboardingStepWelcome,
		CompletedSteps: []OnboardingStep{},
		Skipped:        false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsComplete returns true if onboarding has been completed.
func (p *OnboardingProgress) IsComplete() bool {
	return p.CompletedAt != nil || p.Skipped || p.CurrentStep == OnboardingStepComplete
}

// HasCompletedStep checks if a specific step has been completed.
func (p *OnboardingProgress) HasCompletedStep(step OnboardingStep) bool {
	for _, s := range p.CompletedSteps {
		if s == step {
			return true
		}
	}
	return false
}

// CompleteStep marks a step as completed and advances to the next step.
func (p *OnboardingProgress) CompleteStep(step OnboardingStep) {
	if !p.HasCompletedStep(step) {
		p.CompletedSteps = append(p.CompletedSteps, step)
	}
	p.UpdatedAt = time.Now()

	// Advance to next step
	for i, s := range OnboardingSteps {
		if s == step && i+1 < len(OnboardingSteps) {
			p.CurrentStep = OnboardingSteps[i+1]
			break
		}
	}

	// Check if complete
	if step == OnboardingStepVerify {
		p.CurrentStep = OnboardingStepComplete
		now := time.Now()
		p.CompletedAt = &now
	}
}

// Skip marks the onboarding as skipped.
func (p *OnboardingProgress) Skip() {
	p.Skipped = true
	now := time.Now()
	p.CompletedAt = &now
	p.UpdatedAt = now
}

// UpdateOnboardingRequest represents a request to update onboarding progress.
type UpdateOnboardingRequest struct {
	CurrentStep    *OnboardingStep `json:"current_step,omitempty"`
	CompleteStep   *OnboardingStep `json:"complete_step,omitempty"`
	Skip           *bool           `json:"skip,omitempty"`
}

// OnboardingStatusResponse represents the onboarding status for the frontend.
type OnboardingStatusResponse struct {
	NeedsOnboarding bool             `json:"needs_onboarding"`
	CurrentStep     OnboardingStep   `json:"current_step"`
	CompletedSteps  []OnboardingStep `json:"completed_steps"`
	Skipped         bool             `json:"skipped"`
	IsComplete      bool             `json:"is_complete"`
}
