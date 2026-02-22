//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/handlers"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// dockerAvailable returns true if a Docker daemon is reachable.
func dockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// setupTestDB creates a PostgreSQL testcontainer, runs migrations, and returns a connected DB.
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()

	if !dockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("keldris_integration"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	logger := zerolog.New(zerolog.NewTestWriter(t))
	cfg := db.DefaultConfig(connStr)
	cfg.MaxConns = 5
	cfg.MinConns = 1

	database, err := db.New(ctx, cfg, logger)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	err = database.Migrate(ctx)
	require.NoError(t, err)

	return database
}

// testLogger returns a zerolog.Logger that writes to the test output.
func testLogger(t *testing.T) zerolog.Logger {
	t.Helper()
	return zerolog.New(zerolog.NewTestWriter(t))
}

// createTestOrg creates and persists a test organization.
func createTestOrg(t *testing.T, database *db.DB, name, slug string) *models.Organization {
	t.Helper()
	org := models.NewOrganization(name, slug)
	err := database.CreateOrganization(context.Background(), org)
	require.NoError(t, err)
	return org
}

// createTestUser creates and persists a test user with a membership.
func createTestUser(t *testing.T, database *db.DB, orgID uuid.UUID, email, name string, role models.OrgRole) *models.User {
	t.Helper()
	user := models.NewUser(orgID, "oidc-"+uuid.New().String(), email, name, models.UserRoleAdmin)
	err := database.CreateUser(context.Background(), user)
	require.NoError(t, err)
	membership := models.NewOrgMembership(user.ID, orgID, role)
	err = database.CreateMembership(context.Background(), membership)
	require.NoError(t, err)
	return user
}

// createTestAgent creates and persists a test agent.
func createTestAgent(t *testing.T, database *db.DB, orgID uuid.UUID, hostname string) *models.Agent {
	t.Helper()
	agent := models.NewAgent(orgID, hostname, "hash-"+uuid.New().String())
	err := database.CreateAgent(context.Background(), agent)
	require.NoError(t, err)
	return agent
}

// createTestRepo creates and persists a test repository.
func createTestRepo(t *testing.T, database *db.DB, orgID uuid.UUID, name string) *models.Repository {
	t.Helper()
	repo := models.NewRepository(orgID, name, models.RepositoryTypeLocal, []byte("encrypted-config"))
	err := database.CreateRepository(context.Background(), repo)
	require.NoError(t, err)
	return repo
}

// createTestSchedule creates and persists a test schedule linked to a repository.
func createTestSchedule(t *testing.T, database *db.DB, agentID uuid.UUID, repoID uuid.UUID, name string) *models.Schedule {
	t.Helper()
	schedule := models.NewSchedule(agentID, name, "0 2 * * *", []string{"/data"})
	err := database.CreateSchedule(context.Background(), schedule)
	require.NoError(t, err)

	sr := models.NewScheduleRepository(schedule.ID, repoID, 0)
	err = database.CreateScheduleRepository(context.Background(), sr)
	require.NoError(t, err)

	return schedule
}

// setupTestRouter creates a Gin engine in test mode with a user injected into context.
func setupTestRouter(user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	return r
}

// sessionUser creates an auth.SessionUser from a models.User for test context injection.
func sessionUser(user *models.User, orgID uuid.UUID, role string) *auth.SessionUser {
	return &auth.SessionUser{
		ID:              user.ID,
		OIDCSubject:     user.OIDCSubject,
		Email:           user.Email,
		Name:            user.Name,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    orgID,
		CurrentOrgRole:  role,
	}
}

