package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

const pagerDutyEventsURL = "https://events.pagerduty.com/v2/enqueue"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/httpclient"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

const (
	// PagerDuty Events API v2 endpoint
	pagerDutyEventsAPIURL = "https://events.pagerduty.com/v2/enqueue"
)

// PagerDutyService handles sending notifications via PagerDuty Events API v2.
type PagerDutyService struct {
	config models.PagerDutyChannelConfig
	client *http.Client
	logger zerolog.Logger
}

// NewPagerDutyService creates a new PagerDuty notification service.
func NewPagerDutyService(cfg models.PagerDutyChannelConfig, logger zerolog.Logger) (*PagerDutyService, error) {
	return NewPagerDutyServiceWithProxy(cfg, nil, logger)
}

// NewPagerDutyServiceWithProxy creates a PagerDuty notification service with proxy support.
func NewPagerDutyServiceWithProxy(cfg models.PagerDutyChannelConfig, proxyConfig *config.ProxyConfig, logger zerolog.Logger) (*PagerDutyService, error) {
	if err := ValidatePagerDutyConfig(&cfg); err != nil {
		return nil, err
	}

	// Set default severity if not specified
	if cfg.Severity == "" {
		cfg.Severity = "warning"
	}

	client, err := httpclient.New(httpclient.Options{
		Timeout:     30 * time.Second,
		ProxyConfig: proxyConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	return &PagerDutyService{
		config: cfg,
		client: client,
		logger: logger.With().Str("component", "pagerduty_service").Logger(),
	}, nil
}

// ValidatePagerDutyConfig validates the PagerDuty configuration.
func ValidatePagerDutyConfig(config *models.PagerDutyChannelConfig) error {
	if config.RoutingKey == "" {
		return fmt.Errorf("pagerduty routing key is required")
	}
	return nil
}

// PagerDutyEventAction represents the action to take on an event.
type PagerDutyEventAction string

const (
	PagerDutyEventTrigger     PagerDutyEventAction = "trigger"
	PagerDutyEventAcknowledge PagerDutyEventAction = "acknowledge"
	PagerDutyEventResolve     PagerDutyEventAction = "resolve"
)

// PagerDutySeverity represents the severity of an event.
type PagerDutySeverity string

// PagerDutyEvent represents the data needed to create a PagerDuty event.
type PagerDutyEvent struct {
	Summary  string
	Source   string
	Severity string // info, warning, error, critical
	Group    string
}

// pagerDutyRequest represents a PagerDuty Events API v2 payload.
type pagerDutyRequest struct {
	RoutingKey  string              `json:"routing_key"`
	EventAction string             `json:"event_action"`
	Payload     pagerDutyPayload   `json:"payload"`
}

type pagerDutyPayload struct {
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
	Group    string `json:"group,omitempty"`
}

// PagerDutySender sends notifications via PagerDuty Events API v2.
type PagerDutySender struct {
	client   *http.Client
	logger   zerolog.Logger
	eventURL string
}

// NewPagerDutySender creates a new PagerDuty sender.
func NewPagerDutySender(logger zerolog.Logger) *PagerDutySender {
	return &PagerDutySender{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: ValidatingDialer(),
			},
		},
		client:   &http.Client{},
		logger:   logger.With().Str("component", "pagerduty_sender").Logger(),
		eventURL: pagerDutyEventsURL,
	}
}

// mapSeverity maps notification severity to PagerDuty severity values.
// PagerDuty accepts: critical, error, warning, info.
func mapSeverity(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "error":
		return "error"
	case "warning":
		return "warning"
	default:
		return "info"
	}
}

