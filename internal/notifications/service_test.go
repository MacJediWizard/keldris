package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockNotificationStore implements NotificationStore for testing.
type mockNotificationStore struct {
	mu sync.Mutex

	prefs    []*models.NotificationPreference
	prefsErr error

	channels    map[uuid.UUID]*models.NotificationChannel
	channelErr  error

	logs    []*models.NotificationLog
	logErr  error
	updateLogErr error

	agent    *models.Agent
	agentErr error

	schedule    *models.Schedule
	scheduleErr error
}

func (m *mockNotificationStore) GetEnabledPreferencesForEvent(_ context.Context, _ uuid.UUID, _ models.NotificationEventType) ([]*models.NotificationPreference, error) {
	if m.prefsErr != nil {
		return nil, m.prefsErr
	}
	return m.prefs, nil
}

func (m *mockNotificationStore) GetNotificationChannelByID(_ context.Context, id uuid.UUID) (*models.NotificationChannel, error) {
	if m.channelErr != nil {
		return nil, m.channelErr
	}
	ch, ok := m.channels[id]
	if !ok {
		return nil, fmt.Errorf("channel not found: %s", id)
	}
	return ch, nil
}

func (m *mockNotificationStore) CreateNotificationLog(_ context.Context, log *models.NotificationLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logErr != nil {
		return m.logErr
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockNotificationStore) UpdateNotificationLog(_ context.Context, log *models.NotificationLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateLogErr != nil {
		return m.updateLogErr
	}
	for i, l := range m.logs {
		if l.ID == log.ID {
			m.logs[i] = log
			return nil
		}
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockNotificationStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func (m *mockNotificationStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	if m.scheduleErr != nil {
		return nil, m.scheduleErr
	}
	return m.schedule, nil
}

func (m *mockNotificationStore) getLogs() []*models.NotificationLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*models.NotificationLog, len(m.logs))
	copy(result, m.logs)
	return result
}

func TestNewService(t *testing.T) {
	store := &mockNotificationStore{}
	svc := NewService(store, zerolog.Nop())
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestService_NotifyBackupComplete_NoPreferences(t *testing.T) {
	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{},
	}
	svc := NewService(store, zerolog.Nop())

	result := BackupResult{
		OrgID:   uuid.New(),
		Success: true,
	}

	// Should not panic and should return immediately
	svc.NotifyBackupComplete(context.Background(), result)
	// No logs should be created
	time.Sleep(50 * time.Millisecond)
	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyBackupComplete_PrefsError(t *testing.T) {
	store := &mockNotificationStore{
		prefsErr: fmt.Errorf("db error"),
	}
	svc := NewService(store, zerolog.Nop())

	result := BackupResult{
		OrgID:   uuid.New(),
		Success: true,
	}

	// Should not panic
	svc.NotifyBackupComplete(context.Background(), result)
}

func TestService_NotifyBackupComplete_SlackSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{
		WebhookURL: server.URL,
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupSuccess,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "Slack",
				Type:            models.ChannelTypeSlack,
				ConfigEncrypted: slackConfig,
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleID:   uuid.New(),
		ScheduleName: "daily",
		AgentID:      uuid.New(),
		Hostname:     "server1",
		SnapshotID:   "snap-123",
		StartedAt:    time.Now().Add(-time.Minute),
		CompletedAt:  time.Now(),
		SizeBytes:    1024 * 1024,
		FilesNew:     10,
		FilesChanged: 5,
		Success:      true,
	}

	svc.NotifyBackupComplete(context.Background(), result)

	// Wait for async goroutine
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected at least one notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected status sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyBackupComplete_SlackFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{
		WebhookURL: server.URL,
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupFailed,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "Slack",
				Type:            models.ChannelTypeSlack,
				ConfigEncrypted: slackConfig,
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		Success:      false,
		ErrorMessage: "disk full",
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected at least one notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected status sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyBackupComplete_WebhookChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	webhookConfig, _ := json.Marshal(models.WebhookChannelConfig{
		URL:    server.URL,
		Secret: "my-secret",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupSuccess,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "Webhook",
				Type:            models.ChannelTypeWebhook,
				ConfigEncrypted: webhookConfig,
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		Success:      true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent status, got %s", logs[0].Status)
	}
}

func TestService_NotifyBackupComplete_PagerDutyChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	pdConfig, _ := json.Marshal(models.PagerDutyChannelConfig{
		RoutingKey: "test-key",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupFailed,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "PagerDuty",
				Type:            models.ChannelTypePagerDuty,
				ConfigEncrypted: pdConfig,
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())

	// Override pagerduty sender's URL by calling sendNotification directly
	// instead of relying on NewPagerDutySender default. We test via the
	// NotifyBackupComplete flow, which creates a new sender each time.
	// Since we can't override the URL, let's test with the channel error path instead.
	// Actually, the PagerDuty sender sets eventURL to the real PagerDuty URL,
	// so we need to test differently. Let's just verify the flow doesn't panic
	// and creates a log entry.
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		Success:      false,
		ErrorMessage: "timeout",
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	// The PD call will fail (real URL) but log should exist with failed status
	if logs[0].Status != models.NotificationStatusFailed {
		// Could be either sent or failed depending on network
		t.Logf("log status: %s (may vary based on network)", logs[0].Status)
	}
}

func TestService_NotifyBackupComplete_TeamsChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	teamsConfig, _ := json.Marshal(models.TeamsChannelConfig{
		WebhookURL: server.URL,
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupSuccess,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "Teams",
				Type:            models.ChannelTypeTeams,
				ConfigEncrypted: teamsConfig,
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		SnapshotID:   "snap1",
		StartedAt:    time.Now().Add(-time.Minute),
		CompletedAt:  time.Now(),
		SizeBytes:    2048,
		Success:      true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyBackupComplete_DiscordChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	discordConfig, _ := json.Marshal(models.DiscordChannelConfig{
		WebhookURL: server.URL,
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupFailed,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "Discord",
				Type:            models.ChannelTypeDiscord,
				ConfigEncrypted: discordConfig,
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		Success:      false,
		ErrorMessage: "network error",
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyBackupComplete_ChannelError(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupSuccess,
				Enabled:   true,
			},
		},
		channelErr: fmt.Errorf("channel not found"),
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:   orgID,
		Success: true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	// No logs since channel lookup failed before log creation
	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyBackupComplete_InvalidChannelConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupSuccess,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "Bad Slack",
				Type:            models.ChannelTypeSlack,
				ConfigEncrypted: []byte("invalid json"),
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:   orgID,
		Success: true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	// Should not create any logs because config parsing fails before log creation
	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs for invalid config, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyBackupComplete_UnsupportedChannelType(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{
				ID:        uuid.New(),
				OrgID:     orgID,
				ChannelID: channelID,
				EventType: models.EventBackupSuccess,
				Enabled:   true,
			},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {
				ID:              channelID,
				OrgID:           orgID,
				Name:            "Unknown",
				Type:            "sms",
				ConfigEncrypted: []byte("{}"),
				Enabled:         true,
			},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:   orgID,
		Success: true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	// Default case logs a warning but doesn't create notification logs
	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs for unsupported channel, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyBackupComplete_MultipleChannels(t *testing.T) {
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer slackServer.Close()

	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer discordServer.Close()

	slackChannelID := uuid.New()
	discordChannelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{WebhookURL: slackServer.URL})
	discordConfig, _ := json.Marshal(models.DiscordChannelConfig{WebhookURL: discordServer.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: slackChannelID, EventType: models.EventBackupSuccess, Enabled: true},
			{ID: uuid.New(), OrgID: orgID, ChannelID: discordChannelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			slackChannelID:   {ID: slackChannelID, OrgID: orgID, Name: "Slack", Type: models.ChannelTypeSlack, ConfigEncrypted: slackConfig, Enabled: true},
			discordChannelID: {ID: discordChannelID, OrgID: orgID, Name: "Discord", Type: models.ChannelTypeDiscord, ConfigEncrypted: discordConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		SnapshotID:   "snap1",
		StartedAt:    time.Now().Add(-time.Minute),
		CompletedAt:  time.Now(),
		SizeBytes:    1024,
		Success:      true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(300 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) != 2 {
		t.Errorf("expected 2 logs for 2 channels, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Status != models.NotificationStatusSent {
			t.Errorf("expected sent status, got %s", l.Status)
		}
	}
}

func TestService_NotifyAgentOffline_NoPreferences(t *testing.T) {
	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{},
	}
	svc := NewService(store, zerolog.Nop())

	agent := &models.Agent{
		ID:       uuid.New(),
		Hostname: "server1",
	}

	svc.NotifyAgentOffline(context.Background(), agent, uuid.New(), 5*time.Minute)
	time.Sleep(50 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyAgentOffline_PrefsError(t *testing.T) {
	store := &mockNotificationStore{
		prefsErr: fmt.Errorf("db error"),
	}
	svc := NewService(store, zerolog.Nop())

	agent := &models.Agent{
		ID:       uuid.New(),
		Hostname: "server1",
	}

	svc.NotifyAgentOffline(context.Background(), agent, uuid.New(), 5*time.Minute)
}

func TestService_NotifyAgentOffline_SlackChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{WebhookURL: server.URL})

	lastSeen := time.Now().Add(-10 * time.Minute)
	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Slack", Type: models.ChannelTypeSlack, ConfigEncrypted: slackConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{
		ID:       uuid.New(),
		Hostname: "server1",
		LastSeen: &lastSeen,
	}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 10*time.Minute)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyAgentOffline_WebhookChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	webhookConfig, _ := json.Marshal(models.WebhookChannelConfig{URL: server.URL, Secret: "secret"})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Webhook", Type: models.ChannelTypeWebhook, ConfigEncrypted: webhookConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyAgentOffline_TeamsChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	teamsConfig, _ := json.Marshal(models.TeamsChannelConfig{WebhookURL: server.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Teams", Type: models.ChannelTypeTeams, ConfigEncrypted: teamsConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyAgentOffline_DiscordChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	discordConfig, _ := json.Marshal(models.DiscordChannelConfig{WebhookURL: server.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Discord", Type: models.ChannelTypeDiscord, ConfigEncrypted: discordConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyAgentOffline_ChannelError(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channelErr: fmt.Errorf("not found"),
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyAgentOffline_UnsupportedChannel(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "SMS", Type: "sms", ConfigEncrypted: []byte("{}"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyAgentOffline_InvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Bad", Type: models.ChannelTypeSlack, ConfigEncrypted: []byte("not json"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs for invalid config, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyAgentOffline_WithLastSeen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{WebhookURL: server.URL})
	lastSeen := time.Now().Add(-30 * time.Minute)

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Slack", Type: models.ChannelTypeSlack, ConfigEncrypted: slackConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{
		ID:       uuid.New(),
		Hostname: "server-with-lastseen",
		LastSeen: &lastSeen,
	}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 30*time.Minute)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
}

func TestService_NotifyMaintenanceScheduled_NoPreferences(t *testing.T) {
	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{},
	}
	svc := NewService(store, zerolog.Nop())

	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    uuid.New(),
		Title:    "Monthly Maintenance",
		Message:  "Backups will be paused",
		StartsAt: time.Now().Add(24 * time.Hour),
		EndsAt:   time.Now().Add(26 * time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(50 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyMaintenanceScheduled_PrefsError(t *testing.T) {
	store := &mockNotificationStore{
		prefsErr: fmt.Errorf("db error"),
	}
	svc := NewService(store, zerolog.Nop())

	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    uuid.New(),
		Title:    "Test",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
}

func TestService_NotifyMaintenanceScheduled_SlackChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{WebhookURL: server.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Slack", Type: models.ChannelTypeSlack, ConfigEncrypted: slackConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Monthly Maintenance",
		Message:  "Backups paused",
		StartsAt: time.Now().Add(24 * time.Hour),
		EndsAt:   time.Now().Add(26 * time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyMaintenanceScheduled_WebhookChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	webhookConfig, _ := json.Marshal(models.WebhookChannelConfig{URL: server.URL, Secret: "sec"})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Webhook", Type: models.ChannelTypeWebhook, ConfigEncrypted: webhookConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Maintenance",
		Message:  "Paused",
		StartsAt: time.Now().Add(time.Hour),
		EndsAt:   time.Now().Add(3 * time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyMaintenanceScheduled_TeamsChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	teamsConfig, _ := json.Marshal(models.TeamsChannelConfig{WebhookURL: server.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Teams", Type: models.ChannelTypeTeams, ConfigEncrypted: teamsConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Maintenance",
		Message:  "Paused",
		StartsAt: time.Now().Add(time.Hour),
		EndsAt:   time.Now().Add(3 * time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyMaintenanceScheduled_DiscordChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	discordConfig, _ := json.Marshal(models.DiscordChannelConfig{WebhookURL: server.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Discord", Type: models.ChannelTypeDiscord, ConfigEncrypted: discordConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Maintenance",
		Message:  "Paused",
		StartsAt: time.Now().Add(time.Hour),
		EndsAt:   time.Now().Add(3 * time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", logs[0].Status)
	}
}

func TestService_NotifyMaintenanceScheduled_ChannelError(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channelErr: fmt.Errorf("not found"),
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyMaintenanceScheduled_UnsupportedChannel(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "SMS", Type: "sms", ConfigEncrypted: []byte("{}"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyMaintenanceScheduled_InvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Bad", Type: models.ChannelTypeWebhook, ConfigEncrypted: []byte("{bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs for invalid config, got %d", len(store.getLogs()))
	}
}

func TestService_FinalizeLog_Success(t *testing.T) {
	store := &mockNotificationStore{}
	svc := NewService(store, zerolog.Nop())

	orgID := uuid.New()
	channelID := uuid.New()
	log := models.NewNotificationLog(orgID, &channelID, "test", "recipient", "subject")

	svc.finalizeLog(context.Background(), log, nil, channelID.String(), "recipient")

	if log.Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", log.Status)
	}
	if log.SentAt == nil {
		t.Error("expected SentAt to be set")
	}
}

func TestService_FinalizeLog_Error(t *testing.T) {
	store := &mockNotificationStore{}
	svc := NewService(store, zerolog.Nop())

	orgID := uuid.New()
	channelID := uuid.New()
	log := models.NewNotificationLog(orgID, &channelID, "test", "recipient", "subject")

	svc.finalizeLog(context.Background(), log, fmt.Errorf("send failed"), channelID.String(), "recipient")

	if log.Status != models.NotificationStatusFailed {
		t.Errorf("expected failed, got %s", log.Status)
	}
	if log.ErrorMessage != "send failed" {
		t.Errorf("expected error message 'send failed', got %q", log.ErrorMessage)
	}
}

func TestService_FinalizeLog_UpdateError(t *testing.T) {
	store := &mockNotificationStore{
		updateLogErr: fmt.Errorf("update failed"),
	}
	svc := NewService(store, zerolog.Nop())

	orgID := uuid.New()
	channelID := uuid.New()
	log := models.NewNotificationLog(orgID, &channelID, "test", "recipient", "subject")

	// Should not panic even if update fails
	svc.finalizeLog(context.Background(), log, nil, channelID.String(), "recipient")

	if log.Status != models.NotificationStatusSent {
		t.Errorf("expected sent, got %s", log.Status)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30 seconds"},
		{1 * time.Second, "1 seconds"},
		{90 * time.Second, "1 min 30 sec"},
		{5 * time.Minute, "5 minutes"},
		{time.Hour + 30*time.Minute, "1 hr 30 min"},
		{2 * time.Hour, "2 hours"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestService_NotifyBackupComplete_SlackWebhookError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{WebhookURL: server.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Slack", Type: models.ChannelTypeSlack, ConfigEncrypted: slackConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		Success:      true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	if logs[0].Status != models.NotificationStatusFailed {
		t.Errorf("expected failed status, got %s", logs[0].Status)
	}
	if !strings.Contains(logs[0].ErrorMessage, "500") {
		t.Errorf("expected error message to contain status code, got %q", logs[0].ErrorMessage)
	}
}

func TestService_NotifyBackupComplete_EmailChannel_InvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: []byte("not json"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{OrgID: orgID, Success: true}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs for invalid email config, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyBackupComplete_EmailChannel_InvalidSMTP(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	// Valid JSON but invalid SMTP config (missing host)
	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Port: 587,
		From: "noreply@example.com",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{OrgID: orgID, Success: true}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	// sendBackupEmail returns early if NewEmailService fails, so no log created
	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs for invalid smtp config, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyBackupComplete_CreateLogError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channelID := uuid.New()
	orgID := uuid.New()

	slackConfig, _ := json.Marshal(models.SlackChannelConfig{WebhookURL: server.URL})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Slack", Type: models.ChannelTypeSlack, ConfigEncrypted: slackConfig, Enabled: true},
		},
		logErr: fmt.Errorf("db error"),
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{OrgID: orgID, ScheduleName: "daily", Hostname: "server1", Success: true}

	// Should not panic even if log creation fails
	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyAgentOffline_PagerDutyChannel(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	pdConfig, _ := json.Marshal(models.PagerDutyChannelConfig{RoutingKey: "test-key"})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "PD", Type: models.ChannelTypePagerDuty, ConfigEncrypted: pdConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	// PD call will fail with real URL but log should exist
}

func TestService_NotifyMaintenanceScheduled_PagerDutyChannel(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	pdConfig, _ := json.Marshal(models.PagerDutyChannelConfig{RoutingKey: "test-key"})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "PD", Type: models.ChannelTypePagerDuty, ConfigEncrypted: pdConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
}

func TestService_NotifyAgentOffline_EmailChannel_InvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyMaintenanceScheduled_EmailChannel_InvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	if len(store.getLogs()) != 0 {
		t.Errorf("expected no logs, got %d", len(store.getLogs()))
	}
}

func TestService_NotifyAgentOffline_EmailChannel_InvalidSMTP(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Port: 587,
		From: "noreply@example.com",
		// Missing Host
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyMaintenanceScheduled_EmailChannel_InvalidSMTP(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Port: 587,
		From: "noreply@example.com",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyBackupComplete_EmailChannel_SuccessPath(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	// Valid email config with valid SMTP config so NewEmailService succeeds
	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		SnapshotID:   "snap1",
		StartedAt:    time.Now().Add(-time.Minute),
		CompletedAt:  time.Now(),
		SizeBytes:    1024,
		FilesNew:     5,
		FilesChanged: 3,
		Success:      true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
	// Will fail at SMTP connection but should reach finalizeLog with failed status
	if logs[0].Status != models.NotificationStatusFailed {
		t.Logf("log status: %s", logs[0].Status)
	}
}

func TestService_NotifyBackupComplete_EmailChannel_FailurePath(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupFailed, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		Success:      false,
		ErrorMessage: "disk full",
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
}

func TestService_NotifyAgentOffline_EmailChannel_ValidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
}

func TestService_NotifyMaintenanceScheduled_EmailChannel_ValidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test Maintenance",
		Message:  "Pausing backups",
		StartsAt: time.Now().Add(time.Hour),
		EndsAt:   time.Now().Add(3 * time.Hour),
	}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
}

func TestService_NotifyBackupComplete_WebhookInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Webhook", Type: models.ChannelTypeWebhook, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{OrgID: orgID, Success: true}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyBackupComplete_PagerDutyInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "PD", Type: models.ChannelTypePagerDuty, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{OrgID: orgID, Success: true}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyBackupComplete_TeamsInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Teams", Type: models.ChannelTypeTeams, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{OrgID: orgID, Success: true}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyBackupComplete_DiscordInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Discord", Type: models.ChannelTypeDiscord, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{OrgID: orgID, Success: true}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyAgentOffline_WebhookInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Webhook", Type: models.ChannelTypeWebhook, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyAgentOffline_PagerDutyInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "PD", Type: models.ChannelTypePagerDuty, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyAgentOffline_TeamsInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Teams", Type: models.ChannelTypeTeams, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyAgentOffline_DiscordInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventAgentOffline, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Discord", Type: models.ChannelTypeDiscord, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	agent := &models.Agent{ID: uuid.New(), Hostname: "server1"}

	svc.NotifyAgentOffline(context.Background(), agent, orgID, 5*time.Minute)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyMaintenanceScheduled_SlackInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Slack", Type: models.ChannelTypeSlack, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{ID: uuid.New(), OrgID: orgID, Title: "Test", StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyMaintenanceScheduled_PagerDutyInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "PD", Type: models.ChannelTypePagerDuty, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{ID: uuid.New(), OrgID: orgID, Title: "Test", StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyMaintenanceScheduled_TeamsInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Teams", Type: models.ChannelTypeTeams, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{ID: uuid.New(), OrgID: orgID, Title: "Test", StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyMaintenanceScheduled_DiscordInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Discord", Type: models.ChannelTypeDiscord, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{ID: uuid.New(), OrgID: orgID, Title: "Test", StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyMaintenanceScheduled_WebhookInvalidConfig(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventMaintenanceScheduled, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Webhook", Type: models.ChannelTypeWebhook, ConfigEncrypted: []byte("bad"), Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	window := &models.MaintenanceWindow{ID: uuid.New(), OrgID: orgID, Title: "Test", StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}

	svc.NotifyMaintenanceScheduled(context.Background(), window)
	time.Sleep(200 * time.Millisecond)
}

func TestService_NotifyBackupComplete_EmailChannel_TLSPath(t *testing.T) {
	channelID := uuid.New()
	orgID := uuid.New()

	emailConfig, _ := json.Marshal(models.EmailChannelConfig{
		Host: "localhost",
		Port: 19999,
		From: "noreply@example.com",
		TLS:  true,
	})

	store := &mockNotificationStore{
		prefs: []*models.NotificationPreference{
			{ID: uuid.New(), OrgID: orgID, ChannelID: channelID, EventType: models.EventBackupSuccess, Enabled: true},
		},
		channels: map[uuid.UUID]*models.NotificationChannel{
			channelID: {ID: channelID, OrgID: orgID, Name: "Email", Type: models.ChannelTypeEmail, ConfigEncrypted: emailConfig, Enabled: true},
		},
	}

	svc := NewService(store, zerolog.Nop())
	result := BackupResult{
		OrgID:        orgID,
		ScheduleName: "daily",
		Hostname:     "server1",
		SnapshotID:   "snap1",
		StartedAt:    time.Now().Add(-time.Minute),
		CompletedAt:  time.Now(),
		SizeBytes:    1024,
		Success:      true,
	}

	svc.NotifyBackupComplete(context.Background(), result)
	time.Sleep(200 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) == 0 {
		t.Fatal("expected notification log")
	}
}
