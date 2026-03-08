package security

import (
	"context"
	"fmt"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockRansomwareStore implements RansomwareStore for testing.
type mockRansomwareStore struct {
	settings       *models.RansomwareSettings
	settingsErr    error
	schedule       *models.Schedule
	scheduleErr    error
	agent          *models.Agent
	agentErr       error
	createdAlert   *models.RansomwareAlert
	createAlertErr error
	pausedID       *uuid.UUID
	pauseErr       error
	updatedAlert   *models.RansomwareAlert
	updateErr      error
}

func (m *mockRansomwareStore) GetRansomwareSettingsByScheduleID(_ context.Context, _ uuid.UUID) (*models.RansomwareSettings, error) {
	if m.settingsErr != nil {
		return nil, m.settingsErr
	}
	return m.settings, nil
}

func (m *mockRansomwareStore) GetRansomwareSettingsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.RansomwareSettings, error) {
	return nil, nil
}

func (m *mockRansomwareStore) CreateRansomwareSettings(_ context.Context, _ *models.RansomwareSettings) error {
	return nil
}

func (m *mockRansomwareStore) UpdateRansomwareSettings(_ context.Context, _ *models.RansomwareSettings) error {
	return nil
}

func (m *mockRansomwareStore) DeleteRansomwareSettings(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRansomwareStore) CreateRansomwareAlert(_ context.Context, alert *models.RansomwareAlert) error {
	m.createdAlert = alert
	return m.createAlertErr
}

func (m *mockRansomwareStore) GetRansomwareAlertsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.RansomwareAlert, error) {
	return nil, nil
}

func (m *mockRansomwareStore) GetActiveRansomwareAlertsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.RansomwareAlert, error) {
	return nil, nil
}

func (m *mockRansomwareStore) GetRansomwareAlertByID(_ context.Context, _ uuid.UUID) (*models.RansomwareAlert, error) {
	return nil, nil
}

func (m *mockRansomwareStore) UpdateRansomwareAlert(_ context.Context, alert *models.RansomwareAlert) error {
	m.updatedAlert = alert
	return m.updateErr
}

func (m *mockRansomwareStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	if m.scheduleErr != nil {
		return nil, m.scheduleErr
	}
	return m.schedule, nil
}

