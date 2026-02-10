package db

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	if !dockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("keldris_test"),
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
	cfg := DefaultConfig(connStr)
	cfg.MaxConns = 5
	cfg.MinConns = 1

	database, err := New(ctx, cfg, logger)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	err = database.Migrate(ctx)
	require.NoError(t, err)

	return database
}

// createTestOrg creates and persists a test organization.
func createTestOrg(t *testing.T, db *DB, name, slug string) *models.Organization {
	t.Helper()
	org := models.NewOrganization(name, slug)
	err := db.CreateOrganization(context.Background(), org)
	require.NoError(t, err)
	return org
}

// createTestUser creates and persists a test user with a membership.
func createTestUser(t *testing.T, db *DB, orgID uuid.UUID, email, name string) *models.User {
	t.Helper()
	user := models.NewUser(orgID, "oidc-"+uuid.New().String(), email, name, models.UserRoleAdmin)
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)
	// Create membership so DeleteUser owner checks work
	membership := models.NewOrgMembership(user.ID, orgID, models.OrgRoleOwner)
	err = db.CreateMembership(context.Background(), membership)
	require.NoError(t, err)
	return user
}

// createTestAgent creates and persists a test agent.
func createTestAgent(t *testing.T, db *DB, orgID uuid.UUID, hostname string) *models.Agent {
	t.Helper()
	agent := models.NewAgent(orgID, hostname, "hash-"+uuid.New().String())
	err := db.CreateAgent(context.Background(), agent)
	require.NoError(t, err)
	return agent
}

// createTestRepo creates and persists a test repository.
func createTestRepo(t *testing.T, db *DB, orgID uuid.UUID, name string) *models.Repository {
	t.Helper()
	repo := models.NewRepository(orgID, name, models.RepositoryTypeLocal, []byte("encrypted-config"))
	err := db.CreateRepository(context.Background(), repo)
	require.NoError(t, err)
	return repo
}

