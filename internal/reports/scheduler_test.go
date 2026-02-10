package reports

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockSchedulerStore implements SchedulerStore for testing.
type mockSchedulerStore struct {
	mu sync.Mutex

	// ReportStore fields
	backups      []*models.Backup
	schedules    []*models.Schedule
	storageStats *models.StorageStatsSummary
	agents       []*models.Agent
	alerts       []*models.Alert

	backupsErr      error
	schedulesErr    error
	storageStatsErr error
	agentsErr       error
	alertsErr       error

	// SchedulerStore fields
	reportSchedules []*models.ReportSchedule
	reportHistory   []*models.ReportHistory
	channels        map[uuid.UUID]*models.NotificationChannel
	organizations   map[uuid.UUID]*models.Organization

	reportSchedulesErr     error
	reportScheduleByIDErr  error
	updateLastSentErr      error
	createHistoryErr       error
	channelErr             error
	orgErr                 error

	lastUpdatedScheduleID uuid.UUID
	lastUpdatedSentAt     time.Time
}

func newMockSchedulerStore() *mockSchedulerStore {
	return &mockSchedulerStore{
		storageStats:  &models.StorageStatsSummary{},
		channels:      make(map[uuid.UUID]*models.NotificationChannel),
		organizations: make(map[uuid.UUID]*models.Organization),
	}
}

func (m *mockSchedulerStore) GetBackupsByOrgIDAndDateRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.Backup, error) {
	return m.backups, m.backupsErr
}

func (m *mockSchedulerStore) GetEnabledSchedulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, m.schedulesErr
}

func (m *mockSchedulerStore) GetStorageStatsSummary(_ context.Context, _ uuid.UUID) (*models.StorageStatsSummary, error) {
	return m.storageStats, m.storageStatsErr
}

func (m *mockSchedulerStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return m.agents, m.agentsErr
}

func (m *mockSchedulerStore) GetAlertsByOrgIDAndDateRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.Alert, error) {
	return m.alerts, m.alertsErr
}

func (m *mockSchedulerStore) GetEnabledReportSchedules(_ context.Context) ([]*models.ReportSchedule, error) {
	return m.reportSchedules, m.reportSchedulesErr
}

func (m *mockSchedulerStore) GetReportScheduleByID(_ context.Context, id uuid.UUID) (*models.ReportSchedule, error) {
	if m.reportScheduleByIDErr != nil {
		return nil, m.reportScheduleByIDErr
	}
	for _, s := range m.reportSchedules {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockSchedulerStore) UpdateReportScheduleLastSent(_ context.Context, id uuid.UUID, lastSentAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastUpdatedScheduleID = id
	m.lastUpdatedSentAt = lastSentAt
	return m.updateLastSentErr
}

func (m *mockSchedulerStore) CreateReportHistory(_ context.Context, history *models.ReportHistory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createHistoryErr != nil {
		return m.createHistoryErr
	}
	m.reportHistory = append(m.reportHistory, history)
	return nil
}

func (m *mockSchedulerStore) GetNotificationChannelByID(_ context.Context, id uuid.UUID) (*models.NotificationChannel, error) {
	if m.channelErr != nil {
		return nil, m.channelErr
	}
	ch, ok := m.channels[id]
	if !ok {
		return nil, errors.New("channel not found")
	}
	return ch, nil
}

func (m *mockSchedulerStore) GetOrganizationByID(_ context.Context, id uuid.UUID) (*models.Organization, error) {
	if m.orgErr != nil {
		return nil, m.orgErr
	}
	org, ok := m.organizations[id]
	if !ok {
		return nil, errors.New("org not found")
	}
	return org, nil
}

func (m *mockSchedulerStore) getHistory() []*models.ReportHistory {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*models.ReportHistory, len(m.reportHistory))
	copy(result, m.reportHistory)
	return result
}

func newTestScheduler(store *mockSchedulerStore) *Scheduler {
	logger := zerolog.Nop()
	config := DefaultSchedulerConfig()
	return NewScheduler(store, config, logger)
}