func (m *mockRansomwareStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func (m *mockRansomwareStore) PauseSchedule(_ context.Context, id uuid.UUID) error {
	m.pausedID = &id
	return m.pauseErr
}

func newTestDetector(store RansomwareStore) *RansomwareDetector {
	return NewRansomwareDetector(store, zerolog.Nop())
}

func defaultSettings(scheduleID uuid.UUID) *models.RansomwareSettings {
	s := models.DefaultRansomwareSettings(scheduleID)
	s.Enabled = true
	return s
}

// --- Analyze Tests ---

func TestAnalyze_DisabledReturnsZeroScore(t *testing.T) {
	scheduleID := uuid.New()
	settings := defaultSettings(scheduleID)
	settings.Enabled = false

	store := &mockRansomwareStore{settings: settings}
	d := newTestDetector(store)

	result, err := d.Analyze(context.Background(), AnalysisInput{
		ScheduleID: scheduleID,
		TotalFiles: 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsRansomwareSuspected {
		t.Error("expected ransomware not suspected when disabled")
	}
	if result.RiskScore != 0 {
		t.Errorf("expected risk score 0, got %d", result.RiskScore)
	}
	if result.Recommendation != "Ransomware detection is disabled for this schedule" {
		t.Errorf("unexpected recommendation: %s", result.Recommendation)
	}
}

func TestAnalyze_NoSettingsUsesDefaults(t *testing.T) {
	store := &mockRansomwareStore{
		settingsErr: fmt.Errorf("not found"),
	}
	d := newTestDetector(store)

	// With default settings (30% threshold), 50% change should trigger
	result, err := d.Analyze(context.Background(), AnalysisInput{
		ScheduleID:   uuid.New(),
		TotalFiles:   100,
		FilesChanged: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RiskScore == 0 {
		t.Error("expected non-zero risk score when defaults are used and threshold exceeded")
	}
}

func TestAnalyze_FileChangeRate(t *testing.T) {
	tests := []struct {
		name           string
		totalFiles     int
		filesChanged   int
		filesNew       int
		threshold      int
		wantIndicator  bool
		wantSeverity   string
	}{
		{
			name:          "below threshold",
			totalFiles:    100,
			filesChanged:  10,
			threshold:     30,
			wantIndicator: false,
		},
		{
			name:          "at threshold",
			totalFiles:    100,
			filesChanged:  30,
			threshold:     30,
			wantIndicator: true,
			wantSeverity:  "low",
		},
		{
			name:          "1.5x threshold is medium",
			totalFiles:    100,
			filesChanged:  45,
			threshold:     30,
			wantIndicator: true,
			wantSeverity:  "medium",
		},
		{
			name:          "2x threshold is high",
			totalFiles:    100,
			filesChanged:  60,
			threshold:     30,
			wantIndicator: true,
			wantSeverity:  "high",
		},
		{
			name:          "3x threshold is critical",
			totalFiles:    100,
			filesChanged:  90,
			threshold:     30,
			wantIndicator: true,
			wantSeverity:  "critical",
		},
		{
			name:          "new files count toward change rate",
			totalFiles:    100,
			filesChanged:  10,
			filesNew:      25,
			threshold:     30,
			wantIndicator: true,
			wantSeverity:  "low",
		},
		{
			name:          "zero total files skips check",
			totalFiles:    0,
			filesChanged:  0,
			threshold:     30,
			wantIndicator: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduleID := uuid.New()
			settings := defaultSettings(scheduleID)
			settings.ChangeThresholdPercent = tt.threshold
			settings.EntropyDetectionEnabled = false // isolate this test

			store := &mockRansomwareStore{settings: settings}
			d := newTestDetector(store)

			result, err := d.Analyze(context.Background(), AnalysisInput{
				ScheduleID:   scheduleID,
				TotalFiles:   tt.totalFiles,
				FilesChanged: tt.filesChanged,
				FilesNew:     tt.filesNew,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			found := false
			for _, ind := range result.Indicators {
				if ind.Type == "file_change_rate" {
					found = true
					if tt.wantSeverity != "" && ind.Severity != tt.wantSeverity {
						t.Errorf("severity = %s, want %s", ind.Severity, tt.wantSeverity)
					}
				}
			}
			if found != tt.wantIndicator {
				t.Errorf("file_change_rate indicator found = %v, want %v", found, tt.wantIndicator)
			}
		})
	}
}

func TestAnalyze_RansomwareExtensions(t *testing.T) {
	tests := []struct {
		name          string
		filenames     []string
		wantDetected  bool
		wantCount     int
		customExts    []string
	}{
		{
			name:         "no files",
			filenames:    nil,
			wantDetected: false,
		},
		{
			name:         "normal files",
			filenames:    []string{"report.pdf", "photo.jpg", "document.docx"},
			wantDetected: false,
		},
		{
			name:         "single ransomware extension",
			filenames:    []string{"file.encrypted"},
			wantDetected: true,
			wantCount:    1,
		},
		{
			name:         "multiple ransomware extensions",
			filenames:    []string{"a.encrypted", "b.locked", "c.crypto"},
			wantDetected: true,
			wantCount:    3,
		},
		{
			name:         "mixed normal and ransomware",
			filenames:    []string{"report.pdf", "data.locked", "photo.jpg"},
			wantDetected: true,
			wantCount:    1,
		},
		{
			name:         "case insensitive matching",
			filenames:    []string{"FILE.ENCRYPTED", "data.Locked"},
			wantDetected: true,
			wantCount:    2,
		},
		{
			name:         "duplicate extensions counted once",
			filenames:    []string{"a.encrypted", "b.encrypted", "c.encrypted"},
			wantDetected: true,
			wantCount:    1, // unique extensions
		},
		{
			name:         "wannacry variants",
			filenames:    []string{"file.wncry", "data.wncryt", "doc.wcry"},
			wantDetected: true,
			wantCount:    3,
		},
		{
			name:         "custom extensions",
			filenames:    []string{"file.custom_ransom", "data.txt"},
			customExts:   []string{".custom_ransom"},
			wantDetected: true,
			wantCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduleID := uuid.New()
			settings := defaultSettings(scheduleID)
			settings.ChangeThresholdPercent = 100 // disable change rate check
			settings.EntropyDetectionEnabled = false
			if tt.customExts != nil {
				settings.ExtensionsToDetect = tt.customExts
			}

			store := &mockRansomwareStore{settings: settings}
			d := newTestDetector(store)

			result, err := d.Analyze(context.Background(), AnalysisInput{
				ScheduleID:   scheduleID,
				TotalFiles:   100,
				NewFilenames: tt.filenames,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			found := false
			for _, ind := range result.Indicators {
				if ind.Type == "ransomware_extensions" {
					found = true
					if int(ind.Value) != tt.wantCount {
						t.Errorf("extension count = %d, want %d", int(ind.Value), tt.wantCount)
					}
					if ind.Severity != "critical" {
						t.Errorf("ransomware_extensions severity = %s, want critical", ind.Severity)
					}
				}
			}
			if found != tt.wantDetected {
				t.Errorf("ransomware_extensions detected = %v, want %v", found, tt.wantDetected)
			}
		})
	}
}

func TestAnalyze_EntropyDetection(t *testing.T) {
	tests := []struct {
		name         string
		entropy      float64
		threshold    float64
		enabled      bool
		wantDetected bool
		wantSeverity string
	}{
		{
			name:         "disabled ignores entropy",
			entropy:      7.9,
			threshold:    7.0,
			enabled:      false,
			wantDetected: false,
		},
		{
			name:         "zero entropy ignored",
			entropy:      0,
			threshold:    7.0,
			enabled:      true,
			wantDetected: false,
		},
		{
			name:         "below threshold",
			entropy:      6.5,
			threshold:    7.0,
			enabled:      true,
			wantDetected: false,
		},
		{
			name:         "at threshold is medium",
			entropy:      7.0,
			threshold:    7.0,
			enabled:      true,
			wantDetected: true,
			wantSeverity: "medium",
		},
		{
			name:         "7.5 is high",
			entropy:      7.5,
			threshold:    7.0,
			enabled:      true,
			wantDetected: true,
			wantSeverity: "high",
		},
		{
			name:         "7.8 is critical",
			entropy:      7.8,
			threshold:    7.0,
			enabled:      true,
			wantDetected: true,
			wantSeverity: "critical",
		},
		{
			name:         "7.99 is critical",
			entropy:      7.99,
			threshold:    7.0,
			enabled:      true,
			wantDetected: true,
			wantSeverity: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduleID := uuid.New()
			settings := defaultSettings(scheduleID)
			settings.ChangeThresholdPercent = 100 // disable change rate check
			settings.EntropyDetectionEnabled = tt.enabled
			settings.EntropyThreshold = tt.threshold

			store := &mockRansomwareStore{settings: settings}
			d := newTestDetector(store)

			result, err := d.Analyze(context.Background(), AnalysisInput{
				ScheduleID:     scheduleID,
				TotalFiles:     100,
				AverageEntropy: tt.entropy,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			found := false
			for _, ind := range result.Indicators {
				if ind.Type == "high_entropy" {
					found = true
					if ind.Severity != tt.wantSeverity {
						t.Errorf("severity = %s, want %s", ind.Severity, tt.wantSeverity)
					}
				}
			}
			if found != tt.wantDetected {
				t.Errorf("high_entropy detected = %v, want %v", found, tt.wantDetected)
			}
		})
	}
}

// --- Risk Score Tests ---

func TestCalculateRiskScore(t *testing.T) {
	d := newTestDetector(&mockRansomwareStore{})

	tests := []struct {
		name       string
		indicators []RansomwareIndicator
		wantScore  int
	}{
		{
			name:       "no indicators",
			indicators: nil,
			wantScore:  0,
		},
		{
			name:       "empty indicators",
			indicators: []RansomwareIndicator{},
			wantScore:  0,
		},
		{
			name: "single low",
			indicators: []RansomwareIndicator{
				{Severity: "low"},
			},
			wantScore: 5,
		},
		{
			name: "single medium",
			indicators: []RansomwareIndicator{
				{Severity: "medium"},
			},
			wantScore: 15,
		},
		{
			name: "single high",
			indicators: []RansomwareIndicator{
				{Severity: "high"},
			},
			wantScore: 25,
		},
		{
			name: "single critical",
			indicators: []RansomwareIndicator{
				{Severity: "critical"},
			},
			wantScore: 40,
		},
		{
			name: "critical plus high",
			indicators: []RansomwareIndicator{
				{Severity: "critical"},
				{Severity: "high"},
			},
			wantScore: 65,
		},
		{
			name: "capped at 100",
			indicators: []RansomwareIndicator{
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "critical"},
			},
			wantScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.calculateRiskScore(tt.indicators)
			if got != tt.wantScore {
				t.Errorf("calculateRiskScore() = %d, want %d", got, tt.wantScore)
			}
		})
	}
}

// --- Suspected Flag Tests ---

func TestAnalyze_IsRansomwareSuspected(t *testing.T) {
	tests := []struct {
		name          string
		filesChanged  int
		newFilenames  []string
		entropy       float64
		wantSuspected bool
	}{
		{
			name:          "normal backup not suspected",
			filesChanged:  5,
			wantSuspected: false,
		},
		{
			name:          "critical extension always suspected",
			filesChanged:  0,
			newFilenames:  []string{"file.encrypted"},
			wantSuspected: true, // hasCriticalIndicator returns true
		},
		{
			name:          "high change rate plus high entropy suspected",
			filesChanged:  90,
			entropy:       7.9,
			wantSuspected: true, // score >= 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduleID := uuid.New()
			settings := defaultSettings(scheduleID)
			settings.EntropyDetectionEnabled = true
			settings.EntropyThreshold = 7.5

			store := &mockRansomwareStore{settings: settings}
			d := newTestDetector(store)

			result, err := d.Analyze(context.Background(), AnalysisInput{
				ScheduleID:     scheduleID,
				TotalFiles:     100,
				FilesChanged:   tt.filesChanged,
				NewFilenames:   tt.newFilenames,
				AverageEntropy: tt.entropy,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.IsRansomwareSuspected != tt.wantSuspected {
				t.Errorf("IsRansomwareSuspected = %v, want %v (score=%d, indicators=%d)",
					result.IsRansomwareSuspected, tt.wantSuspected,
					result.RiskScore, len(result.Indicators))
			}
		})
	}
}

// --- Recommendation Tests ---

func TestAnalyze_Recommendations(t *testing.T) {
	tests := []struct {
		name           string
		filesChanged   int
		newFilenames   []string
		entropy        float64
		wantSubstring  string
	}{
		{
			name:          "no suspicious activity",
			filesChanged:  5,
			wantSubstring: "No suspicious activity",
		},
		{
			name:          "critical score",
			filesChanged:  90,
			newFilenames:  []string{"a.encrypted", "b.locked"},
			entropy:       7.9,
			wantSubstring: "CRITICAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduleID := uuid.New()
			settings := defaultSettings(scheduleID)
			settings.EntropyDetectionEnabled = true
			settings.EntropyThreshold = 7.5

			store := &mockRansomwareStore{settings: settings}
			d := newTestDetector(store)

			result, err := d.Analyze(context.Background(), AnalysisInput{
				ScheduleID:     scheduleID,
				TotalFiles:     100,
				FilesChanged:   tt.filesChanged,
				NewFilenames:   tt.newFilenames,
				AverageEntropy: tt.entropy,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Recommendation) == 0 {
				t.Error("expected non-empty recommendation")
			}
			// Just verify the recommendation is set and non-empty.
			// The specific text is tested via the wantSubstring.
			found := false
			if tt.wantSubstring != "" {
				for i := 0; i <= len(result.Recommendation)-len(tt.wantSubstring); i++ {
					if result.Recommendation[i:i+len(tt.wantSubstring)] == tt.wantSubstring {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("recommendation %q does not contain %q", result.Recommendation, tt.wantSubstring)
				}
			}
		})
	}
}

// --- Edge Cases ---

func TestAnalyze_EmptyBackup(t *testing.T) {
	scheduleID := uuid.New()
	settings := defaultSettings(scheduleID)

	store := &mockRansomwareStore{settings: settings}
	d := newTestDetector(store)

	result, err := d.Analyze(context.Background(), AnalysisInput{
		ScheduleID: scheduleID,
		TotalFiles: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RiskScore != 0 {
		t.Errorf("expected 0 risk score for empty backup, got %d", result.RiskScore)
	}
	if result.IsRansomwareSuspected {
		t.Error("empty backup should not be suspected")
	}
}

func TestAnalyze_SingleFileBackup(t *testing.T) {
	scheduleID := uuid.New()
	settings := defaultSettings(scheduleID)

	store := &mockRansomwareStore{settings: settings}
	d := newTestDetector(store)

	// One changed file out of one total = 100% change rate
	result, err := d.Analyze(context.Background(), AnalysisInput{
		ScheduleID:   scheduleID,
		TotalFiles:   1,
		FilesChanged: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 100% change / 30% threshold = ratio ~3.33 -> "critical" severity -> 40 points
	if result.RiskScore == 0 {
		t.Error("expected non-zero risk score for 100% file change")
	}
}

func TestAnalyze_AllFilesEncryptedExtension(t *testing.T) {
	scheduleID := uuid.New()
	settings := defaultSettings(scheduleID)
	settings.EntropyDetectionEnabled = true
	settings.EntropyThreshold = 7.0

	store := &mockRansomwareStore{settings: settings}
	d := newTestDetector(store)

	filenames := make([]string, 100)
	for i := range filenames {
		filenames[i] = fmt.Sprintf("file%d.encrypted", i)
	}

	result, err := d.Analyze(context.Background(), AnalysisInput{
		ScheduleID:     scheduleID,
		TotalFiles:     100,
		FilesNew:       100,
		NewFilenames:   filenames,
		AverageEntropy: 7.95,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsRansomwareSuspected {
		t.Error("expected ransomware to be suspected with all encrypted files")
	}
	// Should have at least 3 indicators: change rate, extensions, entropy
	if len(result.Indicators) < 3 {
		t.Errorf("expected at least 3 indicators, got %d", len(result.Indicators))
	}
	if result.RiskScore < 80 {
		t.Errorf("expected risk score >= 80, got %d", result.RiskScore)
	}
}

// --- detectRansomwareExtensions Tests ---

func TestDetectRansomwareExtensions(t *testing.T) {
	d := newTestDetector(&mockRansomwareStore{})

	tests := []struct {
		name       string
		filenames  []string
		extensions []string
		wantCount  int
	}{
		{
			name:       "empty filenames",
			filenames:  nil,
			extensions: DefaultRansomwareExtensions,
			wantCount:  0,
		},
		{
			name:       "empty extensions list",
			filenames:  []string{"file.encrypted"},
			extensions: nil,
			wantCount:  0,
		},
		{
			name:       "no matches",
			filenames:  []string{"file.txt", "data.csv"},
			extensions: DefaultRansomwareExtensions,
			wantCount:  0,
		},
		{
			name:       "match encrypted",
			filenames:  []string{"file.encrypted"},
			extensions: DefaultRansomwareExtensions,
			wantCount:  1,
		},
		{
			name:       "case insensitive",
			filenames:  []string{"FILE.ENCRYPTED"},
			extensions: []string{".encrypted"},
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.detectRansomwareExtensions(tt.filenames, tt.extensions)
			if len(got) != tt.wantCount {
				t.Errorf("detectRansomwareExtensions() returned %d extensions, want %d: %v", len(got), tt.wantCount, got)
			}
		})
	}
}

// --- Severity Calculation Tests ---

func TestCalculateSeverity(t *testing.T) {
	d := newTestDetector(&mockRansomwareStore{})

	tests := []struct {
		name      string
		value     float64
		threshold float64
		want      string
	}{
		{"ratio 1.0 is low", 30, 30, "low"},
		{"ratio 1.4 is low", 42, 30, "low"},
		{"ratio 1.5 is medium", 45, 30, "medium"},
		{"ratio 1.99 is medium", 59.7, 30, "medium"},
		{"ratio 2.0 is high", 60, 30, "high"},
		{"ratio 2.99 is high", 89.7, 30, "high"},
		{"ratio 3.0 is critical", 90, 30, "critical"},
		{"ratio 5.0 is critical", 150, 30, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.calculateSeverity(tt.value, tt.threshold)
			if got != tt.want {
				t.Errorf("calculateSeverity(%f, %f) = %s, want %s", tt.value, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestCalculateEntropySeverity(t *testing.T) {
	d := newTestDetector(&mockRansomwareStore{})

	tests := []struct {
		name      string
		entropy   float64
		threshold float64
		want      string
	}{
		{"below threshold is low", 6.5, 7.0, "low"},
		{"at threshold is medium", 7.0, 7.0, "medium"},
		{"7.4 is medium", 7.4, 7.0, "medium"},
		{"7.5 is high", 7.5, 7.0, "high"},
		{"7.7 is high", 7.7, 7.0, "high"},
		{"7.8 is critical", 7.8, 7.0, "critical"},
		{"7.99 is critical", 7.99, 7.0, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.calculateEntropySeverity(tt.entropy, tt.threshold)
			if got != tt.want {
				t.Errorf("calculateEntropySeverity(%f, %f) = %s, want %s", tt.entropy, tt.threshold, got, tt.want)
			}
		})
	}
}

// --- hasCriticalIndicator Tests ---

func TestHasCriticalIndicator(t *testing.T) {
	d := newTestDetector(&mockRansomwareStore{})

	tests := []struct {
		name       string
		indicators []RansomwareIndicator
		want       bool
	}{
		{
			name:       "nil indicators",
			indicators: nil,
			want:       false,
		},
		{
			name:       "empty indicators",
			indicators: []RansomwareIndicator{},
			want:       false,
		},
		{
			name: "no critical",
			indicators: []RansomwareIndicator{
				{Severity: "low"},
				{Severity: "medium"},
				{Severity: "high"},
			},
			want: false,
		},
		{
			name: "has critical",
			indicators: []RansomwareIndicator{
				{Severity: "low"},
				{Severity: "critical"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.hasCriticalIndicator(tt.indicators)
			if got != tt.want {
				t.Errorf("hasCriticalIndicator() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- CreateAlertFromAnalysis Tests ---

func TestCreateAlertFromAnalysis_NotSuspected(t *testing.T) {
	store := &mockRansomwareStore{}
	d := newTestDetector(store)

	result := &AnalysisResult{IsRansomwareSuspected: false}
	alert, err := d.CreateAlertFromAnalysis(context.Background(), AnalysisInput{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected nil alert when not suspected")
	}
}

func TestCreateAlertFromAnalysis_CreatesAlert(t *testing.T) {
	scheduleID := uuid.New()
	agentID := uuid.New()

	store := &mockRansomwareStore{
		schedule: &models.Schedule{
			ID:   scheduleID,
			Name: "daily-backup",
		},
		agent: &models.Agent{
			ID:       agentID,
			Hostname: "server-01",
		},
	}
	d := newTestDetector(store)

	input := AnalysisInput{
		OrgID:        uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		BackupID:     uuid.New(),
		TotalFiles:   1000,
		FilesChanged: 500,
		FilesNew:     100,
	}
	result := &AnalysisResult{
		IsRansomwareSuspected: true,
		RiskScore:             85,
		Indicators: []RansomwareIndicator{
			{Type: "file_change_rate", Severity: "critical"},
		},
	}

	alert, err := d.CreateAlertFromAnalysis(context.Background(), input, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert == nil {
		t.Fatal("expected non-nil alert")
	}
	if alert.ScheduleName != "daily-backup" {
		t.Errorf("schedule name = %s, want daily-backup", alert.ScheduleName)
	}
	if alert.AgentHostname != "server-01" {
		t.Errorf("agent hostname = %s, want server-01", alert.AgentHostname)
	}
	if alert.FilesChanged != 500 {
		t.Errorf("files changed = %d, want 500", alert.FilesChanged)
	}
	if alert.RiskScore != 85 {
		t.Errorf("risk score = %d, want 85", alert.RiskScore)
	}
	if store.createdAlert == nil {
		t.Error("expected alert to be persisted to store")
	}
}

// --- PauseBackupsIfRequired Tests ---

func TestPauseBackupsIfRequired_NoPauseWhenDisabled(t *testing.T) {
	scheduleID := uuid.New()
	settings := defaultSettings(scheduleID)
	settings.AutoPauseOnAlert = false

	store := &mockRansomwareStore{settings: settings}
	d := newTestDetector(store)

	alert := &models.RansomwareAlert{ID: uuid.New()}
	err := d.PauseBackupsIfRequired(context.Background(), scheduleID, alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.pausedID != nil {
		t.Error("schedule should not be paused when auto-pause is disabled")
	}
	if alert.BackupsPaused {
		t.Error("alert should not show backups paused")
	}
}

func TestPauseBackupsIfRequired_PausesWhenEnabled(t *testing.T) {
	scheduleID := uuid.New()
	settings := defaultSettings(scheduleID)
	settings.AutoPauseOnAlert = true

	store := &mockRansomwareStore{settings: settings}
	d := newTestDetector(store)

	alert := &models.RansomwareAlert{ID: uuid.New()}
	err := d.PauseBackupsIfRequired(context.Background(), scheduleID, alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.pausedID == nil || *store.pausedID != scheduleID {
		t.Error("expected schedule to be paused")
	}
	if !alert.BackupsPaused {
		t.Error("alert should show backups paused")
	}
	if alert.PausedAt == nil {
		t.Error("paused_at should be set")
	}
}

func TestPauseBackupsIfRequired_NoSettingsDoesNotPause(t *testing.T) {
	store := &mockRansomwareStore{
		settingsErr: fmt.Errorf("not found"),
	}
	d := newTestDetector(store)

	alert := &models.RansomwareAlert{ID: uuid.New()}
	err := d.PauseBackupsIfRequired(context.Background(), uuid.New(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.pausedID != nil {
		t.Error("should not pause when settings not found")
	}
}

// --- DefaultRansomwareExtensions Tests ---

func TestDefaultRansomwareExtensions_CoverCommonThreats(t *testing.T) {
	// Verify the list includes the most common ransomware extensions
	mustInclude := []string{
		".encrypted", ".locked", ".crypto", ".crypt", ".enc",
		".locky", ".cerber", ".petya", ".ransom", ".wannacry",
		".wncry", ".wallet",
	}

	extSet := make(map[string]struct{}, len(DefaultRansomwareExtensions))
	for _, ext := range DefaultRansomwareExtensions {
		extSet[ext] = struct{}{}
	}

	for _, ext := range mustInclude {
		if _, ok := extSet[ext]; !ok {
			t.Errorf("DefaultRansomwareExtensions missing %s", ext)
		}
	}
}
