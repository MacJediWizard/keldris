package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

// DiscordSender sends notifications via Discord webhooks.
type DiscordSender struct {
	client *http.Client
	logger zerolog.Logger
}

// NewDiscordSender creates a new Discord sender.
func NewDiscordSender(logger zerolog.Logger) *DiscordSender {
	return &DiscordSender{
		client: &http.Client{},
		logger: logger.With().Str("component", "discord_sender").Logger(),
	}
}

// discordWebhookPayload represents a Discord webhook payload with embeds.
type discordWebhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

// discordSeverityColor maps notification severity to Discord embed colors (decimal).
func discordSeverityColor(severity string) int {
	switch severity {
	case "critical", "error":
		return 0xdc2626 // red
	case "warning":
		return 0xf59e0b // amber
	default:
		return 0x22c55e // green
	}
}

// Send sends a notification message to a Discord webhook URL.
func (d *DiscordSender) Send(ctx context.Context, webhookURL string, msg NotificationMessage) error {
	payload := discordWebhookPayload{
		Embeds: []discordEmbed{
			{
				Title:       msg.Title,
				Description: msg.Body,
				Color:       discordSeverityColor(msg.Severity),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send discord webhook: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	d.logger.Info().
		Str("event_type", msg.EventType).
		Msg("discord notification sent")

	return nil
}