func TestReportScheduler_Daily(t *testing.T) {
	store := newMockSchedulerStore()
	scheduleID := uuid.New()
	orgID := uuid.New()

	store.reportSchedules = []*models.ReportSchedule{
		{
			ID:         scheduleID,
			OrgID:      orgID,
			Name:       "Daily Report",
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			Enabled:    true,
		},
	}

	sched := newTestScheduler(store)

	err := sched.Reload(context.Background())
	if err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}

	sched.mu.RLock()
	entryCount := len(sched.entries)
	_, exists := sched.entries[scheduleID]
	sched.mu.RUnlock()

	if entryCount != 1 {
		t.Errorf("entries count = %d, want 1", entryCount)
	}
	if !exists {
		t.Error("schedule entry not found for daily schedule")
	}
}

func TestReportScheduler_Weekly(t *testing.T) {
	store := newMockSchedulerStore()
	scheduleID := uuid.New()
	orgID := uuid.New()

	store.reportSchedules = []*models.ReportSchedule{
		{
			ID:         scheduleID,
			OrgID:      orgID,
			Name:       "Weekly Report",
			Frequency:  models.ReportFrequencyWeekly,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			Enabled:    true,
		},
	}

	sched := newTestScheduler(store)

	err := sched.Reload(context.Background())
	if err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}

	sched.mu.RLock()
	_, exists := sched.entries[scheduleID]
	sched.mu.RUnlock()

	if !exists {
		t.Error("schedule entry not found for weekly schedule")
	}
}

func TestReportScheduler_Monthly(t *testing.T) {
	store := newMockSchedulerStore()
	scheduleID := uuid.New()
	orgID := uuid.New()

	store.reportSchedules = []*models.ReportSchedule{
		{
			ID:         scheduleID,
			OrgID:      orgID,
			Name:       "Monthly Report",
			Frequency:  models.ReportFrequencyMonthly,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			Enabled:    true,
		},
	}

	sched := newTestScheduler(store)

	err := sched.Reload(context.Background())
	if err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}

	sched.mu.RLock()
	_, exists := sched.entries[scheduleID]
	sched.mu.RUnlock()

	if !exists {
		t.Error("schedule entry not found for monthly schedule")
	}
}

func TestReportScheduler_Reload(t *testing.T) {
	t.Run("adds new schedules", func(t *testing.T) {
		store := newMockSchedulerStore()
		id1 := uuid.New()
		id2 := uuid.New()
		orgID := uuid.New()

		store.reportSchedules = []*models.ReportSchedule{
			{ID: id1, OrgID: orgID, Frequency: models.ReportFrequencyDaily, Recipients: []string{"a@b.com"}, Timezone: "UTC", Enabled: true},
		}

		sched := newTestScheduler(store)
		if err := sched.Reload(context.Background()); err != nil {
			t.Fatalf("first Reload: %v", err)
		}

		sched.mu.RLock()
		if len(sched.entries) != 1 {
			t.Errorf("after first reload: entries = %d, want 1", len(sched.entries))
		}
		sched.mu.RUnlock()

		// Add another schedule
		store.reportSchedules = append(store.reportSchedules, &models.ReportSchedule{
			ID: id2, OrgID: orgID, Frequency: models.ReportFrequencyWeekly, Recipients: []string{"b@c.com"}, Timezone: "UTC", Enabled: true,
		})

		if err := sched.Reload(context.Background()); err != nil {
			t.Fatalf("second Reload: %v", err)
		}

		sched.mu.RLock()
		if len(sched.entries) != 2 {
			t.Errorf("after second reload: entries = %d, want 2", len(sched.entries))
		}
		sched.mu.RUnlock()
	})

	t.Run("removes disabled schedules", func(t *testing.T) {
		store := newMockSchedulerStore()
		id1 := uuid.New()
		orgID := uuid.New()

		store.reportSchedules = []*models.ReportSchedule{
			{ID: id1, OrgID: orgID, Frequency: models.ReportFrequencyDaily, Recipients: []string{"a@b.com"}, Timezone: "UTC", Enabled: true},
		}

		sched := newTestScheduler(store)
		if err := sched.Reload(context.Background()); err != nil {
			t.Fatalf("first Reload: %v", err)
		}

		// Remove the schedule (simulates being disabled)
		store.reportSchedules = []*models.ReportSchedule{}

		if err := sched.Reload(context.Background()); err != nil {
			t.Fatalf("second Reload: %v", err)
		}

		sched.mu.RLock()
		if len(sched.entries) != 0 {
			t.Errorf("after disabling: entries = %d, want 0", len(sched.entries))
		}
		sched.mu.RUnlock()
	})

	t.Run("does not duplicate existing schedules", func(t *testing.T) {
		store := newMockSchedulerStore()
		id1 := uuid.New()
		orgID := uuid.New()

		store.reportSchedules = []*models.ReportSchedule{
			{ID: id1, OrgID: orgID, Frequency: models.ReportFrequencyDaily, Recipients: []string{"a@b.com"}, Timezone: "UTC", Enabled: true},
		}

		sched := newTestScheduler(store)
		if err := sched.Reload(context.Background()); err != nil {
			t.Fatalf("first Reload: %v", err)
		}

		sched.mu.RLock()
		originalEntryID := sched.entries[id1]
		sched.mu.RUnlock()

		// Reload again with same schedule
		if err := sched.Reload(context.Background()); err != nil {
			t.Fatalf("second Reload: %v", err)
		}

		sched.mu.RLock()
		if sched.entries[id1] != originalEntryID {
			t.Error("existing schedule entry should not have been re-added")
		}
		if len(sched.entries) != 1 {
			t.Errorf("entries = %d, want 1", len(sched.entries))
		}
		sched.mu.RUnlock()
	})

	t.Run("store error", func(t *testing.T) {
		store := newMockSchedulerStore()
		store.reportSchedulesErr = errors.New("db down")

		sched := newTestScheduler(store)
		err := sched.Reload(context.Background())
		if err == nil {
			t.Error("Reload should return error when store fails")
		}
	})
}

