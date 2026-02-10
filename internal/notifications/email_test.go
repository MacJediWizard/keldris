package notifications

import (
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNewEmailService_ValidConfig(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewEmailService_InvalidConfig_MissingHost(t *testing.T) {
	config := SMTPConfig{
		Port: 587,
		From: "noreply@example.com",
	}

	_, err := NewEmailService(config, zerolog.Nop())
	if err == nil {
		t.Fatal("expected error for missing host")
	}
	if !strings.Contains(err.Error(), "smtp host is required") {
		t.Errorf("expected host required error, got: %v", err)
	}
}

func TestNewEmailService_InvalidConfig_MissingPort(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		From: "noreply@example.com",
	}

	_, err := NewEmailService(config, zerolog.Nop())
	if err == nil {
		t.Fatal("expected error for missing port")
	}
	if !strings.Contains(err.Error(), "smtp port is required") {
		t.Errorf("expected port required error, got: %v", err)
	}
}

func TestNewEmailService_InvalidConfig_MissingFrom(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	}

	_, err := NewEmailService(config, zerolog.Nop())
	if err == nil {
		t.Fatal("expected error for missing from")
	}
	if !strings.Contains(err.Error(), "smtp from address is required") {
		t.Errorf("expected from address required error, got: %v", err)
	}
}

func TestSMTPConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SMTPConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  SMTPConfig{Host: "smtp.example.com", Port: 587, From: "test@example.com"},
			wantErr: false,
		},
		{
			name:    "missing host",
			config:  SMTPConfig{Port: 587, From: "test@example.com"},
			wantErr: true,
			errMsg:  "smtp host is required",
		},
		{
			name:    "missing port",
			config:  SMTPConfig{Host: "smtp.example.com", From: "test@example.com"},
			wantErr: true,
			errMsg:  "smtp port is required",
		},
		{
			name:    "missing from",
			config:  SMTPConfig{Host: "smtp.example.com", Port: 587},
			wantErr: true,
			errMsg:  "smtp from address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEmailService_BuildMessage(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	to := []string{"user@example.com"}
	msg := svc.buildMessage(to, "Test Subject", "<h1>Hello</h1>")

	msgStr := string(msg)
	if !strings.Contains(msgStr, "From: noreply@example.com") {
		t.Error("message missing From header")
	}
	if !strings.Contains(msgStr, "To: user@example.com") {
		t.Error("message missing To header")
	}
	if !strings.Contains(msgStr, "Subject: Test Subject") {
		t.Error("message missing Subject header")
	}
	if !strings.Contains(msgStr, "MIME-Version: 1.0") {
		t.Error("message missing MIME-Version header")
	}
	if !strings.Contains(msgStr, "Content-Type: text/html; charset=\"UTF-8\"") {
		t.Error("message missing Content-Type header")
	}
	if !strings.Contains(msgStr, "<h1>Hello</h1>") {
		t.Error("message missing HTML body")
	}
}

func TestEmailService_SendTemplate_BackupSuccess(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := BackupSuccessData{
		Hostname:     "server1",
		ScheduleName: "daily-backup",
		SnapshotID:   "abc123",
		StartedAt:    time.Now().Add(-time.Hour),
		CompletedAt:  time.Now(),
		Duration:     "1 hr 0 min",
		SizeBytes:    1024 * 1024 * 500,
		FilesNew:     100,
		FilesChanged: 50,
	}

	// SendBackupSuccess will fail because there's no SMTP server, but we can test
	// that the template renders properly by calling sendTemplate directly.
	err = svc.sendTemplate([]string{"user@example.com"}, "Test Subject", "backup_success.html", data)
	// This will fail at the SMTP step, but should NOT fail at the template step.
	// If error contains "template" it means template failed; if it contains "send" or
	// connection errors it means the template rendered fine.
	if err != nil && strings.Contains(err.Error(), "execute template") {
		t.Fatalf("template rendering failed: %v", err)
	}
}

func TestEmailService_SendTemplate_BackupFailed(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := BackupFailedData{
		Hostname:     "server1",
		ScheduleName: "daily-backup",
		StartedAt:    time.Now().Add(-time.Minute * 5),
		FailedAt:     time.Now(),
		ErrorMessage: "disk full",
	}

	err = svc.sendTemplate([]string{"user@example.com"}, "Test Subject", "backup_failed.html", data)
	if err != nil && strings.Contains(err.Error(), "execute template") {
		t.Fatalf("template rendering failed: %v", err)
	}
}

func TestEmailService_SendTemplate_AgentOffline(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := AgentOfflineData{
		Hostname:     "server1",
		LastSeen:     time.Now().Add(-time.Hour),
		OfflineSince: "1 hour",
		AgentID:      "abc-123",
	}

	err = svc.sendTemplate([]string{"user@example.com"}, "Agent Offline", "agent_offline.html", data)
	if err != nil && strings.Contains(err.Error(), "execute template") {
		t.Fatalf("template rendering failed: %v", err)
	}
}

