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

func TestTeamsSender_Send(t *testing.T) {
	var received teamsAdaptiveCard

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

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewTeamsSender(zerolog.Nop())
	msg := NotificationMessage{
		Title:     "Backup Failed: server1 - daily",
		Body:      "**Host:** server1\n\n**Error:** disk full",
		EventType: "backup_failed",
		Severity:  "error",
	}

	err := sender.Send(context.Background(), server.URL, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Type != "message" {
		t.Errorf("expected type 'message', got %q", received.Type)
	}
	if len(received.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(received.Attachments))
	}
	att := received.Attachments[0]
	if att.ContentType != "application/vnd.microsoft.card.adaptive" {
		t.Errorf("expected adaptive card content type, got %q", att.ContentType)
	}
	if att.Content.Type != "AdaptiveCard" {
		t.Errorf("expected AdaptiveCard type, got %q", att.Content.Type)
	}
	if len(att.Content.Body) != 2 {
		t.Fatalf("expected 2 body elements, got %d", len(att.Content.Body))
	}
	if att.Content.Body[0].Text != msg.Title {
		t.Errorf("expected title %q, got %q", msg.Title, att.Content.Body[0].Text)
	}
	if att.Content.Body[0].Color != "attention" {
		t.Errorf("expected color 'attention' for error severity, got %q", att.Content.Body[0].Color)
	}
	if att.Content.Body[1].Text != msg.Body {
		t.Errorf("expected body %q, got %q", msg.Body, att.Content.Body[1].Text)
	}
}

func TestTeamsSender_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewTeamsSender(zerolog.Nop())
	msg := NotificationMessage{Title: "Test", Body: "test", EventType: "test", Severity: "info"}

	err := sender.Send(context.Background(), server.URL, msg)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestTeamsSeverityColor(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "attention"},
		{"error", "attention"},
		{"warning", "warning"},
		{"info", "good"},
		{"", "good"},
	}
	for _, tt := range tests {
		got := teamsSeverityColor(tt.severity)
		if got != tt.want {
			t.Errorf("teamsSeverityColor(%q) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}