func TestReportScheduler_StartStop(t *testing.T) {
	store := newMockSchedulerStore()
	sched := newTestScheduler(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := sched.Start(ctx)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	sched.mu.RLock()
	running := sched.running
	sched.mu.RUnlock()
	if !running {
		t.Error("scheduler should be running after Start")
	}

	// Start again should be a no-op
	err = sched.Start(ctx)
	if err != nil {
		t.Fatalf("second Start returned error: %v", err)
	}

	stopCtx := sched.Stop()
	<-stopCtx.Done()

	sched.mu.RLock()
	running = sched.running
	sched.mu.RUnlock()
	if running {
		t.Error("scheduler should not be running after Stop")
	}

	// Stop when already stopped
	stopCtx2 := sched.Stop()
	<-stopCtx2.Done()
}

func TestReportScheduler_Send(t *testing.T) {
	t.Run("no channel configured", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			Name:       "Test Report",
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			ChannelID:  nil,
		}

		reportData := &models.ReportData{}
		now := time.Now()

		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
		if err == nil {
			t.Error("SendReport should return error when no channel configured")
		}

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}
		if history[0].Status != models.ReportStatusFailed {
			t.Errorf("status = %q, want %q", history[0].Status, models.ReportStatusFailed)
		}
	})

	t.Run("channel not found", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		channelID := uuid.New()
		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			Name:       "Test Report",
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			ChannelID:  &channelID,
		}

		reportData := &models.ReportData{}
		now := time.Now()

		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
		if err == nil {
			t.Error("SendReport should return error when channel not found")
		}

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}
		if history[0].Status != models.ReportStatusFailed {
			t.Errorf("status = %q, want %q", history[0].Status, models.ReportStatusFailed)
		}
	})

	t.Run("non-email channel type", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		channelID := uuid.New()
		orgID := uuid.New()

		store.channels[channelID] = &models.NotificationChannel{
			ID:    channelID,
			OrgID: orgID,
			Type:  models.ChannelTypeSlack,
		}

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      orgID,
			Name:       "Test Report",
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			ChannelID:  &channelID,
		}

		reportData := &models.ReportData{}
		now := time.Now()

		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
		if err == nil {
			t.Error("SendReport should return error when channel is not email type")
		}

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}
		if history[0].Status != models.ReportStatusFailed {
			t.Errorf("status = %q, want %q", history[0].Status, models.ReportStatusFailed)
		}
	})

	t.Run("email channel with invalid SMTP config", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		channelID := uuid.New()
		orgID := uuid.New()

		// Email config with empty host will fail validation
		emailConfig := models.EmailChannelConfig{
			Host:     "",
			Port:     0,
			Username: "",
			Password: "",
			From:     "",
			TLS:      false,
		}
		configBytes, _ := json.Marshal(emailConfig)

		store.channels[channelID] = &models.NotificationChannel{
			ID:              channelID,
			OrgID:           orgID,
			Type:            models.ChannelTypeEmail,
			ConfigEncrypted: configBytes,
		}

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      orgID,
			Name:       "Test Report",
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			ChannelID:  &channelID,
		}

		reportData := &models.ReportData{}
		now := time.Now()

		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
		if err == nil {
			t.Error("SendReport should return error for invalid SMTP config")
		}

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}
		if history[0].Status != models.ReportStatusFailed {
			t.Errorf("status = %q, want %q", history[0].Status, models.ReportStatusFailed)
		}
	})

	t.Run("history records correct fields", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		orgID := uuid.New()
		scheduleID := uuid.New()
		schedule := &models.ReportSchedule{
			ID:         scheduleID,
			OrgID:      orgID,
			Name:       "Test Report",
			Frequency:  models.ReportFrequencyWeekly,
			Recipients: []string{"user1@example.com", "user2@example.com"},
			Timezone:   "UTC",
			ChannelID:  nil,
		}

		reportData := &models.ReportData{
			BackupSummary: models.BackupSummary{TotalBackups: 5},
		}
		periodStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		periodEnd := time.Date(2025, 1, 7, 23, 59, 59, 0, time.UTC)

		// Will fail because no channel, but history should still be created
		_ = sched.SendReport(context.Background(), schedule, reportData, periodStart, periodEnd, false)

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}

		h := history[0]
		if h.OrgID != orgID {
			t.Errorf("OrgID = %v, want %v", h.OrgID, orgID)
		}
		if *h.ScheduleID != scheduleID {
			t.Errorf("ScheduleID = %v, want %v", *h.ScheduleID, scheduleID)
		}
		if h.ReportType != string(models.ReportFrequencyWeekly) {
			t.Errorf("ReportType = %q, want %q", h.ReportType, models.ReportFrequencyWeekly)
		}
		if !h.PeriodStart.Equal(periodStart) {
			t.Errorf("PeriodStart = %v, want %v", h.PeriodStart, periodStart)
		}
		if !h.PeriodEnd.Equal(periodEnd) {
			t.Errorf("PeriodEnd = %v, want %v", h.PeriodEnd, periodEnd)
		}
		if len(h.Recipients) != 2 {
			t.Errorf("Recipients len = %d, want 2", len(h.Recipients))
		}
	})

	t.Run("create history error does not panic", func(t *testing.T) {
		store := newMockSchedulerStore()
		store.createHistoryErr = errors.New("db write failed")
		sched := newTestScheduler(store)

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"a@b.com"},
			Timezone:   "UTC",
		}

		reportData := &models.ReportData{}
		now := time.Now()

		// Should not panic even with history creation failure
		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
		if err == nil {
			t.Error("SendReport should still return error (no channel)")
		}
	})
}

