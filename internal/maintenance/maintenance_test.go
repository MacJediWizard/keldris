package maintenance

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockStore implements MaintenanceStore for testing.
type mockStore struct {
	activeWindows   map[uuid.UUID][]*models.MaintenanceWindow
	upcomingWindows map[uuid.UUID][]*models.MaintenanceWindow
	pendingNotifs   []*models.MaintenanceWindow
	organizations   []*models.Organization
	markedSent      []uuid.UUID

	activeErr   error
	upcomingErr error
	pendingErr  error
	markErr     error
	orgsErr     error
}

func newMockStore() *mockStore {
	return &mockStore{
		activeWindows:   make(map[uuid.UUID][]*models.MaintenanceWindow),
		upcomingWindows: make(map[uuid.UUID][]*models.MaintenanceWindow),
	}
}

func (m *mockStore) ListActiveMaintenanceWindows(_ context.Context, orgID uuid.UUID, _ time.Time) ([]*models.MaintenanceWindow, error) {
	if m.activeErr != nil {
		return nil, m.activeErr
	}
	return m.activeWindows[orgID], nil
}

func (m *mockStore) ListUpcomingMaintenanceWindows(_ context.Context, orgID uuid.UUID, _ time.Time, _ int) ([]*models.MaintenanceWindow, error) {
	if m.upcomingErr != nil {
		return nil, m.upcomingErr
	}
	return m.upcomingWindows[orgID], nil
}

func (m *mockStore) ListPendingMaintenanceNotifications(_ context.Context) ([]*models.MaintenanceWindow, error) {
	if m.pendingErr != nil {
		return nil, m.pendingErr
	}
	return m.pendingNotifs, nil
}

func (m *mockStore) MarkMaintenanceNotificationSent(_ context.Context, id uuid.UUID) error {
	if m.markErr != nil {
		return m.markErr
	}
	m.markedSent = append(m.markedSent, id)
	return nil
}

func (m *mockStore) GetAllOrganizations(_ context.Context) ([]*models.Organization, error) {
	if m.orgsErr != nil {
		return nil, m.orgsErr
	}
	return m.organizations, nil
}

// helper to create a test service with a mock store.
func newTestService(store *mockStore) *Service {
	logger := zerolog.Nop()
	return NewService(store, nil, logger)
}

// helper to create a maintenance window relative to now.
func makeWindow(orgID uuid.UUID, title string, startOffset, endOffset time.Duration) *models.MaintenanceWindow {
	now := time.Now()
	return &models.MaintenanceWindow{
		ID:                  uuid.New(),
		OrgID:               orgID,
		Title:               title,
		StartsAt:            now.Add(startOffset),
		EndsAt:              now.Add(endOffset),
		NotifyBeforeMinutes: 60,
	}
}

func TestNewService(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	svc := NewService(store, nil, logger)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.store != store {
		t.Error("expected store to be set")
	}
	if svc.activeWindows == nil {
		t.Error("expected activeWindows map to be initialized")
	}
	if !svc.lastRefresh.IsZero() {
		t.Error("expected lastRefresh to be zero initially")
	}
}

