package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockSnapshotStore implements SnapshotStore for testing.
type mockSnapshotStore struct {
	user             *models.User
	agents           []*models.Agent
	agent            *models.Agent
	repos            []*models.Repository
	repo             *models.Repository
	backups          map[uuid.UUID][]*models.Backup
	backupBySnapshot map[string]*models.Backup
	schedule         *models.Schedule
	restores         map[uuid.UUID][]*models.Restore
	restore          *models.Restore
	comments         []*models.SnapshotComment
	comment          *models.SnapshotComment
	commentCounts    map[string]int

	getUserErr       error
	getAgentsErr     error
	getAgentErr      error
	getRepoErr       error
	getReposErr      error
	getBackupsErr    error
	getBackupErr     error
	getScheduleErr   error
	createRestoreErr error
	getRestoresErr   error
	getRestoreErr    error
	createCommentErr error
	getCommentsErr   error
	getCommentErr    error
	deleteCommentErr error
	getCountsErr     error
}

func (m *mockSnapshotStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockSnapshotStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if m.getAgentErr != nil {
		return nil, m.getAgentErr
	}
	if m.agent != nil && m.agent.ID == id {
		return m.agent, nil
	}
	for _, a := range m.agents {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errors.New("agent not found")
}

func (m *mockSnapshotStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	if m.getAgentsErr != nil {
		return nil, m.getAgentsErr
	}
	return m.agents, nil
}

func (m *mockSnapshotStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	if m.repo != nil && m.repo.ID == id {
		return m.repo, nil
	}
	for _, r := range m.repos {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, errors.New("repository not found")
}

func (m *mockSnapshotStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Repository, error) {
	if m.getReposErr != nil {
		return nil, m.getReposErr
	}
	return m.repos, nil
}

func (m *mockSnapshotStore) GetRepositoryKeyByRepositoryID(_ context.Context, _ uuid.UUID) (*models.RepositoryKey, error) {
	return nil, errors.New("not implemented in test")
}

func (m *mockSnapshotStore) GetBackupsByAgentID(_ context.Context, agentID uuid.UUID) ([]*models.Backup, error) {
	if m.getBackupsErr != nil {
		return nil, m.getBackupsErr
	}
	if m.backups != nil {
		return m.backups[agentID], nil
	}
	return nil, nil
}

func (m *mockSnapshotStore) GetBackupBySnapshotID(_ context.Context, snapshotID string) (*models.Backup, error) {
	if m.getBackupErr != nil {
		return nil, m.getBackupErr
	}
	if b, ok := m.backupBySnapshot[snapshotID]; ok {
		return b, nil
	}
	return nil, errors.New("backup not found")
}

func (m *mockSnapshotStore) GetScheduleByID(_ context.Context, id uuid.UUID) (*models.Schedule, error) {
	if m.getScheduleErr != nil {
		return nil, m.getScheduleErr
	}
	if m.schedule != nil && m.schedule.ID == id {
		return m.schedule, nil
	}
	return nil, errors.New("schedule not found")
}

func (m *mockSnapshotStore) CreateRestore(_ context.Context, _ *models.Restore) error {
	return m.createRestoreErr
}

func (m *mockSnapshotStore) GetRestoresByAgentID(_ context.Context, agentID uuid.UUID) ([]*models.Restore, error) {
	if m.getRestoresErr != nil {
		return nil, m.getRestoresErr
	}
	if m.restores != nil {
		return m.restores[agentID], nil
	}
	return nil, nil
}

func (m *mockSnapshotStore) GetRestoreByID(_ context.Context, id uuid.UUID) (*models.Restore, error) {
	if m.getRestoreErr != nil {
		return nil, m.getRestoreErr
	}
	if m.restore != nil && m.restore.ID == id {
		return m.restore, nil
	}
	return nil, errors.New("restore not found")
}

func (m *mockSnapshotStore) CreateSnapshotComment(_ context.Context, _ *models.SnapshotComment) error {
	return m.createCommentErr
}

func (m *mockSnapshotStore) GetSnapshotCommentsBySnapshotID(_ context.Context, _ string, _ uuid.UUID) ([]*models.SnapshotComment, error) {
	if m.getCommentsErr != nil {
		return nil, m.getCommentsErr
	}
	return m.comments, nil
}

func (m *mockSnapshotStore) GetSnapshotCommentByID(_ context.Context, id uuid.UUID) (*models.SnapshotComment, error) {
	if m.getCommentErr != nil {
		return nil, m.getCommentErr
	}
	if m.comment != nil && m.comment.ID == id {
		return m.comment, nil
	}
	return nil, errors.New("comment not found")
}

func (m *mockSnapshotStore) DeleteSnapshotComment(_ context.Context, _ uuid.UUID) error {
	return m.deleteCommentErr
}

func (m *mockSnapshotStore) GetSnapshotCommentCounts(_ context.Context, _ []string, _ uuid.UUID) (map[string]int, error) {
	if m.getCountsErr != nil {
		return nil, m.getCountsErr
	}
	return m.commentCounts, nil
}

// setupSnapshotsRouter registers all snapshot/restore routes except CompareSnapshots
// to avoid the gin wildcard param conflict between :id and :id1.
func setupSnapshotsRouter(store SnapshotStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewSnapshotsHandler(store, nil, zerolog.Nop())
	handler := NewSnapshotsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")

	snapshots := api.Group("/snapshots")
	snapshots.GET("", handler.ListSnapshots)
	snapshots.GET("/:id", handler.GetSnapshot)
	snapshots.GET("/:id/files", handler.ListFiles)
	snapshots.GET("/:id/comments", handler.ListSnapshotComments)
	snapshots.POST("/:id/comments", handler.CreateSnapshotComment)

	comments := api.Group("/comments")
	comments.DELETE("/:id", handler.DeleteSnapshotComment)

	restores := api.Group("/restores")
	restores.GET("", handler.ListRestores)
	restores.POST("", handler.CreateRestore)
	restores.GET("/:id", handler.GetRestore)

	return r
}

// setupCompareRouter registers only the CompareSnapshots route
// to avoid the gin wildcard param conflict with :id routes.
func setupCompareRouter(store SnapshotStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewSnapshotsHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")

	snapshots := api.Group("/snapshots")
	snapshots.GET("/compare", handler.CompareSnapshots)
	handler := NewSnapshotsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")

	snapshots := api.Group("/snapshots")
	snapshots.GET("/:id1/compare/:id2", handler.CompareSnapshots)

	return r
}

// Helper to create common test fixtures.
func snapshotTestFixtures() (orgID, agentID, scheduleID, repoID uuid.UUID, userID uuid.UUID) {
	return uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

func ptrInt64(v int64) *int64 {
	return &v
}

// ---------------------------------------------------------------------------
// ListSnapshots
// ---------------------------------------------------------------------------

func TestListSnapshots(t *testing.T) {
	orgID, agentID, scheduleID, repoID, userID := snapshotTestFixtures()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Paths: []string{"/data"}}
	backup := &models.Backup{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		RepositoryID: ptrUUID(repoID),
		SnapshotID:   "abc123def456",
		Status:       models.BackupStatusCompleted,
		StartedAt:    time.Now(),
		SizeBytes:    ptrInt64(1024),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success with snapshots", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:    dbUser,
			agents:  []*models.Agent{agent},
			backups: map[uuid.UUID][]*models.Backup{agentID: {backup}},
			schedule: schedule,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if _, ok := resp["snapshots"]; !ok {
			t.Fatal("expected 'snapshots' key in response")
		}
		var snapshots []SnapshotResponse
		if err := json.Unmarshal(resp["snapshots"], &snapshots); err != nil {
			t.Fatalf("unmarshal snapshots: %v", err)
		}
		if len(snapshots) != 1 {
			t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
		}
		if snapshots[0].ID != "abc123def456" {
			t.Errorf("expected snapshot ID abc123def456, got %s", snapshots[0].ID)
		}
		if snapshots[0].ShortID != "abc123de" {
			t.Errorf("expected short_id abc123de, got %s", snapshots[0].ShortID)
		}
		if snapshots[0].Hostname != "host-1" {
			t.Errorf("expected hostname host-1, got %s", snapshots[0].Hostname)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("skips non-completed backups", func(t *testing.T) {
		runningBackup := &models.Backup{
			ID:         uuid.New(),
			ScheduleID: scheduleID,
			AgentID:    agentID,
			SnapshotID: "",
			Status:     models.BackupStatusRunning,
			StartedAt:  time.Now(),
		}
		store := &mockSnapshotStore{
			user:     dbUser,
			agents:   []*models.Agent{agent},
			backups:  map[uuid.UUID][]*models.Backup{agentID: {runningBackup}},
			schedule: schedule,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		// snapshots should be null or empty since running backup is skipped
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			getUserErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agents error", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:         dbUser,
			getAgentsErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent_id filter", func(t *testing.T) {
		agent2 := &models.Agent{ID: uuid.New(), OrgID: orgID, Hostname: "host-2"}
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent, agent2},
			backups: map[uuid.UUID][]*models.Backup{
				agentID:   {backup},
				agent2.ID: {},
			},
			schedule: schedule,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots?agent_id="+agentID.String()))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var snapshots []SnapshotResponse
		_ = json.Unmarshal(resp["snapshots"], &snapshots)
		if len(snapshots) != 1 {
			t.Fatalf("expected 1 snapshot with agent filter, got %d", len(snapshots))
		}
	})

	t.Run("repository_id filter", func(t *testing.T) {
		otherRepoID := uuid.New()
		backupOtherRepo := &models.Backup{
			ID:           uuid.New(),
			ScheduleID:   scheduleID,
			AgentID:      agentID,
			RepositoryID: ptrUUID(otherRepoID),
			SnapshotID:   "othersnap1234",
			Status:       models.BackupStatusCompleted,
			StartedAt:    time.Now(),
		}
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent},
			backups: map[uuid.UUID][]*models.Backup{
				agentID: {backup, backupOtherRepo},
			},
			schedule: schedule,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots?repository_id="+repoID.String()))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var snapshots []SnapshotResponse
		_ = json.Unmarshal(resp["snapshots"], &snapshots)
		if len(snapshots) != 1 {
			t.Fatalf("expected 1 snapshot with repo filter, got %d", len(snapshots))
		}
		if snapshots[0].RepositoryID != repoID.String() {
			t.Errorf("expected repo ID %s, got %s", repoID.String(), snapshots[0].RepositoryID)
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots?agent_id=not-a-uuid"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid repository_id", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:    dbUser,
			agents:  []*models.Agent{agent},
			backups: map[uuid.UUID][]*models.Backup{agentID: {backup}},
			schedule: schedule,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots?repository_id=not-a-uuid"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GetSnapshot
// ---------------------------------------------------------------------------

func TestGetSnapshot(t *testing.T) {
	orgID, agentID, scheduleID, repoID, userID := snapshotTestFixtures()
	snapshotID := "abc123def456"

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Paths: []string{"/data"}}
	backup := &models.Backup{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		RepositoryID: ptrUUID(repoID),
		SnapshotID:   snapshotID,
		Status:       models.BackupStatusCompleted,
		StartedAt:    time.Now(),
		SizeBytes:    ptrInt64(2048),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			schedule:         schedule,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp SnapshotResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if resp.ID != snapshotID {
			t.Errorf("expected snapshot ID %s, got %s", snapshotID, resp.ID)
		}
		if resp.ShortID != "abc123de" {
			t.Errorf("expected short_id abc123de, got %s", resp.ShortID)
		}
		if resp.Hostname != "host-1" {
			t.Errorf("expected hostname host-1, got %s", resp.Hostname)
		}
		if resp.RepositoryID != repoID.String() {
			t.Errorf("expected repo ID %s, got %s", repoID.String(), resp.RepositoryID)
		}
		if resp.SizeBytes == nil || *resp.SizeBytes != 2048 {
			t.Errorf("expected size_bytes 2048, got %v", resp.SizeBytes)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/nonexistent"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			getUserErr:       errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            wrongOrgAgent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			// no agent set, so GetAgentByID returns error
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("schedule error", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			getScheduleErr:   errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("backup with nil repository_id", func(t *testing.T) {
		backupNoRepo := &models.Backup{
			ID:           uuid.New(),
			ScheduleID:   scheduleID,
			AgentID:      agentID,
			RepositoryID: nil,
			SnapshotID:   snapshotID,
			Status:       models.BackupStatusCompleted,
			StartedAt:    time.Now(),
		}
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backupNoRepo},
			schedule:         schedule,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp SnapshotResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.RepositoryID != "" {
			t.Errorf("expected empty repository_id, got %s", resp.RepositoryID)
		}
	})
}

// ---------------------------------------------------------------------------
// ListFiles
// ---------------------------------------------------------------------------

func TestListFiles(t *testing.T) {
	orgID, agentID, _, _, userID := snapshotTestFixtures()
	scheduleID := uuid.New()
	snapshotID := "snapfiles123456"

	repoID := uuid.New()
	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	backup := &models.Backup{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		RepositoryID: &repoID,
		SnapshotID:   snapshotID,
		Status:       models.BackupStatusCompleted,
		StartedAt:    time.Now(),
	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	backup := &models.Backup{
		ID:         uuid.New(),
		ScheduleID: scheduleID,
		AgentID:    agentID,
		SnapshotID: snapshotID,
		Status:     models.BackupStatusCompleted,
		StartedAt:  time.Now(),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("access verified but repo lookup fails", func(t *testing.T) {
		// ListFiles requires real repository credentials for Restic access.
		// This test verifies auth/access passes but returns 500 when repo is not found.
	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/files"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (repo not in mock), got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("access verified with path filter but repo lookup fails", func(t *testing.T) {
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if resp["snapshot_id"] != snapshotID {
			t.Errorf("expected snapshot_id %s, got %v", snapshotID, resp["snapshot_id"])
		}
	})

	t.Run("success with path filter", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/files?path=/data/subdir"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (repo not in mock), got %d: %s", w.Code, w.Body.String())
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["path"] != "/data/subdir" {
			t.Errorf("expected path /data/subdir, got %v", resp["path"])
		}
	})

	t.Run("snapshot not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/nonexistent/files"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			getUserErr:       errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/files"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            wrongOrgAgent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/files"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/files"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// CreateRestore
// ---------------------------------------------------------------------------

func TestCreateRestore(t *testing.T) {
	orgID, agentID, _, repoID, userID := snapshotTestFixtures()
	snapshotID := "snaprestore1234"

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "my-repo", Type: models.RepositoryTypeS3}
	backup := &models.Backup{
		ID:         uuid.New(),
		AgentID:    agentID,
		SnapshotID: snapshotID,
		Status:     models.BackupStatusCompleted,
		StartedAt:  time.Now(),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	validBody := `{"snapshot_id":"` + snapshotID + `","agent_id":"` + agentID.String() + `","repository_id":"` + repoID.String() + `","target_path":"/restore/target"}`

	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			repo:             repo,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp RestoreResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if resp.SnapshotID != snapshotID {
			t.Errorf("expected snapshot_id %s, got %s", snapshotID, resp.SnapshotID)
		}
		if resp.AgentID != agentID.String() {
			t.Errorf("expected agent_id %s, got %s", agentID.String(), resp.AgentID)
		}
		if resp.RepositoryID != repoID.String() {
			t.Errorf("expected repository_id %s, got %s", repoID.String(), resp.RepositoryID)
		}
		if resp.TargetPath != "/restore/target" {
			t.Errorf("expected target_path /restore/target, got %s", resp.TargetPath)
		}
		if resp.Status != "pending" {
			t.Errorf("expected status pending, got %s", resp.Status)
		}
	})

	t.Run("success with include and exclude paths", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			repo:             repo,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		body := `{"snapshot_id":"` + snapshotID + `","agent_id":"` + agentID.String() + `","repository_id":"` + repoID.String() + `","target_path":"/restore","include_paths":["/data/important"],"exclude_paths":["/data/tmp"]}`
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", body))

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp RestoreResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.IncludePaths) != 1 || resp.IncludePaths[0] != "/data/important" {
			t.Errorf("expected include_paths [/data/important], got %v", resp.IncludePaths)
		}
		if len(resp.ExcludePaths) != 1 || resp.ExcludePaths[0] != "/data/tmp" {
			t.Errorf("expected exclude_paths [/data/tmp], got %v", resp.ExcludePaths)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", `{}`))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", `not json`))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		store := &mockSnapshotStore{}
		body := `{"snapshot_id":"snap1","agent_id":"not-a-uuid","repository_id":"` + repoID.String() + `","target_path":"/restore"}`
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", body))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid repository_id", func(t *testing.T) {
		store := &mockSnapshotStore{}
		body := `{"snapshot_id":"snap1","agent_id":"` + agentID.String() + `","repository_id":"not-a-uuid","target_path":"/restore"}`
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", body))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			getUserErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			user: dbUser,
			// agent not set, so GetAgentByID returns error
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockSnapshotStore{
			user:  dbUser,
			agent: wrongOrgAgent,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:       dbUser,
			agent:      agent,
			getRepoErr: errors.New("not found"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo wrong org", func(t *testing.T) {
		wrongOrgRepo := &models.Repository{ID: repoID, OrgID: uuid.New(), Name: "other-repo"}
		store := &mockSnapshotStore{
			user:  dbUser,
			agent: agent,
			repo:  wrongOrgRepo,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("snapshot not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			repo:             repo,
			backupBySnapshot: map[string]*models.Backup{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			repo:             repo,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			createRestoreErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/restores", validBody))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// ListRestores
// ---------------------------------------------------------------------------

func TestListRestores(t *testing.T) {
	orgID, agentID, _, repoID, userID := snapshotTestFixtures()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	restore := &models.Restore{
		ID:           uuid.New(),
		AgentID:      agentID,
		RepositoryID: repoID,
		SnapshotID:   "snaprestore1234",
		TargetPath:   "/restore/target",
		Status:       models.RestoreStatusPending,
		CreatedAt:    time.Now(),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent},
			restores: map[uuid.UUID][]*models.Restore{
				agentID: {restore},
			},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if _, ok := resp["restores"]; !ok {
			t.Fatal("expected 'restores' key in response")
		}
		var restores []RestoreResponse
		_ = json.Unmarshal(resp["restores"], &restores)
		if len(restores) != 1 {
			t.Fatalf("expected 1 restore, got %d", len(restores))
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			getUserErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agents error", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:         dbUser,
			getAgentsErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent_id filter", func(t *testing.T) {
		agent2 := &models.Agent{ID: uuid.New(), OrgID: orgID, Hostname: "host-2"}
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent, agent2},
			restores: map[uuid.UUID][]*models.Restore{
				agentID:   {restore},
				agent2.ID: {},
			},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores?agent_id="+agentID.String()))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var restores []RestoreResponse
		_ = json.Unmarshal(resp["restores"], &restores)
		if len(restores) != 1 {
			t.Fatalf("expected 1 restore with agent filter, got %d", len(restores))
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores?agent_id=bad"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("status filter", func(t *testing.T) {
		completedRestore := &models.Restore{
			ID:           uuid.New(),
			AgentID:      agentID,
			RepositoryID: repoID,
			SnapshotID:   "snapcomplete",
			TargetPath:   "/restore/done",
			Status:       models.RestoreStatusCompleted,
			CreatedAt:    time.Now(),
		}
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent},
			restores: map[uuid.UUID][]*models.Restore{
				agentID: {restore, completedRestore},
			},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores?status=completed"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var restores []RestoreResponse
		_ = json.Unmarshal(resp["restores"], &restores)
		if len(restores) != 1 {
			t.Fatalf("expected 1 completed restore, got %d", len(restores))
		}
		if restores[0].Status != "completed" {
			t.Errorf("expected status completed, got %s", restores[0].Status)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GetRestore
// ---------------------------------------------------------------------------

func TestGetRestore(t *testing.T) {
	orgID, agentID, _, repoID, userID := snapshotTestFixtures()
	restoreID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	now := time.Now()
	startedAt := now.Add(-5 * time.Minute)
	completedAt := now
	restore := &models.Restore{
		ID:           restoreID,
		AgentID:      agentID,
		RepositoryID: repoID,
		SnapshotID:   "snaprestore1234",
		TargetPath:   "/restore/target",
		Status:       models.RestoreStatusCompleted,
		StartedAt:    &startedAt,
		CompletedAt:  &completedAt,
		CreatedAt:    now.Add(-10 * time.Minute),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:    dbUser,
			agent:   agent,
			restore: restore,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores/"+restoreID.String()))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp RestoreResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if resp.ID != restoreID.String() {
			t.Errorf("expected restore ID %s, got %s", restoreID.String(), resp.ID)
		}
		if resp.Status != "completed" {
			t.Errorf("expected status completed, got %s", resp.Status)
		}
		if resp.StartedAt == "" {
			t.Error("expected started_at to be set")
		}
		if resp.CompletedAt == "" {
			t.Error("expected completed_at to be set")
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores/not-a-uuid"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			// restore not set
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores/"+uuid.New().String()))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			restore:    restore,
			getUserErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores/"+restoreID.String()))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockSnapshotStore{
			user:    dbUser,
			agent:   wrongOrgAgent,
			restore: restore,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores/"+restoreID.String()))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("agent not found for restore", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:    dbUser,
			restore: restore,
			// no agent set
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores/"+restoreID.String()))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/restores/"+restoreID.String()))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// ListSnapshotComments
// ---------------------------------------------------------------------------

func TestListSnapshotComments(t *testing.T) {
	orgID, agentID, _, _, userID := snapshotTestFixtures()
	scheduleID := uuid.New()
	snapshotID := "snapcomments123"

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	backup := &models.Backup{
		ID:         uuid.New(),
		ScheduleID: scheduleID,
		AgentID:    agentID,
		SnapshotID: snapshotID,
		Status:     models.BackupStatusCompleted,
		StartedAt:  time.Now(),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Name: "Test User", Email: "test@example.com", Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	comment := &models.SnapshotComment{
		ID:         uuid.New(),
		OrgID:      orgID,
		SnapshotID: snapshotID,
		UserID:     userID,
		Content:    "This backup looks good",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			comments:         []*models.SnapshotComment{comment},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/comments"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if _, ok := resp["comments"]; !ok {
			t.Fatal("expected 'comments' key in response")
		}
		var comments []SnapshotCommentResponse
		_ = json.Unmarshal(resp["comments"], &comments)
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(comments))
		}
		if comments[0].Content != "This backup looks good" {
			t.Errorf("expected comment content 'This backup looks good', got %s", comments[0].Content)
		}
		if comments[0].UserName != "Test User" {
			t.Errorf("expected user_name 'Test User', got %s", comments[0].UserName)
		}
		if comments[0].UserEmail != "test@example.com" {
			t.Errorf("expected user_email 'test@example.com', got %s", comments[0].UserEmail)
		}
	})

	t.Run("empty comments", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			comments:         []*models.SnapshotComment{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/comments"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("snapshot not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/nonexistent/comments"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			getUserErr:       errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/comments"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            wrongOrgAgent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/comments"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("comments error", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			getCommentsErr:   errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/comments"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID+"/comments"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// CreateSnapshotComment
// ---------------------------------------------------------------------------

func TestCreateSnapshotComment(t *testing.T) {
	orgID, agentID, _, _, userID := snapshotTestFixtures()
	scheduleID := uuid.New()
	snapshotID := "snapcomment1234"

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	backup := &models.Backup{
		ID:         uuid.New(),
		ScheduleID: scheduleID,
		AgentID:    agentID,
		SnapshotID: snapshotID,
		Status:     models.BackupStatusCompleted,
		StartedAt:  time.Now(),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Name: "Test User", Email: "test@example.com", Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		body := `{"content":"Great backup!"}`
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/"+snapshotID+"/comments", body))

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp SnapshotCommentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if resp.Content != "Great backup!" {
			t.Errorf("expected content 'Great backup!', got %s", resp.Content)
		}
		if resp.SnapshotID != snapshotID {
			t.Errorf("expected snapshot_id %s, got %s", snapshotID, resp.SnapshotID)
		}
		if resp.UserID != userID.String() {
			t.Errorf("expected user_id %s, got %s", userID.String(), resp.UserID)
		}
		if resp.UserName != "Test User" {
			t.Errorf("expected user_name 'Test User', got %s", resp.UserName)
		}
	})

	t.Run("invalid body empty", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/"+snapshotID+"/comments", `{}`))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/"+snapshotID+"/comments", `not json`))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("snapshot not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/nonexistent/comments", `{"content":"test"}`))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			getUserErr:       errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/"+snapshotID+"/comments", `{"content":"test"}`))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            wrongOrgAgent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/"+snapshotID+"/comments", `{"content":"test"}`))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			agent:            agent,
			backupBySnapshot: map[string]*models.Backup{snapshotID: backup},
			createCommentErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/"+snapshotID+"/comments", `{"content":"test"}`))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/"+snapshotID+"/comments", `{"content":"test"}`))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteSnapshotComment
// ---------------------------------------------------------------------------

func TestDeleteSnapshotComment(t *testing.T) {
	orgID, _, _, _, userID := snapshotTestFixtures()
	commentID := uuid.New()

	comment := &models.SnapshotComment{
		ID:         commentID,
		OrgID:      orgID,
		SnapshotID: "snapdelete12345",
		UserID:     userID,
		Content:    "To be deleted",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Name: "Test User", Email: "test@example.com", Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success owner", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:    dbUser,
			comment: comment,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["message"] != "comment deleted" {
			t.Errorf("expected message 'comment deleted', got %s", resp["message"])
		}
	})

	t.Run("success admin deletes other user comment", func(t *testing.T) {
		otherUserID := uuid.New()
		otherComment := &models.SnapshotComment{
			ID:         commentID,
			OrgID:      orgID,
			SnapshotID: "snapdelete12345",
			UserID:     otherUserID, // different user
			Content:    "Other user's comment",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		adminUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
		store := &mockSnapshotStore{
			user:    adminUser,
			comment: otherComment,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/not-a-uuid"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			// comment not set
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+uuid.New().String()))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			comment:    comment,
			getUserErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongOrgComment := &models.SnapshotComment{
			ID:         commentID,
			OrgID:      uuid.New(), // different org
			SnapshotID: "snap123",
			UserID:     userID,
			Content:    "wrong org",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		store := &mockSnapshotStore{
			user:    dbUser,
			comment: wrongOrgComment,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("not owner non-admin forbidden", func(t *testing.T) {
		otherUserID := uuid.New()
		otherComment := &models.SnapshotComment{
			ID:         commentID,
			OrgID:      orgID,
			SnapshotID: "snap123",
			UserID:     otherUserID, // different user
			Content:    "not mine",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		nonAdminUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleUser} // not admin
		store := &mockSnapshotStore{
			user:    nonAdminUser,
			comment: otherComment,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("viewer non-admin forbidden", func(t *testing.T) {
		otherUserID := uuid.New()
		otherComment := &models.SnapshotComment{
			ID:         commentID,
			OrgID:      orgID,
			SnapshotID: "snap123",
			UserID:     otherUserID,
			Content:    "not mine",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		viewerUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleViewer}
		store := &mockSnapshotStore{
			user:    viewerUser,
			comment: otherComment,
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("delete error", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:             dbUser,
			comment:          comment,
			deleteCommentErr: errors.New("db error"),
		}
		r := setupSnapshotsRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupSnapshotsRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/comments/"+commentID.String()))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// CompareSnapshots
// ---------------------------------------------------------------------------

func TestCompareSnapshots(t *testing.T) {
	orgID, agentID, scheduleID, repoID, userID := snapshotTestFixtures()
	snapshotID1 := "snapcompare1111"
	snapshotID2 := "snapcompare2222"

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host-1"}
	schedule := &models.Schedule{
		ID:      scheduleID,
		AgentID: agentID,
		Paths:   []string{"/data"},
		Repositories: []models.ScheduleRepository{
			{RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}

	backup1 := &models.Backup{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		RepositoryID: &repoID,
		SnapshotID:   snapshotID1,
		Status:       models.BackupStatusCompleted,
		StartedAt:    time.Now().Add(-1 * time.Hour),
		SizeBytes:    ptrInt64(1024),
	}
	backup2 := &models.Backup{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		RepositoryID: &repoID,
		SnapshotID:   snapshotID2,
		Status:       models.BackupStatusCompleted,
		StartedAt:    time.Now(),
		SizeBytes:    ptrInt64(2048),
		ID:         uuid.New(),
		ScheduleID: scheduleID,
		AgentID:    agentID,
		SnapshotID: snapshotID1,
		Status:     models.BackupStatusCompleted,
		StartedAt:  time.Now().Add(-1 * time.Hour),
		SizeBytes:  ptrInt64(1024),
	}
	backup2 := &models.Backup{
		ID:         uuid.New(),
		ScheduleID: scheduleID,
		AgentID:    agentID,
		SnapshotID: snapshotID2,
		Status:     models.BackupStatusCompleted,
		StartedAt:  time.Now(),
		SizeBytes:  ptrInt64(2048),
	}

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	sessionUser := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("access verified but repo lookup fails", func(t *testing.T) {
		// CompareSnapshots requires real repository credentials for Restic diff.
		// This test verifies auth/access passes but returns 500 when repo credentials are unavailable.
	t.Run("success", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:  dbUser,
			agent: agent,
			backupBySnapshot: map[string]*models.Backup{
				snapshotID1: backup1,
				snapshotID2: backup2,
			},
			schedule: schedule,
		}
		r := setupCompareRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1="+snapshotID1+"&id2="+snapshotID2))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (repo not in mock), got %d: %s", w.Code, w.Body.String())
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID1+"/compare/"+snapshotID2))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp SnapshotCompareResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if resp.SnapshotID1 != snapshotID1 {
			t.Errorf("expected snapshot_id_1 %s, got %s", snapshotID1, resp.SnapshotID1)
		}
		if resp.SnapshotID2 != snapshotID2 {
			t.Errorf("expected snapshot_id_2 %s, got %s", snapshotID2, resp.SnapshotID2)
		}
		if resp.Snapshot1 == nil {
			t.Fatal("expected snapshot_1 to be set")
		}
		if resp.Snapshot2 == nil {
			t.Fatal("expected snapshot_2 to be set")
		}
		if resp.Snapshot1.Hostname != "host-1" {
			t.Errorf("expected snapshot_1 hostname host-1, got %s", resp.Snapshot1.Hostname)
		}
		if resp.Snapshot1.RepositoryID != repoID.String() {
			t.Errorf("expected snapshot_1 repo ID %s, got %s", repoID.String(), resp.Snapshot1.RepositoryID)
		}
		if len(resp.Changes) != 0 {
			t.Errorf("expected empty changes (placeholder), got %d", len(resp.Changes))
		}
	})

	t.Run("first snapshot not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			user: dbUser,
			backupBySnapshot: map[string]*models.Backup{
				snapshotID2: backup2,
			},
		}
		r := setupCompareRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1=nonexistent&id2="+snapshotID2))
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/nonexistent/compare/"+snapshotID2))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("second snapshot not found", func(t *testing.T) {
		store := &mockSnapshotStore{
			user:  dbUser,
			agent: agent,
			backupBySnapshot: map[string]*models.Backup{
				snapshotID1: backup1,
			},
		}
		r := setupCompareRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1="+snapshotID1+"&id2=nonexistent"))
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID1+"/compare/nonexistent"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		store := &mockSnapshotStore{
			getUserErr: errors.New("db error"),
		}
		r := setupCompareRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1="+snapshotID1+"&id2="+snapshotID2))
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID1+"/compare/"+snapshotID2))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("first agent wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockSnapshotStore{
			user:  dbUser,
			agent: wrongOrgAgent,
			backupBySnapshot: map[string]*models.Backup{
				snapshotID1: backup1,
				snapshotID2: backup2,
			},
		}
		r := setupCompareRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1="+snapshotID1+"&id2="+snapshotID2))
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID1+"/compare/"+snapshotID2))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("second agent wrong org", func(t *testing.T) {
		// First agent is in the right org, second agent is in the wrong org.
		// Use two distinct agent IDs.
		agent1ID := uuid.New()
		agent2ID := uuid.New()
		agent1 := &models.Agent{ID: agent1ID, OrgID: orgID, Hostname: "host-1"}
		agent2 := &models.Agent{ID: agent2ID, OrgID: uuid.New(), Hostname: "other-host"}

		b1 := &models.Backup{
			ID:         uuid.New(),
			ScheduleID: scheduleID,
			AgentID:    agent1ID,
			SnapshotID: snapshotID1,
			Status:     models.BackupStatusCompleted,
			StartedAt:  time.Now(),
		}
		b2 := &models.Backup{
			ID:         uuid.New(),
			ScheduleID: scheduleID,
			AgentID:    agent2ID,
			SnapshotID: snapshotID2,
			Status:     models.BackupStatusCompleted,
			StartedAt:  time.Now(),
		}

		store := &mockSnapshotStore{
			user:   dbUser,
			agents: []*models.Agent{agent1, agent2},
			backupBySnapshot: map[string]*models.Backup{
				snapshotID1: b1,
				snapshotID2: b2,
			},
			schedule: schedule,
		}
		r := setupCompareRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1="+snapshotID1+"&id2="+snapshotID2))
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID1+"/compare/"+snapshotID2))

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("schedule error still reaches restic call", func(t *testing.T) {
		// CompareSnapshots logs schedule errors but does not fail on them.
		// It will still fail on the restic diff call since repo credentials are unavailable.
	t.Run("schedule error for first snapshot still succeeds", func(t *testing.T) {
		// CompareSnapshots logs schedule errors but does not fail
		store := &mockSnapshotStore{
			user:  dbUser,
			agent: agent,
			backupBySnapshot: map[string]*models.Backup{
				snapshotID1: backup1,
				snapshotID2: backup2,
			},
			getScheduleErr: errors.New("schedule db error"),
		}
		r := setupCompareRouter(store, sessionUser)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1="+snapshotID1+"&id2="+snapshotID2))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (repo not in mock), got %d: %s", w.Code, w.Body.String())
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID1+"/compare/"+snapshotID2))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp SnapshotCompareResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		// Paths should be nil when schedule fails
		if resp.Snapshot1 != nil && resp.Snapshot1.Paths != nil {
			t.Errorf("expected nil paths when schedule error, got %v", resp.Snapshot1.Paths)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSnapshotStore{}
		r := setupCompareRouter(store, nil)
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1="+snapshotID1+"&id2="+snapshotID2))
		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/snapshots/"+snapshotID1+"/compare/"+snapshotID2))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// toRestoreResponse
// ---------------------------------------------------------------------------

func TestToRestoreResponse(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-5 * time.Minute)
	completedAt := now

	t.Run("with started and completed times", func(t *testing.T) {
		restore := &models.Restore{
			ID:           uuid.New(),
			AgentID:      uuid.New(),
			RepositoryID: uuid.New(),
			SnapshotID:   "snap123",
			TargetPath:   "/restore",
			IncludePaths: []string{"/data"},
			ExcludePaths: []string{"/tmp"},
			Status:       models.RestoreStatusCompleted,
			StartedAt:    &startedAt,
			CompletedAt:  &completedAt,
			ErrorMessage: "",
			CreatedAt:    now.Add(-10 * time.Minute),
		}
		resp := toRestoreResponse(restore)
		if resp.StartedAt == "" {
			t.Error("expected started_at to be set")
		}
		if resp.CompletedAt == "" {
			t.Error("expected completed_at to be set")
		}
		if resp.Status != "completed" {
			t.Errorf("expected status completed, got %s", resp.Status)
		}
	})

	t.Run("without started and completed times", func(t *testing.T) {
		restore := &models.Restore{
			ID:           uuid.New(),
			AgentID:      uuid.New(),
			RepositoryID: uuid.New(),
			SnapshotID:   "snap456",
			TargetPath:   "/restore",
			Status:       models.RestoreStatusPending,
			CreatedAt:    now,
		}
		resp := toRestoreResponse(restore)
		if resp.StartedAt != "" {
			t.Errorf("expected empty started_at, got %s", resp.StartedAt)
		}
		if resp.CompletedAt != "" {
			t.Errorf("expected empty completed_at, got %s", resp.CompletedAt)
		}
	})

	t.Run("with error message", func(t *testing.T) {
		restore := &models.Restore{
			ID:           uuid.New(),
			AgentID:      uuid.New(),
			RepositoryID: uuid.New(),
			SnapshotID:   "snap789",
			TargetPath:   "/restore",
			Status:       models.RestoreStatusFailed,
			ErrorMessage: "disk full",
			CreatedAt:    now,
		}
		resp := toRestoreResponse(restore)
		if resp.ErrorMessage != "disk full" {
			t.Errorf("expected error_message 'disk full', got %s", resp.ErrorMessage)
		}
	})
}

// ---------------------------------------------------------------------------
// toSnapshotCommentResponse
// ---------------------------------------------------------------------------

func TestToSnapshotCommentResponse(t *testing.T) {
	t.Run("with user", func(t *testing.T) {
		comment := &models.SnapshotComment{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			SnapshotID: "snap123",
			UserID:     uuid.New(),
			Content:    "test comment",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		user := &models.User{
			ID:    comment.UserID,
			Name:  "Jane Doe",
			Email: "jane@example.com",
		}
		resp := toSnapshotCommentResponse(comment, user)
		if resp.UserName != "Jane Doe" {
			t.Errorf("expected user_name 'Jane Doe', got %s", resp.UserName)
		}
		if resp.UserEmail != "jane@example.com" {
			t.Errorf("expected user_email 'jane@example.com', got %s", resp.UserEmail)
		}
	})

	t.Run("without user nil", func(t *testing.T) {
		comment := &models.SnapshotComment{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			SnapshotID: "snap456",
			UserID:     uuid.New(),
			Content:    "orphan comment",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		resp := toSnapshotCommentResponse(comment, nil)
		if resp.UserName != "" {
			t.Errorf("expected empty user_name, got %s", resp.UserName)
		}
		if resp.UserEmail != "" {
			t.Errorf("expected empty user_email, got %s", resp.UserEmail)
		}
	})
}

// ---------------------------------------------------------------------------
// RegisterRoutes
// ---------------------------------------------------------------------------

func TestSnapshotsRegisterRoutes(t *testing.T) {
	store := &mockSnapshotStore{}
	sessionUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}

	// Test main routes (excluding compare to avoid gin wildcard conflict)
	r := setupSnapshotsRouter(store, sessionUser)

	mainRoutes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/snapshots"},
		{"GET", "/api/v1/snapshots/test-id"},
		{"GET", "/api/v1/snapshots/test-id/files"},
		{"GET", "/api/v1/snapshots/test-id/comments"},
		{"POST", "/api/v1/snapshots/test-id/comments"},
		{"DELETE", "/api/v1/comments/" + uuid.New().String()},
		{"GET", "/api/v1/restores"},
		{"POST", "/api/v1/restores"},
		{"GET", "/api/v1/restores/" + uuid.New().String()},
	}

	for _, rt := range mainRoutes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			var req *http.Request
			if rt.method == "POST" || rt.method == "DELETE" {
				req = JSONRequest(rt.method, rt.path, `{}`)
			} else {
				req = AuthenticatedRequest(rt.method, rt.path)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			// The route should not return 405 Method Not Allowed
			if w.Code == http.StatusMethodNotAllowed {
				t.Errorf("route %s %s returned 405 Method Not Allowed", rt.method, rt.path)
			}
		})
	}

	// Test compare route on a separate router to avoid wildcard conflict
	t.Run("GET compare route", func(t *testing.T) {
		rc := setupCompareRouter(store, sessionUser)
		req := AuthenticatedRequest("GET", "/api/v1/snapshots/compare?id1=id1&id2=id2")
		req := AuthenticatedRequest("GET", "/api/v1/snapshots/id1/compare/id2")
		w := httptest.NewRecorder()
		rc.ServeHTTP(w, req)
		if w.Code == http.StatusMethodNotAllowed {
			t.Error("compare route returned 405 Method Not Allowed")
		}
	})
}