func TestReportScheduler_Preview(t *testing.T) {
	t.Run("generates preview without sending", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		orgID := uuid.New()
		scheduleID := uuid.New()
		schedule := &models.ReportSchedule{
			ID:         scheduleID,
			OrgID:      orgID,
			Name:       "Test Report",
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
		}

		reportData := &models.ReportData{
			BackupSummary: models.BackupSummary{TotalBackups: 10},
		}
		now := time.Now()

		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, true)
		if err != nil {
			t.Fatalf("SendReport with preview=true returned error: %v", err)
		}

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}

		h := history[0]
		if h.Status != models.ReportStatusPreview {
			t.Errorf("status = %q, want %q", h.Status, models.ReportStatusPreview)
		}
		if h.ReportData == nil {
			t.Error("ReportData should not be nil for preview")
		}
		if h.ReportData.BackupSummary.TotalBackups != 10 {
			t.Errorf("TotalBackups = %d, want 10", h.ReportData.BackupSummary.TotalBackups)
		}
	})

	t.Run("preview does not require email channel", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			Frequency:  models.ReportFrequencyWeekly,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			ChannelID:  nil,
		}

		reportData := &models.ReportData{}
		now := time.Now()

		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-7*24*time.Hour), now, true)
		if err != nil {
			t.Errorf("preview should not fail without email channel: %v", err)
		}
	})

	t.Run("preview history creation error", func(t *testing.T) {
		store := newMockSchedulerStore()
		store.createHistoryErr = errors.New("db write failed")
		sched := newTestScheduler(store)

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"a@b.com"},
			Timezone:   "UTC",
		}

		reportData := &models.ReportData{}
		now := time.Now()

		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, true)
		if err == nil {
			t.Error("preview should return error when history creation fails")
		}
	})
}