func TestStore_Organizations(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	t.Run("CreateAndGet", func(t *testing.T) {
		org := models.NewOrganization("Test Org", "test-org")
		err := db.CreateOrganization(ctx, org)
		require.NoError(t, err)

		got, err := db.GetOrganizationByID(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, got.ID)
		assert.Equal(t, "Test Org", got.Name)
		assert.Equal(t, "test-org", got.Slug)
	})

	t.Run("GetBySlug", func(t *testing.T) {
		org := createTestOrg(t, db, "Slug Org", "slug-org-"+uuid.New().String()[:8])

		got, err := db.GetOrganizationBySlug(ctx, org.Slug)
		require.NoError(t, err)
		assert.Equal(t, org.ID, got.ID)
	})

	t.Run("List", func(t *testing.T) {
		orgs, err := db.GetAllOrganizations(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(orgs), 1)
	})

	t.Run("Update", func(t *testing.T) {
		org := createTestOrg(t, db, "Old Name", "update-org-"+uuid.New().String()[:8])
		org.Name = "New Name"
		err := db.UpdateOrganization(ctx, org)
		require.NoError(t, err)

		got, err := db.GetOrganizationByID(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Name", got.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		org := createTestOrg(t, db, "Delete Me", "delete-org-"+uuid.New().String()[:8])
		err := db.DeleteOrganization(ctx, org.ID)
		require.NoError(t, err)

		_, err = db.GetOrganizationByID(ctx, org.ID)
		assert.Error(t, err)
	})

	t.Run("GetOrCreateDefaultOrg", func(t *testing.T) {
		org1, err := db.GetOrCreateDefaultOrg(ctx)
		require.NoError(t, err)
		assert.Equal(t, "default", org1.Slug)

		// Second call should return the same org
		org2, err := db.GetOrCreateDefaultOrg(ctx)
		require.NoError(t, err)
		assert.Equal(t, org1.ID, org2.ID)
	})

	t.Run("DuplicateSlug", func(t *testing.T) {
		slug := "dup-slug-" + uuid.New().String()[:8]
		_ = createTestOrg(t, db, "Org 1", slug)

		org2 := models.NewOrganization("Org 2", slug)
		err := db.CreateOrganization(ctx, org2)
		assert.Error(t, err) // unique constraint violation
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetOrganizationByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_Users(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "User Test Org", "user-test-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		user := models.NewUser(org.ID, "oidc-sub-1", "user1@test.com", "User One", models.UserRoleAdmin)
		err := db.CreateUser(ctx, user)
		require.NoError(t, err)

		got, err := db.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
		assert.Equal(t, "user1@test.com", got.Email)
		assert.Equal(t, models.UserRoleAdmin, got.Role)
	})

	t.Run("GetByOIDCSubject", func(t *testing.T) {
		subject := "oidc-sub-" + uuid.New().String()[:8]
		user := models.NewUser(org.ID, subject, "oidc@test.com", "OIDC User", models.UserRoleUser)
		err := db.CreateUser(ctx, user)
		require.NoError(t, err)

		got, err := db.GetUserByOIDCSubject(ctx, subject)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
	})

	t.Run("List", func(t *testing.T) {
		users, err := db.ListUsers(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 1)
	})

	t.Run("Update", func(t *testing.T) {
		user := models.NewUser(org.ID, "oidc-update-"+uuid.New().String()[:8], "update@test.com", "Old Name", models.UserRoleUser)
		err := db.CreateUser(ctx, user)
		require.NoError(t, err)

		user.Name = "New Name"
		user.Role = models.UserRoleAdmin
		err = db.UpdateUser(ctx, user)
		require.NoError(t, err)

		got, err := db.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Name", got.Name)
	})

	t.Run("DeleteUser", func(t *testing.T) {
		// Create a user that is NOT an owner, so deletion works
		user := models.NewUser(org.ID, "oidc-del-"+uuid.New().String()[:8], "delete@test.com", "Delete Me", models.UserRoleUser)
		err := db.CreateUser(ctx, user)
		require.NoError(t, err)

		err = db.DeleteUser(ctx, user.ID)
		require.NoError(t, err)

		_, err = db.GetUserByID(ctx, user.ID)
		assert.Error(t, err)
	})

	t.Run("DeleteLastOwner", func(t *testing.T) {
		// Create org with a single owner
		ownerOrg := createTestOrg(t, db, "Owner Org", "owner-org-"+uuid.New().String()[:8])
		owner := models.NewUser(ownerOrg.ID, "oidc-owner-"+uuid.New().String()[:8], "owner@test.com", "Owner", models.UserRoleAdmin)
		err := db.CreateUser(ctx, owner)
		require.NoError(t, err)
		membership := models.NewOrgMembership(owner.ID, ownerOrg.ID, models.OrgRoleOwner)
		err = db.CreateMembership(ctx, membership)
		require.NoError(t, err)

		err = db.DeleteUser(ctx, owner.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "last owner")
	})

	t.Run("DuplicateOIDCSubject", func(t *testing.T) {
		subject := "oidc-dup-" + uuid.New().String()[:8]
		user1 := models.NewUser(org.ID, subject, "dup1@test.com", "User 1", models.UserRoleUser)
		err := db.CreateUser(ctx, user1)
		require.NoError(t, err)

		user2 := models.NewUser(org.ID, subject, "dup2@test.com", "User 2", models.UserRoleUser)
		err = db.CreateUser(ctx, user2)
		assert.Error(t, err) // unique constraint
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetUserByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_Agents(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Agent Test Org", "agent-test-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		agent := models.NewAgent(org.ID, "server-01", "api-key-hash")
		err := db.CreateAgent(ctx, agent)
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, got.ID)
		assert.Equal(t, "server-01", got.Hostname)
		assert.Equal(t, models.AgentStatusPending, got.Status)
	})

	t.Run("GetByAPIKeyHash", func(t *testing.T) {
		hash := "unique-hash-" + uuid.New().String()[:8]
		agent := models.NewAgent(org.ID, "hash-server", hash)
		err := db.CreateAgent(ctx, agent)
		require.NoError(t, err)

		got, err := db.GetAgentByAPIKeyHash(ctx, hash)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, got.ID)
	})

	t.Run("ListByOrgID", func(t *testing.T) {
		agents, err := db.GetAgentsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(agents), 1)
	})

	t.Run("Update", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "update-server")
		agent.Hostname = "updated-server"
		agent.Status = models.AgentStatusActive
		now := time.Now()
		agent.LastSeen = &now
		err := db.UpdateAgent(ctx, agent)
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-server", got.Hostname)
		assert.Equal(t, models.AgentStatusActive, got.Status)
	})

	t.Run("UpdateWithHealth", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "health-server")
		agent.HealthStatus = models.HealthStatusHealthy
		agent.HealthMetrics = &models.HealthMetrics{
			CPUUsage:    25.5,
			MemoryUsage: 40.2,
			DiskUsage:   60.0,
		}
		now := time.Now()
		agent.HealthCheckedAt = &now
		err := db.UpdateAgent(ctx, agent)
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, models.HealthStatusHealthy, got.HealthStatus)
		require.NotNil(t, got.HealthMetrics)
		assert.InDelta(t, 25.5, got.HealthMetrics.CPUUsage, 0.01)
	})

	t.Run("UpdateAPIKeyHash", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "rekey-server")
		err := db.UpdateAgentAPIKeyHash(ctx, agent.ID, "new-hash-value")
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, "new-hash-value", got.APIKeyHash)
	})

	t.Run("UpdateAPIKeyHash_NotFound", func(t *testing.T) {
		err := db.UpdateAgentAPIKeyHash(ctx, uuid.New(), "hash")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found")
	})

	t.Run("RevokeAPIKey", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "revoke-server")
		err := db.RevokeAgentAPIKey(ctx, agent.ID)
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, "", got.APIKeyHash)
		assert.Equal(t, models.AgentStatusPending, got.Status)
	})

	t.Run("RevokeAPIKey_NotFound", func(t *testing.T) {
		err := db.RevokeAgentAPIKey(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found")
	})

	t.Run("Delete", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "delete-server")
		err := db.DeleteAgent(ctx, agent.ID)
		require.NoError(t, err)

		_, err = db.GetAgentByID(ctx, agent.ID)
		assert.Error(t, err)
	})

	t.Run("GetOrgIDByAgentID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "orgid-server")
		gotOrgID, err := db.GetOrgIDByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, gotOrgID)
	})

	t.Run("HealthHistory", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "history-server")

		metrics := &models.HealthMetrics{CPUUsage: 50.0, MemoryUsage: 70.0, DiskUsage: 30.0}
		history := models.NewAgentHealthHistory(agent.ID, org.ID, models.HealthStatusWarning, metrics, nil)
		err := db.CreateAgentHealthHistory(ctx, history)
		require.NoError(t, err)

		records, err := db.GetAgentHealthHistory(ctx, agent.ID, 10)
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, models.HealthStatusWarning, records[0].HealthStatus)
	})

	t.Run("FleetHealthSummary", func(t *testing.T) {
		summary, err := db.GetFleetHealthSummary(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, summary.TotalAgents, 1)
	})

	t.Run("AgentStats", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "stats-server")
		stats, err := db.GetAgentStats(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, stats.AgentID)
		assert.Equal(t, 0, stats.TotalBackups)
	})

	t.Run("ForeignKeyViolation", func(t *testing.T) {
		agent := models.NewAgent(uuid.New(), "fk-server", "fk-hash")
		err := db.CreateAgent(ctx, agent)
		assert.Error(t, err) // org_id FK violation
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetAgentByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_Repositories(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Repo Test Org", "repo-test-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		repo := models.NewRepository(org.ID, "My S3 Repo", models.RepositoryTypeS3, []byte("encrypted"))
		err := db.CreateRepository(ctx, repo)
		require.NoError(t, err)

		got, err := db.GetRepositoryByID(ctx, repo.ID)
		require.NoError(t, err)
		assert.Equal(t, "My S3 Repo", got.Name)
		assert.Equal(t, models.RepositoryTypeS3, got.Type)
		assert.Equal(t, []byte("encrypted"), got.ConfigEncrypted)
	})

	t.Run("GetRepositoryAlias", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "alias-repo")
		got, err := db.GetRepository(ctx, repo.ID)
		require.NoError(t, err)
		assert.Equal(t, repo.ID, got.ID)
	})

	t.Run("ListByOrgID", func(t *testing.T) {
		repos, err := db.GetRepositoriesByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(repos), 1)
	})

	t.Run("Update", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "old-repo-name")
		repo.Name = "new-repo-name"
		repo.ConfigEncrypted = []byte("new-encrypted")
		err := db.UpdateRepository(ctx, repo)
		require.NoError(t, err)

		got, err := db.GetRepositoryByID(ctx, repo.ID)
		require.NoError(t, err)
		assert.Equal(t, "new-repo-name", got.Name)
		assert.Equal(t, []byte("new-encrypted"), got.ConfigEncrypted)
	})

	t.Run("Delete", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "delete-repo")
		err := db.DeleteRepository(ctx, repo.ID)
		require.NoError(t, err)

		_, err = db.GetRepositoryByID(ctx, repo.ID)
		assert.Error(t, err)
	})

	t.Run("RepositoryKey_CRUD", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "key-repo")

		rk := models.NewRepositoryKey(repo.ID, []byte("encrypted-key"), false, nil)
		err := db.CreateRepositoryKey(ctx, rk)
		require.NoError(t, err)

		got, err := db.GetRepositoryKeyByRepositoryID(ctx, repo.ID)
		require.NoError(t, err)
		assert.Equal(t, []byte("encrypted-key"), got.EncryptedKey)
		assert.False(t, got.EscrowEnabled)

		// Enable escrow
		err = db.UpdateRepositoryKeyEscrow(ctx, repo.ID, true, []byte("escrow-key"))
		require.NoError(t, err)

		got, err = db.GetRepositoryKeyByRepositoryID(ctx, repo.ID)
		require.NoError(t, err)
		assert.True(t, got.EscrowEnabled)
		assert.Equal(t, []byte("escrow-key"), got.EscrowEncryptedKey)

		// List escrow keys
		keys, err := db.GetRepositoryKeysWithEscrowByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 1)

		// Delete
		err = db.DeleteRepositoryKey(ctx, repo.ID)
		require.NoError(t, err)

		_, err = db.GetRepositoryKeyByRepositoryID(ctx, repo.ID)
		assert.Error(t, err)
	})

	t.Run("ForeignKeyViolation", func(t *testing.T) {
		repo := models.NewRepository(uuid.New(), "FK Repo", models.RepositoryTypeLocal, []byte("cfg"))
		err := db.CreateRepository(ctx, repo)
		assert.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetRepositoryByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_Schedules(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Schedule Test Org", "sched-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "sched-agent")
	repo := createTestRepo(t, db, org.ID, "sched-repo")

	t.Run("CreateAndGetByID", func(t *testing.T) {
		sched := models.NewSchedule(agent.ID, "Daily Backup", "0 2 * * *", []string{"/data"})
		sched.Excludes = []string{"*.tmp"}
		sched.RetentionPolicy = models.DefaultRetentionPolicy()
		sched.Repositories = []models.ScheduleRepository{
			{RepositoryID: repo.ID, Priority: 0, Enabled: true},
		}
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Daily Backup", got.Name)
		assert.Equal(t, "0 2 * * *", got.CronExpression)
		assert.Equal(t, []string{"/data"}, got.Paths)
		assert.Equal(t, []string{"*.tmp"}, got.Excludes)
		require.NotNil(t, got.RetentionPolicy)
		assert.Equal(t, 5, got.RetentionPolicy.KeepLast)
		require.Len(t, got.Repositories, 1)
		assert.Equal(t, repo.ID, got.Repositories[0].RepositoryID)
	})

	t.Run("ListByAgentID", func(t *testing.T) {
		schedules, err := db.GetSchedulesByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(schedules), 1)
	})

	t.Run("Update", func(t *testing.T) {
		sched := models.NewSchedule(agent.ID, "Update Sched", "0 3 * * *", []string{"/home"})
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		sched.Name = "Updated Sched"
		sched.CronExpression = "0 4 * * *"
		sched.Enabled = false
		err = db.UpdateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Sched", got.Name)
		assert.Equal(t, "0 4 * * *", got.CronExpression)
		assert.False(t, got.Enabled)
	})

	t.Run("Delete", func(t *testing.T) {
		sched := models.NewSchedule(agent.ID, "Delete Sched", "0 5 * * *", []string{"/tmp"})
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		err = db.DeleteSchedule(ctx, sched.ID)
		require.NoError(t, err)

		_, err = db.GetScheduleByID(ctx, sched.ID)
		assert.Error(t, err)
	})

	t.Run("MultiRepo", func(t *testing.T) {
		repo2 := createTestRepo(t, db, org.ID, "sched-repo-2")
		sched := models.NewSchedule(agent.ID, "Multi Repo Sched", "0 1 * * *", []string{"/data"})
		sched.Repositories = []models.ScheduleRepository{
			{RepositoryID: repo.ID, Priority: 0, Enabled: true},
			{RepositoryID: repo2.ID, Priority: 1, Enabled: true},
		}
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		require.Len(t, got.Repositories, 2)
		assert.Equal(t, 0, got.Repositories[0].Priority)
		assert.Equal(t, 1, got.Repositories[1].Priority)
	})

	t.Run("SetScheduleRepositories", func(t *testing.T) {
		sched := models.NewSchedule(agent.ID, "SetRepos Sched", "0 6 * * *", []string{"/var"})
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		repo2 := createTestRepo(t, db, org.ID, "set-repos-repo")
		err = db.SetScheduleRepositories(ctx, sched.ID, []models.ScheduleRepository{
			{RepositoryID: repo2.ID, Priority: 0, Enabled: true},
		})
		require.NoError(t, err)

		repos, err := db.GetScheduleRepositories(ctx, sched.ID)
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, repo2.ID, repos[0].RepositoryID)
	})

	t.Run("GetOrgIDByScheduleID", func(t *testing.T) {
		sched := models.NewSchedule(agent.ID, "OrgID Sched", "0 7 * * *", []string{"/"})
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		gotOrgID, err := db.GetOrgIDByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, gotOrgID)
	})

	t.Run("GetEnabledSchedules", func(t *testing.T) {
		schedules, err := db.GetEnabledSchedules(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(schedules), 1)
	})

	t.Run("WithBandwidthAndWindow", func(t *testing.T) {
		bwLimit := 1024
		sched := models.NewSchedule(agent.ID, "BW Sched", "0 8 * * *", []string{"/opt"})
		sched.BandwidthLimitKB = &bwLimit
		sched.BackupWindow = &models.BackupWindow{Start: "02:00", End: "06:00"}
		sched.ExcludedHours = []int{12, 13, 14}
		compression := "max"
		sched.CompressionLevel = &compression
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		require.NotNil(t, got.BandwidthLimitKB)
		assert.Equal(t, 1024, *got.BandwidthLimitKB)
		require.NotNil(t, got.BackupWindow)
		assert.Equal(t, "02:00", got.BackupWindow.Start)
		assert.Equal(t, "06:00", got.BackupWindow.End)
		assert.Equal(t, []int{12, 13, 14}, got.ExcludedHours)
		require.NotNil(t, got.CompressionLevel)
		assert.Equal(t, "max", *got.CompressionLevel)
	})
}

