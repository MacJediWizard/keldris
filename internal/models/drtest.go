package models

import (
	"time"

	"github.com/google/uuid"
)

// DRTestStatus represents the status of a DR test.
type DRTestStatus string

const (
	DRTestStatusScheduled DRTestStatus = "scheduled"
	DRTestStatusRunning   DRTestStatus = "running"
	DRTestStatusCompleted DRTestStatus = "completed"
	DRTestStatusFailed    DRTestStatus = "failed"
	DRTestStatusCanceled  DRTestStatus = "canceled"
)

// DRTest represents a disaster recovery test execution.
type DRTest struct {
	ID                     uuid.UUID    `json:"id"`
	RunbookID              uuid.UUID    `json:"runbook_id"`
	ScheduleID             *uuid.UUID   `json:"schedule_id,omitempty"`
	AgentID                *uuid.UUID   `json:"agent_id,omitempty"`
	SnapshotID             string       `json:"snapshot_id,omitempty"`
	Status                 DRTestStatus `json:"status"`
	StartedAt              *time.Time   `json:"started_at,omitempty"`
	CompletedAt            *time.Time   `json:"completed_at,omitempty"`
	RestoreSizeBytes       *int64       `json:"restore_size_bytes,omitempty"`
	RestoreDurationSeconds *int         `json:"restore_duration_seconds,omitempty"`
	VerificationPassed     *bool        `json:"verification_passed,omitempty"`
	Notes                  string       `json:"notes,omitempty"`
	ErrorMessage           string       `json:"error_message,omitempty"`
	CreatedAt              time.Time    `json:"created_at"`
}

// NewDRTest creates a new DR test record.
func NewDRTest(runbookID uuid.UUID) *DRTest {
	now := time.Now()
	return &DRTest{
		ID:        uuid.New(),
		RunbookID: runbookID,
		Status:    DRTestStatusScheduled,
		CreatedAt: now,
	}
}

// Start marks the DR test as running.
func (t *DRTest) Start() {
	now := time.Now()
	t.Status = DRTestStatusRunning
	t.StartedAt = &now
}

// Complete marks the DR test as completed successfully.
func (t *DRTest) Complete(snapshotID string, sizeBytes int64, durationSecs int, verificationPassed bool) {
	now := time.Now()
	t.Status = DRTestStatusCompleted
	t.CompletedAt = &now
	t.SnapshotID = snapshotID
	t.RestoreSizeBytes = &sizeBytes
	t.RestoreDurationSeconds = &durationSecs
	t.VerificationPassed = &verificationPassed
}

// Fail marks the DR test as failed.
func (t *DRTest) Fail(errMsg string) {
	now := time.Now()
	t.Status = DRTestStatusFailed
	t.CompletedAt = &now
	t.ErrorMessage = errMsg
	verificationFailed := false
	t.VerificationPassed = &verificationFailed
}

// Cancel marks the DR test as canceled.
func (t *DRTest) Cancel(notes string) {
	now := time.Now()
	t.Status = DRTestStatusCanceled
	t.CompletedAt = &now
	t.Notes = notes
}

// SetSchedule associates the test with a backup schedule.
func (t *DRTest) SetSchedule(scheduleID uuid.UUID) {
	t.ScheduleID = &scheduleID
}

// SetAgent associates the test with an agent.
func (t *DRTest) SetAgent(agentID uuid.UUID) {
	t.AgentID = &agentID
}

// DRTestSummary provides a summary of DR test status for dashboard display.
type DRTestSummary struct {
	RunbookID          uuid.UUID  `json:"runbook_id"`
	RunbookName        string     `json:"runbook_name"`
	LastTestID         *uuid.UUID `json:"last_test_id,omitempty"`
	LastTestStatus     string     `json:"last_test_status,omitempty"`
	LastTestAt         *time.Time `json:"last_test_at,omitempty"`
	NextScheduledAt    *time.Time `json:"next_scheduled_at,omitempty"`
	TotalTests         int        `json:"total_tests"`
	PassedTests        int        `json:"passed_tests"`
	FailedTests        int        `json:"failed_tests"`
	SuccessRate        float64    `json:"success_rate"`
	AvgRestoreTimeSecs *int       `json:"avg_restore_time_seconds,omitempty"`
}

// DRStatus provides overall DR readiness status for the dashboard.
type DRStatus struct {
	TotalRunbooks    int              `json:"total_runbooks"`
	ActiveRunbooks   int              `json:"active_runbooks"`
	LastTestAt       *time.Time       `json:"last_test_at,omitempty"`
	NextTestAt       *time.Time       `json:"next_test_at,omitempty"`
	TestsLast30Days  int              `json:"tests_last_30_days"`
	PassRate         float64          `json:"pass_rate"`
	RunbookSummaries []DRTestSummary  `json:"runbook_summaries,omitempty"`
}