func TestRefreshCache_Success(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	store.organizations = []*models.Organization{
		{ID: orgID, Name: "Test Org", Slug: "test-org"},
	}

	window := makeWindow(orgID, "DB Upgrade", -1*time.Hour, 1*time.Hour)
	store.activeWindows[orgID] = []*models.MaintenanceWindow{window}

	svc := newTestService(store)

	err := svc.RefreshCache(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.lastRefresh.IsZero() {
		t.Error("expected lastRefresh to be updated")
	}

	svc.mu.RLock()
	cached := svc.activeWindows[orgID]
	svc.mu.RUnlock()

	if len(cached) != 1 {
		t.Fatalf("expected 1 cached window, got %d", len(cached))
	}
	if cached[0].Title != "DB Upgrade" {
		t.Errorf("expected title 'DB Upgrade', got %q", cached[0].Title)
	}
}

func TestRefreshCache_MultipleOrgs(t *testing.T) {
	store := newMockStore()
	org1 := uuid.New()
	org2 := uuid.New()
	store.organizations = []*models.Organization{
		{ID: org1, Name: "Org 1", Slug: "org-1"},
		{ID: org2, Name: "Org 2", Slug: "org-2"},
	}

	store.activeWindows[org1] = []*models.MaintenanceWindow{
		makeWindow(org1, "Window A", -1*time.Hour, 1*time.Hour),
	}
	store.activeWindows[org2] = []*models.MaintenanceWindow{
		makeWindow(org2, "Window B", -30*time.Minute, 30*time.Minute),
		makeWindow(org2, "Window C", -15*time.Minute, 45*time.Minute),
	}

	svc := newTestService(store)

	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	if len(svc.activeWindows[org1]) != 1 {
		t.Errorf("expected 1 window for org1, got %d", len(svc.activeWindows[org1]))
	}
	if len(svc.activeWindows[org2]) != 2 {
		t.Errorf("expected 2 windows for org2, got %d", len(svc.activeWindows[org2]))
	}
}

func TestRefreshCache_NoOrgs(t *testing.T) {
	store := newMockStore()
	store.organizations = []*models.Organization{}

	svc := newTestService(store)

	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	if len(svc.activeWindows) != 0 {
		t.Errorf("expected empty cache, got %d entries", len(svc.activeWindows))
	}
}

func TestRefreshCache_OrgWithNoActiveWindows(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	store.organizations = []*models.Organization{
		{ID: orgID, Name: "Empty Org", Slug: "empty-org"},
	}
	// No active windows for this org
	store.activeWindows[orgID] = []*models.MaintenanceWindow{}

	svc := newTestService(store)

	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	if _, exists := svc.activeWindows[orgID]; exists {
		t.Error("expected org with no windows to not be in cache")
	}
}

func TestRefreshCache_GetOrgsError(t *testing.T) {
	store := newMockStore()
	store.orgsErr = errors.New("db connection failed")

	svc := newTestService(store)

	err := svc.RefreshCache(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db connection failed" {
		t.Errorf("expected 'db connection failed', got %q", err.Error())
	}
}

func TestRefreshCache_PartialOrgError(t *testing.T) {
	store := newMockStore()
	org1 := uuid.New()
	org2 := uuid.New()
	store.organizations = []*models.Organization{
		{ID: org1, Name: "Org 1", Slug: "org-1"},
		{ID: org2, Name: "Org 2", Slug: "org-2"},
	}

	// org1 has windows, org2 will fail via activeErr
	store.activeWindows[org1] = []*models.MaintenanceWindow{
		makeWindow(org1, "Window A", -1*time.Hour, 1*time.Hour),
	}

	// To simulate a per-org error we need a custom store that errors for org2 only.
	customStore := &perOrgErrorStore{
		mockStore: store,
		errorOrg:  org2,
	}

	svc := NewService(customStore, nil, zerolog.Nop())

	err := svc.RefreshCache(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	// org1 should still be cached even though org2 failed
	if len(svc.activeWindows[org1]) != 1 {
		t.Errorf("expected 1 window for org1, got %d", len(svc.activeWindows[org1]))
	}
}

// perOrgErrorStore returns an error only for a specific org.
type perOrgErrorStore struct {
	*mockStore
	errorOrg uuid.UUID
}

func (s *perOrgErrorStore) ListActiveMaintenanceWindows(ctx context.Context, orgID uuid.UUID, now time.Time) ([]*models.MaintenanceWindow, error) {
	if orgID == s.errorOrg {
		return nil, errors.New("org-specific db error")
	}
	return s.mockStore.ListActiveMaintenanceWindows(ctx, orgID, now)
}

func TestRefreshCache_ReplacesOldCache(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	store.organizations = []*models.Organization{
		{ID: orgID, Name: "Test Org", Slug: "test-org"},
	}

	window1 := makeWindow(orgID, "Old Window", -2*time.Hour, -1*time.Hour)
	store.activeWindows[orgID] = []*models.MaintenanceWindow{window1}

	svc := newTestService(store)
	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now replace with a new window
	window2 := makeWindow(orgID, "New Window", -30*time.Minute, 30*time.Minute)
	store.activeWindows[orgID] = []*models.MaintenanceWindow{window2}

	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error on second refresh: %v", err)
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	cached := svc.activeWindows[orgID]
	if len(cached) != 1 {
		t.Fatalf("expected 1 cached window, got %d", len(cached))
	}
	if cached[0].Title != "New Window" {
		t.Errorf("expected 'New Window', got %q", cached[0].Title)
	}
}

func TestIsMaintenanceActive_ActiveWindow(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	// Manually populate cache with an active window
	window := makeWindow(orgID, "Active", -1*time.Hour, 1*time.Hour)
	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window}
	svc.mu.Unlock()

	if !svc.IsMaintenanceActive(orgID) {
		t.Error("expected maintenance to be active")
	}
}

func TestIsMaintenanceActive_NoActiveWindow(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	// Window in the future - not yet active
	window := makeWindow(orgID, "Future", 1*time.Hour, 2*time.Hour)
	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window}
	svc.mu.Unlock()

	if svc.IsMaintenanceActive(orgID) {
		t.Error("expected maintenance to not be active for future window")
	}
}