func TestStore_Backups(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Backup Test Org", "backup-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "backup-agent")
	repo := createTestRepo(t, db, org.ID, "backup-repo")
	sched := models.NewSchedule(agent.ID, "Backup Sched", "0 1 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	t.Run("CreateAndGetByID", func(t *testing.T) {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		got, err := db.GetBackupByID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Equal(t, backup.ID, got.ID)
		assert.Equal(t, models.BackupStatusRunning, got.Status)
		assert.Equal(t, sched.ID, got.ScheduleID)
		assert.Equal(t, agent.ID, got.AgentID)
	})

	t.Run("ListByScheduleID", func(t *testing.T) {
		backups, err := db.GetBackupsByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(backups), 1)
	})

	t.Run("ListByAgentID", func(t *testing.T) {
		backups, err := db.GetBackupsByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(backups), 1)
	})

	t.Run("Update", func(t *testing.T) {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		backup.Complete("snap-123", 1024000, 10, 5)
		err = db.UpdateBackup(ctx, backup)
		require.NoError(t, err)

		got, err := db.GetBackupByID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BackupStatusCompleted, got.Status)
		assert.Equal(t, "snap-123", got.SnapshotID)
		require.NotNil(t, got.SizeBytes)
		assert.Equal(t, int64(1024000), *got.SizeBytes)
	})

	t.Run("GetBySnapshotID", func(t *testing.T) {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup.Complete("unique-snap-"+uuid.New().String()[:8], 500, 1, 0)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		got, err := db.GetBackupBySnapshotID(ctx, backup.SnapshotID)
		require.NoError(t, err)
		assert.Equal(t, backup.ID, got.ID)
	})

	t.Run("GetLatestByScheduleID", func(t *testing.T) {
		got, err := db.GetLatestBackupByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, sched.ID, got.ScheduleID)
	})

	t.Run("SoftDelete", func(t *testing.T) {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		err = db.DeleteBackup(ctx, backup.ID)
		require.NoError(t, err)

		// Should not be found after soft delete
		_, err = db.GetBackupByID(ctx, backup.ID)
		assert.Error(t, err)
	})

	t.Run("SoftDelete_AlreadyDeleted", func(t *testing.T) {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		err = db.DeleteBackup(ctx, backup.ID)
		require.NoError(t, err)

		// Second delete should fail (already deleted)
		err = db.DeleteBackup(ctx, backup.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "backup not found")
	})

	t.Run("SoftDeleteFiltering", func(t *testing.T) {
		// Create two backups, soft-delete one, verify list only returns the other
		b1 := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		b2 := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		require.NoError(t, db.CreateBackup(ctx, b1))
		require.NoError(t, db.CreateBackup(ctx, b2))

		before, err := db.GetBackupsByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		countBefore := len(before)

		require.NoError(t, db.DeleteBackup(ctx, b1.ID))

		after, err := db.GetBackupsByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, countBefore-1, len(after))
	})

	t.Run("WithRetention", func(t *testing.T) {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup.Complete("ret-snap", 2048, 5, 2)
		backup.RecordRetention(3, 7, nil)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		got, err := db.GetBackupByID(ctx, backup.ID)
		require.NoError(t, err)
		assert.True(t, got.RetentionApplied)
		require.NotNil(t, got.SnapshotsRemoved)
		assert.Equal(t, 3, *got.SnapshotsRemoved)
		require.NotNil(t, got.SnapshotsKept)
		assert.Equal(t, 7, *got.SnapshotsKept)
	})

	t.Run("WithScripts", func(t *testing.T) {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup.RecordPreScript("pre output", nil)
		backup.RecordPostScript("post output", fmt.Errorf("post error"))
		backup.Complete("script-snap", 100, 1, 0)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		got, err := db.GetBackupByID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Equal(t, "pre output", got.PreScriptOutput)
		assert.Equal(t, "post output", got.PostScriptOutput)
		assert.Equal(t, "post error", got.PostScriptError)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetBackupByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_Restores(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Restore Test Org", "restore-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "restore-agent")
	repo := createTestRepo(t, db, org.ID, "restore-repo")

	t.Run("CreateAndGetByID", func(t *testing.T) {
		restore := models.NewRestore(agent.ID, repo.ID, "snap-001", "/restore/target",
			[]string{"/data/important"}, []string{"*.log"})
		err := db.CreateRestore(ctx, restore)
		require.NoError(t, err)

		got, err := db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Equal(t, restore.ID, got.ID)
		assert.Equal(t, models.RestoreStatusPending, got.Status)
		assert.Equal(t, "/restore/target", got.TargetPath)
		assert.Equal(t, []string{"/data/important"}, got.IncludePaths)
		assert.Equal(t, []string{"*.log"}, got.ExcludePaths)
	})

	t.Run("ListByAgentID", func(t *testing.T) {
		restores, err := db.GetRestoresByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(restores), 1)
	})

	t.Run("FullLifecycle", func(t *testing.T) {
		restore := models.NewRestore(agent.ID, repo.ID, "snap-002", "/restore/path", nil, nil)
		err := db.CreateRestore(ctx, restore)
		require.NoError(t, err)

		// Start
		restore.Start()
		err = db.UpdateRestore(ctx, restore)
		require.NoError(t, err)

		got, err := db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Equal(t, models.RestoreStatusRunning, got.Status)
		assert.NotNil(t, got.StartedAt)

		// Complete
		restore.Complete()
		err = db.UpdateRestore(ctx, restore)
		require.NoError(t, err)

		got, err = db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Equal(t, models.RestoreStatusCompleted, got.Status)
		assert.NotNil(t, got.CompletedAt)
	})

	t.Run("FailedRestore", func(t *testing.T) {
		restore := models.NewRestore(agent.ID, repo.ID, "snap-003", "/fail/path", nil, nil)
		err := db.CreateRestore(ctx, restore)
		require.NoError(t, err)

		restore.Start()
		require.NoError(t, db.UpdateRestore(ctx, restore))

		restore.Fail("disk full")
		require.NoError(t, db.UpdateRestore(ctx, restore))

		got, err := db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Equal(t, models.RestoreStatusFailed, got.Status)
		assert.Equal(t, "disk full", got.ErrorMessage)
	})

	t.Run("SoftDelete", func(t *testing.T) {
		restore := models.NewRestore(agent.ID, repo.ID, "snap-del", "/del/path", nil, nil)
		err := db.CreateRestore(ctx, restore)
		require.NoError(t, err)

		err = db.DeleteRestore(ctx, restore.ID)
		require.NoError(t, err)

		_, err = db.GetRestoreByID(ctx, restore.ID)
		assert.Error(t, err)
	})

	t.Run("SoftDelete_AlreadyDeleted", func(t *testing.T) {
		restore := models.NewRestore(agent.ID, repo.ID, "snap-dd", "/dd/path", nil, nil)
		require.NoError(t, db.CreateRestore(ctx, restore))
		require.NoError(t, db.DeleteRestore(ctx, restore.ID))

		err := db.DeleteRestore(ctx, restore.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "restore not found")
	})
}

