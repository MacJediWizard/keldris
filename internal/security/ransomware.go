// Package security provides security-related detection and analysis features.
package security

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Common ransomware file extensions.
var DefaultRansomwareExtensions = []string{
	".encrypted", ".locked", ".crypto", ".crypt", ".enc",
	".locky", ".zepto", ".odin", ".thor", ".aesir",
	".crypted", ".cerber", ".cerber2", ".cerber3",
	".cryp1", ".crypz", ".cryptolocker", ".cryptowall",
	".ctb", ".ctb2", ".ctbl", ".crinf", ".crjoker",
	".coverton", ".keybtc@inbox_com", ".kraken",
	".kkk", ".btc", ".fun", ".gws", ".legion",
	".lesli", ".micro", ".mp3", ".neitrino", ".nocry",
	".onion", ".osiris", ".oops", ".paym", ".paymrss",
	".payms", ".paymts", ".paymt", ".paymst",
	".petya", ".ransom", ".ransomware", ".sage",
	".shit", ".thor", ".ttt", ".vvv", ".wallet",
	".wncry", ".wncryt", ".wcry", ".wanna",
	".wannacry", ".xxx", ".xyz", ".zzz", ".aaa",
	".abc", ".ecc", ".exx", ".ezz", ".rdmk",
	".r5a", ".r4a", ".r3d", ".r2d", ".r1d",
}

// RansomwareDetector analyzes backup statistics for potential ransomware activity.
type RansomwareDetector struct {
	store  RansomwareStore
	logger zerolog.Logger
}

// RansomwareStore defines the database operations needed for ransomware detection.
type RansomwareStore interface {
	GetRansomwareSettingsByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*models.RansomwareSettings, error)
	GetRansomwareSettingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareSettings, error)
	CreateRansomwareSettings(ctx context.Context, settings *models.RansomwareSettings) error
	UpdateRansomwareSettings(ctx context.Context, settings *models.RansomwareSettings) error
	DeleteRansomwareSettings(ctx context.Context, id uuid.UUID) error
	CreateRansomwareAlert(ctx context.Context, alert *models.RansomwareAlert) error
	GetRansomwareAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareAlert, error)
	GetActiveRansomwareAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareAlert, error)
	GetRansomwareAlertByID(ctx context.Context, id uuid.UUID) (*models.RansomwareAlert, error)
	UpdateRansomwareAlert(ctx context.Context, alert *models.RansomwareAlert) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	PauseSchedule(ctx context.Context, scheduleID uuid.UUID) error
}

// NewRansomwareDetector creates a new RansomwareDetector instance.
func NewRansomwareDetector(store RansomwareStore, logger zerolog.Logger) *RansomwareDetector {
	return &RansomwareDetector{
		store:  store,
		logger: logger.With().Str("component", "ransomware_detector").Logger(),
	}
}

// AnalysisInput contains the data needed to analyze a backup for ransomware.
type AnalysisInput struct {
	OrgID           uuid.UUID
	ScheduleID      uuid.UUID
	AgentID         uuid.UUID
	BackupID        uuid.UUID
	TotalFiles      int
	FilesNew        int
	FilesChanged    int
	FilesDeleted    int
	NewFilenames    []string // Names of new files (for extension analysis)
	AverageEntropy  float64  // Average entropy of changed files (0-8 scale)
}

// AnalysisResult contains the results of ransomware analysis.
type AnalysisResult struct {
	IsRansomwareSuspected bool
	RiskScore             int // 0-100
	Indicators            []RansomwareIndicator
	Recommendation        string
}

// RansomwareIndicator represents a single indicator of ransomware activity.
type RansomwareIndicator struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	Severity    string  `json:"severity"` // low, medium, high, critical
}

