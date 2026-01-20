package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DRRunbookStatus represents the status of a DR runbook.
type DRRunbookStatus string

const (
	DRRunbookStatusDraft    DRRunbookStatus = "draft"
	DRRunbookStatusActive   DRRunbookStatus = "active"
	DRRunbookStatusArchived DRRunbookStatus = "archived"
)

// DRRunbookStepType represents the type of a runbook step.
type DRRunbookStepType string

const (
	DRRunbookStepTypeManual  DRRunbookStepType = "manual"
	DRRunbookStepTypeRestore DRRunbookStepType = "restore"
	DRRunbookStepTypeVerify  DRRunbookStepType = "verify"
	DRRunbookStepTypeNotify  DRRunbookStepType = "notify"
)

// DRRunbookStep represents a single step in a DR runbook.
type DRRunbookStep struct {
	Order       int               `json:"order"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        DRRunbookStepType `json:"type"`
	Command     string            `json:"command,omitempty"`
	Expected    string            `json:"expected,omitempty"`
}

// DRRunbookContact represents a contact for DR communications.
type DRRunbookContact struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Email  string `json:"email,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Notify bool   `json:"notify"`
}

// DRRunbook represents a disaster recovery runbook.
type DRRunbook struct {
	ID                          uuid.UUID          `json:"id"`
	OrgID                       uuid.UUID          `json:"org_id"`
	ScheduleID                  *uuid.UUID         `json:"schedule_id,omitempty"`
	Name                        string             `json:"name"`
	Description                 string             `json:"description,omitempty"`
	Steps                       []DRRunbookStep    `json:"steps"`
	Contacts                    []DRRunbookContact `json:"contacts"`
	CredentialsLocation         string             `json:"credentials_location,omitempty"`
	RecoveryTimeObjectiveMins   *int               `json:"recovery_time_objective_minutes,omitempty"`
	RecoveryPointObjectiveMins  *int               `json:"recovery_point_objective_minutes,omitempty"`
	Status                      DRRunbookStatus    `json:"status"`
	CreatedAt                   time.Time          `json:"created_at"`
	UpdatedAt                   time.Time          `json:"updated_at"`
}

// NewDRRunbook creates a new DR runbook with the given details.
func NewDRRunbook(orgID uuid.UUID, name string) *DRRunbook {
	now := time.Now()
	return &DRRunbook{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Steps:     []DRRunbookStep{},
		Contacts:  []DRRunbookContact{},
		Status:    DRRunbookStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetSteps sets the steps from JSON bytes.
func (r *DRRunbook) SetSteps(data []byte) error {
	if len(data) == 0 {
		r.Steps = []DRRunbookStep{}
		return nil
	}
	return json.Unmarshal(data, &r.Steps)
}

// StepsJSON returns the steps as JSON bytes for database storage.
func (r *DRRunbook) StepsJSON() ([]byte, error) {
	if r.Steps == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(r.Steps)
}

// SetContacts sets the contacts from JSON bytes.
func (r *DRRunbook) SetContacts(data []byte) error {
	if len(data) == 0 {
		r.Contacts = []DRRunbookContact{}
		return nil
	}
	return json.Unmarshal(data, &r.Contacts)
}

// ContactsJSON returns the contacts as JSON bytes for database storage.
func (r *DRRunbook) ContactsJSON() ([]byte, error) {
	if r.Contacts == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(r.Contacts)
}

// AddStep appends a step to the runbook.
func (r *DRRunbook) AddStep(title, description string, stepType DRRunbookStepType) {
	step := DRRunbookStep{
		Order:       len(r.Steps) + 1,
		Title:       title,
		Description: description,
		Type:        stepType,
	}
	r.Steps = append(r.Steps, step)
	r.UpdatedAt = time.Now()
}

// AddContact appends a contact to the runbook.
func (r *DRRunbook) AddContact(name, role, email, phone string, notify bool) {
	contact := DRRunbookContact{
		Name:   name,
		Role:   role,
		Email:  email,
		Phone:  phone,
		Notify: notify,
	}
	r.Contacts = append(r.Contacts, contact)
	r.UpdatedAt = time.Now()
}

// Activate sets the runbook status to active.
func (r *DRRunbook) Activate() {
	r.Status = DRRunbookStatusActive
	r.UpdatedAt = time.Now()
}

// Archive sets the runbook status to archived.
func (r *DRRunbook) Archive() {
	r.Status = DRRunbookStatusArchived
	r.UpdatedAt = time.Now()
}

// DRTestSchedule represents a scheduled DR test.
type DRTestSchedule struct {
	ID             uuid.UUID  `json:"id"`
	RunbookID      uuid.UUID  `json:"runbook_id"`
	CronExpression string     `json:"cron_expression"`
	Enabled        bool       `json:"enabled"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// NewDRTestSchedule creates a new DR test schedule.
func NewDRTestSchedule(runbookID uuid.UUID, cronExpr string) *DRTestSchedule {
	now := time.Now()
	return &DRTestSchedule{
		ID:             uuid.New(),
		RunbookID:      runbookID,
		CronExpression: cronExpr,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