func TestStore_Alerts(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Alert Test Org", "alert-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "alert-agent")

	t.Run("CreateAndGetByID", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeAgentOffline, models.AlertSeverityCritical,
			"Agent Offline", "agent-01 has not checked in")
		alert.SetResource(models.ResourceTypeAgent, agent.ID)
		err := db.CreateAlert(ctx, alert)
		require.NoError(t, err)

		got, err := db.GetAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, alert.ID, got.ID)
		assert.Equal(t, models.AlertTypeAgentOffline, got.Type)
		assert.Equal(t, models.AlertSeverityCritical, got.Severity)
		assert.Equal(t, models.AlertStatusActive, got.Status)
		require.NotNil(t, got.ResourceType)
		assert.Equal(t, models.ResourceTypeAgent, *got.ResourceType)
	})

	t.Run("ListByOrgID", func(t *testing.T) {
		alerts, err := db.GetAlertsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 1)
	})

	t.Run("ActiveAlerts", func(t *testing.T) {
		active, err := db.GetActiveAlertsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(active), 1)

		count, err := db.GetActiveAlertCountByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, len(active), count)
	})

	t.Run("AcknowledgeAndResolve", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeBackupSLA, models.AlertSeverityWarning,
			"SLA Violation", "backup missed")
		require.NoError(t, db.CreateAlert(ctx, alert))

		user := createTestUser(t, db, org.ID, "ack@test.com", "Acknowledger")
		alert.Acknowledge(user.ID)
		err := db.UpdateAlert(ctx, alert)
		require.NoError(t, err)

		got, err := db.GetAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AlertStatusAcknowledged, got.Status)
		require.NotNil(t, got.AcknowledgedBy)
		assert.Equal(t, user.ID, *got.AcknowledgedBy)

		alert.Resolve()
		require.NoError(t, db.UpdateAlert(ctx, alert))

		got, err = db.GetAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AlertStatusResolved, got.Status)
		assert.NotNil(t, got.ResolvedAt)
	})

	t.Run("ResolveByResource", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeAgentOffline, models.AlertSeverityCritical,
			"Offline", "agent down")
		alert.SetResource(models.ResourceTypeAgent, agent.ID)
		require.NoError(t, db.CreateAlert(ctx, alert))

		err := db.ResolveAlertsByResource(ctx, models.ResourceTypeAgent, agent.ID)
		require.NoError(t, err)

		got, err := db.GetAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AlertStatusResolved, got.Status)
	})

	t.Run("GetByResourceAndType", func(t *testing.T) {
		agent2 := createTestAgent(t, db, org.ID, "res-agent")
		alert := models.NewAlert(org.ID, models.AlertTypeAgentHealthCritical, models.AlertSeverityCritical,
			"Health Critical", "CPU too high")
		alert.SetResource(models.ResourceTypeAgent, agent2.ID)
		require.NoError(t, db.CreateAlert(ctx, alert))

		got, err := db.GetAlertByResourceAndType(ctx, org.ID, models.ResourceTypeAgent, agent2.ID, models.AlertTypeAgentHealthCritical)
		require.NoError(t, err)
		assert.Equal(t, alert.ID, got.ID)
	})

	t.Run("WithMetadata", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeStorageUsage, models.AlertSeverityWarning,
			"Storage High", "80% used")
		alert.Metadata = map[string]any{"threshold": 80, "current": 85}
		require.NoError(t, db.CreateAlert(ctx, alert))

		got, err := db.GetAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		require.NotNil(t, got.Metadata)
		assert.Equal(t, float64(80), got.Metadata["threshold"])
	})
}