// Analyze checks backup statistics for potential ransomware activity.
func (d *RansomwareDetector) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisResult, error) {
	// Get ransomware settings for this schedule
	settings, err := d.store.GetRansomwareSettingsByScheduleID(ctx, input.ScheduleID)
	if err != nil {
		d.logger.Debug().
			Err(err).
			Str("schedule_id", input.ScheduleID.String()).
			Msg("no ransomware settings found, using defaults")
		settings = models.DefaultRansomwareSettings(input.ScheduleID)
	}

	if !settings.Enabled {
		return &AnalysisResult{
			IsRansomwareSuspected: false,
			RiskScore:             0,
			Indicators:            nil,
			Recommendation:        "Ransomware detection is disabled for this schedule",
		}, nil
	}

	result := &AnalysisResult{
		Indicators: make([]RansomwareIndicator, 0),
	}

	// Calculate file change percentage
	if input.TotalFiles > 0 {
		changePercent := float64(input.FilesChanged+input.FilesNew) / float64(input.TotalFiles) * 100
		if changePercent >= float64(settings.ChangeThresholdPercent) {
			result.Indicators = append(result.Indicators, RansomwareIndicator{
				Type:        "file_change_rate",
				Description: "High percentage of files changed or added",
				Value:       changePercent,
				Threshold:   float64(settings.ChangeThresholdPercent),
				Severity:    d.calculateSeverity(changePercent, float64(settings.ChangeThresholdPercent)),
			})
		}
	}

	// Detect ransomware extensions
	extensions := settings.GetExtensions()
	if len(extensions) == 0 {
		extensions = DefaultRansomwareExtensions
	}

	suspiciousExtensions := d.detectRansomwareExtensions(input.NewFilenames, extensions)
	if len(suspiciousExtensions) > 0 {
		result.Indicators = append(result.Indicators, RansomwareIndicator{
			Type:        "ransomware_extensions",
			Description: "Files with known ransomware extensions detected: " + strings.Join(suspiciousExtensions, ", "),
			Value:       float64(len(suspiciousExtensions)),
			Threshold:   1,
			Severity:    "critical",
		})
	}

	// Detect entropy changes (encrypted files have high entropy)
	if settings.EntropyDetectionEnabled && input.AverageEntropy > 0 {
		if input.AverageEntropy >= settings.EntropyThreshold {
			result.Indicators = append(result.Indicators, RansomwareIndicator{
				Type:        "high_entropy",
				Description: "High file entropy detected (possible encryption)",
				Value:       input.AverageEntropy,
				Threshold:   settings.EntropyThreshold,
				Severity:    d.calculateEntropySeverity(input.AverageEntropy, settings.EntropyThreshold),
			})
		}
	}

	// Calculate overall risk score
	result.RiskScore = d.calculateRiskScore(result.Indicators)
	result.IsRansomwareSuspected = result.RiskScore >= 50 || d.hasCriticalIndicator(result.Indicators)

	// Set recommendation
	result.Recommendation = d.generateRecommendation(result)

	return result, nil
}

// CreateAlertFromAnalysis creates a ransomware alert from analysis results.
func (d *RansomwareDetector) CreateAlertFromAnalysis(
	ctx context.Context,
	input AnalysisInput,
	result *AnalysisResult,
) (*models.RansomwareAlert, error) {
	if !result.IsRansomwareSuspected {
		return nil, nil
	}

	// Get schedule info for alert details
	schedule, err := d.store.GetScheduleByID(ctx, input.ScheduleID)
	if err != nil {
		return nil, err
	}

	agent, err := d.store.GetAgentByID(ctx, input.AgentID)
	if err != nil {
		return nil, err
	}

	alert := models.NewRansomwareAlert(
		input.OrgID,
		input.ScheduleID,
		input.AgentID,
		input.BackupID,
		result.RiskScore,
	)

	alert.ScheduleName = schedule.Name
	alert.AgentHostname = agent.Hostname
	alert.FilesChanged = input.FilesChanged
	alert.FilesNew = input.FilesNew
	alert.TotalFiles = input.TotalFiles

	// Set indicators as metadata
	alert.SetIndicators(result.Indicators)

	if err := d.store.CreateRansomwareAlert(ctx, alert); err != nil {
		return nil, err
	}

	d.logger.Warn().
		Str("alert_id", alert.ID.String()).
		Str("schedule_id", input.ScheduleID.String()).
		Int("risk_score", result.RiskScore).
		Int("indicators", len(result.Indicators)).
		Msg("ransomware alert created")

	return alert, nil
}

