package komodo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// WebhookHandler processes incoming webhooks from Komodo
type WebhookHandler struct {
	logger zerolog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(logger zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{
		logger: logger.With().Str("component", "komodo_webhook").Logger(),
	}
}

// ProcessedWebhook contains the result of processing a webhook
type ProcessedWebhook struct {
	Event         *models.KomodoWebhookEvent
	EventType     models.KomodoWebhookEventType
	BackupTrigger *WebhookBackupTrigger
	ContainerID   string
	StackID       string
	ShouldBackup  bool
}

// ParsePayload parses a raw webhook payload
func (h *WebhookHandler) ParsePayload(payload []byte) (*WebhookPayload, error) {
	var webhookPayload WebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}
	return &webhookPayload, nil
}

// Process processes an incoming webhook and returns the event type and any backup trigger
func (h *WebhookHandler) Process(ctx context.Context, orgID, integrationID uuid.UUID, payload []byte) (*ProcessedWebhook, error) {
	webhookPayload, err := h.ParsePayload(payload)
	if err != nil {
		return nil, err
	}

	eventType := h.mapEventType(webhookPayload.Event)

	h.logger.Info().
		Str("event", webhookPayload.Event).
		Str("event_type", string(eventType)).
		Str("integration_id", integrationID.String()).
		Msg("processing Komodo webhook")

	result := &ProcessedWebhook{
		Event:     models.NewKomodoWebhookEvent(orgID, integrationID, eventType, payload),
		EventType: eventType,
		ContainerID: webhookPayload.ContainerID,
		StackID:     webhookPayload.StackID,
	}

	// Check if this is a backup trigger event
	if eventType == models.KomodoWebhookBackupTrigger {
		trigger, err := h.parseBackupTrigger(webhookPayload)
		if err != nil {
			h.logger.Warn().Err(err).Msg("failed to parse backup trigger data")
		} else {
			result.BackupTrigger = trigger
			result.ShouldBackup = true
		}
	}

	// Check if container events should trigger backup
	if h.shouldTriggerBackup(eventType, webhookPayload) {
		result.ShouldBackup = true
		if result.BackupTrigger == nil {
			result.BackupTrigger = &WebhookBackupTrigger{
				ContainerID:   webhookPayload.ContainerID,
				ContainerName: webhookPayload.ContainerName,
				StackID:       webhookPayload.StackID,
				StackName:     webhookPayload.StackName,
			}
		}
	}

	return result, nil
}

// mapEventType maps a Komodo event string to our event type
func (h *WebhookHandler) mapEventType(event string) models.KomodoWebhookEventType {
	switch event {
	case "container.start", "container.started":
		return models.KomodoWebhookContainerStart
	case "container.stop", "container.stopped":
		return models.KomodoWebhookContainerStop
	case "container.restart", "container.restarted":
		return models.KomodoWebhookContainerRestart
	case "stack.deploy", "stack.deployed":
		return models.KomodoWebhookStackDeploy
	case "stack.update", "stack.updated":
		return models.KomodoWebhookStackUpdate
	case "backup.trigger", "backup.request", "keldris.backup":
		return models.KomodoWebhookBackupTrigger
	default:
		return models.KomodoWebhookUnknown
	}
}

// shouldTriggerBackup determines if an event should trigger a backup
func (h *WebhookHandler) shouldTriggerBackup(eventType models.KomodoWebhookEventType, payload *WebhookPayload) bool {
	// Explicit backup trigger
	if eventType == models.KomodoWebhookBackupTrigger {
		return true
	}

	// Check for backup flag in data
	if payload.Data != nil {
		if triggerBackup, ok := payload.Data["trigger_backup"].(bool); ok && triggerBackup {
			return true
		}
		if backup, ok := payload.Data["backup"].(bool); ok && backup {
			return true
		}
	}

	// Container stop events might trigger backup (configurable)
	if eventType == models.KomodoWebhookContainerStop {
		if payload.Data != nil {
			if backupOnStop, ok := payload.Data["backup_on_stop"].(bool); ok && backupOnStop {
				return true
			}
		}
	}

	return false
}

// parseBackupTrigger extracts backup trigger details from the webhook payload
func (h *WebhookHandler) parseBackupTrigger(payload *WebhookPayload) (*WebhookBackupTrigger, error) {
	trigger := &WebhookBackupTrigger{
		ContainerID:   payload.ContainerID,
		ContainerName: payload.ContainerName,
		StackID:       payload.StackID,
		StackName:     payload.StackName,
		ServerID:      payload.ServerID,
	}

	if payload.Data != nil {
		// Extract paths if specified
		if paths, ok := payload.Data["paths"].([]interface{}); ok {
			for _, p := range paths {
				if path, ok := p.(string); ok {
					trigger.Paths = append(trigger.Paths, path)
				}
			}
		}

		// Extract tags if specified
		if tags, ok := payload.Data["tags"].([]interface{}); ok {
			for _, t := range tags {
				if tag, ok := t.(string); ok {
					trigger.Tags = append(trigger.Tags, tag)
				}
			}
		}

		// Extract priority if specified
		if priority, ok := payload.Data["priority"].(float64); ok {
			trigger.Priority = int(priority)
		}
	}

	return trigger, nil
}

// ValidateWebhookSecret validates the webhook secret if provided
func (h *WebhookHandler) ValidateWebhookSecret(providedSecret, expectedSecret string) bool {
	if expectedSecret == "" {
		// No secret configured, accept all
		return true
	}
	return providedSecret == expectedSecret
}

// GetEventDescription returns a human-readable description of the event
func (h *WebhookHandler) GetEventDescription(eventType models.KomodoWebhookEventType) string {
	switch eventType {
	case models.KomodoWebhookContainerStart:
		return "Container started"
	case models.KomodoWebhookContainerStop:
		return "Container stopped"
	case models.KomodoWebhookContainerRestart:
		return "Container restarted"
	case models.KomodoWebhookStackDeploy:
		return "Stack deployed"
	case models.KomodoWebhookStackUpdate:
		return "Stack updated"
	case models.KomodoWebhookBackupTrigger:
		return "Backup triggered"
	default:
		return "Unknown event"
	}
}