func TestStore_AlertRules(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Rule Test Org", "rule-test-"+uuid.New().String()[:8])

	t.Run("CRUD", func(t *testing.T) {
		rule := models.NewAlertRule(org.ID, "Agent Offline Rule", models.AlertTypeAgentOffline,
			models.AlertRuleConfig{OfflineThresholdMinutes: 15})
		err := db.CreateAlertRule(ctx, rule)
		require.NoError(t, err)

		got, err := db.GetAlertRuleByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, "Agent Offline Rule", got.Name)
		assert.Equal(t, 15, got.Config.OfflineThresholdMinutes)
		assert.True(t, got.Enabled)

		// List
		rules, err := db.GetAlertRulesByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(rules), 1)

		// Enabled
		enabledRules, err := db.GetEnabledAlertRulesByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(enabledRules), 1)

		// Update
		rule.Name = "Updated Rule"
		rule.Enabled = false
		err = db.UpdateAlertRule(ctx, rule)
		require.NoError(t, err)

		got, err = db.GetAlertRuleByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Rule", got.Name)
		assert.False(t, got.Enabled)

		// Delete
		err = db.DeleteAlertRule(ctx, rule.ID)
		require.NoError(t, err)
		_, err = db.GetAlertRuleByID(ctx, rule.ID)
		assert.Error(t, err)
	})
}

func TestStore_AuditLogs(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Audit Test Org", "audit-test-"+uuid.New().String()[:8])
	user := createTestUser(t, db, org.ID, "auditor@test.com", "Auditor")
	agent := createTestAgent(t, db, org.ID, "audit-agent")

	t.Run("Create", func(t *testing.T) {
		log := models.NewAuditLog(org.ID, models.AuditActionCreate, "agent", models.AuditResultSuccess)
		log.WithUser(user.ID).WithResource(agent.ID).
			WithRequestInfo("192.168.1.1", "test-agent").
			WithDetails("Created agent audit-agent")
		err := db.CreateAuditLog(ctx, log)
		require.NoError(t, err)

		got, err := db.GetAuditLogByID(ctx, log.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AuditActionCreate, got.Action)
		assert.Equal(t, models.AuditResultSuccess, got.Result)
		assert.Equal(t, "agent", got.ResourceType)
		assert.Equal(t, "192.168.1.1", got.IPAddress)
	})

	t.Run("ListWithFilters", func(t *testing.T) {
		// Create several audit logs
		for i := 0; i < 5; i++ {
			action := models.AuditActionCreate
			if i%2 == 0 {
				action = models.AuditActionUpdate
			}
			log := models.NewAuditLog(org.ID, action, "agent", models.AuditResultSuccess)
			log.WithUser(user.ID)
			require.NoError(t, db.CreateAuditLog(ctx, log))
		}

		// No filters
		logs, err := db.GetAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 5)

		// Filter by action
		logs, err = db.GetAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{Action: "create"})
		require.NoError(t, err)
		for _, l := range logs {
			assert.Equal(t, models.AuditActionCreate, l.Action)
		}

		// Filter by resource type
		logs, err = db.GetAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{ResourceType: "agent"})
		require.NoError(t, err)
		for _, l := range logs {
			assert.Equal(t, "agent", l.ResourceType)
		}

		// Filter by result
		logs, err = db.GetAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{Result: "success"})
		require.NoError(t, err)
		for _, l := range logs {
			assert.Equal(t, models.AuditResultSuccess, l.Result)
		}

		// With limit and offset
		logs, err = db.GetAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{Limit: 2, Offset: 1})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(logs), 2)

		// With date filters
		startDate := time.Now().Add(-1 * time.Hour)
		endDate := time.Now().Add(1 * time.Hour)
		logs, err = db.GetAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 1)
	})

	t.Run("Count", func(t *testing.T) {
		count, err := db.CountAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{})
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))

		// Count with filter
		count, err = db.CountAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{Action: "create"})
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
	})

	t.Run("Search", func(t *testing.T) {
		log := models.NewAuditLog(org.ID, models.AuditActionDelete, "repository", models.AuditResultSuccess)
		log.WithDetails("Deleted repository backup-repo-xyz")
		require.NoError(t, db.CreateAuditLog(ctx, log))

		logs, err := db.GetAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{Search: "backup-repo-xyz"})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 1)
	})
}

func TestStore_Policies(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Policy Test Org", "policy-test-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		policy := models.NewPolicy(org.ID, "Daily Production")
		policy.Description = "Standard daily backup policy"
		policy.Paths = []string{"/data", "/etc"}
		policy.Excludes = []string{"*.tmp", "*.log"}
		policy.RetentionPolicy = models.DefaultRetentionPolicy()
		bwLimit := 512
		policy.BandwidthLimitKB = &bwLimit
		policy.BackupWindow = &models.BackupWindow{Start: "01:00", End: "05:00"}
		policy.CronExpression = "0 2 * * *"
		err := db.CreatePolicy(ctx, policy)
		require.NoError(t, err)

		got, err := db.GetPolicyByID(ctx, policy.ID)
		require.NoError(t, err)
		assert.Equal(t, "Daily Production", got.Name)
		assert.Equal(t, "Standard daily backup policy", got.Description)
		assert.Equal(t, []string{"/data", "/etc"}, got.Paths)
		assert.Equal(t, []string{"*.tmp", "*.log"}, got.Excludes)
		require.NotNil(t, got.RetentionPolicy)
		require.NotNil(t, got.BandwidthLimitKB)
		assert.Equal(t, 512, *got.BandwidthLimitKB)
		require.NotNil(t, got.BackupWindow)
		assert.Equal(t, "01:00", got.BackupWindow.Start)
		assert.Equal(t, "0 2 * * *", got.CronExpression)
	})

	t.Run("ListByOrgID", func(t *testing.T) {
		policies, err := db.GetPoliciesByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(policies), 1)
	})

	t.Run("Update", func(t *testing.T) {
		policy := models.NewPolicy(org.ID, "Old Policy")
		require.NoError(t, db.CreatePolicy(ctx, policy))

		policy.Name = "Updated Policy"
		policy.Paths = []string{"/new/path"}
		err := db.UpdatePolicy(ctx, policy)
		require.NoError(t, err)

		got, err := db.GetPolicyByID(ctx, policy.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Policy", got.Name)
		assert.Equal(t, []string{"/new/path"}, got.Paths)
	})

	t.Run("Delete", func(t *testing.T) {
		policy := models.NewPolicy(org.ID, "Delete Policy")
		require.NoError(t, db.CreatePolicy(ctx, policy))

		err := db.DeletePolicy(ctx, policy.ID)
		require.NoError(t, err)

		_, err = db.GetPolicyByID(ctx, policy.ID)
		assert.Error(t, err)
	})

	t.Run("ApplyToSchedule", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "policy-apply-agent")
		policy := models.NewPolicy(org.ID, "Apply Policy")
		policy.Paths = []string{"/policy/path"}
		policy.CronExpression = "0 3 * * *"
		policy.RetentionPolicy = models.DefaultRetentionPolicy()
		require.NoError(t, db.CreatePolicy(ctx, policy))

		sched := models.NewSchedule(agent.ID, "Policy Sched", "0 1 * * *", []string{"/old"})
		sched.PolicyID = &policy.ID
		require.NoError(t, db.CreateSchedule(ctx, sched))

		// Verify schedule is linked to policy
		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		require.NotNil(t, got.PolicyID)
		assert.Equal(t, policy.ID, *got.PolicyID)
	})
}