func TestReportScheduler_GeneratePreview(t *testing.T) {
	store := newMockSchedulerStore()
	store.backups = []*models.Backup{}
	store.schedules = []*models.Schedule{}
	store.agents = []*models.Agent{}
	store.alerts = []*models.Alert{}

	sched := newTestScheduler(store)
	orgID := uuid.New()

	data, periodStart, periodEnd, err := sched.GeneratePreview(context.Background(), orgID, models.ReportFrequencyDaily, "UTC")
	if err != nil {
		t.Fatalf("GeneratePreview returned error: %v", err)
	}
	if data == nil {
		t.Fatal("report data should not be nil")
	}
	if periodStart.IsZero() || periodEnd.IsZero() {
		t.Error("period times should not be zero")
	}
	if !periodEnd.After(periodStart) {
		t.Error("period end should be after period start")
	}
}

func TestReportScheduler_GetGenerator(t *testing.T) {
	store := newMockSchedulerStore()
	sched := newTestScheduler(store)

	gen := sched.GetGenerator()
	if gen == nil {
		t.Fatal("GetGenerator should not return nil")
	}
}

func TestReportScheduler_FrequencyToCron(t *testing.T) {
	store := newMockSchedulerStore()
	sched := newTestScheduler(store)

	tests := []struct {
		frequency models.ReportFrequency
		want      string
	}{
		{models.ReportFrequencyDaily, "0 0 8 * * *"},
		{models.ReportFrequencyWeekly, "0 0 8 * * 1"},
		{models.ReportFrequencyMonthly, "0 0 8 1 * *"},
		{"unknown", "0 0 8 * * 1"},
	}

	for _, tt := range tests {
		t.Run(string(tt.frequency), func(t *testing.T) {
			got := sched.frequencyToCron(tt.frequency)
			if got != tt.want {
				t.Errorf("frequencyToCron(%q) = %q, want %q", tt.frequency, got, tt.want)
			}
		})
	}
}

func TestReportScheduler_SendWithOrganization(t *testing.T) {
	store := newMockSchedulerStore()
	sched := newTestScheduler(store)

	orgID := uuid.New()
	channelID := uuid.New()

	// Set up org
	store.organizations[orgID] = &models.Organization{
		ID:   orgID,
		Name: "Test Corp",
		Slug: "test-corp",
	}

	// Non-email channel - will still fail at "no email channel" but org lookup code is exercised
	store.channels[channelID] = &models.NotificationChannel{
		ID:    channelID,
		OrgID: orgID,
		Type:  models.ChannelTypeWebhook,
	}

	schedule := &models.ReportSchedule{
		ID:         uuid.New(),
		OrgID:      orgID,
		Name:       "Test Report",
		Frequency:  models.ReportFrequencyDaily,
		Recipients: []string{"admin@example.com"},
		Timezone:   "UTC",
		ChannelID:  &channelID,
	}

	reportData := &models.ReportData{}
	now := time.Now()

	err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
	if err == nil {
		t.Error("SendReport should fail when channel is not email type")
	}
}

