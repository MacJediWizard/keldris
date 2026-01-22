package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RansomwareAlertStatus represents the current status of a ransomware alert.
type RansomwareAlertStatus string

const (
	// RansomwareAlertStatusActive indicates the alert is active and requires attention.
	RansomwareAlertStatusActive RansomwareAlertStatus = "active"
	// RansomwareAlertStatusInvestigating indicates the alert is being investigated.
	RansomwareAlertStatusInvestigating RansomwareAlertStatus = "investigating"
	// RansomwareAlertStatusFalsePositive indicates the alert was determined to be a false positive.
	RansomwareAlertStatusFalsePositive RansomwareAlertStatus = "false_positive"
	// RansomwareAlertStatusConfirmed indicates ransomware was confirmed.
	RansomwareAlertStatusConfirmed RansomwareAlertStatus = "confirmed"
	// RansomwareAlertStatusResolved indicates the alert has been resolved.
	RansomwareAlertStatusResolved RansomwareAlertStatus = "resolved"
)

// RansomwareSettings holds ransomware detection configuration for a schedule.
type RansomwareSettings struct {
	ID                      uuid.UUID `json:"id"`
	ScheduleID              uuid.UUID `json:"schedule_id"`
	Enabled                 bool      `json:"enabled"`
	ChangeThresholdPercent  int       `json:"change_threshold_percent"`  // Alert if > X% files changed
	ExtensionsToDetect      []string  `json:"extensions_to_detect"`      // Ransomware extensions to detect
	EntropyDetectionEnabled bool      `json:"entropy_detection_enabled"` // Detect high entropy files
	EntropyThreshold        float64   `json:"entropy_threshold"`         // Entropy threshold (0-8 scale)
	AutoPauseOnAlert        bool      `json:"auto_pause_on_alert"`       // Pause backups when ransomware detected
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// NewRansomwareSettings creates a new RansomwareSettings with the given schedule ID.
func NewRansomwareSettings(scheduleID uuid.UUID) *RansomwareSettings {
	now := time.Now()
	return &RansomwareSettings{
		ID:                      uuid.New(),
		ScheduleID:              scheduleID,
		Enabled:                 true,
		ChangeThresholdPercent:  30, // Default: alert if > 30% files changed
		ExtensionsToDetect:      nil, // Use default list
		EntropyDetectionEnabled: true,
		EntropyThreshold:        7.5, // High entropy indicates encryption
		AutoPauseOnAlert:        false,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

// DefaultRansomwareSettings returns default ransomware settings for a schedule.
func DefaultRansomwareSettings(scheduleID uuid.UUID) *RansomwareSettings {
	return NewRansomwareSettings(scheduleID)
}

// SetExtensions sets the extensions from JSON bytes.
func (r *RansomwareSettings) SetExtensions(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &r.ExtensionsToDetect)
}

// ExtensionsJSON returns the extensions as JSON bytes for database storage.
func (r *RansomwareSettings) ExtensionsJSON() ([]byte, error) {
	if r.ExtensionsToDetect == nil {
		return nil, nil
	}
	return json.Marshal(r.ExtensionsToDetect)
}

// GetExtensions returns the extensions list, or nil if using defaults.
func (r *RansomwareSettings) GetExtensions() []string {
	return r.ExtensionsToDetect
}

// RansomwareAlert represents a detected ransomware activity alert.
type RansomwareAlert struct {
	ID            uuid.UUID             `json:"id"`
	OrgID         uuid.UUID             `json:"org_id"`
	ScheduleID    uuid.UUID             `json:"schedule_id"`
	ScheduleName  string                `json:"schedule_name"`
	AgentID       uuid.UUID             `json:"agent_id"`
	AgentHostname string                `json:"agent_hostname"`
	BackupID      uuid.UUID             `json:"backup_id"`
	Status        RansomwareAlertStatus `json:"status"`
	RiskScore     int                   `json:"risk_score"` // 0-100
	Indicators    map[string]any        `json:"indicators,omitempty"`
	FilesChanged  int                   `json:"files_changed"`
	FilesNew      int                   `json:"files_new"`
	TotalFiles    int                   `json:"total_files"`
	BackupsPaused bool                  `json:"backups_paused"`
	PausedAt      *time.Time            `json:"paused_at,omitempty"`
	ResumedAt     *time.Time            `json:"resumed_at,omitempty"`
	ResolvedBy    *uuid.UUID            `json:"resolved_by,omitempty"`
	ResolvedAt    *time.Time            `json:"resolved_at,omitempty"`
	Resolution    string                `json:"resolution,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// NewRansomwareAlert creates a new RansomwareAlert.
func NewRansomwareAlert(
	orgID, scheduleID, agentID, backupID uuid.UUID,
	riskScore int,
) *RansomwareAlert {
	now := time.Now()
	return &RansomwareAlert{
		ID:         uuid.New(),
		OrgID:      orgID,
		ScheduleID: scheduleID,
		AgentID:    agentID,
		BackupID:   backupID,
		Status:     RansomwareAlertStatusActive,
		RiskScore:  riskScore,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// SetIndicators sets the indicators metadata.
func (r *RansomwareAlert) SetIndicators(indicators any) error {
	data, err := json.Marshal(indicators)
	if err != nil {
		return err
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		// If it's an array, wrap it
		r.Indicators = map[string]any{"items": indicators}
		return nil
	}
	r.Indicators = metadata
	return nil
}

// SetIndicatorsFromBytes sets the indicators from JSON bytes.
func (r *RansomwareAlert) SetIndicatorsFromBytes(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var indicators map[string]any
	if err := json.Unmarshal(data, &indicators); err != nil {
		return err
	}
	r.Indicators = indicators
	return nil
}

// IndicatorsJSON returns the indicators as JSON bytes for database storage.
func (r *RansomwareAlert) IndicatorsJSON() ([]byte, error) {
	if r.Indicators == nil {
		return nil, nil
	}
	return json.Marshal(r.Indicators)
}

// Investigate marks the alert as under investigation.
func (r *RansomwareAlert) Investigate() {
	r.Status = RansomwareAlertStatusInvestigating
	r.UpdatedAt = time.Now()
}

// MarkFalsePositive marks the alert as a false positive.
func (r *RansomwareAlert) MarkFalsePositive(userID uuid.UUID, resolution string) {
	now := time.Now()
	r.Status = RansomwareAlertStatusFalsePositive
	r.ResolvedBy = &userID
	r.ResolvedAt = &now
	r.Resolution = resolution
	r.UpdatedAt = now
}

// MarkConfirmed marks the alert as confirmed ransomware.
func (r *RansomwareAlert) MarkConfirmed(userID uuid.UUID, resolution string) {
	now := time.Now()
	r.Status = RansomwareAlertStatusConfirmed
	r.ResolvedBy = &userID
	r.ResolvedAt = &now
	r.Resolution = resolution
	r.UpdatedAt = now
}

// Resolve marks the alert as resolved.
func (r *RansomwareAlert) Resolve(userID uuid.UUID, resolution string) {
	now := time.Now()
	r.Status = RansomwareAlertStatusResolved
	r.ResolvedBy = &userID
	r.ResolvedAt = &now
	r.Resolution = resolution
	r.UpdatedAt = now
}

// ResumeBackups marks that backups have been resumed.
func (r *RansomwareAlert) ResumeBackups() {
	now := time.Now()
	r.BackupsPaused = false
	r.ResumedAt = &now
	r.UpdatedAt = now
}

// IsCritical returns true if the risk score is >= 80.
func (r *RansomwareAlert) IsCritical() bool {
	return r.RiskScore >= 80
}

// IsActive returns true if the alert is still active.
func (r *RansomwareAlert) IsActive() bool {
	return r.Status == RansomwareAlertStatusActive ||
		r.Status == RansomwareAlertStatusInvestigating
}

// ChangePercentage calculates the percentage of files that changed.
func (r *RansomwareAlert) ChangePercentage() float64 {
	if r.TotalFiles == 0 {
		return 0
	}
	return float64(r.FilesChanged+r.FilesNew) / float64(r.TotalFiles) * 100
}
