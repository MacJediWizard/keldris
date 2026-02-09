package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

const pagerDutyEventsURL = "https://events.pagerduty.com/v2/enqueue"

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