func TestReportScheduler_SendWithInvalidChannelConfig(t *testing.T) {
	store := newMockSchedulerStore()
	sched := newTestScheduler(store)

	orgID := uuid.New()
	channelID := uuid.New()

	// Email channel with invalid JSON config
	store.channels[channelID] = &models.NotificationChannel{
		ID:              channelID,
		OrgID:           orgID,
		Type:            models.ChannelTypeEmail,
		ConfigEncrypted: []byte("not valid json"),
	}

	schedule := &models.ReportSchedule{
		ID:         uuid.New(),
		OrgID:      orgID,
		Name:       "Test Report",
		Frequency:  models.ReportFrequencyDaily,
		Recipients: []string{"admin@example.com"},
		Timezone:   "UTC",
		ChannelID:  &channelID,
	}

	reportData := &models.ReportData{}
	now := time.Now()

	err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
	if err == nil {
		t.Error("SendReport should fail when channel config is invalid JSON")
	}
}

func TestReportScheduler_SendWithValidSMTP(t *testing.T) {
	t.Run("valid SMTP config reaches email sending", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		orgID := uuid.New()
		channelID := uuid.New()

		emailConfig := models.EmailChannelConfig{
			Host:     "localhost",
			Port:     25,
			Username: "user",
			Password: "pass",
			From:     "reports@example.com",
			TLS:      false,
		}
		configBytes, _ := json.Marshal(emailConfig)

		store.channels[channelID] = &models.NotificationChannel{
			ID:              channelID,
			OrgID:           orgID,
			Type:            models.ChannelTypeEmail,
			ConfigEncrypted: configBytes,
		}

		store.organizations[orgID] = &models.Organization{
			ID:   orgID,
			Name: "Acme Corp",
		}

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      orgID,
			Name:       "Weekly Report",
			Frequency:  models.ReportFrequencyWeekly,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			ChannelID:  &channelID,
		}

		reportData := &models.ReportData{
			BackupSummary: models.BackupSummary{
				TotalBackups:      10,
				SuccessfulBackups: 8,
				FailedBackups:     2,
				SuccessRate:       80.0,
				TotalDataBacked:   1024 * 1024 * 100,
				SchedulesActive:   3,
			},
			StorageSummary: models.StorageSummary{
				TotalRawSize:     1024 * 1024 * 50,
				TotalRestoreSize: 1024 * 1024 * 200,
				SpaceSaved:       1024 * 1024 * 150,
				SpaceSavedPct:    75.0,
				RepositoryCount:  2,
				TotalSnapshots:   20,
			},
			AgentSummary: models.AgentSummary{
				TotalAgents:   5,
				ActiveAgents:  3,
				OfflineAgents: 1,
				PendingAgents: 1,
			},
			AlertSummary: models.AlertSummary{
				TotalAlerts:    4,
				CriticalAlerts: 1,
				WarningAlerts:  3,
			},
		}
		now := time.Now()

		// Will fail at SMTP connection but exercises the email data conversion code
		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-7*24*time.Hour), now, false)
		if err == nil {
			t.Error("SendReport should fail when SMTP server is unreachable")
		}

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}
		if history[0].Status != models.ReportStatusFailed {
			t.Errorf("status = %q, want %q", history[0].Status, models.ReportStatusFailed)
		}
	})

	t.Run("valid SMTP with top issues", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		orgID := uuid.New()
		channelID := uuid.New()

		emailConfig := models.EmailChannelConfig{
			Host: "localhost",
			Port: 25,
			From: "reports@example.com",
		}
		configBytes, _ := json.Marshal(emailConfig)

		store.channels[channelID] = &models.NotificationChannel{
			ID:              channelID,
			OrgID:           orgID,
			Type:            models.ChannelTypeEmail,
			ConfigEncrypted: configBytes,
		}

		now := time.Now()
		reportData := &models.ReportData{
			TopIssues: []models.ReportIssue{
				{Type: "agent_offline", Severity: "critical", Title: "Agent down", Description: "Host-1 offline", OccurredAt: now},
				{Type: "backup_sla", Severity: "critical", Title: "SLA breach", Description: "Overdue", OccurredAt: now},
			},
		}

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      orgID,
			Name:       "Daily Report",
			Frequency:  models.ReportFrequencyDaily,
			Recipients: []string{"admin@example.com"},
			Timezone:   "UTC",
			ChannelID:  &channelID,
		}

		// Will fail at SMTP but exercises the top issues conversion
		err := sched.SendReport(context.Background(), schedule, reportData, now.Add(-24*time.Hour), now, false)
		if err == nil {
			t.Error("SendReport should fail when SMTP server is unreachable")
		}

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}
		if history[0].Status != models.ReportStatusFailed {
			t.Errorf("status = %q, want %q", history[0].Status, models.ReportStatusFailed)
		}
	})

	t.Run("org not found uses schedule name", func(t *testing.T) {
		store := newMockSchedulerStore()
		sched := newTestScheduler(store)

		orgID := uuid.New()
		channelID := uuid.New()

		emailConfig := models.EmailChannelConfig{
			Host: "localhost",
			Port: 25,
			From: "reports@example.com",
		}
		configBytes, _ := json.Marshal(emailConfig)

		store.channels[channelID] = &models.NotificationChannel{
			ID:              channelID,
			OrgID:           orgID,
			Type:            models.ChannelTypeEmail,
			ConfigEncrypted: configBytes,
		}

		// No org in store - will fall back to schedule.Name
		store.orgErr = errors.New("org not found")

		schedule := &models.ReportSchedule{
			ID:         uuid.New(),
			OrgID:      orgID,
			Name:       "Fallback Name",
			Frequency:  models.ReportFrequencyMonthly,
			Recipients: []string{"a@b.com"},
			Timezone:   "America/New_York",
			ChannelID:  &channelID,
		}

		reportData := &models.ReportData{}
		now := time.Now()

		// Will fail at SMTP but exercises the org-not-found fallback
		_ = sched.SendReport(context.Background(), schedule, reportData, now.Add(-30*24*time.Hour), now, false)

		history := store.getHistory()
		if len(history) != 1 {
			t.Fatalf("history len = %d, want 1", len(history))
		}
	})
}

