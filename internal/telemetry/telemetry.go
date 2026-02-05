// Package telemetry provides anonymous usage telemetry collection and reporting.
//
// PRIVACY NOTICE:
// This telemetry system is opt-in only and collects only aggregate, anonymous data:
// - Server version and build information
// - Count of agents (not names, hostnames, or IPs)
// - Count of backups (not file paths, sizes, or content)
// - Feature usage flags (which features are enabled, not how they're used)
//
// We explicitly DO NOT collect:
// - Hostnames, IP addresses, or any network identifiers
// - File paths, backup content, or user data
// - Organization names, user emails, or any PII
// - Repository locations or credentials
// - Any data that could identify specific users or systems
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DefaultEndpoint is the telemetry collection endpoint.
const DefaultEndpoint = "https://telemetry.keldris.io/v1/collect"

// CollectionInterval is how often telemetry is sent (weekly).
const CollectionInterval = 7 * 24 * time.Hour

// TelemetryData represents the anonymous data collected for telemetry.
// All fields are aggregate counts or version strings - no PII.
type TelemetryData struct {
	// InstallID is a random UUID generated on first setup.
	// It is NOT tied to any user, org, or system identifier.
	InstallID string `json:"install_id"`

	// Timestamp of when this data was collected.
	CollectedAt time.Time `json:"collected_at"`

	// Version information
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	GoVersion string `json:"go_version,omitempty"`

	// Aggregate counts only - no identifying information
	Counts TelemetryCounts `json:"counts"`

	// Feature flags - just booleans indicating if features are enabled
	Features TelemetryFeatures `json:"features"`
}

// TelemetryCounts contains aggregate count data.
type TelemetryCounts struct {
	TotalAgents        int `json:"total_agents"`
	ActiveAgents       int `json:"active_agents"`
	TotalBackups       int `json:"total_backups"`
	SuccessfulBackups  int `json:"successful_backups"`
	TotalRepositories  int `json:"total_repositories"`
	TotalSchedules     int `json:"total_schedules"`
	TotalOrganizations int `json:"total_organizations"`
	TotalUsers         int `json:"total_users"`
}

// TelemetryFeatures indicates which features are enabled.
type TelemetryFeatures struct {
	OIDCEnabled           bool `json:"oidc_enabled"`
	SMTPEnabled           bool `json:"smtp_enabled"`
	DockerBackupsEnabled  bool `json:"docker_backups_enabled"`
	GeoReplicationEnabled bool `json:"geo_replication_enabled"`
	SLAMonitoringEnabled  bool `json:"sla_monitoring_enabled"`
	RansomwareProtection  bool `json:"ransomware_protection"`
	LegalHoldsUsed        bool `json:"legal_holds_used"`
	ClassificationUsed    bool `json:"classification_used"`
	StorageTieringUsed    bool `json:"storage_tiering_used"`
	DRRunbooksUsed        bool `json:"dr_runbooks_used"`
}

// Settings holds telemetry configuration.
type Settings struct {
	// Enabled indicates if telemetry is enabled (opt-in, default false).
	Enabled bool `json:"enabled"`

	// InstallID is a random UUID for this installation.
	InstallID string `json:"install_id"`

	// LastSentAt is when telemetry was last successfully sent.
	LastSentAt *time.Time `json:"last_sent_at,omitempty"`

	// LastData contains the most recently collected telemetry data.
	// Users can view this to see exactly what's being sent.
	LastData *TelemetryData `json:"last_data,omitempty"`

	// Endpoint allows overriding the telemetry endpoint (for testing).
	Endpoint string `json:"endpoint,omitempty"`

	// ConsentGivenAt records when the user opted in.
	ConsentGivenAt *time.Time `json:"consent_given_at,omitempty"`

	// ConsentVersion tracks which privacy policy version was consented to.
	ConsentVersion string `json:"consent_version,omitempty"`
}

// DefaultSettings returns Settings with sensible defaults (telemetry disabled).
func DefaultSettings() Settings {
	return Settings{
		Enabled:   false,
		InstallID: uuid.New().String(),
		Endpoint:  DefaultEndpoint,
	}
}

// Validate validates the telemetry settings.
func (s *Settings) Validate() error {
	if s.InstallID == "" {
		return fmt.Errorf("install_id is required")
	}
	if _, err := uuid.Parse(s.InstallID); err != nil {
		return fmt.Errorf("install_id must be a valid UUID: %w", err)
	}
	return nil
}

// DataCollector is the interface for collecting telemetry data from the database.
type DataCollector interface {
	CollectTelemetryData(ctx context.Context) (*TelemetryCounts, *TelemetryFeatures, error)
}