func TestIsMaintenanceActive_PastWindow(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	// Window in the past - already ended
	window := makeWindow(orgID, "Past", -2*time.Hour, -1*time.Hour)
	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window}
	svc.mu.Unlock()

	if svc.IsMaintenanceActive(orgID) {
		t.Error("expected maintenance to not be active for past window")
	}
}

func TestIsMaintenanceActive_NoWindowsForOrg(t *testing.T) {
	store := newMockStore()
	svc := newTestService(store)

	// No windows loaded for this org
	if svc.IsMaintenanceActive(uuid.New()) {
		t.Error("expected maintenance to not be active for unknown org")
	}
}

func TestIsMaintenanceActive_EmptyCache(t *testing.T) {
	store := newMockStore()
	svc := newTestService(store)

	if svc.IsMaintenanceActive(uuid.New()) {
		t.Error("expected maintenance to not be active with empty cache")
	}
}

func TestIsMaintenanceActive_MultipleWindows_OneActive(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	past := makeWindow(orgID, "Past", -3*time.Hour, -2*time.Hour)
	active := makeWindow(orgID, "Active", -1*time.Hour, 1*time.Hour)
	future := makeWindow(orgID, "Future", 2*time.Hour, 3*time.Hour)

	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{past, active, future}
	svc.mu.Unlock()

	if !svc.IsMaintenanceActive(orgID) {
		t.Error("expected maintenance to be active when one of multiple windows is active")
	}
}

func TestGetActiveWindow_ReturnsActiveWindow(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	window := makeWindow(orgID, "Active Maintenance", -1*time.Hour, 1*time.Hour)
	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window}
	svc.mu.Unlock()

	result := svc.GetActiveWindow(orgID)
	if result == nil {
		t.Fatal("expected non-nil window")
	}
	if result.Title != "Active Maintenance" {
		t.Errorf("expected 'Active Maintenance', got %q", result.Title)
	}
}

func TestGetActiveWindow_NoActiveWindow(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	window := makeWindow(orgID, "Future", 1*time.Hour, 2*time.Hour)
	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window}
	svc.mu.Unlock()

	result := svc.GetActiveWindow(orgID)
	if result != nil {
		t.Error("expected nil for no active window")
	}
}

func TestGetActiveWindow_NoWindowsForOrg(t *testing.T) {
	store := newMockStore()
	svc := newTestService(store)

	result := svc.GetActiveWindow(uuid.New())
	if result != nil {
		t.Error("expected nil for unknown org")
	}
}

func TestGetActiveWindow_MultipleActive_ReturnsFirst(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	window1 := makeWindow(orgID, "First Active", -2*time.Hour, 1*time.Hour)
	window2 := makeWindow(orgID, "Second Active", -1*time.Hour, 2*time.Hour)

	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window1, window2}
	svc.mu.Unlock()

	result := svc.GetActiveWindow(orgID)
	if result == nil {
		t.Fatal("expected non-nil window")
	}
	if result.Title != "First Active" {
		t.Errorf("expected 'First Active', got %q", result.Title)
	}
}