func TestStore_NotificationChannels(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Notif Test Org", "notif-test-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Email Channel", models.ChannelTypeEmail, []byte("smtp-config"))
		err := db.CreateNotificationChannel(ctx, channel)
		require.NoError(t, err)

		got, err := db.GetNotificationChannelByID(ctx, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, "Email Channel", got.Name)
		assert.Equal(t, models.ChannelTypeEmail, got.Type)
		assert.True(t, got.Enabled)
		assert.Equal(t, []byte("smtp-config"), got.ConfigEncrypted)
	})

	t.Run("ListByOrgID", func(t *testing.T) {
		channels, err := db.GetNotificationChannelsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 1)
	})

	t.Run("GetEnabledEmailChannels", func(t *testing.T) {
		channels, err := db.GetEnabledEmailChannelsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 1)
		for _, c := range channels {
			assert.Equal(t, models.ChannelTypeEmail, c.Type)
			assert.True(t, c.Enabled)
		}
	})

	t.Run("Update", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Old Channel", models.ChannelTypeSlack, []byte("slack-config"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		channel.Name = "Updated Channel"
		channel.Enabled = false
		channel.ConfigEncrypted = []byte("new-slack-config")
		err := db.UpdateNotificationChannel(ctx, channel)
		require.NoError(t, err)

		got, err := db.GetNotificationChannelByID(ctx, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Channel", got.Name)
		assert.False(t, got.Enabled)
	})

	t.Run("Delete", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Delete Channel", models.ChannelTypeWebhook, []byte("wh-config"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		err := db.DeleteNotificationChannel(ctx, channel.ID)
		require.NoError(t, err)

		_, err = db.GetNotificationChannelByID(ctx, channel.ID)
		assert.Error(t, err)
	})

	t.Run("Preferences", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Pref Channel", models.ChannelTypeEmail, []byte("cfg"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		pref := models.NewNotificationPreference(org.ID, channel.ID, models.EventBackupSuccess)
		err := db.CreateNotificationPreference(ctx, pref)
		require.NoError(t, err)

		// List by org
		prefs, err := db.GetNotificationPreferencesByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(prefs), 1)

		// List by channel
		prefs, err = db.GetNotificationPreferencesByChannelID(ctx, channel.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(prefs), 1)

		// Get enabled for event
		enabledPrefs, err := db.GetEnabledPreferencesForEvent(ctx, org.ID, models.EventBackupSuccess)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(enabledPrefs), 1)

		// Update
		pref.Enabled = false
		err = db.UpdateNotificationPreference(ctx, pref)
		require.NoError(t, err)

		// Should no longer appear in enabled prefs
		enabledPrefs, err = db.GetEnabledPreferencesForEvent(ctx, org.ID, models.EventBackupSuccess)
		require.NoError(t, err)
		found := false
		for _, p := range enabledPrefs {
			if p.ID == pref.ID {
				found = true
			}
		}
		assert.False(t, found)

		// Delete
		err = db.DeleteNotificationPreference(ctx, pref.ID)
		require.NoError(t, err)
	})

	t.Run("NotificationLogs", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Log Channel", models.ChannelTypeEmail, []byte("cfg"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		log := models.NewNotificationLog(org.ID, &channel.ID, "backup_success", "user@test.com", "Backup Complete")
		err := db.CreateNotificationLog(ctx, log)
		require.NoError(t, err)

		// Mark sent
		log.MarkSent()
		err = db.UpdateNotificationLog(ctx, log)
		require.NoError(t, err)

		// List
		logs, err := db.GetNotificationLogsByOrgID(ctx, org.ID, 100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 1)

		foundLog := false
		for _, l := range logs {
			if l.ID == log.ID {
				foundLog = true
				assert.Equal(t, models.NotificationStatusSent, l.Status)
			}
		}
		assert.True(t, foundLog)
	})
}

func TestStore_Memberships(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Member Test Org", "member-test-"+uuid.New().String()[:8])
	user := models.NewUser(org.ID, "oidc-member-"+uuid.New().String()[:8], "member@test.com", "Member", models.UserRoleUser)
	require.NoError(t, db.CreateUser(ctx, user))

	t.Run("CreateAndGet", func(t *testing.T) {
		m := models.NewOrgMembership(user.ID, org.ID, models.OrgRoleMember)
		err := db.CreateMembership(ctx, m)
		require.NoError(t, err)

		got, err := db.GetMembershipByUserAndOrg(ctx, user.ID, org.ID)
		require.NoError(t, err)
		assert.Equal(t, models.OrgRoleMember, got.Role)
	})

	t.Run("ListByUserID", func(t *testing.T) {
		memberships, err := db.GetMembershipsByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(memberships), 1)
	})

	t.Run("ListByOrgID", func(t *testing.T) {
		memberships, err := db.GetMembershipsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(memberships), 1)
	})

	t.Run("GetUserOrganizations", func(t *testing.T) {
		orgs, err := db.GetUserOrganizations(ctx, user.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(orgs), 1)
	})

	t.Run("Update", func(t *testing.T) {
		m, err := db.GetMembershipByUserAndOrg(ctx, user.ID, org.ID)
		require.NoError(t, err)

		m.Role = models.OrgRoleAdmin
		err = db.UpdateMembership(ctx, m)
		require.NoError(t, err)

		got, err := db.GetMembershipByUserAndOrg(ctx, user.ID, org.ID)
		require.NoError(t, err)
		assert.Equal(t, models.OrgRoleAdmin, got.Role)
	})

	t.Run("Delete", func(t *testing.T) {
		err := db.DeleteMembership(ctx, user.ID, org.ID)
		require.NoError(t, err)

		_, err = db.GetMembershipByUserAndOrg(ctx, user.ID, org.ID)
		assert.Error(t, err)
	})
}