// Service manages telemetry collection and reporting.
type Service struct {
	collector DataCollector
	logger    zerolog.Logger
	version   string
	commit    string
	buildDate string
	client    *http.Client

	mu       sync.RWMutex
	settings Settings
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewService creates a new telemetry service.
func NewService(
	collector DataCollector,
	version, commit, buildDate string,
	logger zerolog.Logger,
) *Service {
	return &Service{
		collector: collector,
		logger:    logger.With().Str("component", "telemetry").Logger(),
		version:   version,
		commit:    commit,
		buildDate: buildDate,
		settings:  DefaultSettings(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetSettings updates the telemetry settings.
func (s *Service) SetSettings(settings Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
}

// GetSettings returns the current telemetry settings.
func (s *Service) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// IsEnabled returns whether telemetry is enabled.
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.Enabled
}

// Start begins the periodic telemetry collection if enabled.
func (s *Service) Start() {
	s.mu.Lock()
	if s.stopCh != nil {
		s.mu.Unlock()
		return
	}
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.run()
}

// Stop halts the telemetry collection.
func (s *Service) Stop() {
	s.mu.Lock()
	if s.stopCh == nil {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	doneCh := s.doneCh
	s.mu.Unlock()

	<-doneCh
}

func (s *Service) run() {
	defer close(s.doneCh)

	// Send initial telemetry if enabled and not sent recently
	s.tryCollectAndSend()

	ticker := time.NewTicker(CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tryCollectAndSend()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) tryCollectAndSend() {
	if !s.IsEnabled() {
		return
	}

	settings := s.GetSettings()

	// Check if we've sent recently
	if settings.LastSentAt != nil {
		if time.Since(*settings.LastSentAt) < CollectionInterval {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	data, err := s.Collect(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to collect telemetry data")
		return
	}

	if err := s.Send(ctx, data); err != nil {
		s.logger.Warn().Err(err).Msg("failed to send telemetry data")
		return
	}

	s.logger.Info().Msg("telemetry data sent successfully")
}

// Collect gathers the current telemetry data.
func (s *Service) Collect(ctx context.Context) (*TelemetryData, error) {
	settings := s.GetSettings()

	counts, features, err := s.collector.CollectTelemetryData(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect telemetry counts: %w", err)
	}

	data := &TelemetryData{
		InstallID:   settings.InstallID,
		CollectedAt: time.Now().UTC(),
		Version:     s.version,
		Commit:      s.commit,
		BuildDate:   s.buildDate,
		Counts:      *counts,
		Features:    *features,
	}

	// Store the last collected data so users can view it
	s.mu.Lock()
	s.settings.LastData = data
	s.mu.Unlock()

	return data, nil
}

// Send transmits the telemetry data to the collection endpoint.
func (s *Service) Send(ctx context.Context, data *TelemetryData) error {
	settings := s.GetSettings()
	endpoint := settings.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal telemetry data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("Keldris/%s", s.version))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Update last sent time
	now := time.Now()
	s.mu.Lock()
	s.settings.LastSentAt = &now
	s.mu.Unlock()

	return nil
}

// Preview returns what telemetry data would be sent, without actually sending it.
// This allows users to see exactly what information would be collected.
func (s *Service) Preview(ctx context.Context) (*TelemetryData, error) {
	return s.Collect(ctx)
}

// GetPrivacyExplanation returns a human-readable explanation of what telemetry collects.
func GetPrivacyExplanation() string {
	return `KELDRIS ANONYMOUS TELEMETRY

WHY WE COLLECT TELEMETRY:
Telemetry helps us understand how Keldris is being used so we can:
- Prioritize features that matter most to users
- Identify and fix common issues
- Make informed decisions about platform development
- Ensure compatibility with popular configurations

WHAT WE COLLECT (all anonymous, aggregate data):
- Version: Server version, build info (helps us support your version)
- Agent count: Total and active agents (not names or hostnames)
- Backup count: Total and successful backups (not file paths or content)
- Feature flags: Which features are enabled (not how they're configured)
- Organization/user counts: Just numbers (not names or emails)

WHAT WE DO NOT COLLECT:
- Hostnames, IP addresses, or MAC addresses
- File paths, backup content, or file names
- User names, emails, or any personal information
- Organization names or identifiers
- Repository URLs, credentials, or locations
- Any data that could identify you or your systems

YOUR CONTROL:
- Telemetry is opt-in only (disabled by default)
- You can view exactly what would be sent before enabling
- You can disable telemetry at any time
- Disabling telemetry has no effect on functionality

DATA HANDLING:
- Data is transmitted over HTTPS
- Data is stored in aggregate form only
- We do not sell or share telemetry data
- Data is used solely for product improvement`
}

// UpdateSettingsRequest is the request for updating telemetry settings.
type UpdateSettingsRequest struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// TelemetryStatusResponse is the response for telemetry status.
type TelemetryStatusResponse struct {
	Settings    Settings `json:"settings"`
	Explanation string   `json:"explanation"`
}

// TelemetryPreviewResponse is the response for telemetry preview.
type TelemetryPreviewResponse struct {
	Data        *TelemetryData `json:"data"`
	Explanation string         `json:"explanation"`
}
