package notifications

import (
	"bytes"
	"crypto/tls"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/rs/zerolog"
)

//go:embed templates/*.html
var templateFS embed.FS

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"-"`
	From     string `yaml:"from" json:"from"`
	TLS      bool   `yaml:"tls" json:"tls"`
}

// Validate checks if the SMTP configuration is valid
func (c *SMTPConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("smtp host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("smtp port is required")
	}
	if c.From == "" {
		return fmt.Errorf("smtp from address is required")
	}
	return nil
}

// EmailService handles sending email notifications
type EmailService struct {
	config    SMTPConfig
	templates *template.Template
	logger    zerolog.Logger
}

// NewEmailService creates a new email service
func NewEmailService(config SMTPConfig, logger zerolog.Logger) (*EmailService, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid smtp config: %w", err)
	}

	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse email templates: %w", err)
	}

	return &EmailService{
		config:    config,
		templates: tmpl,
		logger:    logger.With().Str("component", "email_service").Logger(),
	}, nil
}

// BackupSuccessData holds data for backup success email template
type BackupSuccessData struct {
	Hostname     string
	ScheduleName string
	SnapshotID   string
	StartedAt    time.Time
	CompletedAt  time.Time
	Duration     string
	SizeBytes    int64
	FilesNew     int
	FilesChanged int
}

// BackupFailedData holds data for backup failed email template
type BackupFailedData struct {
	Hostname     string
	ScheduleName string
	StartedAt    time.Time
	FailedAt     time.Time
	ErrorMessage string
}

// AgentOfflineData holds data for agent offline email template
type AgentOfflineData struct {
	Hostname     string
	LastSeen     time.Time
	OfflineSince string
	AgentID      string
}

// MaintenanceScheduledData holds data for maintenance scheduled email template
type MaintenanceScheduledData struct {
	Title    string
	Message  string
	StartsAt time.Time
	EndsAt   time.Time
	Duration string
}

// TestRestoreFailedData holds data for test restore failed email template
type TestRestoreFailedData struct {
	RepositoryName   string
	RepositoryID     string
	SnapshotID       string
	SamplePercentage int
	StartedAt        time.Time
	FailedAt         time.Time
	FilesRestored    int
	FilesVerified    int
	ErrorMessage     string
	ConsecutiveFails int
}

// ValidationFailedData holds data for validation failed email template
type ValidationFailedData struct {
	Hostname           string
	ScheduleName       string
	SnapshotID         string
	BackupCompletedAt  time.Time
	ValidationFailedAt time.Time
	ErrorMessage       string
	ValidationSummary  string
	ValidationDetails  string
}

// ReportEmailData holds data for report summary email template
type ReportEmailData struct {
	OrgName              string
	Frequency            string
	FrequencyLower       string
	PeriodStart          time.Time
	PeriodEnd            time.Time
	Data                 *ReportData
	TotalDataFormatted   string
	RawSizeFormatted     string
	RestoreSizeFormatted string
	SpaceSavedFormatted  string
}

// ReportData contains the aggregated data for a report
type ReportData struct {
	BackupSummary  ReportBackupSummary  `json:"backup_summary"`
	StorageSummary ReportStorageSummary `json:"storage_summary"`
	AgentSummary   ReportAgentSummary   `json:"agent_summary"`
	AlertSummary   ReportAlertSummary   `json:"alert_summary"`
	TopIssues      []ReportIssue        `json:"top_issues,omitempty"`
}

// ReportBackupSummary contains backup statistics for the report period
type ReportBackupSummary struct {
	TotalBackups      int     `json:"total_backups"`
	SuccessfulBackups int     `json:"successful_backups"`
	FailedBackups     int     `json:"failed_backups"`
	SuccessRate       float64 `json:"success_rate"`
	TotalDataBacked   int64   `json:"total_data_backed"`
	SchedulesActive   int     `json:"schedules_active"`
}

// ReportStorageSummary contains storage statistics
type ReportStorageSummary struct {
	TotalRawSize     int64   `json:"total_raw_size"`
	TotalRestoreSize int64   `json:"total_restore_size"`
	SpaceSaved       int64   `json:"space_saved"`
	SpaceSavedPct    float64 `json:"space_saved_pct"`
	RepositoryCount  int     `json:"repository_count"`
	TotalSnapshots   int     `json:"total_snapshots"`
}

// ReportAgentSummary contains agent health statistics
type ReportAgentSummary struct {
	TotalAgents   int `json:"total_agents"`
	ActiveAgents  int `json:"active_agents"`
	OfflineAgents int `json:"offline_agents"`
	PendingAgents int `json:"pending_agents"`
}

// ReportAlertSummary contains alert statistics
type ReportAlertSummary struct {
	TotalAlerts        int `json:"total_alerts"`
	CriticalAlerts     int `json:"critical_alerts"`
	WarningAlerts      int `json:"warning_alerts"`
	AcknowledgedAlerts int `json:"acknowledged_alerts"`
	ResolvedAlerts     int `json:"resolved_alerts"`
}

// ReportIssue represents a notable issue to highlight in the report
type ReportIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// SendBackupSuccess sends a backup success notification email
func (s *EmailService) SendBackupSuccess(to []string, data BackupSuccessData) error {
	subject := fmt.Sprintf("Backup Successful: %s - %s", data.Hostname, data.ScheduleName)
	return s.sendTemplate(to, subject, "backup_success.html", data)
}

// SendBackupFailed sends a backup failed notification email
func (s *EmailService) SendBackupFailed(to []string, data BackupFailedData) error {
	subject := fmt.Sprintf("Backup Failed: %s - %s", data.Hostname, data.ScheduleName)
	return s.sendTemplate(to, subject, "backup_failed.html", data)
}

// SendAgentOffline sends an agent offline notification email
func (s *EmailService) SendAgentOffline(to []string, data AgentOfflineData) error {
	subject := fmt.Sprintf("Agent Offline: %s", data.Hostname)
	return s.sendTemplate(to, subject, "agent_offline.html", data)
}

// SendMaintenanceScheduled sends a maintenance scheduled notification email
func (s *EmailService) SendMaintenanceScheduled(to []string, data MaintenanceScheduledData) error {
	subject := fmt.Sprintf("Scheduled Maintenance: %s", data.Title)
	return s.sendTemplate(to, subject, "maintenance_scheduled.html", data)
}

// SendTestRestoreFailed sends a test restore failed notification email
func (s *EmailService) SendTestRestoreFailed(to []string, data TestRestoreFailedData) error {
	subject := fmt.Sprintf("Test Restore Failed: %s", data.RepositoryName)
	return s.sendTemplate(to, subject, "test_restore_failed.html", data)
}

// SendValidationFailed sends a validation failed notification email
func (s *EmailService) SendValidationFailed(to []string, data ValidationFailedData) error {
	subject := fmt.Sprintf("Backup Validation Failed: %s - %s", data.Hostname, data.ScheduleName)
	return s.sendTemplate(to, subject, "validation_failed.html", data)
}

// SendReport sends a report summary email
func (s *EmailService) SendReport(to []string, data ReportEmailData) error {
	subject := fmt.Sprintf("%s Backup Report: %s", data.Frequency, data.OrgName)
	return s.sendTemplate(to, subject, "report_summary.html", data)
}

// FormatBytes formats bytes into a human-readable string
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// sendTemplate renders a template and sends the email
func (s *EmailService) sendTemplate(to []string, subject, templateName string, data interface{}) error {
	var body bytes.Buffer
	if err := s.templates.ExecuteTemplate(&body, templateName, data); err != nil {
		return fmt.Errorf("execute template %s: %w", templateName, err)
	}

	return s.send(to, subject, body.String())
}

// send sends an email with the given subject and HTML body
func (s *EmailService) send(to []string, subject, htmlBody string) error {
	s.logger.Debug().
		Strs("to", to).
		Str("subject", subject).
		Msg("sending email")

	// Build the message
	msg := s.buildMessage(to, subject, htmlBody)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var err error
	if s.config.TLS {
		err = s.sendTLS(addr, to, msg)
	} else {
		err = s.sendPlain(addr, to, msg)
	}

	if err != nil {
		s.logger.Error().
			Err(err).
			Strs("to", to).
			Str("subject", subject).
			Msg("failed to send email")
		return fmt.Errorf("send email: %w", err)
	}

	s.logger.Info().
		Strs("to", to).
		Str("subject", subject).
		Msg("email sent successfully")

	return nil
}

// buildMessage constructs the email message with headers
func (s *EmailService) buildMessage(to []string, subject, htmlBody string) []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", s.config.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to[0]))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlBody)
	return buf.Bytes()
}

// sendPlain sends email without TLS (for port 25 or trusted networks)
func (s *EmailService) sendPlain(addr string, to []string, msg []byte) error {
	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	return smtp.SendMail(addr, auth, s.config.From, to, msg)
}

// sendTLS sends email with TLS (for port 465 or STARTTLS on port 587)
func (s *EmailService) sendTLS(addr string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials provided
	if s.config.Username != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	// Set sender
	if err = client.Mail(s.config.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp rcpt to %s: %w", recipient, err)
		}
	}

	// Send the email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("close message writer: %w", err)
	}

	return client.Quit()
}
