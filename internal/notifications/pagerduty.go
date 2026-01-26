package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
func NewPagerDutyService(config models.PagerDutyChannelConfig, logger zerolog.Logger) (*PagerDutyService, error) {
	if err := ValidatePagerDutyConfig(&config); err != nil {
		return nil, err
	}

	// Set default severity if not specified
	if config.Severity == "" {
		config.Severity = "warning"
	}

	return &PagerDutyService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
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

const (
	PagerDutySeverityCritical PagerDutySeverity = "critical"
	PagerDutySeverityError    PagerDutySeverity = "error"
	PagerDutySeverityWarning  PagerDutySeverity = "warning"
	PagerDutySeverityInfo     PagerDutySeverity = "info"
)

// PagerDutyEvent represents a PagerDuty Events API v2 event.
type PagerDutyEvent struct {
	RoutingKey  string              `json:"routing_key"`
	EventAction PagerDutyEventAction `json:"event_action"`
	DedupKey    string              `json:"dedup_key,omitempty"`
	Payload     PagerDutyPayload    `json:"payload"`
	Images      []PagerDutyImage    `json:"images,omitempty"`
	Links       []PagerDutyLink     `json:"links,omitempty"`
}

// PagerDutyPayload represents the payload of a PagerDuty event.
type PagerDutyPayload struct {
	Summary       string                 `json:"summary"`
	Source        string                 `json:"source"`
	Severity      PagerDutySeverity      `json:"severity"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	Component     string                 `json:"component,omitempty"`
	Group         string                 `json:"group,omitempty"`
	Class         string                 `json:"class,omitempty"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// PagerDutyImage represents an image attachment.
type PagerDutyImage struct {
	Src  string `json:"src"`
	Href string `json:"href,omitempty"`
	Alt  string `json:"alt,omitempty"`
}

// PagerDutyLink represents a link attachment.
type PagerDutyLink struct {
	Href string `json:"href"`
	Text string `json:"text,omitempty"`
}

// PagerDutyResponse represents the response from PagerDuty Events API.
type PagerDutyResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key,omitempty"`
}

// Send sends an event to PagerDuty.
func (s *PagerDutyService) Send(event *PagerDutyEvent) error {
	event.RoutingKey = s.config.RoutingKey

	// Apply config defaults
	if event.Payload.Component == "" && s.config.Component != "" {
		event.Payload.Component = s.config.Component
	}
	if event.Payload.Group == "" && s.config.Group != "" {
		event.Payload.Group = s.config.Group
	}
	if event.Payload.Class == "" && s.config.Class != "" {
		event.Payload.Class = s.config.Class
	}

	// Merge custom fields from config
	if len(s.config.CustomFields) > 0 {
		if event.Payload.CustomDetails == nil {
			event.Payload.CustomDetails = make(map[string]interface{})
		}
		for k, v := range s.config.CustomFields {
			if _, exists := event.Payload.CustomDetails[k]; !exists {
				event.Payload.CustomDetails[k] = v
			}
		}
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal pagerduty event: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, pagerDutyEventsAPIURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create pagerduty request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send pagerduty request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pagerduty API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var pdResp PagerDutyResponse
	if err := json.Unmarshal(body, &pdResp); err == nil {
		s.logger.Debug().
			Str("status", pdResp.Status).
			Str("dedup_key", pdResp.DedupKey).
			Msg("pagerduty event sent")
	}

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