// Send sends a PagerDuty event using the Events API v2.
func (p *PagerDutySender) Send(ctx context.Context, routingKey string, event PagerDutyEvent) error {
	payload := pagerDutyRequest{
		RoutingKey:  routingKey,
		EventAction: "trigger",
		Payload: pagerDutyPayload{
			Summary:  event.Summary,
			Source:   event.Source,
			Severity: mapSeverity(event.Severity),
			Group:    event.Group,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal pagerduty payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.eventURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create pagerduty request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("send pagerduty event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("pagerduty returned status %d", resp.StatusCode)
	}

	p.logger.Info().
		Str("summary", event.Summary).
		Str("severity", event.Severity).
		Msg("pagerduty event sent")

	return nil
}

// getSeverity returns the appropriate PagerDuty severity.
func (s *PagerDutyService) getSeverity(defaultSeverity PagerDutySeverity) PagerDutySeverity {
	if s.config.Severity != "" {
		return PagerDutySeverity(s.config.Severity)
	}
	return defaultSeverity
}

// SendBackupSuccess sends a backup success notification to PagerDuty.
// Note: Backup success typically resolves any previous backup failure alerts.
func (s *PagerDutyService) SendBackupSuccess(data BackupSuccessData) error {
	event := &PagerDutyEvent{
		EventAction: PagerDutyEventResolve,
		DedupKey:    fmt.Sprintf("backup-%s-%s", data.Hostname, data.ScheduleName),
		Payload: PagerDutyPayload{
			Summary:   fmt.Sprintf("Backup Successful: %s - %s", data.Hostname, data.ScheduleName),
			Source:    data.Hostname,
			Severity:  PagerDutySeverityInfo,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"hostname":      data.Hostname,
				"schedule":      data.ScheduleName,
				"snapshot_id":   data.SnapshotID,
				"duration":      data.Duration,
				"files_new":     data.FilesNew,
				"files_changed": data.FilesChanged,
				"size_bytes":    data.SizeBytes,
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Msg("sending backup success (resolve) to PagerDuty")

	return s.Send(event)
}

// SendBackupFailed sends a backup failed notification to PagerDuty.
func (s *PagerDutyService) SendBackupFailed(data BackupFailedData) error {
	event := &PagerDutyEvent{
		EventAction: PagerDutyEventTrigger,
		DedupKey:    fmt.Sprintf("backup-%s-%s", data.Hostname, data.ScheduleName),
		Payload: PagerDutyPayload{
			Summary:   fmt.Sprintf("Backup Failed: %s - %s: %s", data.Hostname, data.ScheduleName, data.ErrorMessage),
			Source:    data.Hostname,
			Severity:  s.getSeverity(PagerDutySeverityError),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"hostname":      data.Hostname,
				"schedule":      data.ScheduleName,
				"started_at":    data.StartedAt.Format(time.RFC3339),
				"failed_at":     data.FailedAt.Format(time.RFC3339),
				"error_message": data.ErrorMessage,
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending backup failed notification to PagerDuty")

	return s.Send(event)
}

// SendAgentOffline sends an agent offline notification to PagerDuty.
func (s *PagerDutyService) SendAgentOffline(data AgentOfflineData) error {
	event := &PagerDutyEvent{
		EventAction: PagerDutyEventTrigger,
		DedupKey:    fmt.Sprintf("agent-offline-%s", data.AgentID),
		Payload: PagerDutyPayload{
			Summary:   fmt.Sprintf("Agent Offline: %s (offline for %s)", data.Hostname, data.OfflineSince),
			Source:    data.Hostname,
			Severity:  s.getSeverity(PagerDutySeverityWarning),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"hostname":       data.Hostname,
				"agent_id":       data.AgentID,
				"last_seen":      data.LastSeen.Format(time.RFC3339),
				"offline_since":  data.OfflineSince,
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("agent_id", data.AgentID).
		Msg("sending agent offline notification to PagerDuty")

	return s.Send(event)
}

// SendMaintenanceScheduled sends a maintenance scheduled notification to PagerDuty.
func (s *PagerDutyService) SendMaintenanceScheduled(data MaintenanceScheduledData) error {
	event := &PagerDutyEvent{
		EventAction: PagerDutyEventTrigger,
		DedupKey:    fmt.Sprintf("maintenance-%s-%d", data.Title, data.StartsAt.Unix()),
		Payload: PagerDutyPayload{
			Summary:   fmt.Sprintf("Scheduled Maintenance: %s", data.Title),
			Source:    "keldris-backup",
			Severity:  PagerDutySeverityInfo,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"title":     data.Title,
				"message":   data.Message,
				"starts_at": data.StartsAt.Format(time.RFC3339),
				"ends_at":   data.EndsAt.Format(time.RFC3339),
				"duration":  data.Duration,
			},
		},
	}

	s.logger.Debug().
		Str("title", data.Title).
		Time("starts_at", data.StartsAt).
		Msg("sending maintenance scheduled notification to PagerDuty")

	return s.Send(event)
}

// SendTestRestoreFailed sends a test restore failed notification to PagerDuty.
func (s *PagerDutyService) SendTestRestoreFailed(data TestRestoreFailedData) error {
	severity := s.getSeverity(PagerDutySeverityError)
	if data.ConsecutiveFails > 2 {
		severity = PagerDutySeverityCritical
	}

	event := &PagerDutyEvent{
		EventAction: PagerDutyEventTrigger,
		DedupKey:    fmt.Sprintf("test-restore-%s", data.RepositoryID),
		Payload: PagerDutyPayload{
			Summary:   fmt.Sprintf("Test Restore Failed: %s - %s", data.RepositoryName, data.ErrorMessage),
			Source:    "keldris-backup",
			Severity:  severity,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"repository_name":    data.RepositoryName,
				"repository_id":      data.RepositoryID,
				"snapshot_id":        data.SnapshotID,
				"sample_percentage":  data.SamplePercentage,
				"files_restored":     data.FilesRestored,
				"files_verified":     data.FilesVerified,
				"started_at":         data.StartedAt.Format(time.RFC3339),
				"failed_at":          data.FailedAt.Format(time.RFC3339),
				"error_message":      data.ErrorMessage,
				"consecutive_fails":  data.ConsecutiveFails,
			},
		},
	}

	s.logger.Debug().
		Str("repository", data.RepositoryName).
		Str("error", data.ErrorMessage).
		Int("consecutive_fails", data.ConsecutiveFails).
		Msg("sending test restore failed notification to PagerDuty")

	return s.Send(event)
}

// TestConnection sends a test event to verify the PagerDuty integration is working.
func (s *PagerDutyService) TestConnection() error {
	event := &PagerDutyEvent{
		EventAction: PagerDutyEventTrigger,
		DedupKey:    fmt.Sprintf("keldris-test-%d", time.Now().Unix()),
		Payload: PagerDutyPayload{
			Summary:   "Keldris Backup - Test Notification",
			Source:    "keldris-backup",
			Severity:  PagerDutySeverityInfo,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"message": "Your PagerDuty integration is working correctly!",
				"test":    true,
			},
		},
	}
	return s.Send(event)
}

// ResolveAgent sends a resolve event for an agent that has come back online.
func (s *PagerDutyService) ResolveAgent(agentID, hostname string) error {
	event := &PagerDutyEvent{
		EventAction: PagerDutyEventResolve,
		DedupKey:    fmt.Sprintf("agent-offline-%s", agentID),
		Payload: PagerDutyPayload{
			Summary:   fmt.Sprintf("Agent Online: %s", hostname),
			Source:    hostname,
			Severity:  PagerDutySeverityInfo,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}

	s.logger.Debug().
		Str("hostname", hostname).
		Str("agent_id", agentID).
		Msg("resolving agent offline alert in PagerDuty")

	return s.Send(event)
}

// SendValidationFailed sends a backup validation failed notification to PagerDuty.
func (s *PagerDutyService) SendValidationFailed(data ValidationFailedData) error {
	event := &PagerDutyEvent{
		EventAction: PagerDutyEventTrigger,
		DedupKey:    fmt.Sprintf("validation-%s-%s-%s", data.Hostname, data.ScheduleName, data.SnapshotID),
		Payload: PagerDutyPayload{
			Summary:   fmt.Sprintf("Backup Validation Failed: %s - %s: %s", data.Hostname, data.ScheduleName, data.ErrorMessage),
			Source:    data.Hostname,
			Severity:  s.getSeverity(PagerDutySeverityError),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"hostname":             data.Hostname,
				"schedule":             data.ScheduleName,
				"snapshot_id":          data.SnapshotID,
				"backup_completed_at":  data.BackupCompletedAt.Format(time.RFC3339),
				"validation_failed_at": data.ValidationFailedAt.Format(time.RFC3339),
				"error_message":        data.ErrorMessage,
				"validation_summary":   data.ValidationSummary,
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending validation failed notification to PagerDuty")

	return s.Send(event)
}
