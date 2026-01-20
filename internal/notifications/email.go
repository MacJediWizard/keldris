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