func TestStore_Invitations(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Invite Test Org", "invite-test-"+uuid.New().String()[:8])
	user := createTestUser(t, db, org.ID, "inviter@test.com", "Inviter")

	t.Run("CreateAndGetByToken", func(t *testing.T) {
		token := "invite-token-" + uuid.New().String()[:8]
		inv := models.NewOrgInvitation(org.ID, "new@test.com", models.OrgRoleMember, token, user.ID, time.Now().Add(72*time.Hour))
		err := db.CreateInvitation(ctx, inv)
		require.NoError(t, err)

		got, err := db.GetInvitationByToken(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, "new@test.com", got.Email)
		assert.Equal(t, models.OrgRoleMember, got.Role)
	})

	t.Run("GetPendingByOrgID", func(t *testing.T) {
		invitations, err := db.GetPendingInvitationsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(invitations), 1)
	})

	t.Run("GetPendingByEmail", func(t *testing.T) {
		email := "pending-" + uuid.New().String()[:8] + "@test.com"
		token := "token-" + uuid.New().String()[:8]
		inv := models.NewOrgInvitation(org.ID, email, models.OrgRoleMember, token, user.ID, time.Now().Add(72*time.Hour))
		require.NoError(t, db.CreateInvitation(ctx, inv))

		invitations, err := db.GetPendingInvitationsByEmail(ctx, email)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(invitations), 1)
	})

	t.Run("Accept", func(t *testing.T) {
		token := "accept-token-" + uuid.New().String()[:8]
		inv := models.NewOrgInvitation(org.ID, "accept@test.com", models.OrgRoleMember, token, user.ID, time.Now().Add(72*time.Hour))
		require.NoError(t, db.CreateInvitation(ctx, inv))

		err := db.AcceptInvitation(ctx, inv.ID)
		require.NoError(t, err)

		got, err := db.GetInvitationByToken(ctx, token)
		require.NoError(t, err)
		assert.NotNil(t, got.AcceptedAt)
	})

	t.Run("Delete", func(t *testing.T) {
		token := "delete-token-" + uuid.New().String()[:8]
		inv := models.NewOrgInvitation(org.ID, "delete@test.com", models.OrgRoleMember, token, user.ID, time.Now().Add(72*time.Hour))
		require.NoError(t, db.CreateInvitation(ctx, inv))

		err := db.DeleteInvitation(ctx, inv.ID)
		require.NoError(t, err)

		_, err = db.GetInvitationByToken(ctx, token)
		assert.Error(t, err)
	})
}

func TestStore_BackupScripts(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Script Test Org", "script-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "script-agent")
	sched := models.NewSchedule(agent.ID, "Script Sched", "0 1 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	t.Run("CRUD", func(t *testing.T) {
		script := models.NewBackupScript(sched.ID, models.BackupScriptTypePreBackup, "#!/bin/bash\necho hello")
		err := db.CreateBackupScript(ctx, script)
		require.NoError(t, err)

		got, err := db.GetBackupScriptByID(ctx, script.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BackupScriptTypePreBackup, got.Type)
		assert.Equal(t, "#!/bin/bash\necho hello", got.Script)
		assert.Equal(t, 300, got.TimeoutSeconds)
		assert.True(t, got.Enabled)

		// List by schedule
		scripts, err := db.GetBackupScriptsByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(scripts), 1)

		// By schedule and type
		got, err = db.GetBackupScriptByScheduleAndType(ctx, sched.ID, models.BackupScriptTypePreBackup)
		require.NoError(t, err)
		assert.Equal(t, script.ID, got.ID)

		// Enabled only
		enabled, err := db.GetEnabledBackupScriptsByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(enabled), 1)

		// Update
		script.Script = "#!/bin/bash\necho updated"
		script.FailOnError = true
		err = db.UpdateBackupScript(ctx, script)
		require.NoError(t, err)

		got, err = db.GetBackupScriptByID(ctx, script.ID)
		require.NoError(t, err)
		assert.Equal(t, "#!/bin/bash\necho updated", got.Script)
		assert.True(t, got.FailOnError)

		// Delete
		err = db.DeleteBackupScript(ctx, script.ID)
		require.NoError(t, err)
		_, err = db.GetBackupScriptByID(ctx, script.ID)
		assert.Error(t, err)
	})
}

func TestStore_StorageStats(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Stats Test Org", "stats-test-"+uuid.New().String()[:8])
	repo := createTestRepo(t, db, org.ID, "stats-repo")

	t.Run("CreateAndGetLatest", func(t *testing.T) {
		stats := models.NewStorageStats(repo.ID)
		stats.SetStats(1024*1024, 100, 512*1024, 1024*1024, 10)
		err := db.CreateStorageStats(ctx, stats)
		require.NoError(t, err)

		got, err := db.GetLatestStorageStats(ctx, repo.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1024*1024), got.TotalSize)
		assert.Equal(t, 100, got.TotalFileCount)
		assert.Equal(t, 10, got.SnapshotCount)
	})

	t.Run("ListByRepositoryID", func(t *testing.T) {
		// Create a second stats entry
		stats2 := models.NewStorageStats(repo.ID)
		stats2.SetStats(2*1024*1024, 200, 1024*1024, 2*1024*1024, 15)
		require.NoError(t, db.CreateStorageStats(ctx, stats2))

		allStats, err := db.GetStorageStatsByRepositoryID(ctx, repo.ID, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allStats), 2)
	})

	t.Run("Summary", func(t *testing.T) {
		summary, err := db.GetStorageStatsSummary(ctx, org.ID)
		require.NoError(t, err)
		assert.Greater(t, summary.RepositoryCount, 0)
	})

	t.Run("LatestForAllRepos", func(t *testing.T) {
		allStats, err := db.GetLatestStatsForAllRepos(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allStats), 1)
	})

	t.Run("StorageGrowth", func(t *testing.T) {
		points, err := db.GetStorageGrowth(ctx, repo.ID, 30)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(points), 1)
	})

	t.Run("AllStorageGrowth", func(t *testing.T) {
		points, err := db.GetAllStorageGrowth(ctx, org.ID, 30)
		require.NoError(t, err)
		// May be empty depending on the complex query, but shouldn't error
		_ = points
	})
}

func TestStore_ReplicationStatus(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Repl Test Org", "repl-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "repl-agent")
	repo1 := createTestRepo(t, db, org.ID, "repl-source")
	repo2 := createTestRepo(t, db, org.ID, "repl-target")
	sched := models.NewSchedule(agent.ID, "Repl Sched", "0 1 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	t.Run("GetOrCreate", func(t *testing.T) {
		rs, err := db.GetOrCreateReplicationStatus(ctx, sched.ID, repo1.ID, repo2.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ReplicationStatusPending, rs.Status)

		// Second call returns existing
		rs2, err := db.GetOrCreateReplicationStatus(ctx, sched.ID, repo1.ID, repo2.ID)
		require.NoError(t, err)
		assert.Equal(t, rs.ID, rs2.ID)
	})

	t.Run("Update", func(t *testing.T) {
		rs, err := db.GetOrCreateReplicationStatus(ctx, sched.ID, repo1.ID, repo2.ID)
		require.NoError(t, err)

		rs.MarkSynced("snap-repl-1")
		err = db.UpdateReplicationStatus(ctx, rs)
		require.NoError(t, err)

		rs2, err := db.GetOrCreateReplicationStatus(ctx, sched.ID, repo1.ID, repo2.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ReplicationStatusSynced, rs2.Status)
		require.NotNil(t, rs2.LastSnapshotID)
		assert.Equal(t, "snap-repl-1", *rs2.LastSnapshotID)
	})

	t.Run("ListBySchedule", func(t *testing.T) {
		statuses, err := db.GetReplicationStatusBySchedule(ctx, sched.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(statuses), 1)
	})
}