// TestFullBackupWorkflow tests the complete backup lifecycle:
// create org -> user -> agent -> repository -> schedule -> backup -> verify -> restore -> verify.
func TestFullBackupWorkflow(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	// Step 1: Create organization
	org := createTestOrg(t, database, "Backup Test Org", "backup-test-"+uuid.New().String()[:8])
	require.NotEqual(t, uuid.Nil, org.ID)

	// Step 2: Create user with owner role
	user := createTestUser(t, database, org.ID, "admin@backuptest.com", "Admin User", models.OrgRoleOwner)
	require.NotEqual(t, uuid.Nil, user.ID)

	// Step 3: Create agent
	agent := createTestAgent(t, database, org.ID, "backup-server-01")
	require.NotEqual(t, uuid.Nil, agent.ID)
	assert.Equal(t, models.AgentStatusPending, agent.Status)

	// Activate the agent
	agent.MarkSeen()
	err := database.UpdateAgent(ctx, agent)
	require.NoError(t, err)

	// Verify agent is active
	updatedAgent, err := database.GetAgentByID(ctx, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentStatusActive, updatedAgent.Status)
	assert.NotNil(t, updatedAgent.LastSeen)

	// Step 4: Create repository
	repo := createTestRepo(t, database, org.ID, "local-backup-store")

	// Step 5: Create schedule with repository
	schedule := createTestSchedule(t, database, agent.ID, repo.ID, "Daily Backup")
	assert.True(t, schedule.Enabled)
	assert.Equal(t, "0 2 * * *", schedule.CronExpression)

	// Verify schedule-repository association
	scheduleRepos, err := database.GetScheduleRepositories(ctx, schedule.ID)
	require.NoError(t, err)
	require.Len(t, scheduleRepos, 1)
	assert.Equal(t, repo.ID, scheduleRepos[0].RepositoryID)
	assert.Equal(t, 0, scheduleRepos[0].Priority) // primary

	// Step 6: Run backup (simulate)
	backup := models.NewBackup(schedule.ID, agent.ID, &repo.ID)
	err = database.CreateBackup(ctx, backup)
	require.NoError(t, err)
	assert.Equal(t, models.BackupStatusRunning, backup.Status)

	// Verify backup exists and is running
	fetchedBackup, err := database.GetBackupByID(ctx, backup.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BackupStatusRunning, fetchedBackup.Status)
	assert.Equal(t, schedule.ID, fetchedBackup.ScheduleID)
	assert.Equal(t, agent.ID, fetchedBackup.AgentID)

	// Complete the backup
	backup.Complete("snap-abc123", 1024*1024*100, 42, 7)
	err = database.UpdateBackup(ctx, backup)
	require.NoError(t, err)

	// Verify backup completed
	completedBackup, err := database.GetBackupByID(ctx, backup.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BackupStatusCompleted, completedBackup.Status)
	assert.Equal(t, "snap-abc123", completedBackup.SnapshotID)
	assert.NotNil(t, completedBackup.CompletedAt)
	assert.NotNil(t, completedBackup.SizeBytes)
	assert.Equal(t, int64(1024*1024*100), *completedBackup.SizeBytes)
	assert.NotNil(t, completedBackup.FilesNew)
	assert.Equal(t, 42, *completedBackup.FilesNew)

	// Verify backup shows up in agent backups
	agentBackups, err := database.GetBackupsByAgentID(ctx, agent.ID)
	require.NoError(t, err)
	require.Len(t, agentBackups, 1)
	assert.Equal(t, backup.ID, agentBackups[0].ID)

	// Step 7: Run restore
	restore := models.NewRestore(agent.ID, repo.ID, "snap-abc123", "/restore/target", nil, nil)
	err = database.CreateRestore(ctx, restore)
	require.NoError(t, err)
	assert.Equal(t, models.RestoreStatusPending, restore.Status)

	// Start restore
	restore.Start()
	err = database.UpdateRestore(ctx, restore)
	require.NoError(t, err)

	// Verify restore is running
	runningRestore, err := database.GetRestoreByID(ctx, restore.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RestoreStatusRunning, runningRestore.Status)
	assert.NotNil(t, runningRestore.StartedAt)

	// Complete restore
	restore.Complete()
	err = database.UpdateRestore(ctx, restore)
	require.NoError(t, err)

	// Verify restore completed
	completedRestore, err := database.GetRestoreByID(ctx, restore.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RestoreStatusCompleted, completedRestore.Status)
	assert.NotNil(t, completedRestore.CompletedAt)
	assert.True(t, completedRestore.IsComplete())
	assert.Empty(t, completedRestore.ErrorMessage)

	// Verify restore shows up in agent restores
	agentRestores, err := database.GetRestoresByAgentID(ctx, agent.ID)
	require.NoError(t, err)
	require.Len(t, agentRestores, 1)
	assert.Equal(t, restore.ID, agentRestores[0].ID)
}