func TestReportScheduler_ExecuteReport(t *testing.T) {
	store := newMockSchedulerStore()
	store.backups = []*models.Backup{}
	store.schedules = []*models.Schedule{}
	store.agents = []*models.Agent{}
	store.alerts = []*models.Alert{}

	sched := newTestScheduler(store)

	schedule := &models.ReportSchedule{
		ID:         uuid.New(),
		OrgID:      uuid.New(),
		Name:       "Test Report",
		Frequency:  models.ReportFrequencyDaily,
		Recipients: []string{"admin@example.com"},
		Timezone:   "UTC",
		ChannelID:  nil,
	}

	// executeReport is a private method that handles errors internally (logging)
	// It will fail because no email channel is configured
	sched.executeReport(schedule)

	// No history should be created because SendReport fails before creating history
	// Actually, history IS created in the "no email channel" path
	history := store.getHistory()
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
}

func TestReportScheduler_ExecuteReportWithSMTP(t *testing.T) {
	store := newMockSchedulerStore()
	store.backups = []*models.Backup{}
	store.schedules = []*models.Schedule{}
	store.agents = []*models.Agent{}
	store.alerts = []*models.Alert{}

	sched := newTestScheduler(store)

	orgID := uuid.New()
	channelID := uuid.New()

	emailConfig := models.EmailChannelConfig{
		Host: "localhost",
		Port: 25,
		From: "reports@example.com",
	}
	configBytes, _ := json.Marshal(emailConfig)

	store.channels[channelID] = &models.NotificationChannel{
		ID:              channelID,
		OrgID:           orgID,
		Type:            models.ChannelTypeEmail,
		ConfigEncrypted: configBytes,
	}

	schedule := &models.ReportSchedule{
		ID:         uuid.New(),
		OrgID:      orgID,
		Name:       "Test Report",
		Frequency:  models.ReportFrequencyWeekly,
		Recipients: []string{"admin@example.com"},
		Timezone:   "UTC",
		ChannelID:  &channelID,
	}

	// Will generate report and try to send - fails at SMTP but exercises executeReport fully
	sched.executeReport(schedule)

	history := store.getHistory()
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
}