func TestStore_Verifications(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Verif Test Org", "verif-test-"+uuid.New().String()[:8])
	repo := createTestRepo(t, db, org.ID, "verif-repo")

	t.Run("VerificationSchedule_CRUD", func(t *testing.T) {
		vs := models.NewVerificationSchedule(repo.ID, models.VerificationTypeCheck, "0 3 * * 0")
		err := db.CreateVerificationSchedule(ctx, vs)
		require.NoError(t, err)

		got, err := db.GetVerificationScheduleByID(ctx, vs.ID)
		require.NoError(t, err)
		assert.Equal(t, models.VerificationTypeCheck, got.Type)
		assert.Equal(t, "0 3 * * 0", got.CronExpression)
		assert.True(t, got.Enabled)

		// List by repo
		schedules, err := db.GetVerificationSchedulesByRepoID(ctx, repo.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(schedules), 1)

		// Enabled
		enabled, err := db.GetEnabledVerificationSchedules(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(enabled), 1)

		// Update
		vs.CronExpression = "0 4 * * 0"
		vs.Enabled = false
		err = db.UpdateVerificationSchedule(ctx, vs)
		require.NoError(t, err)

		got, err = db.GetVerificationScheduleByID(ctx, vs.ID)
		require.NoError(t, err)
		assert.Equal(t, "0 4 * * 0", got.CronExpression)
		assert.False(t, got.Enabled)

		// Delete
		err = db.DeleteVerificationSchedule(ctx, vs.ID)
		require.NoError(t, err)
		_, err = db.GetVerificationScheduleByID(ctx, vs.ID)
		assert.Error(t, err)
	})

	t.Run("Verification_CRUD", func(t *testing.T) {
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		detailsBytes, _ := v.DetailsJSON()
		_ = detailsBytes // no details yet

		// We need to insert via raw SQL since there's no CreateVerification exposed,
		// but let's check if it exists
		_, err := db.Pool.Exec(ctx, `
			INSERT INTO verifications (id, repository_id, type, started_at, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, v.ID, v.RepositoryID, string(v.Type), v.StartedAt, string(v.Status), v.CreatedAt)
		require.NoError(t, err)

		got, err := db.GetVerificationByID(ctx, v.ID)
		require.NoError(t, err)
		assert.Equal(t, models.VerificationTypeCheck, got.Type)
		assert.Equal(t, models.VerificationStatusRunning, got.Status)

		// List by repo
		verifs, err := db.GetVerificationsByRepoID(ctx, repo.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(verifs), 1)

		// Latest
		latest, err := db.GetLatestVerificationByRepoID(ctx, repo.ID)
		require.NoError(t, err)
		assert.Equal(t, v.ID, latest.ID)
	})
}

func TestStore_Transactions(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	t.Run("SuccessfulTransaction", func(t *testing.T) {
		err := db.ExecTx(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "SELECT 1")
			return err
		})
		require.NoError(t, err)
	})

	t.Run("RollbackOnError", func(t *testing.T) {
		org := createTestOrg(t, db, "TX Test Org", "tx-test-"+uuid.New().String()[:8])

		err := db.ExecTx(ctx, func(tx pgx.Tx) error {
			// Insert an org
			_, err := tx.Exec(ctx, `
				INSERT INTO organizations (id, name, slug, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5)
			`, uuid.New(), "TX Org", "tx-org-unique", time.Now(), time.Now())
			if err != nil {
				return err
			}
			// Force error
			return fmt.Errorf("forced rollback")
		})
		assert.Error(t, err)

		// Verify the org was not created
		_, err = db.GetOrganizationBySlug(ctx, "tx-org-unique")
		assert.Error(t, err)

		// But the original org still exists
		_, err = db.GetOrganizationByID(ctx, org.ID)
		require.NoError(t, err)
	})
}

func TestStore_EdgeCases(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Edge Test Org", "edge-test-"+uuid.New().String()[:8])

	t.Run("EmptyLists", func(t *testing.T) {
		newOrgID := uuid.New()
		// These should return nil/empty slices, not errors
		agents, err := db.GetAgentsByOrgID(ctx, newOrgID)
		require.NoError(t, err)
		assert.Empty(t, agents)
	})

	t.Run("AgentWithOSInfo", func(t *testing.T) {
		agent := models.NewAgent(org.ID, "os-info-server", "os-hash")
		agent.OSInfo = &models.OSInfo{
			OS:      "linux",
			Arch:    "amd64",
			Version: "Ubuntu 22.04",
		}
		err := db.CreateAgent(ctx, agent)
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		require.NotNil(t, got.OSInfo)
		assert.Equal(t, "linux", got.OSInfo.OS)
		assert.Equal(t, "amd64", got.OSInfo.Arch)
	})

	t.Run("AgentWithNetworkMounts", func(t *testing.T) {
		agent := models.NewAgent(org.ID, "mount-server", "mount-hash")
		agent.NetworkMounts = []models.NetworkMount{
			{
				Path:   "/mnt/backup",
				Type:   models.MountTypeNFS,
				Remote: "nfs.server:/share",
				Status: models.MountStatusConnected,
			},
		}
		err := db.CreateAgent(ctx, agent)
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		require.Len(t, got.NetworkMounts, 1)
		assert.Equal(t, "/mnt/backup", got.NetworkMounts[0].Path)
		assert.Equal(t, models.MountTypeNFS, got.NetworkMounts[0].Type)
	})

	t.Run("HealthHistoryDefaultLimit", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "hist-limit-server")
		// Passing 0 should use default limit of 100
		history, err := db.GetAgentHealthHistory(ctx, agent.ID, 0)
		require.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("BackupWithNilRepoID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "nil-repo-agent")
		sched := models.NewSchedule(agent.ID, "Nil Repo Sched", "0 1 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		backup := models.NewBackup(sched.ID, agent.ID, nil)
		err := db.CreateBackup(ctx, backup)
		require.NoError(t, err)

		got, err := db.GetBackupByID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Nil(t, got.RepositoryID)
	})

	t.Run("RestoreWithEmptyPaths", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "empty-paths-agent")
		repo := createTestRepo(t, db, org.ID, "empty-paths-repo")

		restore := models.NewRestore(agent.ID, repo.ID, "snap-empty", "/target", nil, nil)
		err := db.CreateRestore(ctx, restore)
		require.NoError(t, err)

		got, err := db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Empty(t, got.IncludePaths)
		assert.Empty(t, got.ExcludePaths)
	})

	t.Run("AlertWithoutResource", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeBackupSLA, models.AlertSeverityInfo,
			"No Resource", "test alert without resource")
		err := db.CreateAlert(ctx, alert)
		require.NoError(t, err)

		got, err := db.GetAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Nil(t, got.ResourceType)
		assert.Nil(t, got.ResourceID)
	})

	t.Run("ScheduleWithOnMountUnavailable", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "mount-behavior-agent")
		sched := models.NewSchedule(agent.ID, "Mount Sched", "0 1 * * *", []string{"/mnt/share"})
		sched.OnMountUnavailable = models.MountBehaviorSkip
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, models.MountBehaviorSkip, got.OnMountUnavailable)
	})
}