func TestCheckAndSendNotifications_Success(t *testing.T) {
	store := newMockStore()
	w1 := makeWindow(uuid.New(), "Notify Me", 30*time.Minute, 2*time.Hour)
	w2 := makeWindow(uuid.New(), "Also Notify", 45*time.Minute, 3*time.Hour)
	store.pendingNotifs = []*models.MaintenanceWindow{w1, w2}

	svc := newTestService(store)

	svc.CheckAndSendNotifications(context.Background())

	if len(store.markedSent) != 2 {
		t.Fatalf("expected 2 notifications marked sent, got %d", len(store.markedSent))
	}
	if store.markedSent[0] != w1.ID {
		t.Errorf("expected first marked ID %s, got %s", w1.ID, store.markedSent[0])
	}
	if store.markedSent[1] != w2.ID {
		t.Errorf("expected second marked ID %s, got %s", w2.ID, store.markedSent[1])
	}
}

func TestCheckAndSendNotifications_NoPending(t *testing.T) {
	store := newMockStore()
	store.pendingNotifs = []*models.MaintenanceWindow{}

	svc := newTestService(store)

	// Should not panic or error
	svc.CheckAndSendNotifications(context.Background())

	if len(store.markedSent) != 0 {
		t.Errorf("expected no notifications marked sent, got %d", len(store.markedSent))
	}
}

func TestCheckAndSendNotifications_ListError(t *testing.T) {
	store := newMockStore()
	store.pendingErr = errors.New("db error")

	svc := newTestService(store)

	// Should log error but not panic
	svc.CheckAndSendNotifications(context.Background())

	if len(store.markedSent) != 0 {
		t.Errorf("expected no notifications marked sent on error, got %d", len(store.markedSent))
	}
}

func TestCheckAndSendNotifications_MarkError(t *testing.T) {
	store := newMockStore()
	w := makeWindow(uuid.New(), "Fail Mark", 30*time.Minute, 2*time.Hour)
	store.pendingNotifs = []*models.MaintenanceWindow{w}
	store.markErr = errors.New("mark failed")

	svc := newTestService(store)

	// Should log error but not panic
	svc.CheckAndSendNotifications(context.Background())

	// markedSent should be empty since mark returned an error
	if len(store.markedSent) != 0 {
		t.Errorf("expected no notifications marked sent on mark error, got %d", len(store.markedSent))
	}
}

func TestCheckAndSendNotifications_NilNotifier(t *testing.T) {
	store := newMockStore()
	w := makeWindow(uuid.New(), "No Notifier", 30*time.Minute, 2*time.Hour)
	store.pendingNotifs = []*models.MaintenanceWindow{w}

	// Service created with nil notifier
	svc := newTestService(store)

	// Should not panic and should still mark sent
	svc.CheckAndSendNotifications(context.Background())

	if len(store.markedSent) != 1 {
		t.Fatalf("expected 1 notification marked sent, got %d", len(store.markedSent))
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	store.organizations = []*models.Organization{
		{ID: orgID, Name: "Concurrent Org", Slug: "concurrent-org"},
	}
	store.activeWindows[orgID] = []*models.MaintenanceWindow{
		makeWindow(orgID, "Concurrent Window", -1*time.Hour, 1*time.Hour),
	}

	svc := newTestService(store)

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	// Concurrent RefreshCache calls
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.RefreshCache(context.Background()); err != nil {
				errCh <- err
			}
		}()
	}

	// Concurrent read calls
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.IsMaintenanceActive(orgID)
			svc.GetActiveWindow(orgID)
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent error: %v", err)
	}
}

func TestWindowOverlap(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	svc := newTestService(store)

	// Two overlapping windows
	window1 := makeWindow(orgID, "Window 1", -2*time.Hour, 30*time.Minute)
	window2 := makeWindow(orgID, "Window 2", -1*time.Hour, 1*time.Hour)

	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window1, window2}
	svc.mu.Unlock()

	// Both are active, so maintenance should be active
	if !svc.IsMaintenanceActive(orgID) {
		t.Error("expected maintenance to be active with overlapping windows")
	}

	// GetActiveWindow returns the first active one
	result := svc.GetActiveWindow(orgID)
	if result == nil {
		t.Fatal("expected non-nil window")
	}
	if result.Title != "Window 1" {
		t.Errorf("expected 'Window 1', got %q", result.Title)
	}
}

