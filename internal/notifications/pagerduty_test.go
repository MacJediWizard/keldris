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

func TestPagerDutySender_Send(t *testing.T) {
	var received pagerDutyRequest

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

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := NewPagerDutySender(zerolog.Nop())
	sender.client = &http.Client{}
	sender.eventURL = server.URL

	event := PagerDutyEvent{
		Summary:  "Backup Failed: server1 - daily",
		Source:   "server1",
		Severity: "critical",
		Group:    "backup",
	}

	err := sender.Send(context.Background(), "test-routing-key", event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.RoutingKey != "test-routing-key" {
		t.Errorf("expected routing key %q, got %q", "test-routing-key", received.RoutingKey)
	}
	if received.EventAction != "trigger" {
		t.Errorf("expected event action trigger, got %s", received.EventAction)
	}
	if received.Payload.Summary != event.Summary {
		t.Errorf("expected summary %q, got %q", event.Summary, received.Payload.Summary)
	}
	if received.Payload.Source != "server1" {
		t.Errorf("expected source server1, got %s", received.Payload.Source)
	}
	if received.Payload.Severity != "critical" {
		t.Errorf("expected severity critical, got %s", received.Payload.Severity)
	}
	if received.Payload.Group != "backup" {
		t.Errorf("expected group backup, got %s", received.Payload.Group)
	}
}

func TestPagerDutySender_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewPagerDutySender(zerolog.Nop())
	sender.client = &http.Client{}
	sender.eventURL = server.URL

	event := PagerDutyEvent{Summary: "Test", Source: "test", Severity: "info"}

	err := sender.Send(context.Background(), "key", event)
	if err == nil {
		t.Fatal("expected error for non-202 response")
	}
}

func TestPagerDutySender_SSRFProtection(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://127.0.0.1:8080/v2/enqueue"},
		{"private 10.x", "http://10.0.0.1:8080/v2/enqueue"},
		{"private 172.16.x", "http://172.16.0.1:8080/v2/enqueue"},
		{"private 192.168.x", "http://192.168.1.1:8080/v2/enqueue"},
		{"link-local", "http://169.254.1.1:8080/v2/enqueue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewPagerDutySender(zerolog.Nop())
			sender.eventURL = tt.url
			event := PagerDutyEvent{Summary: "Test", Source: "test", Severity: "info"}
			err := sender.Send(context.Background(), "key", event)
			if err == nil {
				t.Error("expected SSRF protection to block request to private IP")
			}
		})
	}
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"critical", "critical"},
		{"error", "error"},
		{"warning", "warning"},
		{"info", "info"},
		{"", "info"},
		{"unknown", "info"},
	}
	for _, tt := range tests {
		got := mapSeverity(tt.input)
		if got != tt.want {
			t.Errorf("mapSeverity(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