// PauseBackupsIfRequired pauses backups for a schedule if the settings require it.
func (d *RansomwareDetector) PauseBackupsIfRequired(ctx context.Context, scheduleID uuid.UUID, alert *models.RansomwareAlert) error {
	settings, err := d.store.GetRansomwareSettingsByScheduleID(ctx, scheduleID)
	if err != nil {
		return nil // No settings, don't pause
	}

	if !settings.AutoPauseOnAlert {
		return nil
	}

	if err := d.store.PauseSchedule(ctx, scheduleID); err != nil {
		return err
	}

	alert.BackupsPaused = true
	alert.PausedAt = timePtr(time.Now())

	if err := d.store.UpdateRansomwareAlert(ctx, alert); err != nil {
		return err
	}

	d.logger.Warn().
		Str("schedule_id", scheduleID.String()).
		Str("alert_id", alert.ID.String()).
		Msg("backups paused due to ransomware detection")

	return nil
}

func (d *RansomwareDetector) detectRansomwareExtensions(filenames []string, extensions []string) []string {
	found := make(map[string]struct{})
	for _, filename := range filenames {
		ext := strings.ToLower(filepath.Ext(filename))
		for _, ransomExt := range extensions {
			if ext == strings.ToLower(ransomExt) {
				found[ext] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(found))
	for ext := range found {
		result = append(result, ext)
	}
	return result
}

func (d *RansomwareDetector) calculateSeverity(value, threshold float64) string {
	ratio := value / threshold
	switch {
	case ratio >= 3:
		return "critical"
	case ratio >= 2:
		return "high"
	case ratio >= 1.5:
		return "medium"
	default:
		return "low"
	}
}

func (d *RansomwareDetector) calculateEntropySeverity(entropy, threshold float64) string {
	// Entropy scale is 0-8 (bits per byte)
	// Random/encrypted data is typically > 7.5
	switch {
	case entropy >= 7.8:
		return "critical"
	case entropy >= 7.5:
		return "high"
	case entropy >= threshold:
		return "medium"
	default:
		return "low"
	}
}

func (d *RansomwareDetector) calculateRiskScore(indicators []RansomwareIndicator) int {
	if len(indicators) == 0 {
		return 0
	}

	score := 0.0
	for _, ind := range indicators {
		switch ind.Severity {
		case "critical":
			score += 40
		case "high":
			score += 25
		case "medium":
			score += 15
		case "low":
			score += 5
		}
	}

	// Cap at 100
	return int(math.Min(score, 100))
}

func (d *RansomwareDetector) hasCriticalIndicator(indicators []RansomwareIndicator) bool {
	for _, ind := range indicators {
		if ind.Severity == "critical" {
			return true
		}
	}
	return false
}

func (d *RansomwareDetector) generateRecommendation(result *AnalysisResult) string {
	if !result.IsRansomwareSuspected {
		return "No suspicious activity detected"
	}

	switch {
	case result.RiskScore >= 80:
		return "CRITICAL: Immediate investigation required. Consider isolating affected systems and reviewing recent file changes. Do not restore from potentially compromised backups."
	case result.RiskScore >= 60:
		return "HIGH RISK: Review file changes immediately. Verify with users if recent bulk changes were expected. Consider pausing automatic backups pending review."
	case result.RiskScore >= 40:
		return "MODERATE RISK: Unusual backup patterns detected. Review recent file changes and verify with system users."
	default:
		return "LOW RISK: Minor anomalies detected. Monitor for continued unusual activity."
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
