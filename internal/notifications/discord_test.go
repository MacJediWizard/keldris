package notifications

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestDiscordSender_Send(t *testing.T) {
	var received discordWebhookPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	sender := NewDiscordSender(zerolog.Nop())
	msg := NotificationMessage{
		Title:     "Backup Failed: server1 - daily",
		Body:      "**Host:** server1\n**Error:** disk full",
		EventType: "backup_failed",
		Severity:  "error",
	}

	err := sender.Send(context.Background(), server.URL, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(received.Embeds))
	}
	embed := received.Embeds[0]
	if embed.Title != msg.Title {
		t.Errorf("expected title %q, got %q", msg.Title, embed.Title)
	}
	if embed.Description != msg.Body {
		t.Errorf("expected description %q, got %q", msg.Body, embed.Description)
	}
	if embed.Color != 0xdc2626 {
		t.Errorf("expected red color for error severity, got %d", embed.Color)
	}
}

func TestDiscordSender_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewDiscordSender(zerolog.Nop())
	msg := NotificationMessage{Title: "Test", Body: "test", EventType: "test", Severity: "info"}

	err := sender.Send(context.Background(), server.URL, msg)
	if err == nil {
		t.Fatal("expected error for non-200/204 response")
	}
}

func TestDiscordSeverityColor(t *testing.T) {
	tests := []struct {
		severity string
		want     int
	}{
		{"critical", 0xdc2626},
		{"error", 0xdc2626},
		{"warning", 0xf59e0b},
		{"info", 0x22c55e},
		{"", 0x22c55e},
	}
	for _, tt := range tests {
		got := discordSeverityColor(tt.severity)
		if got != tt.want {
			t.Errorf("discordSeverityColor(%q) = %d, want %d", tt.severity, got, tt.want)
		}
	}
}