// TestAuthWorkflow tests session-based auth: login flow, session management, and logout.
// Since OIDC requires an external provider, we test the session layer and API handler auth
// directly, simulating the post-OIDC callback state.
func TestAuthWorkflow(t *testing.T) {
	database := setupTestDB(t)
	logger := testLogger(t)

	// Create session store
	secret := []byte("test-secret-that-is-at-least-32-bytes-long!")
	sessionCfg := auth.DefaultSessionConfig(secret, false, 0, 0)
	sessions, err := auth.NewSessionStore(sessionCfg, logger)
	require.NoError(t, err)

	// Create test org and user
	org := createTestOrg(t, database, "Auth Test Org", "auth-test-"+uuid.New().String()[:8])
	user := createTestUser(t, database, org.ID, "authuser@example.com", "Auth User", models.OrgRoleOwner)

	t.Run("SessionSetAndGetUser", func(t *testing.T) {
		// Simulate setting user in session (post-OIDC callback)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		su := &auth.SessionUser{
			ID:              user.ID,
			OIDCSubject:     user.OIDCSubject,
			Email:           user.Email,
			Name:            user.Name,
			AuthenticatedAt: time.Now(),
			CurrentOrgID:    org.ID,
			CurrentOrgRole:  string(models.OrgRoleOwner),
		}

		err := sessions.SetUser(req, w, su)
		require.NoError(t, err)

		// Extract the session cookie from response
		cookies := w.Result().Cookies()
		require.NotEmpty(t, cookies)

		// Create new request with session cookie and retrieve user
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}

		gotUser, err := sessions.GetUser(req2)
		require.NoError(t, err)
		assert.Equal(t, user.ID, gotUser.ID)
		assert.Equal(t, user.Email, gotUser.Email)
		assert.Equal(t, user.Name, gotUser.Name)
		assert.Equal(t, org.ID, gotUser.CurrentOrgID)
		assert.Equal(t, string(models.OrgRoleOwner), gotUser.CurrentOrgRole)
	})

	t.Run("SessionAuthentication", func(t *testing.T) {
		// Unauthenticated request
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.False(t, sessions.IsAuthenticated(req))

		// Authenticated request
		w := httptest.NewRecorder()
		su := &auth.SessionUser{
			ID:              user.ID,
			OIDCSubject:     user.OIDCSubject,
			Email:           user.Email,
			Name:            user.Name,
			AuthenticatedAt: time.Now(),
			CurrentOrgID:    org.ID,
			CurrentOrgRole:  string(models.OrgRoleOwner),
		}
		err := sessions.SetUser(req, w, su)
		require.NoError(t, err)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, c := range w.Result().Cookies() {
			req2.AddCookie(c)
		}
		assert.True(t, sessions.IsAuthenticated(req2))
	})

	t.Run("SessionLogout", func(t *testing.T) {
		// Set up authenticated session
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		su := &auth.SessionUser{
			ID:              user.ID,
			OIDCSubject:     user.OIDCSubject,
			Email:           user.Email,
			Name:            user.Name,
			AuthenticatedAt: time.Now(),
			CurrentOrgID:    org.ID,
			CurrentOrgRole:  string(models.OrgRoleOwner),
		}
		err := sessions.SetUser(req, w, su)
		require.NoError(t, err)

		cookies := w.Result().Cookies()

		// Clear user (logout)
		req2 := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()
		err = sessions.ClearUser(req2, w2)
		require.NoError(t, err)

		// Verify session is cleared with logout cookies
		req3 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, c := range w2.Result().Cookies() {
			req3.AddCookie(c)
		}
		assert.False(t, sessions.IsAuthenticated(req3))
	})

	t.Run("MeEndpointAuthenticated", func(t *testing.T) {
		su := sessionUser(user, org.ID, string(models.OrgRoleOwner))
		router := setupTestRouter(su)

		authHandler := handlers.NewAuthHandler(nil, sessions, database, logger)
		authGroup := router.Group("/auth")
		authHandler.RegisterRoutes(authGroup)

		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		// Set up session cookie
		w := httptest.NewRecorder()
		err := sessions.SetUser(req, w, su)
		require.NoError(t, err)

		req2 := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		for _, c := range w.Result().Cookies() {
			req2.AddCookie(c)
		}

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var meResp handlers.MeResponse
		err = json.Unmarshal(w2.Body.Bytes(), &meResp)
		require.NoError(t, err)
		assert.Equal(t, user.ID, meResp.ID)
		assert.Equal(t, user.Email, meResp.Email)
	})

	t.Run("AuthMiddlewareRejectsUnauthenticated", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.AuthMiddleware(sessions, logger))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("AuthMiddlewareAcceptsAuthenticated", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.AuthMiddleware(sessions, logger))
		router.GET("/protected", func(c *gin.Context) {
			u := middleware.GetUser(c)
			require.NotNil(t, u)
			c.JSON(http.StatusOK, gin.H{"user_id": u.ID.String()})
		})

		// Create authenticated request
		su := &auth.SessionUser{
			ID:              user.ID,
			OIDCSubject:     user.OIDCSubject,
			Email:           user.Email,
			Name:            user.Name,
			AuthenticatedAt: time.Now(),
			CurrentOrgID:    org.ID,
			CurrentOrgRole:  string(models.OrgRoleOwner),
		}
		setupReq := httptest.NewRequest(http.MethodGet, "/", nil)
		setupW := httptest.NewRecorder()
		err := sessions.SetUser(setupReq, setupW, su)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		for _, c := range setupW.Result().Cookies() {
			req.AddCookie(c)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestNotificationWorkflow tests notification channel creation, preference configuration,
// and notification logging when a backup failure triggers a notification.
func TestNotificationWorkflow(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	// Set up org, user, agent, repo, schedule
	org := createTestOrg(t, database, "Notification Test Org", "notif-test-"+uuid.New().String()[:8])
	createTestUser(t, database, org.ID, "notify@example.com", "Notify User", models.OrgRoleOwner)
	agent := createTestAgent(t, database, org.ID, "notif-agent-01")
	repo := createTestRepo(t, database, org.ID, "notif-repo")
	schedule := createTestSchedule(t, database, agent.ID, repo.ID, "Notified Backup")

	// Step 1: Create notification channel (webhook)
	channel := models.NewNotificationChannel(org.ID, "Ops Webhook", models.ChannelTypeWebhook, []byte(`{"url":"https://hooks.example.com/backup","secret":"s3cr3t"}`))
	err := database.CreateNotificationChannel(ctx, channel)
	require.NoError(t, err)

	// Verify channel was created
	channels, err := database.GetNotificationChannelsByOrgID(ctx, org.ID)
	require.NoError(t, err)
	require.Len(t, channels, 1)
	assert.Equal(t, "Ops Webhook", channels[0].Name)
	assert.Equal(t, models.ChannelTypeWebhook, channels[0].Type)
	assert.True(t, channels[0].Enabled)

	// Step 2: Create notification preference for backup failures
	pref := models.NewNotificationPreference(org.ID, channel.ID, models.EventBackupFailed)
	err = database.CreateNotificationPreference(ctx, pref)
	require.NoError(t, err)

	// Verify preference exists
	prefs, err := database.GetNotificationPreferencesByOrgID(ctx, org.ID)
	require.NoError(t, err)
	require.Len(t, prefs, 1)
	assert.Equal(t, models.EventBackupFailed, prefs[0].EventType)
	assert.True(t, prefs[0].Enabled)

	// Also create a preference for backup success to verify filtering
	successPref := models.NewNotificationPreference(org.ID, channel.ID, models.EventBackupSuccess)
	err = database.CreateNotificationPreference(ctx, successPref)
	require.NoError(t, err)

	// Step 3: Simulate backup failure
	backup := models.NewBackup(schedule.ID, agent.ID, &repo.ID)
	err = database.CreateBackup(ctx, backup)
	require.NoError(t, err)

	backup.Fail("disk full: /dev/sda1 has no space left")
	err = database.UpdateBackup(ctx, backup)
	require.NoError(t, err)

	// Verify backup is failed
	failedBackup, err := database.GetBackupByID(ctx, backup.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BackupStatusFailed, failedBackup.Status)
	assert.Equal(t, "disk full: /dev/sda1 has no space left", failedBackup.ErrorMessage)

	// Step 4: Check enabled preferences for the backup_failed event
	enabledPrefs, err := database.GetEnabledPreferencesForEvent(ctx, org.ID, models.EventBackupFailed)
	require.NoError(t, err)
	require.Len(t, enabledPrefs, 1)
	assert.Equal(t, channel.ID, enabledPrefs[0].ChannelID)

	// Step 5: Create notification log (simulating the notification service sending)
	notifLog := models.NewNotificationLog(org.ID, &channel.ID, string(models.EventBackupFailed), "https://hooks.example.com/backup", "Backup Failed: Notified Backup")
	notifLog.MarkSent()
	err = database.CreateNotificationLog(ctx, notifLog)
	require.NoError(t, err)

	// Verify notification log
	logs, err := database.GetNotificationLogsByOrgID(ctx, org.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, models.NotificationStatusSent, logs[0].Status)
	assert.Equal(t, string(models.EventBackupFailed), logs[0].EventType)
	assert.NotNil(t, logs[0].SentAt)
	assert.NotNil(t, logs[0].ChannelID)
	assert.Equal(t, channel.ID, *logs[0].ChannelID)

	// Step 6: Verify disabling a channel stops notifications
	channel.Enabled = false
	err = database.UpdateNotificationChannel(ctx, channel)
	require.NoError(t, err)

	enabledPrefsAfterDisable, err := database.GetEnabledPreferencesForEvent(ctx, org.ID, models.EventBackupFailed)
	require.NoError(t, err)
	assert.Empty(t, enabledPrefsAfterDisable, "disabled channel should not return enabled preferences")
}

// TestMultiOrgIsolation verifies that data is isolated between organizations
// and that RBAC enforcement prevents unauthorized cross-org access.
func TestMultiOrgIsolation(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	// Create two separate organizations
	orgA := createTestOrg(t, database, "Organization Alpha", "org-alpha-"+uuid.New().String()[:8])
	orgB := createTestOrg(t, database, "Organization Beta", "org-beta-"+uuid.New().String()[:8])

	// Create users in each org
	userA := createTestUser(t, database, orgA.ID, "alice@alpha.com", "Alice", models.OrgRoleOwner)
	userB := createTestUser(t, database, orgB.ID, "bob@beta.com", "Bob", models.OrgRoleOwner)

	// Create agents in each org
	agentA := createTestAgent(t, database, orgA.ID, "alpha-server-01")
	agentB := createTestAgent(t, database, orgB.ID, "beta-server-01")

	// Create repositories in each org
	repoA := createTestRepo(t, database, orgA.ID, "alpha-repo")
	repoB := createTestRepo(t, database, orgB.ID, "beta-repo")

	// Create schedules and backups in each org
	scheduleA := createTestSchedule(t, database, agentA.ID, repoA.ID, "Alpha Daily")
	scheduleB := createTestSchedule(t, database, agentB.ID, repoB.ID, "Beta Daily")

	backupA := models.NewBackup(scheduleA.ID, agentA.ID, &repoA.ID)
	backupA.Complete("snap-alpha-001", 5000, 10, 2)
	err := database.CreateBackup(ctx, backupA)
	require.NoError(t, err)

	backupB := models.NewBackup(scheduleB.ID, agentB.ID, &repoB.ID)
	backupB.Complete("snap-beta-001", 8000, 15, 3)
	err = database.CreateBackup(ctx, backupB)
	require.NoError(t, err)

	// Create notification channels in each org
	channelA := models.NewNotificationChannel(orgA.ID, "Alpha Slack", models.ChannelTypeSlack, []byte(`{"webhook_url":"https://hooks.slack.com/alpha"}`))
	err = database.CreateNotificationChannel(ctx, channelA)
	require.NoError(t, err)

	channelB := models.NewNotificationChannel(orgB.ID, "Beta Slack", models.ChannelTypeSlack, []byte(`{"webhook_url":"https://hooks.slack.com/beta"}`))
	err = database.CreateNotificationChannel(ctx, channelB)
	require.NoError(t, err)

	t.Run("AgentIsolation", func(t *testing.T) {
		// Org A should only see its agents
		agentsA, err := database.GetAgentsByOrgID(ctx, orgA.ID)
		require.NoError(t, err)
		require.Len(t, agentsA, 1)
		assert.Equal(t, agentA.ID, agentsA[0].ID)
		assert.Equal(t, "alpha-server-01", agentsA[0].Hostname)

		// Org B should only see its agents
		agentsB, err := database.GetAgentsByOrgID(ctx, orgB.ID)
		require.NoError(t, err)
		require.Len(t, agentsB, 1)
		assert.Equal(t, agentB.ID, agentsB[0].ID)
		assert.Equal(t, "beta-server-01", agentsB[0].Hostname)
	})

	t.Run("UserIsolation", func(t *testing.T) {
		// Org A should only see its users
		usersA, err := database.ListUsers(ctx, orgA.ID)
		require.NoError(t, err)
		require.Len(t, usersA, 1)
		assert.Equal(t, userA.ID, usersA[0].ID)

		// Org B should only see its users
		usersB, err := database.ListUsers(ctx, orgB.ID)
		require.NoError(t, err)
		require.Len(t, usersB, 1)
		assert.Equal(t, userB.ID, usersB[0].ID)
	})

	t.Run("BackupIsolation", func(t *testing.T) {
		// Backups by agent should be isolated (agent belongs to specific org)
		backupsA, err := database.GetBackupsByAgentID(ctx, agentA.ID)
		require.NoError(t, err)
		require.Len(t, backupsA, 1)
		assert.Equal(t, backupA.ID, backupsA[0].ID)

		backupsB, err := database.GetBackupsByAgentID(ctx, agentB.ID)
		require.NoError(t, err)
		require.Len(t, backupsB, 1)
		assert.Equal(t, backupB.ID, backupsB[0].ID)

		// Schedule-based queries should also be isolated
		scheduleBackupsA, err := database.GetBackupsByScheduleID(ctx, scheduleA.ID)
		require.NoError(t, err)
		require.Len(t, scheduleBackupsA, 1)
		assert.Equal(t, backupA.ID, scheduleBackupsA[0].ID)
	})

	t.Run("NotificationChannelIsolation", func(t *testing.T) {
		// Org A should only see its channels
		channelsA, err := database.GetNotificationChannelsByOrgID(ctx, orgA.ID)
		require.NoError(t, err)
		require.Len(t, channelsA, 1)
		assert.Equal(t, channelA.ID, channelsA[0].ID)

		// Org B should only see its channels
		channelsB, err := database.GetNotificationChannelsByOrgID(ctx, orgB.ID)
		require.NoError(t, err)
		require.Len(t, channelsB, 1)
		assert.Equal(t, channelB.ID, channelsB[0].ID)
	})

	t.Run("ScheduleIsolation", func(t *testing.T) {
		// Schedules are queried via agent which belongs to a specific org
		schedulesA, err := database.GetSchedulesByAgentID(ctx, agentA.ID)
		require.NoError(t, err)
		require.Len(t, schedulesA, 1)
		assert.Equal(t, scheduleA.ID, schedulesA[0].ID)

		schedulesB, err := database.GetSchedulesByAgentID(ctx, agentB.ID)
		require.NoError(t, err)
		require.Len(t, schedulesB, 1)
		assert.Equal(t, scheduleB.ID, schedulesB[0].ID)
	})

	t.Run("RepositoryIsolation", func(t *testing.T) {
		// Repositories should be isolated by org
		reposA, err := database.GetRepositoriesByOrgID(ctx, orgA.ID)
		require.NoError(t, err)
		require.Len(t, reposA, 1)
		assert.Equal(t, repoA.ID, reposA[0].ID)

		reposB, err := database.GetRepositoriesByOrgID(ctx, orgB.ID)
		require.NoError(t, err)
		require.Len(t, reposB, 1)
		assert.Equal(t, repoB.ID, reposB[0].ID)
	})

	t.Run("CrossOrgAccessPrevention", func(t *testing.T) {
		// Verify that directly accessing another org's resource by ID returns the correct data
		// (i.e., the DB stores it, but handler-level org_id checks would prevent access)
		directAgent, err := database.GetAgentByID(ctx, agentA.ID)
		require.NoError(t, err)
		assert.Equal(t, orgA.ID, directAgent.OrgID, "agent's org_id should match org A")

		directAgent2, err := database.GetAgentByID(ctx, agentB.ID)
		require.NoError(t, err)
		assert.Equal(t, orgB.ID, directAgent2.OrgID, "agent's org_id should match org B")
	})

	t.Run("RBACEnforcement", func(t *testing.T) {
		logger := testLogger(t)

		// Create session store
		secret := []byte("test-secret-that-is-at-least-32-bytes-long!")
		sessionCfg := auth.DefaultSessionConfig(secret, false, 0, 0)
		sessions, err := auth.NewSessionStore(sessionCfg, logger)
		require.NoError(t, err)

		// Create a member (non-admin) user in org A
		memberUser := createTestUser(t, database, orgA.ID, "member@alpha.com", "Member User", models.OrgRoleMember)

		t.Run("AdminCanAccessOrgSettings", func(t *testing.T) {
			su := sessionUser(userA, orgA.ID, string(models.OrgRoleOwner))
			router := setupTestRouter(su)
			router.Use(middleware.AuthMiddleware(sessions, logger))
			router.GET("/api/v1/org", func(c *gin.Context) {
				u := middleware.GetUser(c)
				if u == nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"org_id": u.CurrentOrgID, "role": u.CurrentOrgRole})
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/org", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("MemberHasLimitedAccess", func(t *testing.T) {
			su := sessionUser(memberUser, orgA.ID, string(models.OrgRoleMember))
			router := setupTestRouter(su)
			router.GET("/api/v1/org/settings", func(c *gin.Context) {
				u := middleware.GetUser(c)
				if u == nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					return
				}
				// Simulate RBAC check: only owner/admin can modify org settings
				if u.CurrentOrgRole != string(models.OrgRoleOwner) && u.CurrentOrgRole != string(models.OrgRoleAdmin) {
					c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"settings": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/org/settings", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		})

		t.Run("UserCannotAccessOtherOrg", func(t *testing.T) {
			// User A tries to access Org B's data
			su := sessionUser(userA, orgA.ID, string(models.OrgRoleOwner))
			router := setupTestRouter(su)
			router.GET("/api/v1/agents", func(c *gin.Context) {
				u := middleware.GetUser(c)
				if u == nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					return
				}
				// Handler enforces org isolation by using session org_id
				agents, err := database.GetAgentsByOrgID(ctx, u.CurrentOrgID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
					return
				}
				c.JSON(http.StatusOK, agents)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var agents []*models.Agent
			err := json.Unmarshal(w.Body.Bytes(), &agents)
			require.NoError(t, err)
			require.Len(t, agents, 1)
			// User A only sees org A's agent, not org B's
			assert.Equal(t, "alpha-server-01", agents[0].Hostname)
		})
	})
}

// TestBackupFailureWorkflow tests the failure path: a backup that fails and produces correct state.
func TestBackupFailureWorkflow(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	org := createTestOrg(t, database, "Failure Test Org", "fail-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, database, org.ID, "fail-agent-01")
	repo := createTestRepo(t, database, org.ID, "fail-repo")
	schedule := createTestSchedule(t, database, agent.ID, repo.ID, "Failing Backup")

	// Start backup
	backup := models.NewBackup(schedule.ID, agent.ID, &repo.ID)
	err := database.CreateBackup(ctx, backup)
	require.NoError(t, err)
	assert.Equal(t, models.BackupStatusRunning, backup.Status)

	// Fail backup
	backup.Fail("connection timeout: unable to reach storage backend")
	err = database.UpdateBackup(ctx, backup)
	require.NoError(t, err)

	// Verify failure state
	failedBackup, err := database.GetBackupByID(ctx, backup.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BackupStatusFailed, failedBackup.Status)
	assert.Equal(t, "connection timeout: unable to reach storage backend", failedBackup.ErrorMessage)
	assert.NotNil(t, failedBackup.CompletedAt)
	assert.True(t, failedBackup.IsComplete())
	assert.Nil(t, failedBackup.SizeBytes)
	assert.Nil(t, failedBackup.FilesNew)
}

// TestRestoreFailureWorkflow tests the restore failure path.
func TestRestoreFailureWorkflow(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	org := createTestOrg(t, database, "Restore Fail Org", "restore-fail-"+uuid.New().String()[:8])
	agent := createTestAgent(t, database, org.ID, "restore-fail-agent")
	repo := createTestRepo(t, database, org.ID, "restore-fail-repo")

	// Create restore
	restore := models.NewRestore(agent.ID, repo.ID, "snap-404", "/restore/target", []string{"/data/important"}, []string{"/data/cache"})
	err := database.CreateRestore(ctx, restore)
	require.NoError(t, err)

	// Start restore
	restore.Start()
	err = database.UpdateRestore(ctx, restore)
	require.NoError(t, err)

	// Fail restore
	restore.Fail("snapshot not found: snap-404")
	err = database.UpdateRestore(ctx, restore)
	require.NoError(t, err)

	// Verify failure state
	failedRestore, err := database.GetRestoreByID(ctx, restore.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RestoreStatusFailed, failedRestore.Status)
	assert.Equal(t, "snapshot not found: snap-404", failedRestore.ErrorMessage)
	assert.NotNil(t, failedRestore.CompletedAt)
	assert.True(t, failedRestore.IsComplete())
	assert.Equal(t, []string{"/data/important"}, failedRestore.IncludePaths)
	assert.Equal(t, []string{"/data/cache"}, failedRestore.ExcludePaths)
}

// TestScheduleManagementWorkflow tests creating, updating, and deleting schedules.
func TestScheduleManagementWorkflow(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	org := createTestOrg(t, database, "Schedule Test Org", "sched-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, database, org.ID, "sched-agent-01")
	repo := createTestRepo(t, database, org.ID, "sched-repo")

	// Create schedule
	schedule := createTestSchedule(t, database, agent.ID, repo.ID, "Hourly Backup")

	// Verify schedule was created
	fetched, err := database.GetScheduleByID(ctx, schedule.ID)
	require.NoError(t, err)
	assert.Equal(t, "Hourly Backup", fetched.Name)
	assert.Equal(t, "0 2 * * *", fetched.CronExpression)
	assert.True(t, fetched.Enabled)
	assert.Equal(t, []string{"/data"}, fetched.Paths)

	// Update schedule
	fetched.Name = "Updated Hourly Backup"
	fetched.CronExpression = "0 * * * *"
	fetched.Enabled = false
	err = database.UpdateSchedule(ctx, fetched)
	require.NoError(t, err)

	// Verify update
	updated, err := database.GetScheduleByID(ctx, schedule.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Hourly Backup", updated.Name)
	assert.Equal(t, "0 * * * *", updated.CronExpression)
	assert.False(t, updated.Enabled)

	// Delete schedule
	err = database.DeleteSchedule(ctx, schedule.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = database.GetScheduleByID(ctx, schedule.ID)
	assert.Error(t, err, "schedule should not be found after deletion")
}
