package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockAggregatorStore struct {
	orgs            []*models.Organization
	backups         []*models.Backup
	savedSummary    *models.MetricsDailySummary

	orgsErr         error
	backupsErr      error
	upsertErr       error
}

func (m *mockAggregatorStore) GetAllOrganizations(_ context.Context) ([]*models.Organization, error) {
	return m.orgs, m.orgsErr
}

func (m *mockAggregatorStore) GetBackupsByOrgIDAndDateRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.Backup, error) {
	return m.backups, m.backupsErr
}

func (m *mockAggregatorStore) CreateOrUpdateDailySummary(_ context.Context, summary *models.MetricsDailySummary) error {
	m.savedSummary = summary
	return m.upsertErr
}

func TestAggregator_AggregateDailyMetrics(t *testing.T) {
	orgID := uuid.New()
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	completedAt := date.Add(10 * time.Minute)
	size := int64(2048)

	t.Run("aggregates backup stats correctly", func(t *testing.T) {
		agentID1 := uuid.New()
		agentID2 := uuid.New()

		store := &mockAggregatorStore{
			backups: []*models.Backup{
				{
					ID:          uuid.New(),
					AgentID:     agentID1,
					Status:      models.BackupStatusCompleted,
					SizeBytes:   &size,
					StartedAt:   date,
					CompletedAt: &completedAt,
				},
				{
					ID:          uuid.New(),
					AgentID:     agentID2,
					Status:      models.BackupStatusCompleted,
					SizeBytes:   &size,
					StartedAt:   date.Add(time.Hour),
					CompletedAt: &completedAt,
				},
				{
					ID:        uuid.New(),
					AgentID:   agentID1,
					Status:    models.BackupStatusFailed,
					StartedAt: date.Add(2 * time.Hour),
				},
			},
		}

		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateDailyMetrics(context.Background(), orgID, date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := store.savedSummary
		if s == nil {
			t.Fatal("expected summary to be saved")
		}
		if s.TotalBackups != 3 {
			t.Errorf("expected 3 total backups, got %d", s.TotalBackups)
		}
		if s.SuccessfulBackups != 2 {
			t.Errorf("expected 2 successful, got %d", s.SuccessfulBackups)
		}
		if s.FailedBackups != 1 {
			t.Errorf("expected 1 failed, got %d", s.FailedBackups)
		}
		if s.TotalSizeBytes != 4096 {
			t.Errorf("expected total size 4096, got %d", s.TotalSizeBytes)
		}
		if s.AgentsActive != 2 {
			t.Errorf("expected 2 active agents, got %d", s.AgentsActive)
		}
		if s.OrgID != orgID {
			t.Errorf("expected org_id %s, got %s", orgID, s.OrgID)
		}
	})

	t.Run("handles empty backups", func(t *testing.T) {
		store := &mockAggregatorStore{
			backups: []*models.Backup{},
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateDailyMetrics(context.Background(), orgID, date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := store.savedSummary
		if s.TotalBackups != 0 {
			t.Errorf("expected 0 total backups, got %d", s.TotalBackups)
		}
		if s.AgentsActive != 0 {
			t.Errorf("expected 0 active agents, got %d", s.AgentsActive)
		}
	})

	t.Run("handles nil size bytes", func(t *testing.T) {
		store := &mockAggregatorStore{
			backups: []*models.Backup{
				{
					ID:          uuid.New(),
					AgentID:     uuid.New(),
					Status:      models.BackupStatusCompleted,
					SizeBytes:   nil,
					StartedAt:   date,
					CompletedAt: &completedAt,
				},
			},
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateDailyMetrics(context.Background(), orgID, date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.savedSummary.TotalSizeBytes != 0 {
			t.Errorf("expected 0 total size for nil, got %d", store.savedSummary.TotalSizeBytes)
		}
	})

	t.Run("calculates duration in seconds", func(t *testing.T) {
		start := date
		end := date.Add(5 * time.Minute)
		store := &mockAggregatorStore{
			backups: []*models.Backup{
				{
					ID:          uuid.New(),
					AgentID:     uuid.New(),
					Status:      models.BackupStatusCompleted,
					StartedAt:   start,
					CompletedAt: &end,
				},
			},
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateDailyMetrics(context.Background(), orgID, date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.savedSummary.TotalDurationSecs != 300 {
			t.Errorf("expected 300 seconds, got %d", store.savedSummary.TotalDurationSecs)
		}
	})

	t.Run("error fetching backups", func(t *testing.T) {
		store := &mockAggregatorStore{
			backupsErr: errors.New("db error"),
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateDailyMetrics(context.Background(), orgID, date)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("error upserting summary", func(t *testing.T) {
		store := &mockAggregatorStore{
			backups:   []*models.Backup{},
			upsertErr: errors.New("db error"),
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateDailyMetrics(context.Background(), orgID, date)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAggregator_AggregateAllOrgs(t *testing.T) {
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("aggregates all organizations", func(t *testing.T) {
		store := &mockAggregatorStore{
			orgs: []*models.Organization{
				{ID: uuid.New(), Name: "Org1"},
				{ID: uuid.New(), Name: "Org2"},
			},
			backups: []*models.Backup{},
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateAllOrgs(context.Background(), date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error fetching organizations", func(t *testing.T) {
		store := &mockAggregatorStore{
			orgsErr: errors.New("db error"),
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateAllOrgs(context.Background(), date)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("continues on individual org failure", func(t *testing.T) {
		store := &mockAggregatorStore{
			orgs: []*models.Organization{
				{ID: uuid.New(), Name: "Org1"},
			},
			backupsErr: errors.New("db error"),
		}
		a := NewAggregator(store, zerolog.Nop())
		err := a.AggregateAllOrgs(context.Background(), date)
		if err == nil {
			t.Fatal("expected error for failed org aggregation")
		}
	})
}

func TestNewAggregator(t *testing.T) {
	store := &mockAggregatorStore{}
	a := NewAggregator(store, zerolog.Nop())
	if a == nil {
		t.Fatal("expected non-nil aggregator")
	}
}