func TestBackupBlockingDuringMaintenance(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	otherOrgID := uuid.New()
	svc := newTestService(store)

	// Only orgID has an active window
	window := makeWindow(orgID, "Blocking Window", -1*time.Hour, 1*time.Hour)
	svc.mu.Lock()
	svc.activeWindows[orgID] = []*models.MaintenanceWindow{window}
	svc.mu.Unlock()

	// orgID should be blocked
	if !svc.IsMaintenanceActive(orgID) {
		t.Error("expected backups to be blocked for org with active maintenance")
	}

	// other org should not be blocked
	if svc.IsMaintenanceActive(otherOrgID) {
		t.Error("expected backups to not be blocked for org without maintenance")
	}
}

func TestRecurringMaintenanceSchedule(t *testing.T) {
	store := newMockStore()
	orgID := uuid.New()
	store.organizations = []*models.Organization{
		{ID: orgID, Name: "Recurring Org", Slug: "recurring-org"},
	}

	// Simulate recurring schedule: multiple windows at regular intervals
	now := time.Now()
	windows := []*models.MaintenanceWindow{
		{
			ID:       uuid.New(),
			OrgID:    orgID,
			Title:    "Weekly Maintenance - Week 1",
			StartsAt: now.Add(-30 * time.Minute),
			EndsAt:   now.Add(30 * time.Minute),
		},
	}
	store.activeWindows[orgID] = windows

	svc := newTestService(store)

	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !svc.IsMaintenanceActive(orgID) {
		t.Error("expected recurring maintenance window to be active")
	}

	// After the window ends, simulate next refresh with the window gone
	store.activeWindows[orgID] = []*models.MaintenanceWindow{}

	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.IsMaintenanceActive(orgID) {
		t.Error("expected maintenance to not be active after window ends")
	}

	// Simulate next recurring window becoming active
	nextWindow := &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Weekly Maintenance - Week 2",
		StartsAt: now.Add(-10 * time.Minute),
		EndsAt:   now.Add(50 * time.Minute),
	}
	store.activeWindows[orgID] = []*models.MaintenanceWindow{nextWindow}

	if err := svc.RefreshCache(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !svc.IsMaintenanceActive(orgID) {
		t.Error("expected next recurring window to be active")
	}

	active := svc.GetActiveWindow(orgID)
	if active == nil {
		t.Fatal("expected non-nil active window")
	}
	if active.Title != "Weekly Maintenance - Week 2" {
		t.Errorf("expected 'Weekly Maintenance - Week 2', got %q", active.Title)
	}
}

func TestCheckAndSendNotifications_MultipleWithPartialMarkError(t *testing.T) {
	// Use a custom store that fails on the second mark
	store := &selectiveMarkErrorStore{
		mockStore: newMockStore(),
		failID:    uuid.Nil, // will be set below
	}

	w1 := makeWindow(uuid.New(), "Success", 30*time.Minute, 2*time.Hour)
	w2 := makeWindow(uuid.New(), "Fail Mark", 45*time.Minute, 3*time.Hour)
	store.failID = w2.ID
	store.pendingNotifs = []*models.MaintenanceWindow{w1, w2}

	svc := NewService(store, nil, zerolog.Nop())

	svc.CheckAndSendNotifications(context.Background())

	// w1 should be marked, w2 should not
	if len(store.markedSent) != 1 {
		t.Fatalf("expected 1 notification marked sent, got %d", len(store.markedSent))
	}
	if store.markedSent[0] != w1.ID {
		t.Errorf("expected marked ID %s, got %s", w1.ID, store.markedSent[0])
	}
}

// selectiveMarkErrorStore fails MarkMaintenanceNotificationSent for a specific window ID.
type selectiveMarkErrorStore struct {
	*mockStore
	failID uuid.UUID
}

func (s *selectiveMarkErrorStore) MarkMaintenanceNotificationSent(_ context.Context, id uuid.UUID) error {
	if id == s.failID {
		return errors.New("mark failed for specific window")
	}
	s.markedSent = append(s.markedSent, id)
	return nil
}
