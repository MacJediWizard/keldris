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

func TestSlackSender_Send(t *testing.T) {
	var received slackMessage

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

	sender := NewSlackSender(zerolog.Nop())
	sender.client = &http.Client{}
	msg := NotificationMessage{
		Title:     "Backup Failed: server1 - daily",
		Body:      "*Host:* server1\n*Error:* disk full",
		EventType: "backup_failed",
		Severity:  "error",
	}

	err := sender.Send(context.Background(), server.URL, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(received.Attachments))
	}
	att := received.Attachments[0]
	if att.Color != "#dc2626" {
		t.Errorf("expected red color for error severity, got %s", att.Color)
	}
	if len(att.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(att.Blocks))
	}
	if att.Blocks[0].Type != "header" {
		t.Errorf("expected header block, got %s", att.Blocks[0].Type)
	}
	if att.Blocks[0].Text.Text != msg.Title {
		t.Errorf("expected title %q, got %q", msg.Title, att.Blocks[0].Text.Text)
	}
	if att.Blocks[1].Text.Text != msg.Body {
		t.Errorf("expected body %q, got %q", msg.Body, att.Blocks[1].Text.Text)
	}
}

func TestSlackSender_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewSlackSender(zerolog.Nop())
	sender.client = &http.Client{}
	msg := NotificationMessage{Title: "Test", Body: "test", EventType: "test", Severity: "info"}

	err := sender.Send(context.Background(), server.URL, msg)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestSlackSender_SSRFProtection(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://127.0.0.1:8080/webhook"},
		{"private 10.x", "http://10.0.0.1:8080/webhook"},
		{"private 172.16.x", "http://172.16.0.1:8080/webhook"},
		{"private 192.168.x", "http://192.168.1.1:8080/webhook"},
		{"link-local", "http://169.254.1.1:8080/webhook"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewSlackSender(zerolog.Nop())
			msg := NotificationMessage{Title: "Test", Body: "test", EventType: "test", Severity: "info"}
			err := sender.Send(context.Background(), tt.url, msg)
			if err == nil {
				t.Error("expected SSRF protection to block request to private IP")
			}
		})
	}
}

func TestSeverityColor(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "#dc2626"},
		{"error", "#dc2626"},
		{"warning", "#f59e0b"},
		{"info", "#22c55e"},
		{"", "#22c55e"},
	}
	for _, tt := range tests {
		got := severityColor(tt.severity)
		if got != tt.want {
			t.Errorf("severityColor(%q) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}