func TestEmailService_SendTemplate_MaintenanceScheduled(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := MaintenanceScheduledData{
		Title:    "Monthly Maintenance",
		Message:  "Backups will be paused.",
		StartsAt: time.Now().Add(24 * time.Hour),
		EndsAt:   time.Now().Add(26 * time.Hour),
		Duration: "2 hours",
	}

	err = svc.sendTemplate([]string{"user@example.com"}, "Maintenance", "maintenance_scheduled.html", data)
	if err != nil && strings.Contains(err.Error(), "execute template") {
		t.Fatalf("template rendering failed: %v", err)
	}
}

func TestEmailService_SendTemplate_ReportSummary(t *testing.T) {
	config := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := ReportEmailData{
		OrgName:        "Acme Corp",
		Frequency:      "Weekly",
		FrequencyLower: "weekly",
		PeriodStart:    time.Now().Add(-7 * 24 * time.Hour),
		PeriodEnd:      time.Now(),
		Data: &ReportData{
			BackupSummary:  ReportBackupSummary{TotalBackups: 50, SuccessfulBackups: 48, FailedBackups: 2, SuccessRate: 96.0},
			StorageSummary: ReportStorageSummary{TotalRawSize: 1024 * 1024 * 1024, TotalRestoreSize: 500 * 1024 * 1024},
			AgentSummary:   ReportAgentSummary{TotalAgents: 10, ActiveAgents: 9, OfflineAgents: 1},
			AlertSummary:   ReportAlertSummary{TotalAlerts: 5, CriticalAlerts: 1},
		},
		TotalDataFormatted:   "1.00 GB",
		RawSizeFormatted:     "1.00 GB",
		RestoreSizeFormatted: "500.00 MB",
		SpaceSavedFormatted:  "500.00 MB",
	}

	err = svc.sendTemplate([]string{"user@example.com"}, "Report", "report_summary.html", data)
	if err != nil && strings.Contains(err.Error(), "execute template") {
		t.Fatalf("template rendering failed: %v", err)
	}
}

func TestEmailService_SendBackupSuccess_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999, // nothing listening here
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := BackupSuccessData{
		Hostname:     "server1",
		ScheduleName: "daily",
		SnapshotID:   "snap1",
		StartedAt:    time.Now().Add(-time.Minute),
		CompletedAt:  time.Now(),
		Duration:     "1 min",
		SizeBytes:    1024,
		FilesNew:     1,
		FilesChanged: 0,
	}

	err = svc.SendBackupSuccess([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestEmailService_SendBackupFailed_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := BackupFailedData{
		Hostname:     "server1",
		ScheduleName: "daily",
		StartedAt:    time.Now(),
		FailedAt:     time.Now(),
		ErrorMessage: "disk full",
	}

	err = svc.SendBackupFailed([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestEmailService_SendAgentOffline_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := AgentOfflineData{
		Hostname:     "server1",
		LastSeen:     time.Now(),
		OfflineSince: "5 min",
		AgentID:      "abc",
	}

	err = svc.SendAgentOffline([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestEmailService_SendMaintenanceScheduled_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := MaintenanceScheduledData{
		Title:    "Test",
		Message:  "Test message",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
		Duration: "1 hour",
	}

	err = svc.SendMaintenanceScheduled([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestEmailService_SendReport_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := ReportEmailData{
		OrgName:   "Test Org",
		Frequency: "Weekly",
		Data:      &ReportData{},
	}

	err = svc.SendReport([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestEmailService_SendTLS_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
		TLS:  true,
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := BackupSuccessData{
		Hostname:     "server1",
		ScheduleName: "daily",
		SnapshotID:   "snap1",
		StartedAt:    time.Now(),
		CompletedAt:  time.Now(),
		Duration:     "0 sec",
		SizeBytes:    0,
	}

	err = svc.SendBackupSuccess([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected TLS connection error")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
		{1024*1024*1024*1024 + 512*1024*1024*1024, "1.50 TB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEmailService_SendBackupSuccess_SubjectFormat(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := BackupSuccessData{
		Hostname:     "web-01",
		ScheduleName: "nightly",
	}

	// Call will fail at SMTP but we verify it doesn't fail at template level
	err = svc.SendBackupSuccess([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected error (no SMTP server)")
	}
	// Should be a send/connection error, not a template error
	if strings.Contains(err.Error(), "execute template") {
		t.Errorf("unexpected template error: %v", err)
	}
}

func TestEmailService_SendBackupFailed_SubjectFormat(t *testing.T) {
	config := SMTPConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	}

	svc, err := NewEmailService(config, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := BackupFailedData{
		Hostname:     "web-01",
		ScheduleName: "nightly",
		ErrorMessage: "timeout",
	}

	err = svc.SendBackupFailed([]string{"user@example.com"}, data)
	if err == nil {
		t.Fatal("expected error (no SMTP server)")
	}
	if strings.Contains(err.Error(), "execute template") {
		t.Errorf("unexpected template error: %v", err)
	}
}
