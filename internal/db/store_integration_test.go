//go:build integration

package db

import (
	"context"
	"fmt"
	"log"
	"os"
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

var testDB *DB

func TestMain(m *testing.M) {
	if !dockerAvailable() {
		fmt.Println("Docker is not available, skipping integration tests")
		os.Exit(0)
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
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		pgContainer.Terminate(ctx)
		log.Fatalf("failed to get connection string: %v", err)
	}

	logger := zerolog.New(zerolog.NewConsoleWriter())
	cfg := DefaultConfig(connStr)
	cfg.MaxConns = 5
	cfg.MinConns = 1

	testDB, err = New(ctx, cfg, logger)
	if err != nil {
		pgContainer.Terminate(ctx)
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := testDB.Migrate(ctx); err != nil {
		testDB.Close()
		pgContainer.Terminate(ctx)
		log.Fatalf("failed to run migrations: %v", err)
	}

	code := m.Run()

	testDB.Close()
	pgContainer.Terminate(ctx)

	os.Exit(code)
}

// dockerAvailable returns true if a Docker daemon is reachable.
func dockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// setupTestDB returns the shared test database after cleaning all tables.
func setupTestDB(t *testing.T) *DB {
	t.Helper()
	cleanTables(t, testDB)
	return testDB
}

// cleanTables truncates all user tables between tests for isolation.
func cleanTables(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `
		DO $$ DECLARE r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename != 'schema_migrations') LOOP
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`)
	require.NoError(t, err)
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
		assert.Contains(t, got.BackupWindow.Start, "02:00")
		assert.Contains(t, got.BackupWindow.End, "06:00")
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
		assert.Contains(t, got.BackupWindow.Start, "01:00")
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

		err := db.CreateVerification(ctx, v)
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

func TestStore_MaintenanceWindows(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "MW Test Org", "mw-test-"+uuid.New().String()[:8])
	user := createTestUser(t, db, org.ID, "mw-user@test.com", "MW User")

	t.Run("CreateAndGetByID", func(t *testing.T) {
		startsAt := time.Now().Add(2 * time.Hour)
		endsAt := time.Now().Add(4 * time.Hour)
		mw := models.NewMaintenanceWindow(org.ID, "Server Upgrade", startsAt, endsAt)
		mw.Message = "Upgrading database servers"
		mw.CreatedBy = &user.ID

		err := db.CreateMaintenanceWindow(ctx, mw)
		require.NoError(t, err)

		got, err := db.GetMaintenanceWindowByID(ctx, mw.ID)
		require.NoError(t, err)
		assert.Equal(t, mw.ID, got.ID)
		assert.Equal(t, org.ID, got.OrgID)
		assert.Equal(t, "Server Upgrade", got.Title)
		assert.Equal(t, "Upgrading database servers", got.Message)
		assert.Equal(t, 60, got.NotifyBeforeMinutes)
		assert.False(t, got.NotificationSent)
		require.NotNil(t, got.CreatedBy)
		assert.Equal(t, user.ID, *got.CreatedBy)
	})

	t.Run("ListByOrg", func(t *testing.T) {
		windows, err := db.ListMaintenanceWindowsByOrg(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(windows), 1)
	})

	t.Run("ListActiveMaintenanceWindows", func(t *testing.T) {
		// Create a window that is currently active (startsAt in the past, endsAt in the future)
		startsAt := time.Now().Add(-1 * time.Hour)
		endsAt := time.Now().Add(1 * time.Hour)
		mw := models.NewMaintenanceWindow(org.ID, "Active Window", startsAt, endsAt)
		err := db.CreateMaintenanceWindow(ctx, mw)
		require.NoError(t, err)

		active, err := db.ListActiveMaintenanceWindows(ctx, org.ID, time.Now())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(active), 1)

		found := false
		for _, w := range active {
			if w.ID == mw.ID {
				found = true
			}
		}
		assert.True(t, found, "expected the active window to be returned")
	})

	t.Run("ListUpcomingMaintenanceWindows", func(t *testing.T) {
		// Create a window starting within 30 minutes (within the 60-minute query range)
		startsAt := time.Now().Add(30 * time.Minute)
		endsAt := time.Now().Add(90 * time.Minute)
		mw := models.NewMaintenanceWindow(org.ID, "Upcoming Window", startsAt, endsAt)
		err := db.CreateMaintenanceWindow(ctx, mw)
		require.NoError(t, err)

		upcoming, err := db.ListUpcomingMaintenanceWindows(ctx, org.ID, time.Now(), 60)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(upcoming), 1)

		found := false
		for _, w := range upcoming {
			if w.ID == mw.ID {
				found = true
			}
		}
		assert.True(t, found, "expected the upcoming window to be returned")
	})

	t.Run("ListPendingMaintenanceNotifications", func(t *testing.T) {
		// Create a window that starts within notify_before_minutes (default 60) from now,
		// with notification_sent = false.
		startsAt := time.Now().Add(30 * time.Minute)
		endsAt := time.Now().Add(2 * time.Hour)
		mw := models.NewMaintenanceWindow(org.ID, "Pending Notification", startsAt, endsAt)
		err := db.CreateMaintenanceWindow(ctx, mw)
		require.NoError(t, err)

		pending, err := db.ListPendingMaintenanceNotifications(ctx)
		require.NoError(t, err)
		// Should find at least the one we just created
		found := false
		for _, w := range pending {
			if w.ID == mw.ID {
				found = true
				assert.False(t, w.NotificationSent)
			}
		}
		assert.True(t, found, "expected the pending notification window to be returned")
	})

	t.Run("UpdateMaintenanceWindow", func(t *testing.T) {
		startsAt := time.Now().Add(3 * time.Hour)
		endsAt := time.Now().Add(5 * time.Hour)
		mw := models.NewMaintenanceWindow(org.ID, "Update Me", startsAt, endsAt)
		require.NoError(t, db.CreateMaintenanceWindow(ctx, mw))

		mw.Title = "Updated Title"
		mw.Message = "Updated message"
		mw.NotifyBeforeMinutes = 30
		err := db.UpdateMaintenanceWindow(ctx, mw)
		require.NoError(t, err)

		got, err := db.GetMaintenanceWindowByID(ctx, mw.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", got.Title)
		assert.Equal(t, "Updated message", got.Message)
		assert.Equal(t, 30, got.NotifyBeforeMinutes)
	})

	t.Run("DeleteMaintenanceWindow", func(t *testing.T) {
		startsAt := time.Now().Add(6 * time.Hour)
		endsAt := time.Now().Add(8 * time.Hour)
		mw := models.NewMaintenanceWindow(org.ID, "Delete Me", startsAt, endsAt)
		require.NoError(t, db.CreateMaintenanceWindow(ctx, mw))

		err := db.DeleteMaintenanceWindow(ctx, mw.ID)
		require.NoError(t, err)

		_, err = db.GetMaintenanceWindowByID(ctx, mw.ID)
		assert.Error(t, err)
	})

	t.Run("MarkMaintenanceNotificationSent", func(t *testing.T) {
		startsAt := time.Now().Add(10 * time.Minute)
		endsAt := time.Now().Add(2 * time.Hour)
		mw := models.NewMaintenanceWindow(org.ID, "Mark Sent", startsAt, endsAt)
		require.NoError(t, db.CreateMaintenanceWindow(ctx, mw))
		assert.False(t, mw.NotificationSent)

		err := db.MarkMaintenanceNotificationSent(ctx, mw.ID)
		require.NoError(t, err)

		got, err := db.GetMaintenanceWindowByID(ctx, mw.ID)
		require.NoError(t, err)
		assert.True(t, got.NotificationSent)
	})
}

func TestStore_ExcludePatterns(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "EP Test Org", "ep-test-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		ep := models.NewExcludePattern(org.ID, "Node Modules", "Exclude node_modules directories", "development", []string{"**/node_modules/**", "**/.npm/**"})
		err := db.CreateExcludePattern(ctx, ep)
		require.NoError(t, err)

		got, err := db.GetExcludePatternByID(ctx, ep.ID)
		require.NoError(t, err)
		assert.Equal(t, ep.ID, got.ID)
		assert.Equal(t, "Node Modules", got.Name)
		assert.Equal(t, "Exclude node_modules directories", got.Description)
		assert.Equal(t, "development", got.Category)
		assert.False(t, got.IsBuiltin)
		require.NotNil(t, got.OrgID)
		assert.Equal(t, org.ID, *got.OrgID)
		assert.Equal(t, []string{"**/node_modules/**", "**/.npm/**"}, got.Patterns)
	})

	t.Run("GetByOrgID", func(t *testing.T) {
		patterns, err := db.GetExcludePatternsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(patterns), 1)
	})

	t.Run("GetByCategory", func(t *testing.T) {
		// Create another pattern in the same category
		ep2 := models.NewExcludePattern(org.ID, "Python Cache", "Exclude Python cache", "development", []string{"**/__pycache__/**", "*.pyc"})
		require.NoError(t, db.CreateExcludePattern(ctx, ep2))

		patterns, err := db.GetExcludePatternsByCategory(ctx, org.ID, "development")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(patterns), 2)
		for _, p := range patterns {
			assert.Equal(t, "development", p.Category)
		}
	})

	t.Run("BuiltinPatterns", func(t *testing.T) {
		// Seed some builtin patterns
		builtins := []*models.ExcludePattern{
			models.NewBuiltinExcludePattern("OS Temporary Files", "Common OS temp files", "system", []string{"/tmp/**", "/var/tmp/**"}),
			models.NewBuiltinExcludePattern("Cache Directories", "Common cache dirs", "system", []string{"**/.cache/**", "**/Cache/**"}),
		}
		err := db.SeedBuiltinExcludePatterns(ctx, builtins)
		require.NoError(t, err)

		// Get builtin patterns
		got, err := db.GetBuiltinExcludePatterns(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(got), 2)
		for _, p := range got {
			assert.True(t, p.IsBuiltin)
			assert.Nil(t, p.OrgID)
		}

		// Seeding again should not fail (upsert behavior)
		builtins[0].Description = "Updated description"
		err = db.SeedBuiltinExcludePatterns(ctx, builtins)
		require.NoError(t, err)
	})

	t.Run("UpdateExcludePattern", func(t *testing.T) {
		ep := models.NewExcludePattern(org.ID, "Update Pattern", "old desc", "misc", []string{"*.old"})
		require.NoError(t, db.CreateExcludePattern(ctx, ep))

		ep.Name = "Updated Pattern"
		ep.Description = "new desc"
		ep.Patterns = []string{"*.old", "*.bak"}
		ep.Category = "cleanup"
		err := db.UpdateExcludePattern(ctx, ep)
		require.NoError(t, err)

		got, err := db.GetExcludePatternByID(ctx, ep.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Pattern", got.Name)
		assert.Equal(t, "new desc", got.Description)
		assert.Equal(t, "cleanup", got.Category)
		assert.Equal(t, []string{"*.old", "*.bak"}, got.Patterns)
	})

	t.Run("DeleteExcludePattern", func(t *testing.T) {
		ep := models.NewExcludePattern(org.ID, "Delete Pattern", "to be deleted", "temp", []string{"*.del"})
		require.NoError(t, db.CreateExcludePattern(ctx, ep))

		err := db.DeleteExcludePattern(ctx, ep.ID)
		require.NoError(t, err)

		_, err = db.GetExcludePatternByID(ctx, ep.ID)
		assert.Error(t, err)
	})
}

func TestStore_SnapshotComments(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Comment Test Org", "comment-test-"+uuid.New().String()[:8])
	user := createTestUser(t, db, org.ID, "commenter@test.com", "Commenter")

	t.Run("CreateAndGetByID", func(t *testing.T) {
		snapshotID := "snap-" + uuid.New().String()[:8]
		comment := models.NewSnapshotComment(org.ID, snapshotID, user.ID, "This backup looks good")
		err := db.CreateSnapshotComment(ctx, comment)
		require.NoError(t, err)

		got, err := db.GetSnapshotCommentByID(ctx, comment.ID)
		require.NoError(t, err)
		assert.Equal(t, comment.ID, got.ID)
		assert.Equal(t, org.ID, got.OrgID)
		assert.Equal(t, snapshotID, got.SnapshotID)
		assert.Equal(t, user.ID, got.UserID)
		assert.Equal(t, "This backup looks good", got.Content)
	})

	t.Run("GetBySnapshotID", func(t *testing.T) {
		snapshotID := "snap-list-" + uuid.New().String()[:8]
		c1 := models.NewSnapshotComment(org.ID, snapshotID, user.ID, "Comment 1")
		c2 := models.NewSnapshotComment(org.ID, snapshotID, user.ID, "Comment 2")
		require.NoError(t, db.CreateSnapshotComment(ctx, c1))
		require.NoError(t, db.CreateSnapshotComment(ctx, c2))

		comments, err := db.GetSnapshotCommentsBySnapshotID(ctx, snapshotID, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(comments), 2)
	})

	t.Run("UpdateSnapshotComment", func(t *testing.T) {
		snapshotID := "snap-upd-" + uuid.New().String()[:8]
		comment := models.NewSnapshotComment(org.ID, snapshotID, user.ID, "Old content")
		require.NoError(t, db.CreateSnapshotComment(ctx, comment))

		comment.Content = "Updated content"
		err := db.UpdateSnapshotComment(ctx, comment)
		require.NoError(t, err)

		got, err := db.GetSnapshotCommentByID(ctx, comment.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated content", got.Content)
	})

	t.Run("DeleteSnapshotComment", func(t *testing.T) {
		snapshotID := "snap-del-" + uuid.New().String()[:8]
		comment := models.NewSnapshotComment(org.ID, snapshotID, user.ID, "Delete me")
		require.NoError(t, db.CreateSnapshotComment(ctx, comment))

		err := db.DeleteSnapshotComment(ctx, comment.ID)
		require.NoError(t, err)

		_, err = db.GetSnapshotCommentByID(ctx, comment.ID)
		assert.Error(t, err)
	})

	t.Run("GetSnapshotCommentCounts", func(t *testing.T) {
		snap1 := "snap-cnt1-" + uuid.New().String()[:8]
		snap2 := "snap-cnt2-" + uuid.New().String()[:8]
		snap3 := "snap-cnt3-" + uuid.New().String()[:8]

		// Add 2 comments to snap1, 1 to snap2, 0 to snap3
		require.NoError(t, db.CreateSnapshotComment(ctx, models.NewSnapshotComment(org.ID, snap1, user.ID, "a")))
		require.NoError(t, db.CreateSnapshotComment(ctx, models.NewSnapshotComment(org.ID, snap1, user.ID, "b")))
		require.NoError(t, db.CreateSnapshotComment(ctx, models.NewSnapshotComment(org.ID, snap2, user.ID, "c")))

		counts, err := db.GetSnapshotCommentCounts(ctx, []string{snap1, snap2, snap3}, org.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, counts[snap1])
		assert.Equal(t, 1, counts[snap2])
		assert.Equal(t, 0, counts[snap3]) // not present in map, zero value

		// Empty list should return empty map
		emptyCounts, err := db.GetSnapshotCommentCounts(ctx, []string{}, org.ID)
		require.NoError(t, err)
		assert.Empty(t, emptyCounts)
	})
}

func TestStore_Verification_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "VL Test Org", "vl-test-"+uuid.New().String()[:8])
	repo := createTestRepo(t, db, org.ID, "vl-repo")

	t.Run("UpdateVerification", func(t *testing.T) {
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		require.NoError(t, db.CreateVerification(ctx, v))

		// Verify initial status is running
		got, err := db.GetVerificationByID(ctx, v.ID)
		require.NoError(t, err)
		assert.Equal(t, models.VerificationStatusRunning, got.Status)

		// Mark as passed with details
		v.Pass(&models.VerificationDetails{
			ErrorsFound: []string{},
		})
		err = db.UpdateVerification(ctx, v)
		require.NoError(t, err)

		got, err = db.GetVerificationByID(ctx, v.ID)
		require.NoError(t, err)
		assert.Equal(t, models.VerificationStatusPassed, got.Status)
		assert.NotNil(t, got.CompletedAt)
		assert.NotNil(t, got.DurationMs)
	})

	t.Run("UpdateVerification_Failed", func(t *testing.T) {
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		require.NoError(t, db.CreateVerification(ctx, v))

		v.Fail("repository data corrupted", &models.VerificationDetails{
			ErrorsFound: []string{"pack abc123 is damaged"},
		})
		err := db.UpdateVerification(ctx, v)
		require.NoError(t, err)

		got, err := db.GetVerificationByID(ctx, v.ID)
		require.NoError(t, err)
		assert.Equal(t, models.VerificationStatusFailed, got.Status)
		assert.Equal(t, "repository data corrupted", got.ErrorMessage)
		assert.NotNil(t, got.CompletedAt)
	})

	t.Run("DeleteVerification", func(t *testing.T) {
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		require.NoError(t, db.CreateVerification(ctx, v))

		err := db.DeleteVerification(ctx, v.ID)
		require.NoError(t, err)

		// Soft-deleted verification should not be found
		_, err = db.GetVerificationByID(ctx, v.ID)
		assert.Error(t, err)
	})

	t.Run("DeleteVerification_AlreadyDeleted", func(t *testing.T) {
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		require.NoError(t, db.CreateVerification(ctx, v))

		require.NoError(t, db.DeleteVerification(ctx, v.ID))

		err := db.DeleteVerification(ctx, v.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "verification not found")
	})

	t.Run("GetConsecutiveFailedVerifications", func(t *testing.T) {
		// Use a separate repo so results are isolated
		failRepo := createTestRepo(t, db, org.ID, "fail-repo-"+uuid.New().String()[:8])

		// Create 3 consecutive failed verifications
		for i := 0; i < 3; i++ {
			v := models.NewVerification(failRepo.ID, models.VerificationTypeCheck)
			require.NoError(t, db.CreateVerification(ctx, v))
			v.Fail(fmt.Sprintf("failure %d", i+1), nil)
			require.NoError(t, db.UpdateVerification(ctx, v))
		}

		count, err := db.GetConsecutiveFailedVerifications(ctx, failRepo.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		// Add a passing verification, which should break the consecutive streak
		v := models.NewVerification(failRepo.ID, models.VerificationTypeCheck)
		require.NoError(t, db.CreateVerification(ctx, v))
		v.Pass(nil)
		require.NoError(t, db.UpdateVerification(ctx, v))

		count, err = db.GetConsecutiveFailedVerifications(ctx, failRepo.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Add 2 more failures after the pass
		for i := 0; i < 2; i++ {
			vf := models.NewVerification(failRepo.ID, models.VerificationTypeCheck)
			require.NoError(t, db.CreateVerification(ctx, vf))
			vf.Fail(fmt.Sprintf("post-pass failure %d", i+1), nil)
			require.NoError(t, db.UpdateVerification(ctx, vf))
		}

		count, err = db.GetConsecutiveFailedVerifications(ctx, failRepo.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("GetConsecutiveFailedVerifications_NoVerifications", func(t *testing.T) {
		emptyRepo := createTestRepo(t, db, org.ID, "empty-verif-repo-"+uuid.New().String()[:8])
		count, err := db.GetConsecutiveFailedVerifications(ctx, emptyRepo.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestStore_GetAllAgentsAndSchedules(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	t.Run("GetAllAgents", func(t *testing.T) {
		// Create agents in two different orgs
		org1 := createTestOrg(t, db, "All Agents Org 1", "all-agents-1-"+uuid.New().String()[:8])
		org2 := createTestOrg(t, db, "All Agents Org 2", "all-agents-2-"+uuid.New().String()[:8])

		a1 := createTestAgent(t, db, org1.ID, "agent-org1-"+uuid.New().String()[:8])
		a2 := createTestAgent(t, db, org2.ID, "agent-org2-"+uuid.New().String()[:8])

		allAgents, err := db.GetAllAgents(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allAgents), 2)

		// Verify both agents from different orgs are returned
		foundA1, foundA2 := false, false
		for _, a := range allAgents {
			if a.ID == a1.ID {
				foundA1 = true
			}
			if a.ID == a2.ID {
				foundA2 = true
			}
		}
		assert.True(t, foundA1, "expected agent from org1 to be in GetAllAgents result")
		assert.True(t, foundA2, "expected agent from org2 to be in GetAllAgents result")
	})

	t.Run("GetAllSchedules", func(t *testing.T) {
		org := createTestOrg(t, db, "All Sched Org", "all-sched-"+uuid.New().String()[:8])
		agent := createTestAgent(t, db, org.ID, "all-sched-agent-"+uuid.New().String()[:8])

		sched := models.NewSchedule(agent.ID, "All Sched Test", "0 1 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		allSchedules, err := db.GetAllSchedules(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allSchedules), 1)

		found := false
		for _, s := range allSchedules {
			if s.ID == sched.ID {
				found = true
			}
		}
		assert.True(t, found, "expected schedule to be in GetAllSchedules result")
	})

	t.Run("GetSchedulesByPolicyID", func(t *testing.T) {
		org := createTestOrg(t, db, "Policy Sched Org", "policy-sched-"+uuid.New().String()[:8])
		agent := createTestAgent(t, db, org.ID, "policy-sched-agent-"+uuid.New().String()[:8])

		policy := models.NewPolicy(org.ID, "Test Policy")
		policy.Paths = []string{"/policy/data"}
		policy.CronExpression = "0 3 * * *"
		require.NoError(t, db.CreatePolicy(ctx, policy))

		sched := models.NewSchedule(agent.ID, "Policy Linked Sched", "0 3 * * *", []string{"/policy/data"})
		sched.PolicyID = &policy.ID
		require.NoError(t, db.CreateSchedule(ctx, sched))

		schedules, err := db.GetSchedulesByPolicyID(ctx, policy.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(schedules), 1)

		found := false
		for _, s := range schedules {
			if s.ID == sched.ID {
				found = true
				require.NotNil(t, s.PolicyID)
				assert.Equal(t, policy.ID, *s.PolicyID)
			}
		}
		assert.True(t, found, "expected schedule linked to policy")
	})

	t.Run("GetEnabledSchedulesByOrgID", func(t *testing.T) {
		org := createTestOrg(t, db, "Enabled Sched Org", "enabled-sched-"+uuid.New().String()[:8])
		agent := createTestAgent(t, db, org.ID, "enabled-sched-agent-"+uuid.New().String()[:8])

		// Create an enabled schedule
		enabledSched := models.NewSchedule(agent.ID, "Enabled Sched", "0 1 * * *", []string{"/enabled"})
		require.NoError(t, db.CreateSchedule(ctx, enabledSched))

		// Create a disabled schedule
		disabledSched := models.NewSchedule(agent.ID, "Disabled Sched", "0 2 * * *", []string{"/disabled"})
		disabledSched.Enabled = false
		require.NoError(t, db.CreateSchedule(ctx, disabledSched))

		schedules, err := db.GetEnabledSchedulesByOrgID(ctx, org.ID)
		require.NoError(t, err)

		// Should contain the enabled schedule but not the disabled one
		foundEnabled, foundDisabled := false, false
		for _, s := range schedules {
			if s.ID == enabledSched.ID {
				foundEnabled = true
			}
			if s.ID == disabledSched.ID {
				foundDisabled = true
			}
		}
		assert.True(t, foundEnabled, "expected enabled schedule to be returned")
		assert.False(t, foundDisabled, "expected disabled schedule to NOT be returned")
	})

	t.Run("GetSchedulesByPolicyID_Empty", func(t *testing.T) {
		// Query for a policy with no linked schedules
		schedules, err := db.GetSchedulesByPolicyID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Empty(t, schedules)
	})
}

func TestStore_ReportSchedules(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Report Sched Org", "rpt-sched-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		sched := models.NewReportSchedule(org.ID, "Daily Report", models.ReportFrequencyDaily, []string{"admin@test.com", "ops@test.com"})
		err := db.CreateReportSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetReportScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, sched.ID, got.ID)
		assert.Equal(t, org.ID, got.OrgID)
		assert.Equal(t, "Daily Report", got.Name)
		assert.Equal(t, models.ReportFrequencyDaily, got.Frequency)
		assert.Equal(t, []string{"admin@test.com", "ops@test.com"}, got.Recipients)
		assert.Nil(t, got.ChannelID)
		assert.Equal(t, "UTC", got.Timezone)
		assert.True(t, got.Enabled)
		assert.Nil(t, got.LastSentAt)
	})

	t.Run("GetReportSchedulesByOrgID", func(t *testing.T) {
		schedules, err := db.GetReportSchedulesByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(schedules), 1)
	})

	t.Run("GetEnabledReportSchedules", func(t *testing.T) {
		enabled, err := db.GetEnabledReportSchedules(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(enabled), 1)
		for _, s := range enabled {
			assert.True(t, s.Enabled)
		}
	})

	t.Run("UpdateReportSchedule", func(t *testing.T) {
		sched := models.NewReportSchedule(org.ID, "Update Me", models.ReportFrequencyWeekly, []string{"user@test.com"})
		require.NoError(t, db.CreateReportSchedule(ctx, sched))

		sched.Name = "Updated Report"
		sched.Frequency = models.ReportFrequencyMonthly
		sched.Recipients = []string{"new@test.com", "another@test.com"}
		sched.Timezone = "America/New_York"
		sched.Enabled = false
		err := db.UpdateReportSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetReportScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Report", got.Name)
		assert.Equal(t, models.ReportFrequencyMonthly, got.Frequency)
		assert.Equal(t, []string{"new@test.com", "another@test.com"}, got.Recipients)
		assert.Equal(t, "America/New_York", got.Timezone)
		assert.False(t, got.Enabled)
	})

	t.Run("UpdateReportScheduleLastSent", func(t *testing.T) {
		sched := models.NewReportSchedule(org.ID, "LastSent Report", models.ReportFrequencyDaily, []string{"test@test.com"})
		require.NoError(t, db.CreateReportSchedule(ctx, sched))

		lastSent := time.Now().Add(-1 * time.Hour)
		err := db.UpdateReportScheduleLastSent(ctx, sched.ID, lastSent)
		require.NoError(t, err)

		got, err := db.GetReportScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		require.NotNil(t, got.LastSentAt)
		assert.WithinDuration(t, lastSent, *got.LastSentAt, 2*time.Second)
	})

	t.Run("DeleteReportSchedule", func(t *testing.T) {
		sched := models.NewReportSchedule(org.ID, "Delete Me", models.ReportFrequencyWeekly, []string{"del@test.com"})
		require.NoError(t, db.CreateReportSchedule(ctx, sched))

		err := db.DeleteReportSchedule(ctx, sched.ID)
		require.NoError(t, err)

		_, err = db.GetReportScheduleByID(ctx, sched.ID)
		assert.Error(t, err)
	})
}

func TestStore_ReportHistory(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Report Hist Org", "rpt-hist-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		periodStart := time.Now().Add(-24 * time.Hour)
		periodEnd := time.Now()
		history := models.NewReportHistory(org.ID, nil, "daily_summary", periodStart, periodEnd, []string{"user@test.com"})
		err := db.CreateReportHistory(ctx, history)
		require.NoError(t, err)

		got, err := db.GetReportHistoryByID(ctx, history.ID)
		require.NoError(t, err)
		assert.Equal(t, history.ID, got.ID)
		assert.Equal(t, org.ID, got.OrgID)
		assert.Nil(t, got.ScheduleID)
		assert.Equal(t, "daily_summary", got.ReportType)
		assert.Equal(t, models.ReportStatusSent, got.Status)
		assert.Equal(t, []string{"user@test.com"}, got.Recipients)
	})

	t.Run("CreateWithScheduleID", func(t *testing.T) {
		sched := models.NewReportSchedule(org.ID, "Hist Link", models.ReportFrequencyDaily, []string{"a@test.com"})
		require.NoError(t, db.CreateReportSchedule(ctx, sched))

		periodStart := time.Now().Add(-24 * time.Hour)
		periodEnd := time.Now()
		history := models.NewReportHistory(org.ID, &sched.ID, "daily_summary", periodStart, periodEnd, []string{"a@test.com"})
		err := db.CreateReportHistory(ctx, history)
		require.NoError(t, err)

		got, err := db.GetReportHistoryByID(ctx, history.ID)
		require.NoError(t, err)
		require.NotNil(t, got.ScheduleID)
		assert.Equal(t, sched.ID, *got.ScheduleID)
	})

	t.Run("GetReportHistoryByOrgID", func(t *testing.T) {
		history, err := db.GetReportHistoryByOrgID(ctx, org.ID, 100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(history), 1)
	})
}

func TestStore_AgentGroups(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "AgentGroup Org", "agrp-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Production", "Production servers", "#FF5733")
		err := db.CreateAgentGroup(ctx, group)
		require.NoError(t, err)

		got, err := db.GetAgentGroupByID(ctx, group.ID)
		require.NoError(t, err)
		assert.Equal(t, group.ID, got.ID)
		assert.Equal(t, org.ID, got.OrgID)
		assert.Equal(t, "Production", got.Name)
		assert.Equal(t, "Production servers", got.Description)
		assert.Equal(t, "#FF5733", got.Color)
	})

	t.Run("GetAgentGroupsByOrgID", func(t *testing.T) {
		groups, err := db.GetAgentGroupsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("UpdateAgentGroup", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Old Group", "Old desc", "#000000")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		group.Name = "Updated Group"
		group.Description = "Updated description"
		group.Color = "#FFFFFF"
		err := db.UpdateAgentGroup(ctx, group)
		require.NoError(t, err)

		got, err := db.GetAgentGroupByID(ctx, group.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Group", got.Name)
		assert.Equal(t, "Updated description", got.Description)
		assert.Equal(t, "#FFFFFF", got.Color)
	})

	t.Run("DeleteAgentGroup", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Delete Group", "Will be deleted", "#111111")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		err := db.DeleteAgentGroup(ctx, group.ID)
		require.NoError(t, err)

		_, err = db.GetAgentGroupByID(ctx, group.ID)
		assert.Error(t, err)
	})

	t.Run("AddAgentToGroupAndGetMembers", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Members Group", "Test members", "#222222")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		agent1 := createTestAgent(t, db, org.ID, "grp-agent-1-"+uuid.New().String()[:8])
		agent2 := createTestAgent(t, db, org.ID, "grp-agent-2-"+uuid.New().String()[:8])

		err := db.AddAgentToGroup(ctx, group.ID, agent1.ID)
		require.NoError(t, err)
		err = db.AddAgentToGroup(ctx, group.ID, agent2.ID)
		require.NoError(t, err)

		members, err := db.GetAgentGroupMembers(ctx, group.ID)
		require.NoError(t, err)
		assert.Len(t, members, 2)
	})

	t.Run("RemoveAgentFromGroup", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Remove Group", "Test removal", "#333333")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		agent := createTestAgent(t, db, org.ID, "rmv-agent-"+uuid.New().String()[:8])
		require.NoError(t, db.AddAgentToGroup(ctx, group.ID, agent.ID))

		err := db.RemoveAgentFromGroup(ctx, group.ID, agent.ID)
		require.NoError(t, err)

		members, err := db.GetAgentGroupMembers(ctx, group.ID)
		require.NoError(t, err)
		assert.Len(t, members, 0)
	})

	t.Run("GetGroupsByAgentID", func(t *testing.T) {
		group1 := models.NewAgentGroup(org.ID, "Agent Groups A", "Group A", "#444444")
		group2 := models.NewAgentGroup(org.ID, "Agent Groups B", "Group B", "#555555")
		require.NoError(t, db.CreateAgentGroup(ctx, group1))
		require.NoError(t, db.CreateAgentGroup(ctx, group2))

		agent := createTestAgent(t, db, org.ID, "multi-grp-agent-"+uuid.New().String()[:8])
		require.NoError(t, db.AddAgentToGroup(ctx, group1.ID, agent.ID))
		require.NoError(t, db.AddAgentToGroup(ctx, group2.ID, agent.ID))

		groups, err := db.GetGroupsByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Len(t, groups, 2)
	})

	t.Run("GetAgentsWithGroupsByOrgID", func(t *testing.T) {
		agentsWithGroups, err := db.GetAgentsWithGroupsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(agentsWithGroups), 1)
	})

	t.Run("GetAgentsByGroupID", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "IDList Group", "ID list test", "#666666")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		agent := createTestAgent(t, db, org.ID, "idlist-agent-"+uuid.New().String()[:8])
		require.NoError(t, db.AddAgentToGroup(ctx, group.ID, agent.ID))

		agentIDs, err := db.GetAgentsByGroupID(ctx, group.ID)
		require.NoError(t, err)
		assert.Len(t, agentIDs, 1)
		assert.Equal(t, agent.ID, agentIDs[0])
	})

	t.Run("GetSchedulesByAgentGroupID", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Sched Group", "Schedules test", "#777777")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		grpAgent := createTestAgent(t, db, org.ID, "sched-grp-agent-"+uuid.New().String()[:8])
		require.NoError(t, db.AddAgentToGroup(ctx, group.ID, grpAgent.ID))

		sched := models.NewSchedule(grpAgent.ID, "Group Sched", "0 1 * * *", []string{"/data"})
		sched.AgentGroupID = &group.ID
		require.NoError(t, db.CreateSchedule(ctx, sched))

		schedules, err := db.GetSchedulesByAgentGroupID(ctx, group.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(schedules), 1)
	})
}

func TestStore_Onboarding(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Onboard Org", "onboard-"+uuid.New().String()[:8])

	t.Run("GetOnboardingProgress_NotFound", func(t *testing.T) {
		_, err := db.GetOnboardingProgress(ctx, org.ID)
		assert.Error(t, err)
	})

	t.Run("GetOrCreateOnboardingProgress_Create", func(t *testing.T) {
		progress, err := db.GetOrCreateOnboardingProgress(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, progress.OrgID)
		assert.Equal(t, models.OnboardingStepWelcome, progress.CurrentStep)
		assert.Empty(t, progress.CompletedSteps)
		assert.False(t, progress.Skipped)
		assert.Nil(t, progress.CompletedAt)
	})

	t.Run("GetOrCreateOnboardingProgress_ReturnExisting", func(t *testing.T) {
		first, err := db.GetOrCreateOnboardingProgress(ctx, org.ID)
		require.NoError(t, err)

		second, err := db.GetOrCreateOnboardingProgress(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
	})

	t.Run("UpdateOnboardingProgress", func(t *testing.T) {
		progress, err := db.GetOrCreateOnboardingProgress(ctx, org.ID)
		require.NoError(t, err)

		progress.CompleteStep(models.OnboardingStepWelcome)
		err = db.UpdateOnboardingProgress(ctx, progress)
		require.NoError(t, err)

		got, err := db.GetOnboardingProgress(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, models.OnboardingStepOrganization, got.CurrentStep)
		assert.Contains(t, got.CompletedSteps, models.OnboardingStepWelcome)
	})

	t.Run("SkipOnboarding", func(t *testing.T) {
		skipOrg := createTestOrg(t, db, "Skip Org", "skip-"+uuid.New().String()[:8])
		_, err := db.GetOrCreateOnboardingProgress(ctx, skipOrg.ID)
		require.NoError(t, err)

		err = db.SkipOnboarding(ctx, skipOrg.ID)
		require.NoError(t, err)

		got, err := db.GetOnboardingProgress(ctx, skipOrg.ID)
		require.NoError(t, err)
		assert.True(t, got.Skipped)
		assert.NotNil(t, got.CompletedAt)
	})
}

func TestStore_StoragePricing(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Pricing Org", "pricing-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByOrgID", func(t *testing.T) {
		pricing := models.NewStoragePricing(org.ID, "s3")
		pricing.StoragePerGBMonth = 0.023
		pricing.EgressPerGB = 0.09
		pricing.OperationsPerK = 0.005
		pricing.ProviderName = "AWS S3"
		pricing.ProviderDescription = "Amazon S3 Standard"
		err := db.CreateStoragePricing(ctx, pricing)
		require.NoError(t, err)

		pricings, err := db.GetStoragePricingByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(pricings), 1)

		found := false
		for _, p := range pricings {
			if p.ID == pricing.ID {
				found = true
				assert.Equal(t, "s3", p.RepositoryType)
				assert.InDelta(t, 0.023, p.StoragePerGBMonth, 0.0001)
				assert.InDelta(t, 0.09, p.EgressPerGB, 0.0001)
				assert.InDelta(t, 0.005, p.OperationsPerK, 0.0001)
				assert.Equal(t, "AWS S3", p.ProviderName)
				assert.Equal(t, "Amazon S3 Standard", p.ProviderDescription)
			}
		}
		assert.True(t, found)
	})

	t.Run("GetStoragePricingByType", func(t *testing.T) {
		got, err := db.GetStoragePricingByType(ctx, org.ID, "s3")
		require.NoError(t, err)
		assert.Equal(t, "s3", got.RepositoryType)
		assert.Equal(t, org.ID, got.OrgID)
	})

	t.Run("UpdateStoragePricing", func(t *testing.T) {
		pricing := models.NewStoragePricing(org.ID, "gcs")
		pricing.StoragePerGBMonth = 0.020
		pricing.ProviderName = "Google Cloud Storage"
		require.NoError(t, db.CreateStoragePricing(ctx, pricing))

		pricing.StoragePerGBMonth = 0.026
		pricing.ProviderName = "GCS Updated"
		err := db.UpdateStoragePricing(ctx, pricing)
		require.NoError(t, err)

		got, err := db.GetStoragePricingByType(ctx, org.ID, "gcs")
		require.NoError(t, err)
		assert.InDelta(t, 0.026, got.StoragePerGBMonth, 0.0001)
		assert.Equal(t, "GCS Updated", got.ProviderName)
	})

	t.Run("DeleteStoragePricing", func(t *testing.T) {
		pricing := models.NewStoragePricing(org.ID, "azure-del")
		pricing.StoragePerGBMonth = 0.018
		require.NoError(t, db.CreateStoragePricing(ctx, pricing))

		err := db.DeleteStoragePricing(ctx, pricing.ID)
		require.NoError(t, err)

		_, err = db.GetStoragePricingByType(ctx, org.ID, "azure-del")
		assert.Error(t, err)
	})
}

func TestStore_CostEstimates(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Cost Est Org", "cost-est-"+uuid.New().String()[:8])
	repo := createTestRepo(t, db, org.ID, "cost-est-repo")

	t.Run("CreateAndGetLatest", func(t *testing.T) {
		est := models.NewCostEstimateRecord(org.ID, repo.ID)
		est.StorageSizeBytes = 1024 * 1024 * 1024 // 1 GB
		est.MonthlyCost = 0.023
		est.YearlyCost = 0.276
		est.CostPerGB = 0.023
		err := db.CreateCostEstimate(ctx, est)
		require.NoError(t, err)

		latest, err := db.GetLatestCostEstimates(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(latest), 1)

		found := false
		for _, e := range latest {
			if e.ID == est.ID {
				found = true
				assert.Equal(t, repo.ID, e.RepositoryID)
				assert.Equal(t, int64(1024*1024*1024), e.StorageSizeBytes)
				assert.InDelta(t, 0.023, e.MonthlyCost, 0.0001)
				assert.InDelta(t, 0.276, e.YearlyCost, 0.0001)
				assert.InDelta(t, 0.023, e.CostPerGB, 0.0001)
			}
		}
		assert.True(t, found)
	})

	t.Run("GetCostEstimateHistory", func(t *testing.T) {
		// Create a second estimate
		est2 := models.NewCostEstimateRecord(org.ID, repo.ID)
		est2.StorageSizeBytes = 2 * 1024 * 1024 * 1024
		est2.MonthlyCost = 0.046
		est2.YearlyCost = 0.552
		est2.CostPerGB = 0.023
		require.NoError(t, db.CreateCostEstimate(ctx, est2))

		history, err := db.GetCostEstimateHistory(ctx, repo.ID, 30)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(history), 2)
	})
}

func TestStore_CostAlerts(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Cost Alert Org", "cost-alert-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		alert := models.NewCostAlert(org.ID, "Monthly Budget Alert", 100.0)
		err := db.CreateCostAlert(ctx, alert)
		require.NoError(t, err)

		got, err := db.GetCostAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, alert.ID, got.ID)
		assert.Equal(t, org.ID, got.OrgID)
		assert.Equal(t, "Monthly Budget Alert", got.Name)
		assert.InDelta(t, 100.0, got.MonthlyThreshold, 0.01)
		assert.True(t, got.Enabled)
		assert.True(t, got.NotifyOnExceed)
		assert.False(t, got.NotifyOnForecast)
		assert.Equal(t, 3, got.ForecastMonths)
		assert.Nil(t, got.LastTriggeredAt)
	})

	t.Run("GetCostAlertsByOrgID", func(t *testing.T) {
		alerts, err := db.GetCostAlertsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 1)
	})

	t.Run("GetEnabledCostAlerts", func(t *testing.T) {
		enabled, err := db.GetEnabledCostAlerts(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(enabled), 1)
		for _, a := range enabled {
			assert.True(t, a.Enabled)
		}
	})

	t.Run("UpdateCostAlert", func(t *testing.T) {
		alert := models.NewCostAlert(org.ID, "Update Alert", 50.0)
		require.NoError(t, db.CreateCostAlert(ctx, alert))

		alert.Name = "Updated Alert"
		alert.MonthlyThreshold = 200.0
		alert.Enabled = false
		alert.NotifyOnForecast = true
		alert.ForecastMonths = 6
		err := db.UpdateCostAlert(ctx, alert)
		require.NoError(t, err)

		got, err := db.GetCostAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Alert", got.Name)
		assert.InDelta(t, 200.0, got.MonthlyThreshold, 0.01)
		assert.False(t, got.Enabled)
		assert.True(t, got.NotifyOnForecast)
		assert.Equal(t, 6, got.ForecastMonths)
	})

	t.Run("UpdateCostAlertTriggered", func(t *testing.T) {
		alert := models.NewCostAlert(org.ID, "Trigger Alert", 75.0)
		require.NoError(t, db.CreateCostAlert(ctx, alert))

		err := db.UpdateCostAlertTriggered(ctx, alert.ID)
		require.NoError(t, err)

		got, err := db.GetCostAlertByID(ctx, alert.ID)
		require.NoError(t, err)
		require.NotNil(t, got.LastTriggeredAt)
		assert.WithinDuration(t, time.Now(), *got.LastTriggeredAt, 5*time.Second)
	})

	t.Run("DeleteCostAlert", func(t *testing.T) {
		alert := models.NewCostAlert(org.ID, "Delete Alert", 25.0)
		require.NoError(t, db.CreateCostAlert(ctx, alert))

		err := db.DeleteCostAlert(ctx, alert.ID)
		require.NoError(t, err)

		_, err = db.GetCostAlertByID(ctx, alert.ID)
		assert.Error(t, err)
	})
}

func TestStore_SSOGroupMappings(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "SSO Org", "sso-"+uuid.New().String()[:8])

	t.Run("CreateAndGetByID", func(t *testing.T) {
		mapping := models.NewSSOGroupMapping(org.ID, "engineering", models.OrgRoleAdmin)
		err := db.CreateSSOGroupMapping(ctx, mapping)
		require.NoError(t, err)

		got, err := db.GetSSOGroupMappingByID(ctx, mapping.ID)
		require.NoError(t, err)
		assert.Equal(t, mapping.ID, got.ID)
		assert.Equal(t, org.ID, got.OrgID)
		assert.Equal(t, "engineering", got.OIDCGroupName)
		assert.Equal(t, models.OrgRoleAdmin, got.Role)
		assert.False(t, got.AutoCreateOrg)
	})

	t.Run("GetSSOGroupMappingsByOrgID", func(t *testing.T) {
		mappings, err := db.GetSSOGroupMappingsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(mappings), 1)
	})

	t.Run("GetSSOGroupMappingsByGroupNames", func(t *testing.T) {
		mapping2 := models.NewSSOGroupMapping(org.ID, "devops-"+uuid.New().String()[:8], models.OrgRoleMember)
		require.NoError(t, db.CreateSSOGroupMapping(ctx, mapping2))

		mappings, err := db.GetSSOGroupMappingsByGroupNames(ctx, []string{mapping2.OIDCGroupName})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(mappings), 1)

		found := false
		for _, m := range mappings {
			if m.OIDCGroupName == mapping2.OIDCGroupName {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("UpdateSSOGroupMapping", func(t *testing.T) {
		mapping := models.NewSSOGroupMapping(org.ID, "update-grp-"+uuid.New().String()[:8], models.OrgRoleMember)
		require.NoError(t, db.CreateSSOGroupMapping(ctx, mapping))

		mapping.Role = models.OrgRoleAdmin
		mapping.AutoCreateOrg = true
		err := db.UpdateSSOGroupMapping(ctx, mapping)
		require.NoError(t, err)

		got, err := db.GetSSOGroupMappingByID(ctx, mapping.ID)
		require.NoError(t, err)
		assert.Equal(t, models.OrgRoleAdmin, got.Role)
		assert.True(t, got.AutoCreateOrg)
	})

	t.Run("DeleteSSOGroupMapping", func(t *testing.T) {
		mapping := models.NewSSOGroupMapping(org.ID, "del-grp-"+uuid.New().String()[:8], models.OrgRoleMember)
		require.NoError(t, db.CreateSSOGroupMapping(ctx, mapping))

		err := db.DeleteSSOGroupMapping(ctx, mapping.ID)
		require.NoError(t, err)

		_, err = db.GetSSOGroupMappingByID(ctx, mapping.ID)
		assert.Error(t, err)
	})

	t.Run("UserSSOGroups", func(t *testing.T) {
		user := createTestUser(t, db, org.ID, "sso-user-"+uuid.New().String()[:8]+"@test.com", "SSO User")

		// Initially no groups
		_, err := db.GetUserSSOGroups(ctx, user.ID)
		assert.Error(t, err)

		// Upsert groups
		err = db.UpsertUserSSOGroups(ctx, user.ID, []string{"engineering", "devops"})
		require.NoError(t, err)

		got, err := db.GetUserSSOGroups(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.UserID)
		assert.Equal(t, []string{"engineering", "devops"}, got.OIDCGroups)

		// Upsert again to update
		err = db.UpsertUserSSOGroups(ctx, user.ID, []string{"engineering", "devops", "admins"})
		require.NoError(t, err)

		got, err = db.GetUserSSOGroups(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"engineering", "devops", "admins"}, got.OIDCGroups)
	})

	t.Run("OrganizationSSOSettings", func(t *testing.T) {
		// Get default settings
		_, autoCreate, err := db.GetOrganizationSSOSettings(ctx, org.ID)
		require.NoError(t, err)
		assert.False(t, autoCreate)

		// Update settings
		memberRole := "member"
		err = db.UpdateOrganizationSSOSettings(ctx, org.ID, &memberRole, true)
		require.NoError(t, err)

		defaultRole, autoCreate, err := db.GetOrganizationSSOSettings(ctx, org.ID)
		require.NoError(t, err)
		require.NotNil(t, defaultRole)
		assert.Equal(t, "member", *defaultRole)
		assert.True(t, autoCreate)
	})

	t.Run("UpdateMembershipRole", func(t *testing.T) {
		user := createTestUser(t, db, org.ID, "role-user-"+uuid.New().String()[:8]+"@test.com", "Role User")

		membership, err := db.GetMembershipByUserAndOrg(ctx, user.ID, org.ID)
		require.NoError(t, err)

		err = db.UpdateMembershipRole(ctx, membership.ID, models.OrgRoleMember)
		require.NoError(t, err)

		got, err := db.GetMembershipByUserAndOrg(ctx, user.ID, org.ID)
		require.NoError(t, err)
		assert.Equal(t, models.OrgRoleMember, got.Role)
	})
}

func TestStore_DRRunbooks(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "DR Runbook Org", "dr-runbook-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "dr-agent")
	sched := models.NewSchedule(agent.ID, "DR Sched", "0 2 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	t.Run("CreateAndGetByID", func(t *testing.T) {
		runbook := models.NewDRRunbook(org.ID, "Primary DR Plan")
		runbook.Description = "Main disaster recovery plan"
		runbook.CredentialsLocation = "vault://secrets/dr"
		rtoMins := 60
		rpoMins := 15
		runbook.RecoveryTimeObjectiveMins = &rtoMins
		runbook.RecoveryPointObjectiveMins = &rpoMins
		err := db.CreateDRRunbook(ctx, runbook)
		require.NoError(t, err)

		got, err := db.GetDRRunbookByID(ctx, runbook.ID)
		require.NoError(t, err)
		assert.Equal(t, runbook.ID, got.ID)
		assert.Equal(t, "Primary DR Plan", got.Name)
		assert.Equal(t, "Main disaster recovery plan", got.Description)
		assert.Equal(t, models.DRRunbookStatusDraft, got.Status)
		assert.Equal(t, "vault://secrets/dr", got.CredentialsLocation)
		require.NotNil(t, got.RecoveryTimeObjectiveMins)
		assert.Equal(t, 60, *got.RecoveryTimeObjectiveMins)
		require.NotNil(t, got.RecoveryPointObjectiveMins)
		assert.Equal(t, 15, *got.RecoveryPointObjectiveMins)
		assert.Empty(t, got.Steps)
		assert.Empty(t, got.Contacts)
	})

	t.Run("GetByOrgID", func(t *testing.T) {
		runbooks, err := db.GetDRRunbooksByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(runbooks), 1)
	})

	t.Run("GetByScheduleID", func(t *testing.T) {
		runbook := models.NewDRRunbook(org.ID, "Schedule Linked Runbook")
		runbook.ScheduleID = &sched.ID
		err := db.CreateDRRunbook(ctx, runbook)
		require.NoError(t, err)

		got, err := db.GetDRRunbookByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, runbook.ID, got.ID)
		require.NotNil(t, got.ScheduleID)
		assert.Equal(t, sched.ID, *got.ScheduleID)
	})

	t.Run("Update", func(t *testing.T) {
		runbook := models.NewDRRunbook(org.ID, "Update Runbook")
		err := db.CreateDRRunbook(ctx, runbook)
		require.NoError(t, err)

		runbook.Name = "Updated DR Plan"
		runbook.Description = "Updated description"
		runbook.Steps = []models.DRRunbookStep{
			{
				Order:       1,
				Title:       "Stop services",
				Description: "Stop all running services",
				Type:        models.DRRunbookStepTypeManual,
				Command:     "systemctl stop all",
				Expected:    "Services stopped",
			},
			{
				Order:       2,
				Title:       "Restore data",
				Description: "Restore from latest snapshot",
				Type:        models.DRRunbookStepTypeRestore,
			},
		}
		runbook.Contacts = []models.DRRunbookContact{
			{
				Name:   "John Doe",
				Role:   "SRE Lead",
				Email:  "john@example.com",
				Phone:  "+1234567890",
				Notify: true,
			},
		}
		runbook.Status = models.DRRunbookStatusActive
		err = db.UpdateDRRunbook(ctx, runbook)
		require.NoError(t, err)

		got, err := db.GetDRRunbookByID(ctx, runbook.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated DR Plan", got.Name)
		assert.Equal(t, "Updated description", got.Description)
		assert.Equal(t, models.DRRunbookStatusActive, got.Status)
		require.Len(t, got.Steps, 2)
		assert.Equal(t, "Stop services", got.Steps[0].Title)
		assert.Equal(t, models.DRRunbookStepTypeManual, got.Steps[0].Type)
		assert.Equal(t, "systemctl stop all", got.Steps[0].Command)
		assert.Equal(t, "Restore data", got.Steps[1].Title)
		assert.Equal(t, models.DRRunbookStepTypeRestore, got.Steps[1].Type)
		require.Len(t, got.Contacts, 1)
		assert.Equal(t, "John Doe", got.Contacts[0].Name)
		assert.Equal(t, "SRE Lead", got.Contacts[0].Role)
		assert.Equal(t, "john@example.com", got.Contacts[0].Email)
		assert.True(t, got.Contacts[0].Notify)
	})

	t.Run("Delete", func(t *testing.T) {
		runbook := models.NewDRRunbook(org.ID, "Delete Runbook")
		err := db.CreateDRRunbook(ctx, runbook)
		require.NoError(t, err)

		err = db.DeleteDRRunbook(ctx, runbook.ID)
		require.NoError(t, err)

		_, err = db.GetDRRunbookByID(ctx, runbook.ID)
		assert.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetDRRunbookByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_DRTests(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "DR Test Org", "dr-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "dr-test-agent")
	sched := models.NewSchedule(agent.ID, "DR Test Sched", "0 2 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	runbook := models.NewDRRunbook(org.ID, "Test Runbook")
	require.NoError(t, db.CreateDRRunbook(ctx, runbook))

	t.Run("CreateAndGetByID", func(t *testing.T) {
		drTest := models.NewDRTest(runbook.ID)
		err := db.CreateDRTest(ctx, drTest)
		require.NoError(t, err)

		got, err := db.GetDRTestByID(ctx, drTest.ID)
		require.NoError(t, err)
		assert.Equal(t, drTest.ID, got.ID)
		assert.Equal(t, runbook.ID, got.RunbookID)
		assert.Equal(t, models.DRTestStatusScheduled, got.Status)
		assert.Nil(t, got.ScheduleID)
		assert.Nil(t, got.AgentID)
		assert.Empty(t, got.SnapshotID)
	})

	t.Run("GetByRunbookID", func(t *testing.T) {
		tests, err := db.GetDRTestsByRunbookID(ctx, runbook.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tests), 1)
	})

	t.Run("GetByOrgID", func(t *testing.T) {
		tests, err := db.GetDRTestsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tests), 1)
	})

	t.Run("GetLatestByRunbookID", func(t *testing.T) {
		drTest2 := models.NewDRTest(runbook.ID)
		drTest2.Notes = "second test"
		err := db.CreateDRTest(ctx, drTest2)
		require.NoError(t, err)

		latest, err := db.GetLatestDRTestByRunbookID(ctx, runbook.ID)
		require.NoError(t, err)
		assert.Equal(t, drTest2.ID, latest.ID)
	})

	t.Run("Update", func(t *testing.T) {
		drTest := models.NewDRTest(runbook.ID)
		drTest.SetSchedule(sched.ID)
		drTest.SetAgent(agent.ID)
		err := db.CreateDRTest(ctx, drTest)
		require.NoError(t, err)

		drTest.Start()
		err = db.UpdateDRTest(ctx, drTest)
		require.NoError(t, err)

		got, err := db.GetDRTestByID(ctx, drTest.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DRTestStatusRunning, got.Status)
		assert.NotNil(t, got.StartedAt)
		require.NotNil(t, got.ScheduleID)
		assert.Equal(t, sched.ID, *got.ScheduleID)
		require.NotNil(t, got.AgentID)
		assert.Equal(t, agent.ID, *got.AgentID)

		drTest.Complete("snap-dr-001", 2048, 120, true)
		err = db.UpdateDRTest(ctx, drTest)
		require.NoError(t, err)

		got, err = db.GetDRTestByID(ctx, drTest.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DRTestStatusCompleted, got.Status)
		assert.NotNil(t, got.CompletedAt)
		assert.Equal(t, "snap-dr-001", got.SnapshotID)
		require.NotNil(t, got.RestoreSizeBytes)
		assert.Equal(t, int64(2048), *got.RestoreSizeBytes)
		require.NotNil(t, got.RestoreDurationSeconds)
		assert.Equal(t, 120, *got.RestoreDurationSeconds)
		require.NotNil(t, got.VerificationPassed)
		assert.True(t, *got.VerificationPassed)
	})

	t.Run("FailedTest", func(t *testing.T) {
		drTest := models.NewDRTest(runbook.ID)
		err := db.CreateDRTest(ctx, drTest)
		require.NoError(t, err)

		drTest.Start()
		require.NoError(t, db.UpdateDRTest(ctx, drTest))

		drTest.Fail("restore verification failed")
		require.NoError(t, db.UpdateDRTest(ctx, drTest))

		got, err := db.GetDRTestByID(ctx, drTest.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DRTestStatusFailed, got.Status)
		assert.Equal(t, "restore verification failed", got.ErrorMessage)
		require.NotNil(t, got.VerificationPassed)
		assert.False(t, *got.VerificationPassed)
	})

	t.Run("Delete", func(t *testing.T) {
		drTest := models.NewDRTest(runbook.ID)
		err := db.CreateDRTest(ctx, drTest)
		require.NoError(t, err)

		err = db.DeleteDRTest(ctx, drTest.ID)
		require.NoError(t, err)

		_, err = db.GetDRTestByID(ctx, drTest.ID)
		assert.Error(t, err)
	})

	t.Run("DeleteAlreadyDeleted", func(t *testing.T) {
		drTest := models.NewDRTest(runbook.ID)
		require.NoError(t, db.CreateDRTest(ctx, drTest))
		require.NoError(t, db.DeleteDRTest(ctx, drTest.ID))

		err := db.DeleteDRTest(ctx, drTest.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DR test not found")
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetDRTestByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_DRTestSchedules(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "DR Sched Org", "dr-sched-"+uuid.New().String()[:8])

	runbook := models.NewDRRunbook(org.ID, "Sched Runbook")
	require.NoError(t, db.CreateDRRunbook(ctx, runbook))

	t.Run("CreateAndGetByRunbookID", func(t *testing.T) {
		testSched := models.NewDRTestSchedule(runbook.ID, "0 3 * * 0")
		err := db.CreateDRTestSchedule(ctx, testSched)
		require.NoError(t, err)

		schedules, err := db.GetDRTestSchedulesByRunbookID(ctx, runbook.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(schedules), 1)

		found := false
		for _, s := range schedules {
			if s.ID == testSched.ID {
				found = true
				assert.Equal(t, "0 3 * * 0", s.CronExpression)
				assert.True(t, s.Enabled)
				assert.Equal(t, runbook.ID, s.RunbookID)
			}
		}
		assert.True(t, found)
	})

	t.Run("GetEnabledSchedules", func(t *testing.T) {
		enabled, err := db.GetEnabledDRTestSchedules(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(enabled), 1)
		for _, s := range enabled {
			assert.True(t, s.Enabled)
		}
	})

	t.Run("Update", func(t *testing.T) {
		testSched := models.NewDRTestSchedule(runbook.ID, "0 4 * * 1")
		err := db.CreateDRTestSchedule(ctx, testSched)
		require.NoError(t, err)

		testSched.CronExpression = "0 5 * * 2"
		testSched.Enabled = false
		now := time.Now()
		testSched.LastRunAt = &now
		nextRun := now.Add(7 * 24 * time.Hour)
		testSched.NextRunAt = &nextRun
		err = db.UpdateDRTestSchedule(ctx, testSched)
		require.NoError(t, err)

		schedules, err := db.GetDRTestSchedulesByRunbookID(ctx, runbook.ID)
		require.NoError(t, err)

		var got *models.DRTestSchedule
		for _, s := range schedules {
			if s.ID == testSched.ID {
				got = s
				break
			}
		}
		require.NotNil(t, got)
		assert.Equal(t, "0 5 * * 2", got.CronExpression)
		assert.False(t, got.Enabled)
		assert.NotNil(t, got.LastRunAt)
		assert.NotNil(t, got.NextRunAt)
	})

	t.Run("Delete", func(t *testing.T) {
		testSched := models.NewDRTestSchedule(runbook.ID, "0 6 * * 3")
		err := db.CreateDRTestSchedule(ctx, testSched)
		require.NoError(t, err)

		err = db.DeleteDRTestSchedule(ctx, testSched.ID)
		require.NoError(t, err)

		schedules, err := db.GetDRTestSchedulesByRunbookID(ctx, runbook.ID)
		require.NoError(t, err)
		for _, s := range schedules {
			assert.NotEqual(t, testSched.ID, s.ID)
		}
	})
}

func TestStore_DRStatus(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "DR Status Org", "dr-status-"+uuid.New().String()[:8])

	t.Run("EmptyOrg", func(t *testing.T) {
		status, err := db.GetDRStatus(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, status.TotalRunbooks)
		assert.Equal(t, 0, status.ActiveRunbooks)
	})

	t.Run("WithRunbooksAndTests", func(t *testing.T) {
		runbook1 := models.NewDRRunbook(org.ID, "Active Runbook")
		runbook1.Status = models.DRRunbookStatusActive
		require.NoError(t, db.CreateDRRunbook(ctx, runbook1))

		runbook2 := models.NewDRRunbook(org.ID, "Draft Runbook")
		require.NoError(t, db.CreateDRRunbook(ctx, runbook2))

		drTest := models.NewDRTest(runbook1.ID)
		drTest.Complete("snap-status", 1024, 60, true)
		require.NoError(t, db.CreateDRTest(ctx, drTest))

		status, err := db.GetDRStatus(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, status.TotalRunbooks)
		assert.Equal(t, 1, status.ActiveRunbooks)
		assert.GreaterOrEqual(t, status.TestsLast30Days, 1)
	})
}

func TestStore_Tags(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Tag Test Org", "tag-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "tag-agent")
	repo := createTestRepo(t, db, org.ID, "tag-repo")
	sched := models.NewSchedule(agent.ID, "Tag Sched", "0 1 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	t.Run("CreateAndGetByID", func(t *testing.T) {
		tag := models.NewTag(org.ID, "production", "#ff0000")
		err := db.CreateTag(ctx, tag)
		require.NoError(t, err)

		got, err := db.GetTagByID(ctx, tag.ID)
		require.NoError(t, err)
		assert.Equal(t, tag.ID, got.ID)
		assert.Equal(t, "production", got.Name)
		assert.Equal(t, "#ff0000", got.Color)
		assert.Equal(t, org.ID, got.OrgID)
	})

	t.Run("DefaultColor", func(t *testing.T) {
		tag := models.NewTag(org.ID, "default-color", "")
		err := db.CreateTag(ctx, tag)
		require.NoError(t, err)

		got, err := db.GetTagByID(ctx, tag.ID)
		require.NoError(t, err)
		assert.Equal(t, "#6366f1", got.Color)
	})

	t.Run("GetByOrgID", func(t *testing.T) {
		tags, err := db.GetTagsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tags), 1)
	})

	t.Run("GetByNameAndOrgID", func(t *testing.T) {
		tagName := "unique-tag-" + uuid.New().String()[:8]
		tag := models.NewTag(org.ID, tagName, "#00ff00")
		require.NoError(t, db.CreateTag(ctx, tag))

		got, err := db.GetTagByNameAndOrgID(ctx, tagName, org.ID)
		require.NoError(t, err)
		assert.Equal(t, tag.ID, got.ID)
	})

	t.Run("Update", func(t *testing.T) {
		tag := models.NewTag(org.ID, "old-tag", "#111111")
		require.NoError(t, db.CreateTag(ctx, tag))

		tag.Name = "updated-tag"
		tag.Color = "#222222"
		err := db.UpdateTag(ctx, tag)
		require.NoError(t, err)

		got, err := db.GetTagByID(ctx, tag.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-tag", got.Name)
		assert.Equal(t, "#222222", got.Color)
	})

	t.Run("Delete", func(t *testing.T) {
		tag := models.NewTag(org.ID, "delete-tag", "#333333")
		require.NoError(t, db.CreateTag(ctx, tag))

		err := db.DeleteTag(ctx, tag.ID)
		require.NoError(t, err)

		_, err = db.GetTagByID(ctx, tag.ID)
		assert.Error(t, err)
	})

	t.Run("BackupTags", func(t *testing.T) {
		tag1 := models.NewTag(org.ID, "backup-tag-1", "#aaaaaa")
		tag2 := models.NewTag(org.ID, "backup-tag-2", "#bbbbbb")
		require.NoError(t, db.CreateTag(ctx, tag1))
		require.NoError(t, db.CreateTag(ctx, tag2))

		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup.Complete("snap-tag-1", 1024, 10, 5)
		require.NoError(t, db.CreateBackup(ctx, backup))

		err := db.AssignTagToBackup(ctx, backup.ID, tag1.ID)
		require.NoError(t, err)
		err = db.AssignTagToBackup(ctx, backup.ID, tag2.ID)
		require.NoError(t, err)

		tags, err := db.GetTagsByBackupID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 2)

		backupIDs, err := db.GetBackupIDsByTagID(ctx, tag1.ID)
		require.NoError(t, err)
		assert.Contains(t, backupIDs, backup.ID)

		err = db.RemoveTagFromBackup(ctx, backup.ID, tag2.ID)
		require.NoError(t, err)

		tags, err = db.GetTagsByBackupID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, tag1.ID, tags[0].ID)
	})

	t.Run("SetBackupTags", func(t *testing.T) {
		tag1 := models.NewTag(org.ID, "set-tag-1", "#cccccc")
		tag2 := models.NewTag(org.ID, "set-tag-2", "#dddddd")
		tag3 := models.NewTag(org.ID, "set-tag-3", "#eeeeee")
		require.NoError(t, db.CreateTag(ctx, tag1))
		require.NoError(t, db.CreateTag(ctx, tag2))
		require.NoError(t, db.CreateTag(ctx, tag3))

		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup.Complete("snap-set-tag", 512, 3, 1)
		require.NoError(t, db.CreateBackup(ctx, backup))

		err := db.SetBackupTags(ctx, backup.ID, []uuid.UUID{tag1.ID, tag2.ID})
		require.NoError(t, err)

		tags, err := db.GetTagsByBackupID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 2)

		err = db.SetBackupTags(ctx, backup.ID, []uuid.UUID{tag3.ID})
		require.NoError(t, err)

		tags, err = db.GetTagsByBackupID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, tag3.ID, tags[0].ID)
	})

	t.Run("SnapshotTags", func(t *testing.T) {
		tag := models.NewTag(org.ID, "snapshot-tag", "#ff00ff")
		require.NoError(t, db.CreateTag(ctx, tag))

		snapshotID := "snap-tagged-" + uuid.New().String()[:8]

		err := db.AssignTagToSnapshot(ctx, snapshotID, tag.ID)
		require.NoError(t, err)

		tags, err := db.GetTagsBySnapshotID(ctx, snapshotID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, tag.ID, tags[0].ID)

		err = db.RemoveTagFromSnapshot(ctx, snapshotID, tag.ID)
		require.NoError(t, err)

		tags, err = db.GetTagsBySnapshotID(ctx, snapshotID)
		require.NoError(t, err)
		assert.Empty(t, tags)
	})

	t.Run("SetSnapshotTags", func(t *testing.T) {
		tag1 := models.NewTag(org.ID, "snap-set-1", "#001122")
		tag2 := models.NewTag(org.ID, "snap-set-2", "#334455")
		require.NoError(t, db.CreateTag(ctx, tag1))
		require.NoError(t, db.CreateTag(ctx, tag2))

		snapshotID := "snap-set-" + uuid.New().String()[:8]

		err := db.SetSnapshotTags(ctx, snapshotID, []uuid.UUID{tag1.ID, tag2.ID})
		require.NoError(t, err)

		tags, err := db.GetTagsBySnapshotID(ctx, snapshotID)
		require.NoError(t, err)
		assert.Len(t, tags, 2)

		err = db.SetSnapshotTags(ctx, snapshotID, []uuid.UUID{tag1.ID})
		require.NoError(t, err)

		tags, err = db.GetTagsBySnapshotID(ctx, snapshotID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
	})

	t.Run("GetBackupsByTagIDs", func(t *testing.T) {
		filterTag := models.NewTag(org.ID, "filter-tag", "#abcdef")
		require.NoError(t, db.CreateTag(ctx, filterTag))

		backup1 := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup1.Complete("snap-filter-1", 100, 1, 0)
		require.NoError(t, db.CreateBackup(ctx, backup1))
		require.NoError(t, db.AssignTagToBackup(ctx, backup1.ID, filterTag.ID))

		backup2 := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup2.Complete("snap-filter-2", 200, 2, 0)
		require.NoError(t, db.CreateBackup(ctx, backup2))

		backups, err := db.GetBackupsByTagIDs(ctx, []uuid.UUID{filterTag.ID})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(backups), 1)

		foundBackup1 := false
		for _, b := range backups {
			if b.ID == backup1.ID {
				foundBackup1 = true
			}
			assert.NotEqual(t, backup2.ID, b.ID)
		}
		assert.True(t, foundBackup1)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := db.GetTagByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestStore_Search(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Search Test Org", "search-test-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "searchable-agent")
	_ = createTestRepo(t, db, org.ID, "searchable-repo")
	sched := models.NewSchedule(agent.ID, "searchable-schedule", "0 1 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	t.Run("NoTypeFilter", func(t *testing.T) {
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query: "searchable",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 3)

		types := map[string]bool{}
		for _, r := range results {
			types[r.Type] = true
		}
		assert.True(t, types["agent"])
		assert.True(t, types["repository"])
		assert.True(t, types["schedule"])
	})

	t.Run("WithTypeFilter", func(t *testing.T) {
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query: "searchable",
			Types: []string{"agent"},
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
		for _, r := range results {
			assert.Equal(t, "agent", r.Type)
		}
	})

	t.Run("WithStatusFilter", func(t *testing.T) {
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query:  "searchable",
			Types:  []string{"agent"},
			Status: "pending",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
		for _, r := range results {
			assert.Equal(t, "pending", r.Status)
		}
	})

	t.Run("EmptyResults", func(t *testing.T) {
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query: "nonexistent-item-xyz-" + uuid.New().String(),
		})
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("WithLimit", func(t *testing.T) {
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query: "searchable",
			Limit: 1,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 4)
	})
}

func TestStore_DashboardAndMetrics(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Metrics Org", "metrics-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "metrics-agent")
	repo := createTestRepo(t, db, org.ID, "metrics-repo")
	sched := models.NewSchedule(agent.ID, "Metrics Sched", "0 1 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	// Create several completed backups
	for i := 0; i < 3; i++ {
		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup.Complete(fmt.Sprintf("snap-metric-%d", i), int64(1024*(i+1)), 10+i, 5+i)
		require.NoError(t, db.CreateBackup(ctx, backup))
	}
	// Create a failed backup
	failedBackup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
	failedBackup.Fail("test failure")
	require.NoError(t, db.CreateBackup(ctx, failedBackup))

	t.Run("GetBackupsByOrgIDSince", func(t *testing.T) {
		since := time.Now().Add(-1 * time.Hour)
		backups, err := db.GetBackupsByOrgIDSince(ctx, org.ID, since)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(backups), 4)
	})

	t.Run("GetBackupCountsByOrgID", func(t *testing.T) {
		total, running, failed24h, err := db.GetBackupCountsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 4)
		assert.GreaterOrEqual(t, running, 0)
		assert.GreaterOrEqual(t, failed24h, 1)
	})

	t.Run("CreateMetricsHistory", func(t *testing.T) {
		metrics := models.NewMetricsHistory(org.ID)
		metrics.BackupCount = 10
		metrics.BackupSuccessCount = 8
		metrics.BackupFailedCount = 2
		metrics.BackupTotalSize = 1024 * 1024
		metrics.BackupTotalDuration = 5000
		metrics.AgentTotalCount = 5
		metrics.AgentOnlineCount = 3
		metrics.AgentOfflineCount = 2
		metrics.StorageUsedBytes = 2048 * 1024
		metrics.StorageRawBytes = 4096 * 1024
		metrics.StorageSpaceSaved = 2048 * 1024
		metrics.RepositoryCount = 2
		metrics.TotalSnapshots = 15
		err := db.CreateMetricsHistory(ctx, metrics)
		require.NoError(t, err)
	})

	t.Run("GetDashboardStats", func(t *testing.T) {
		stats, err := db.GetDashboardStats(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, stats.AgentTotal, 1)
		assert.GreaterOrEqual(t, stats.BackupTotal, 4)
		assert.GreaterOrEqual(t, stats.RepositoryCount, 1)
		assert.GreaterOrEqual(t, stats.ScheduleCount, 1)
		assert.GreaterOrEqual(t, stats.ScheduleEnabled, 1)
	})

	t.Run("GetBackupSuccessRates", func(t *testing.T) {
		rate7d, rate30d, err := db.GetBackupSuccessRates(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, "7d", rate7d.Period)
		assert.Equal(t, "30d", rate30d.Period)
		assert.GreaterOrEqual(t, rate7d.Total, 4)
		assert.GreaterOrEqual(t, rate7d.Successful, 3)
		assert.GreaterOrEqual(t, rate7d.Failed, 1)
		assert.Greater(t, rate7d.SuccessPercent, float64(0))
		assert.GreaterOrEqual(t, rate30d.Total, 4)
	})

	t.Run("GetStorageGrowthTrend", func(t *testing.T) {
		stats := models.NewStorageStats(repo.ID)
		stats.SetStats(1024*1024, 100, 512*1024, 1024*1024, 10)
		require.NoError(t, db.CreateStorageStats(ctx, stats))

		trends, err := db.GetStorageGrowthTrend(ctx, org.ID, 30)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(trends), 1)
	})

	t.Run("GetBackupDurationTrend", func(t *testing.T) {
		trends, err := db.GetBackupDurationTrend(ctx, org.ID, 30)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(trends), 1)
		for _, trend := range trends {
			assert.Greater(t, trend.BackupCount, 0)
		}
	})

	t.Run("GetDailyBackupStats", func(t *testing.T) {
		stats, err := db.GetDailyBackupStats(ctx, org.ID, 30)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(stats), 1)
		for _, s := range stats {
			assert.GreaterOrEqual(t, s.Total, 1)
		}
	})

	t.Run("GetBackupsByOrgIDAndDateRange", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now().Add(1 * time.Hour)
		backups, err := db.GetBackupsByOrgIDAndDateRange(ctx, org.ID, start, end)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(backups), 4)
	})

	t.Run("GetAlertsByOrgIDAndDateRange", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeBackupSLA, models.AlertSeverityWarning,
			"Metrics Alert", "test alert for metrics")
		require.NoError(t, db.CreateAlert(ctx, alert))

		start := time.Now().Add(-1 * time.Hour)
		end := time.Now().Add(1 * time.Hour)
		alerts, err := db.GetAlertsByOrgIDAndDateRange(ctx, org.ID, start, end)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 1)
	})

	t.Run("EmptyOrg", func(t *testing.T) {
		emptyOrg := createTestOrg(t, db, "Empty Metrics Org", "empty-metrics-"+uuid.New().String()[:8])

		stats, err := db.GetDashboardStats(ctx, emptyOrg.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, stats.AgentTotal)
		assert.Equal(t, 0, stats.BackupTotal)

		since := time.Now().Add(-1 * time.Hour)
		backups, err := db.GetBackupsByOrgIDSince(ctx, emptyOrg.ID, since)
		require.NoError(t, err)
		assert.Empty(t, backups)

		rate7d, rate30d, err := db.GetBackupSuccessRates(ctx, emptyOrg.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, rate7d.Total)
		assert.Equal(t, 0, rate30d.Total)
	})

	t.Run("DailySummary_CreateAndGet", func(t *testing.T) {
		now := time.Now()
		date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		summary := &models.MetricsDailySummary{
			ID:                uuid.New(),
			OrgID:             org.ID,
			Date:              date,
			TotalBackups:      10,
			SuccessfulBackups: 8,
			FailedBackups:     2,
			TotalSizeBytes:    1024 * 1024,
			TotalDurationSecs: 600,
			AgentsActive:      3,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		err := db.CreateOrUpdateDailySummary(ctx, summary)
		require.NoError(t, err)

		got, err := db.GetDailySummary(ctx, org.ID, date)
		require.NoError(t, err)
		assert.Equal(t, 10, got.TotalBackups)
		assert.Equal(t, 8, got.SuccessfulBackups)
		assert.Equal(t, 2, got.FailedBackups)
		assert.Equal(t, int64(1024*1024), got.TotalSizeBytes)
		assert.Equal(t, int64(600), got.TotalDurationSecs)
		assert.Equal(t, 3, got.AgentsActive)
	})

	t.Run("DailySummary_Upsert", func(t *testing.T) {
		now := time.Now()
		date := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)

		summary := &models.MetricsDailySummary{
			ID:                uuid.New(),
			OrgID:             org.ID,
			Date:              date,
			TotalBackups:      5,
			SuccessfulBackups: 4,
			FailedBackups:     1,
			TotalSizeBytes:    512,
			TotalDurationSecs: 300,
			AgentsActive:      2,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		require.NoError(t, db.CreateOrUpdateDailySummary(ctx, summary))

		// Upsert with different values
		summary.ID = uuid.New()
		summary.TotalBackups = 20
		summary.SuccessfulBackups = 18
		summary.UpdatedAt = time.Now()
		require.NoError(t, db.CreateOrUpdateDailySummary(ctx, summary))

		got, err := db.GetDailySummary(ctx, org.ID, date)
		require.NoError(t, err)
		assert.Equal(t, 20, got.TotalBackups)
		assert.Equal(t, 18, got.SuccessfulBackups)
	})

	t.Run("DailySummary_GetRange", func(t *testing.T) {
		now := time.Now()
		startDate := time.Date(now.Year(), now.Month(), now.Day()-2, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		summaries, err := db.GetDailySummaries(ctx, org.ID, startDate, endDate)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(summaries), 1)
	})

	t.Run("DailySummary_Delete", func(t *testing.T) {
		now := time.Now()
		oldDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

		summary := &models.MetricsDailySummary{
			ID:        uuid.New(),
			OrgID:     org.ID,
			Date:      oldDate,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, db.CreateOrUpdateDailySummary(ctx, summary))

		err := db.DeleteDailySummariesBefore(ctx, org.ID, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		_, err = db.GetDailySummary(ctx, org.ID, oldDate)
		assert.Error(t, err)
	})
}

func TestStore_SearchBackupsAdvanced(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Search Adv Org", "search-adv-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "search-adv-agent")
	repo := createTestRepo(t, db, org.ID, "search-adv-repo")
	sched := models.NewSchedule(agent.ID, "Search Adv Sched", "0 1 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
	backup.Complete("searchable-snap-adv", 2048, 5, 2)
	require.NoError(t, db.CreateBackup(ctx, backup))

	t.Run("SearchBackupBySnapshotID", func(t *testing.T) {
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query: "searchable-snap-adv",
			Types: []string{"backup"},
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("SearchBackupWithDateFilter", func(t *testing.T) {
		from := time.Now().Add(-1 * time.Hour)
		to := time.Now().Add(1 * time.Hour)
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query:    "searchable-snap-adv",
			Types:    []string{"backup"},
			DateFrom: &from,
			DateTo:   &to,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("SearchBackupWithSizeFilter", func(t *testing.T) {
		sizeMin := int64(100)
		sizeMax := int64(100000)
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query:   "searchable-snap-adv",
			Types:   []string{"backup"},
			SizeMin: &sizeMin,
			SizeMax: &sizeMax,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("SearchBackupWithStatusFilter", func(t *testing.T) {
		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query:  "searchable-snap-adv",
			Types:  []string{"backup"},
			Status: "completed",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("SearchBackupWithTagFilter", func(t *testing.T) {
		tag := models.NewTag(org.ID, "search-tag-adv", "#ff0000")
		require.NoError(t, db.CreateTag(ctx, tag))
		require.NoError(t, db.AssignTagToBackup(ctx, backup.ID, tag.ID))

		results, err := db.Search(ctx, org.ID, SearchFilter{
			Query:  "searchable-snap-adv",
			Types:  []string{"backup"},
			TagIDs: []uuid.UUID{tag.ID},
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})
}

func TestStore_AuditLogCountFilters(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Count Org", "count-"+uuid.New().String()[:8])
	user := createTestUser(t, db, org.ID, "counter@test.com", "Counter")

	for i := 0; i < 3; i++ {
		log := models.NewAuditLog(org.ID, models.AuditActionCreate, "agent", models.AuditResultSuccess)
		log.WithUser(user.ID).WithDetails(fmt.Sprintf("Created agent %d", i))
		require.NoError(t, db.CreateAuditLog(ctx, log))
	}
	for i := 0; i < 2; i++ {
		log := models.NewAuditLog(org.ID, models.AuditActionDelete, "repository", models.AuditResultFailure)
		log.WithUser(user.ID)
		require.NoError(t, db.CreateAuditLog(ctx, log))
	}

	t.Run("CountWithResourceTypeFilter", func(t *testing.T) {
		count, err := db.CountAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{ResourceType: "agent"})
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("CountWithResultFilter", func(t *testing.T) {
		count, err := db.CountAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{Result: "failure"})
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("CountWithDateFilters", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now().Add(1 * time.Hour)
		count, err := db.CountAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{
			StartDate: &start,
			EndDate:   &end,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("CountWithSearchFilter", func(t *testing.T) {
		count, err := db.CountAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{Search: "Created agent"})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
	})

	t.Run("CountWithCombinedFilters", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour)
		count, err := db.CountAuditLogsByOrgID(ctx, org.ID, AuditLogFilter{
			Action:    "create",
			StartDate: &start,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}

func TestStore_PolicyUpdateAllFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Policy Full Org", "pol-full-"+uuid.New().String()[:8])

	t.Run("UpdateAllFields", func(t *testing.T) {
		policy := models.NewPolicy(org.ID, "Full Policy")
		policy.Paths = []string{"/data"}
		policy.Excludes = []string{"*.tmp"}
		policy.RetentionPolicy = models.DefaultRetentionPolicy()
		bwLimit := 512
		policy.BandwidthLimitKB = &bwLimit
		policy.BackupWindow = &models.BackupWindow{Start: "01:00", End: "05:00"}
		policy.ExcludedHours = []int{12, 13}
		policy.CronExpression = "0 2 * * *"
		require.NoError(t, db.CreatePolicy(ctx, policy))

		policy.Name = "Updated Full Policy"
		policy.Description = "Updated description"
		policy.Paths = []string{"/new/data", "/etc"}
		policy.Excludes = []string{"*.log", "*.bak"}
		policy.RetentionPolicy = &models.RetentionPolicy{KeepLast: 10}
		newBW := 1024
		policy.BandwidthLimitKB = &newBW
		policy.BackupWindow = &models.BackupWindow{Start: "02:00", End: "06:00"}
		policy.ExcludedHours = []int{9, 10, 11}
		policy.CronExpression = "0 3 * * *"
		err := db.UpdatePolicy(ctx, policy)
		require.NoError(t, err)

		got, err := db.GetPolicyByID(ctx, policy.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Full Policy", got.Name)
		assert.Equal(t, "Updated description", got.Description)
		assert.Equal(t, []string{"/new/data", "/etc"}, got.Paths)
		assert.Equal(t, []string{"*.log", "*.bak"}, got.Excludes)
		require.NotNil(t, got.RetentionPolicy)
		assert.Equal(t, 10, got.RetentionPolicy.KeepLast)
		require.NotNil(t, got.BandwidthLimitKB)
		assert.Equal(t, 1024, *got.BandwidthLimitKB)
		require.NotNil(t, got.BackupWindow)
		assert.Contains(t, got.BackupWindow.Start, "02:00")
		assert.Contains(t, got.BackupWindow.End, "06:00")
		assert.Equal(t, "0 3 * * *", got.CronExpression)
	})
}

func TestStore_ScheduleUpdateAllFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Sched Full Org", "sched-full-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "sched-full-agent")
	repo := createTestRepo(t, db, org.ID, "sched-full-repo")

	t.Run("UpdateWithAllOptionalFields", func(t *testing.T) {
		sched := models.NewSchedule(agent.ID, "Full Sched", "0 1 * * *", []string{"/data"})
		sched.Repositories = []models.ScheduleRepository{
			{RepositoryID: repo.ID, Priority: 0, Enabled: true},
		}
		sched.RetentionPolicy = models.DefaultRetentionPolicy()
		require.NoError(t, db.CreateSchedule(ctx, sched))

		bwLimit := 2048
		sched.BandwidthLimitKB = &bwLimit
		sched.BackupWindow = &models.BackupWindow{Start: "00:00", End: "04:00"}
		sched.ExcludedHours = []int{8, 9, 10, 11, 12}
		compression := "auto"
		sched.CompressionLevel = &compression
		sched.OnMountUnavailable = models.MountBehaviorFail
		sched.Excludes = []string{"*.cache", "node_modules"}
		sched.Name = "Updated Full Sched"
		sched.CronExpression = "0 3 * * *"
		sched.Enabled = false
		err := db.UpdateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Full Sched", got.Name)
		require.NotNil(t, got.BandwidthLimitKB)
		assert.Equal(t, 2048, *got.BandwidthLimitKB)
		require.NotNil(t, got.BackupWindow)
		assert.Contains(t, got.BackupWindow.Start, "00:00")
		assert.Equal(t, []int{8, 9, 10, 11, 12}, got.ExcludedHours)
		require.NotNil(t, got.CompressionLevel)
		assert.Equal(t, "auto", *got.CompressionLevel)
		assert.Equal(t, models.MountBehaviorFail, got.OnMountUnavailable)
		assert.Equal(t, []string{"*.cache", "node_modules"}, got.Excludes)
		assert.False(t, got.Enabled)
	})
}

func TestStore_ReportHistoryWithData(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "RptData Org", "rptdata-"+uuid.New().String()[:8])

	t.Run("CreateWithReportData", func(t *testing.T) {
		now := time.Now()
		history := models.NewReportHistory(org.ID, nil, "weekly", now.Add(-7*24*time.Hour), now, []string{"admin@test.com"})
		history.ReportData = &models.ReportData{
			BackupSummary: models.BackupSummary{
				TotalBackups:      50,
				SuccessfulBackups: 48,
				FailedBackups:     2,
				SuccessRate:       96.0,
			},
			StorageSummary: models.StorageSummary{
				TotalRawSize:     1024 * 1024 * 1024,
				TotalRestoreSize: 512 * 1024 * 1024,
				RepositoryCount:  3,
			},
			AgentSummary: models.AgentSummary{
				TotalAgents:  5,
				ActiveAgents: 4,
			},
			AlertSummary: models.AlertSummary{
				TotalAlerts:    10,
				CriticalAlerts: 2,
			},
		}
		sentAt := time.Now()
		history.SentAt = &sentAt
		err := db.CreateReportHistory(ctx, history)
		require.NoError(t, err)

		got, err := db.GetReportHistoryByID(ctx, history.ID)
		require.NoError(t, err)
		require.NotNil(t, got.ReportData)
		assert.Equal(t, 50, got.ReportData.BackupSummary.TotalBackups)
		assert.Equal(t, 3, got.ReportData.StorageSummary.RepositoryCount)
		assert.NotNil(t, got.SentAt)
	})

	t.Run("CreateWithNilData", func(t *testing.T) {
		now := time.Now()
		history := models.NewReportHistory(org.ID, nil, "daily", now.Add(-24*time.Hour), now, []string{"test@test.com"})
		err := db.CreateReportHistory(ctx, history)
		require.NoError(t, err)

		got, err := db.GetReportHistoryByID(ctx, history.ID)
		require.NoError(t, err)
		assert.Nil(t, got.ReportData)
	})
}

func TestStore_DBHealthAndVersion(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	t.Run("Health", func(t *testing.T) {
		health := db.Health()
		require.NotNil(t, health)
		assert.Contains(t, health, "total_conns")
		assert.Contains(t, health, "acquired_conns")
		assert.Contains(t, health, "idle_conns")
		assert.Contains(t, health, "max_conns")
	})

	t.Run("CurrentVersion", func(t *testing.T) {
		version, err := db.CurrentVersion(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, version, 29)
	})

	t.Run("ExecTxRollback", func(t *testing.T) {
		err := db.ExecTx(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "SELECT 1")
			if err != nil {
				return err
			}
			return fmt.Errorf("intentional rollback")
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "intentional rollback")
	})
}

func TestStore_AgentWithFullStats(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Stats Full Org", "stats-full-"+uuid.New().String()[:8])

	t.Run("AgentStatsWithBackups", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "stats-full-agent")
		repo := createTestRepo(t, db, org.ID, "stats-full-repo")
		sched := models.NewSchedule(agent.ID, "Stats Sched", "0 1 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		backup := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		backup.Complete("stats-snap", 4096, 20, 5)
		require.NoError(t, db.CreateBackup(ctx, backup))

		stats, err := db.GetAgentStats(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, stats.AgentID)
		assert.GreaterOrEqual(t, stats.TotalBackups, 1)
	})
}

// TestStore_NotFoundErrors tests that operations on non-existent records return proper errors.
func TestStore_NotFoundErrors(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	nonExistentID := uuid.New()

	t.Run("DeleteExcludePattern_NonExistent", func(t *testing.T) {
		err := db.DeleteExcludePattern(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteExcludePattern_Builtin", func(t *testing.T) {
		// Seed builtin patterns, then try to delete one
		builtinPatterns := []*models.ExcludePattern{
			models.NewBuiltinExcludePattern("OS Temp Files", "Operating system temp files", "system", []string{"*.tmp", "*.swp"}),
		}
		err := db.SeedBuiltinExcludePatterns(ctx, builtinPatterns)
		require.NoError(t, err)

		// Find the builtin pattern
		builtins, err := db.GetBuiltinExcludePatterns(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, builtins)

		// Try to delete a builtin pattern — should fail
		err = db.DeleteExcludePattern(ctx, builtins[0].ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found or is built-in")
	})

	t.Run("DeleteSnapshotComment_NonExistent", func(t *testing.T) {
		err := db.DeleteSnapshotComment(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UpdateSnapshotComment_NonExistent", func(t *testing.T) {
		comment := &models.SnapshotComment{
			ID:      nonExistentID,
			Content: "won't be saved",
		}
		err := db.UpdateSnapshotComment(ctx, comment)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteVerification_NonExistent", func(t *testing.T) {
		err := db.DeleteVerification(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteRestore_NonExistent", func(t *testing.T) {
		err := db.DeleteRestore(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UpdateAgentAPIKeyHash_NonExistent", func(t *testing.T) {
		err := db.UpdateAgentAPIKeyHash(ctx, nonExistentID, "hash123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("RevokeAgentAPIKey_NonExistent", func(t *testing.T) {
		err := db.RevokeAgentAPIKey(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetAgentByAPIKeyHash_NotFound", func(t *testing.T) {
		_, err := db.GetAgentByAPIKeyHash(ctx, "nonexistent-hash-value")
		require.Error(t, err)
	})

	t.Run("GetBackupBySnapshotID_NotFound", func(t *testing.T) {
		_, err := db.GetBackupBySnapshotID(ctx, "nonexistent-snapshot-id")
		require.Error(t, err)
	})

	t.Run("GetUserByOIDCSubject_NotFound", func(t *testing.T) {
		_, err := db.GetUserByOIDCSubject(ctx, "nonexistent-oidc-subject")
		require.Error(t, err)
	})
}

// TestStore_NilInputPaths tests that nil/empty inputs are handled correctly.
func TestStore_NilInputPaths(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Nil Input Org", "nil-input-"+uuid.New().String()[:8])

	t.Run("CreateReportSchedule_NilRecipients", func(t *testing.T) {
		sched := models.NewReportSchedule(org.ID, "Nil Recipients Report", models.ReportFrequencyDaily, nil)
		err := db.CreateReportSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetReportScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, sched.Name, got.Name)
	})

	t.Run("UpdateReportSchedule_NilRecipients", func(t *testing.T) {
		sched := models.NewReportSchedule(org.ID, "Update Nil Report", models.ReportFrequencyWeekly, []string{"a@b.com"})
		require.NoError(t, db.CreateReportSchedule(ctx, sched))

		sched.Recipients = nil
		sched.Name = "Updated Nil Report"
		err := db.UpdateReportSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetReportScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Nil Report", got.Name)
	})

	t.Run("CreateReportHistory_NilRecipients_NilData", func(t *testing.T) {
		now := time.Now()
		history := models.NewReportHistory(org.ID, nil, "daily", now.Add(-24*time.Hour), now, nil)
		history.ReportData = nil
		err := db.CreateReportHistory(ctx, history)
		require.NoError(t, err)

		got, err := db.GetReportHistoryByID(ctx, history.ID)
		require.NoError(t, err)
		assert.Equal(t, "daily", got.ReportType)
	})

	t.Run("CreateReportHistory_WithData", func(t *testing.T) {
		now := time.Now()
		history := models.NewReportHistory(org.ID, nil, "weekly", now.Add(-7*24*time.Hour), now, []string{"x@y.com"})
		history.ReportData = &models.ReportData{
			BackupSummary: models.BackupSummary{TotalBackups: 10, SuccessfulBackups: 9, FailedBackups: 1, SuccessRate: 90.0},
			AgentSummary:  models.AgentSummary{TotalAgents: 3, ActiveAgents: 2, OfflineAgents: 1},
		}
		err := db.CreateReportHistory(ctx, history)
		require.NoError(t, err)

		got, err := db.GetReportHistoryByID(ctx, history.ID)
		require.NoError(t, err)
		assert.NotNil(t, got.ReportData)
	})

	t.Run("CreateVerification_EmptySnapshotID", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "verif-empty-snap-repo")
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		v.SnapshotID = "" // empty — should be handled as nil
		err := db.CreateVerification(ctx, v)
		require.NoError(t, err)

		got, err := db.GetVerificationByID(ctx, v.ID)
		require.NoError(t, err)
		assert.Empty(t, got.SnapshotID)
	})

	t.Run("UpdateVerification_EmptySnapshotID", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "verif-upd-empty-snap")
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		v.SnapshotID = "snap-123"
		require.NoError(t, db.CreateVerification(ctx, v))

		v.SnapshotID = ""
		v.Status = models.VerificationStatusPassed
		now := time.Now()
		v.CompletedAt = &now
		dur := int64(500)
		v.DurationMs = &dur
		err := db.UpdateVerification(ctx, v)
		require.NoError(t, err)
	})

	t.Run("CreateVerificationSchedule_EmptyReadDataSubset", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "vs-no-subset")
		vs := models.NewVerificationSchedule(repo.ID, models.VerificationTypeCheck, "0 3 * * *")
		vs.ReadDataSubset = "" // empty
		err := db.CreateVerificationSchedule(ctx, vs)
		require.NoError(t, err)
	})

	t.Run("UpdateVerificationSchedule_EmptyReadDataSubset", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "vs-upd-no-subset")
		vs := models.NewVerificationSchedule(repo.ID, models.VerificationTypeCheck, "0 4 * * *")
		vs.ReadDataSubset = "1%"
		require.NoError(t, db.CreateVerificationSchedule(ctx, vs))

		vs.ReadDataSubset = ""
		err := db.UpdateVerificationSchedule(ctx, vs)
		require.NoError(t, err)
	})

	t.Run("CreateRestore_NilPaths", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "restore-nil-paths")
		repo := createTestRepo(t, db, org.ID, "restore-nil-paths-repo")
		restore := models.NewRestore(agent.ID, repo.ID, "snap-nil", "/target", nil, nil)
		err := db.CreateRestore(ctx, restore)
		require.NoError(t, err)

		got, err := db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Equal(t, "/target", got.TargetPath)
	})

	t.Run("CreateSchedule_EmptyOptionalFields", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "sched-empty-opts")
		sched := models.NewSchedule(agent.ID, "Minimal Schedule", "0 0 * * *", []string{"/data"})
		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Minimal Schedule", got.Name)
	})
}

// TestStore_AdditionalMethodCoverage covers specific methods needing more coverage.
func TestStore_AdditionalMethodCoverage(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Extra Cov Org", "extra-cov-"+uuid.New().String()[:8])

	t.Run("MarkMaintenanceNotificationSent", func(t *testing.T) {
		mw := models.NewMaintenanceWindow(org.ID, "Notify Test", time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
		require.NoError(t, db.CreateMaintenanceWindow(ctx, mw))

		err := db.MarkMaintenanceNotificationSent(ctx, mw.ID)
		require.NoError(t, err)
	})

	t.Run("GetEnabledSchedulesByOrgID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "enabled-sched-agent")
		sched := models.NewSchedule(agent.ID, "Enabled Sched", "0 2 * * *", []string{"/data"})
		sched.Enabled = true
		require.NoError(t, db.CreateSchedule(ctx, sched))

		scheds, err := db.GetEnabledSchedulesByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, scheds)
	})

	t.Run("GetEnabledSchedules", func(t *testing.T) {
		scheds, err := db.GetEnabledSchedules(ctx)
		require.NoError(t, err)
		assert.NotNil(t, scheds)
	})

	t.Run("GetBackupsByOrgIDSince", func(t *testing.T) {
		since := time.Now().Add(-24 * time.Hour)
		_, err := db.GetBackupsByOrgIDSince(ctx, org.ID, since)
		require.NoError(t, err)
	})

	t.Run("GetBackupsByOrgIDAndDateRange", func(t *testing.T) {
		now := time.Now()
		_, err := db.GetBackupsByOrgIDAndDateRange(ctx, org.ID, now.Add(-7*24*time.Hour), now)
		require.NoError(t, err)
	})

	t.Run("GetAlertsByOrgIDAndDateRange", func(t *testing.T) {
		now := time.Now()
		_, err := db.GetAlertsByOrgIDAndDateRange(ctx, org.ID, now.Add(-7*24*time.Hour), now)
		require.NoError(t, err)
	})

	t.Run("GetBackupCountsByOrgID", func(t *testing.T) {
		total, running, failed, err := db.GetBackupCountsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 0)
		assert.GreaterOrEqual(t, running, 0)
		assert.GreaterOrEqual(t, failed, 0)
	})

	t.Run("NotificationLogLifecycle", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Test Email", models.ChannelTypeEmail, []byte("{}"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		log := models.NewNotificationLog(org.ID, &channel.ID, "backup_success", "user@test.com", "Backup completed")
		err := db.CreateNotificationLog(ctx, log)
		require.NoError(t, err)

		log.Status = models.NotificationStatusFailed
		log.ErrorMessage = "SMTP timeout"
		err = db.UpdateNotificationLog(ctx, log)
		require.NoError(t, err)

		logs, err := db.GetNotificationLogsByOrgID(ctx, org.ID, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, logs)
	})

	t.Run("NotificationPreferenceLifecycle", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Pref Channel", models.ChannelTypeSlack, []byte("{}"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		pref := models.NewNotificationPreference(org.ID, channel.ID, models.EventBackupSuccess)
		err := db.CreateNotificationPreference(ctx, pref)
		require.NoError(t, err)

		prefs, err := db.GetNotificationPreferencesByChannelID(ctx, channel.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, prefs)

		enabledPrefs, err := db.GetEnabledPreferencesForEvent(ctx, org.ID, models.EventBackupSuccess)
		require.NoError(t, err)
		assert.NotEmpty(t, enabledPrefs)

		pref.Enabled = false
		err = db.UpdateNotificationPreference(ctx, pref)
		require.NoError(t, err)

		err = db.DeleteNotificationPreference(ctx, pref.ID)
		require.NoError(t, err)
	})

	t.Run("RepositoryKeyLifecycle", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "repokey-cov-repo")
		rk := models.NewRepositoryKey(repo.ID, []byte("encrypted-key"), false, nil)
		err := db.CreateRepositoryKey(ctx, rk)
		require.NoError(t, err)

		err = db.UpdateRepositoryKeyEscrow(ctx, repo.ID, true, []byte("escrow-key"))
		require.NoError(t, err)

		keys, err := db.GetRepositoryKeysWithEscrowByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, keys)

		err = db.DeleteRepositoryKey(ctx, rk.ID)
		require.NoError(t, err)
	})

	t.Run("StorageStatsGrowth", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "ss-growth-repo")
		ss := models.NewStorageStats(repo.ID)
		ss.TotalSize = 1000
		ss.TotalFileCount = 50
		ss.RawDataSize = 2000
		ss.SnapshotCount = 5
		require.NoError(t, db.CreateStorageStats(ctx, ss))

		growth, err := db.GetStorageGrowth(ctx, repo.ID, 30)
		require.NoError(t, err)
		assert.NotNil(t, growth)

		allGrowth, err := db.GetAllStorageGrowth(ctx, org.ID, 30)
		require.NoError(t, err)
		assert.NotNil(t, allGrowth)

		latest, err := db.GetLatestStatsForAllRepos(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, latest)

		summary, err := db.GetStorageStatsSummary(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, summary)
	})

	t.Run("UpdateAgent_WithHealthMetrics", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "agent-health-upd")
		agent.HealthStatus = models.HealthStatusHealthy
		now := time.Now()
		agent.HealthCheckedAt = &now
		agent.HealthMetrics = &models.HealthMetrics{
			CPUUsage:    25.5,
			MemoryUsage: 60.0,
			DiskUsage:   40.0,
		}
		agent.NetworkMounts = []models.NetworkMount{
			{Path: "/mnt/nfs", Type: models.MountTypeNFS, Status: models.MountStatusConnected},
		}
		err := db.UpdateAgent(ctx, agent)
		require.NoError(t, err)

		got, err := db.GetAgentByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, models.HealthStatusHealthy, got.HealthStatus)
		assert.NotNil(t, got.HealthMetrics)
		assert.NotNil(t, got.NetworkMounts)
	})

	t.Run("CreateAgentHealthHistory_Full", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "health-hist-full")
		history := models.NewAgentHealthHistory(agent.ID, org.ID, models.HealthStatusHealthy,
			&models.HealthMetrics{CPUUsage: 10, MemoryUsage: 50, DiskUsage: 30},
			[]models.HealthIssue{{Component: "disk", Severity: models.HealthStatusWarning, Message: "disk approaching threshold"}})
		err := db.CreateAgentHealthHistory(ctx, history)
		require.NoError(t, err)

		records, err := db.GetAgentHealthHistory(ctx, agent.ID, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, records)
	})

	t.Run("GetFleetHealthSummary", func(t *testing.T) {
		summary, err := db.GetFleetHealthSummary(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, summary)
	})

	t.Run("BackupScript_Enabled", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "bs-enabled-agent")
		sched := models.NewSchedule(agent.ID, "BS Enabled Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		script := models.NewBackupScript(sched.ID, models.BackupScriptTypePreBackup, "echo pre")
		script.Enabled = true
		require.NoError(t, db.CreateBackupScript(ctx, script))

		enabled, err := db.GetEnabledBackupScriptsByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, enabled)
	})

	t.Run("UpdateOrganization", func(t *testing.T) {
		testOrg := createTestOrg(t, db, "Update Org Test", "upd-org-"+uuid.New().String()[:8])
		testOrg.Name = "Updated Org Name"
		err := db.UpdateOrganization(ctx, testOrg)
		require.NoError(t, err)
	})

	t.Run("DeleteOrganization", func(t *testing.T) {
		testOrg := createTestOrg(t, db, "Delete Org Test", "del-org-"+uuid.New().String()[:8])
		err := db.DeleteOrganization(ctx, testOrg.ID)
		require.NoError(t, err)
	})

	t.Run("UserOrganizations", func(t *testing.T) {
		user := createTestUser(t, db, org.ID, "userorg@test.com", "UserOrg Test")
		orgs, err := db.GetUserOrganizations(ctx, user.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, orgs)
	})

	t.Run("AcceptInvitation", func(t *testing.T) {
		inviter := createTestUser(t, db, org.ID, "inviter-accept@test.com", "Inviter Accept")
		inv := models.NewOrgInvitation(org.ID, "accept@test.com", models.OrgRoleMember, "token-accept-"+uuid.New().String()[:8], inviter.ID, time.Now().Add(24*time.Hour))
		require.NoError(t, db.CreateInvitation(ctx, inv))

		err := db.AcceptInvitation(ctx, inv.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteInvitation", func(t *testing.T) {
		inviter := createTestUser(t, db, org.ID, "inviter-del@test.com", "Inviter Del")
		inv := models.NewOrgInvitation(org.ID, "delete-inv@test.com", models.OrgRoleMember, "token-del-"+uuid.New().String()[:8], inviter.ID, time.Now().Add(24*time.Hour))
		require.NoError(t, db.CreateInvitation(ctx, inv))

		err := db.DeleteInvitation(ctx, inv.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteScheduleRepositories", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "del-sched-repo-agent")
		sched := models.NewSchedule(agent.ID, "Del Repo Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		repo := createTestRepo(t, db, org.ID, "del-sched-repo")
		sr := models.NewScheduleRepository(sched.ID, repo.ID, 1)
		require.NoError(t, db.CreateScheduleRepository(ctx, sr))

		err := db.DeleteScheduleRepositories(ctx, sched.ID)
		require.NoError(t, err)

		repos, err := db.GetScheduleRepositories(ctx, sched.ID)
		require.NoError(t, err)
		assert.Empty(t, repos)
	})

	t.Run("SetScheduleRepositories_MultipleRepos", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "set-sched-repos-agent")
		sched := models.NewSchedule(agent.ID, "Set Repos Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		repo1 := createTestRepo(t, db, org.ID, "set-repo1")
		repo2 := createTestRepo(t, db, org.ID, "set-repo2")

		repos := []models.ScheduleRepository{
			*models.NewScheduleRepository(sched.ID, repo1.ID, 1),
			*models.NewScheduleRepository(sched.ID, repo2.ID, 2),
		}
		err := db.SetScheduleRepositories(ctx, sched.ID, repos)
		require.NoError(t, err)

		gotRepos, err := db.GetScheduleRepositories(ctx, sched.ID)
		require.NoError(t, err)
		assert.Len(t, gotRepos, 2)
	})

	t.Run("GetBackupsByScheduleID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "bkp-sched-id-agent")
		sched := models.NewSchedule(agent.ID, "BkpSchedID Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))
		repo := createTestRepo(t, db, org.ID, "bkp-sched-id-repo")

		b := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		b.Complete("snap-bsi", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		backups, err := db.GetBackupsByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, backups)
	})

	t.Run("GetBackupsByAgentID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "bkp-agent-id")
		sched := models.NewSchedule(agent.ID, "BkpAgentID Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))
		repo := createTestRepo(t, db, org.ID, "bkp-agent-id-repo")

		b := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		b.Complete("snap-bai", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		backups, err := db.GetBackupsByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, backups)
	})

	t.Run("GetLatestBackupByScheduleID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "latest-bkp-agent")
		sched := models.NewSchedule(agent.ID, "Latest Bkp Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))
		repo := createTestRepo(t, db, org.ID, "latest-bkp-repo")

		b := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		b.Complete("snap-latest", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		latest, err := db.GetLatestBackupByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.Equal(t, "snap-latest", latest.SnapshotID)
	})

	t.Run("GetRestoresByAgentID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "restores-agent-id")
		repo := createTestRepo(t, db, org.ID, "restores-agent-repo")
		restore := models.NewRestore(agent.ID, repo.ID, "snap-restores", "/restore-path", []string{"/include"}, []string{"/exclude"})
		require.NoError(t, db.CreateRestore(ctx, restore))

		restores, err := db.GetRestoresByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, restores)
	})

	t.Run("GetOrgIDByAgentID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "orgid-by-agent")
		gotOrgID, err := db.GetOrgIDByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, gotOrgID)
	})

	t.Run("GetOrgIDByScheduleID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "orgid-by-sched-agent")
		sched := models.NewSchedule(agent.ID, "OrgID Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		gotOrgID, err := db.GetOrgIDByScheduleID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, gotOrgID)
	})

	t.Run("DeleteUser_LastOwner", func(t *testing.T) {
		delOrg := createTestOrg(t, db, "Del User Org", "del-user-org-"+uuid.New().String()[:8])
		owner := createTestUser(t, db, delOrg.ID, "owner-del@test.com", "Owner Del")

		// createTestUser already creates an owner membership, so this should fail
		err := db.DeleteUser(ctx, owner.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "last owner")
	})

	t.Run("GetEnabledVerificationSchedules", func(t *testing.T) {
		_, err := db.GetEnabledVerificationSchedules(ctx)
		require.NoError(t, err)
	})

	t.Run("GetVerificationSchedulesByRepoID", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "vsched-repo")
		vs := models.NewVerificationSchedule(repo.ID, models.VerificationTypeCheck, "0 5 * * *")
		require.NoError(t, db.CreateVerificationSchedule(ctx, vs))

		schedules, err := db.GetVerificationSchedulesByRepoID(ctx, repo.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, schedules)
	})

	t.Run("DeleteVerificationSchedule", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "vsched-del-repo")
		vs := models.NewVerificationSchedule(repo.ID, models.VerificationTypeCheck, "0 6 * * *")
		require.NoError(t, db.CreateVerificationSchedule(ctx, vs))

		err := db.DeleteVerificationSchedule(ctx, vs.ID)
		require.NoError(t, err)
	})

	t.Run("ConsecutiveFailedVerifications", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "consec-fail-repo")
		count, err := db.GetConsecutiveFailedVerifications(ctx, repo.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("DRRunbook_UpdateAllFields", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "dr-update-agent")
		sched := models.NewSchedule(agent.ID, "DR Update Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		runbook := models.NewDRRunbook(org.ID, "Update DR Runbook")
		runbook.ScheduleID = &sched.ID
		runbook.Description = "Initial"
		runbook.Steps = []models.DRRunbookStep{{Order: 1, Title: "Step 1", Description: "Do thing"}}
		runbook.Contacts = []models.DRRunbookContact{{Name: "Alice", Email: "alice@test.com"}}
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		runbook.Description = "Updated"
		runbook.Steps = []models.DRRunbookStep{{Order: 1, Title: "Step 1 Updated", Description: "Do updated thing"}}
		runbook.Contacts = []models.DRRunbookContact{{Name: "Bob", Email: "bob@test.com"}}
		runbook.CredentialsLocation = "vault://secrets/dr"
		rto := 60
		rpo := 15
		runbook.RecoveryTimeObjectiveMins = &rto
		runbook.RecoveryPointObjectiveMins = &rpo
		err := db.UpdateDRRunbook(ctx, runbook)
		require.NoError(t, err)
	})

	t.Run("DRTest_Update", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "dr-test-upd-agent")
		sched := models.NewSchedule(agent.ID, "DR Test Upd Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		runbook := models.NewDRRunbook(org.ID, "DR Test Update Runbook")
		runbook.ScheduleID = &sched.ID
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		drTest := models.NewDRTest(runbook.ID)
		require.NoError(t, db.CreateDRTest(ctx, drTest))

		now := time.Now()
		drTest.CompletedAt = &now
		drTest.Status = "completed"
		passed := true
		drTest.VerificationPassed = &passed
		drTest.Notes = "Test passed successfully"
		err := db.UpdateDRTest(ctx, drTest)
		require.NoError(t, err)
	})

	t.Run("DRTestSchedule_Update", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "dr-ts-upd-agent")
		sched := models.NewSchedule(agent.ID, "DR TS Upd Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		runbook := models.NewDRRunbook(org.ID, "DR TS Update Runbook")
		runbook.ScheduleID = &sched.ID
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		ts := models.NewDRTestSchedule(runbook.ID, "0 0 * * 0")
		require.NoError(t, db.CreateDRTestSchedule(ctx, ts))

		ts.CronExpression = "0 6 * * 1"
		ts.Enabled = false
		err := db.UpdateDRTestSchedule(ctx, ts)
		require.NoError(t, err)
	})

	t.Run("Tag_Update", func(t *testing.T) {
		tag := models.NewTag(org.ID, "update-tag-"+uuid.New().String()[:8], "#FF0000")
		require.NoError(t, db.CreateTag(ctx, tag))

		tag.Color = "#00FF00"
		err := db.UpdateTag(ctx, tag)
		require.NoError(t, err)
	})

	t.Run("SnapshotTags", func(t *testing.T) {
		tag := models.NewTag(org.ID, "snap-tag-"+uuid.New().String()[:8], "#0000FF")
		require.NoError(t, db.CreateTag(ctx, tag))

		snapID := "snap-tag-test-" + uuid.New().String()[:8]
		err := db.AssignTagToSnapshot(ctx, snapID, tag.ID)
		require.NoError(t, err)

		tags, err := db.GetTagsBySnapshotID(ctx, snapID)
		require.NoError(t, err)
		assert.NotEmpty(t, tags)

		err = db.RemoveTagFromSnapshot(ctx, snapID, tag.ID)
		require.NoError(t, err)
	})

	t.Run("SetSnapshotTags", func(t *testing.T) {
		tag1 := models.NewTag(org.ID, "set-snap-t1-"+uuid.New().String()[:8], "#FF0000")
		tag2 := models.NewTag(org.ID, "set-snap-t2-"+uuid.New().String()[:8], "#00FF00")
		require.NoError(t, db.CreateTag(ctx, tag1))
		require.NoError(t, db.CreateTag(ctx, tag2))

		snapID := "set-snap-tags-" + uuid.New().String()[:8]
		err := db.SetSnapshotTags(ctx, snapID, []uuid.UUID{tag1.ID, tag2.ID})
		require.NoError(t, err)

		tags, err := db.GetTagsBySnapshotID(ctx, snapID)
		require.NoError(t, err)
		assert.Len(t, tags, 2)
	})

	t.Run("SetBackupTags", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "set-bkp-tags-agent")
		schedObj := models.NewSchedule(agent.ID, "SetBkpTags Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))
		repo := createTestRepo(t, db, org.ID, "set-bkp-tags-repo")

		b := models.NewBackup(schedObj.ID, agent.ID, &repo.ID)
		b.Complete("snap-set-tags", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		tag := models.NewTag(org.ID, "set-bkp-tag-"+uuid.New().String()[:8], "#FF0000")
		require.NoError(t, db.CreateTag(ctx, tag))

		err := db.SetBackupTags(ctx, b.ID, []uuid.UUID{tag.ID})
		require.NoError(t, err)

		tags, err := db.GetTagsByBackupID(ctx, b.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
	})

	t.Run("ResolveAlertsByResource", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeBackupSLA, models.AlertSeverityCritical, "Test Alert", "Test")
		resourceID := uuid.New()
		resType := models.ResourceTypeAgent
		alert.ResourceID = &resourceID
		alert.ResourceType = &resType
		require.NoError(t, db.CreateAlert(ctx, alert))

		err := db.ResolveAlertsByResource(ctx, models.ResourceTypeAgent, resourceID)
		require.NoError(t, err)
	})

	t.Run("GetAlertByResourceAndType", func(t *testing.T) {
		resourceID := uuid.New()
		resType := models.ResourceTypeAgent
		alert := models.NewAlert(org.ID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Resource Alert", "Test")
		alert.ResourceID = &resourceID
		alert.ResourceType = &resType
		require.NoError(t, db.CreateAlert(ctx, alert))

		got, err := db.GetAlertByResourceAndType(ctx, org.ID, models.ResourceTypeAgent, resourceID, models.AlertTypeAgentOffline)
		require.NoError(t, err)
		assert.Equal(t, alert.ID, got.ID)
	})

	t.Run("GetActiveAlertCountByOrgID", func(t *testing.T) {
		count, err := db.GetActiveAlertCountByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})

	t.Run("GetActiveAlertsByOrgID", func(t *testing.T) {
		alerts, err := db.GetActiveAlertsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, alerts)
	})

	t.Run("SSOGroupMapping_Update", func(t *testing.T) {
		mapping := models.NewSSOGroupMapping(org.ID, "update-group-"+uuid.New().String()[:8], models.OrgRoleMember)
		require.NoError(t, db.CreateSSOGroupMapping(ctx, mapping))

		mapping.Role = models.OrgRoleAdmin
		err := db.UpdateSSOGroupMapping(ctx, mapping)
		require.NoError(t, err)
	})

	t.Run("CostAlert_UpdateTriggered", func(t *testing.T) {
		costAlert := models.NewCostAlert(org.ID, "Trigger Test", 100.0)
		require.NoError(t, db.CreateCostAlert(ctx, costAlert))

		err := db.UpdateCostAlertTriggered(ctx, costAlert.ID)
		require.NoError(t, err)

		alerts, err := db.GetEnabledCostAlerts(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, alerts)
	})

	t.Run("StoragePricing_Update", func(t *testing.T) {
		pricing := models.NewStoragePricing(org.ID, "s3")
		pricing.StoragePerGBMonth = 0.023
		require.NoError(t, db.CreateStoragePricing(ctx, pricing))

		pricing.StoragePerGBMonth = 0.025
		err := db.UpdateStoragePricing(ctx, pricing)
		require.NoError(t, err)
	})

	t.Run("AgentGroup_Update", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Update Group", "desc", "#FF0000")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		group.Name = "Updated Group"
		group.Color = "#00FF00"
		err := db.UpdateAgentGroup(ctx, group)
		require.NoError(t, err)
	})

	t.Run("MembershipUpdate", func(t *testing.T) {
		membOrg := createTestOrg(t, db, "Memb Upd Org", "memb-upd-"+uuid.New().String()[:8])
		user := createTestUser(t, db, membOrg.ID, "memb-upd@test.com", "Memb Upd")

		memberships, err := db.GetMembershipsByUserID(ctx, user.ID)
		require.NoError(t, err)
		require.NotEmpty(t, memberships)

		memberships[0].Role = models.OrgRoleAdmin
		err = db.UpdateMembership(ctx, memberships[0])
		require.NoError(t, err)
	})

	t.Run("UpdateMembershipRole", func(t *testing.T) {
		membOrg := createTestOrg(t, db, "Role Upd Org", "role-upd-"+uuid.New().String()[:8])
		user := createTestUser(t, db, membOrg.ID, "role-upd@test.com", "Role Upd")

		memberships, err := db.GetMembershipsByUserID(ctx, user.ID)
		require.NoError(t, err)
		require.NotEmpty(t, memberships)

		err = db.UpdateMembershipRole(ctx, memberships[0].ID, models.OrgRoleAdmin)
		require.NoError(t, err)
	})

	t.Run("GetOrganizationSSOSettings", func(t *testing.T) {
		defaultRole, autoCreate, err := db.GetOrganizationSSOSettings(ctx, org.ID)
		require.NoError(t, err)
		assert.Nil(t, defaultRole)
		assert.False(t, autoCreate)
	})

	t.Run("UpdateOrganizationSSOSettings", func(t *testing.T) {
		role := "member"
		err := db.UpdateOrganizationSSOSettings(ctx, org.ID, &role, true)
		require.NoError(t, err)

		defaultRole, autoCreate, err := db.GetOrganizationSSOSettings(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, defaultRole)
		assert.True(t, autoCreate)
	})

	t.Run("UpsertUserSSOGroups", func(t *testing.T) {
		user := createTestUser(t, db, org.ID, "sso-groups@test.com", "SSO User")
		err := db.UpsertUserSSOGroups(ctx, user.ID, []string{"admins", "developers"})
		require.NoError(t, err)
	})

	t.Run("GetSSOGroupMappingsByGroupNames", func(t *testing.T) {
		mapping := models.NewSSOGroupMapping(org.ID, "devs-"+uuid.New().String()[:8], models.OrgRoleMember)
		require.NoError(t, db.CreateSSOGroupMapping(ctx, mapping))

		mappings, err := db.GetSSOGroupMappingsByGroupNames(ctx, []string{mapping.OIDCGroupName})
		require.NoError(t, err)
		assert.NotEmpty(t, mappings)
	})

	t.Run("ListPendingMaintenanceNotifications", func(t *testing.T) {
		_, err := db.ListPendingMaintenanceNotifications(ctx)
		require.NoError(t, err)
	})

	t.Run("ListUpcomingMaintenanceWindows", func(t *testing.T) {
		_, err := db.ListUpcomingMaintenanceWindows(ctx, org.ID, time.Now(), 10)
		require.NoError(t, err)
	})

	t.Run("GetExcludePatternsByCategory", func(t *testing.T) {
		ep := models.NewExcludePattern(org.ID, "Cat Pattern", "test", "logs", []string{"*.log"})
		require.NoError(t, db.CreateExcludePattern(ctx, ep))

		patterns, err := db.GetExcludePatternsByCategory(ctx, org.ID, "logs")
		require.NoError(t, err)
		assert.NotEmpty(t, patterns)
	})

	t.Run("GetLatestVerificationByRepoID", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "latest-verif-repo")
		v := models.NewVerification(repo.ID, models.VerificationTypeCheckReadData)
		v.SnapshotID = "latest-verif-snap"
		require.NoError(t, db.CreateVerification(ctx, v))

		latest, err := db.GetLatestVerificationByRepoID(ctx, repo.ID)
		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.Equal(t, v.ID, latest.ID)
	})

	t.Run("GetEnabledDRTestSchedules", func(t *testing.T) {
		_, err := db.GetEnabledDRTestSchedules(ctx)
		require.NoError(t, err)
	})

	t.Run("GetLatestDRTestByRunbookID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "latest-dr-test-agent")
		sched := models.NewSchedule(agent.ID, "Latest DR Test Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		runbook := models.NewDRRunbook(org.ID, "Latest DR Runbook")
		runbook.ScheduleID = &sched.ID
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		drTest := models.NewDRTest(runbook.ID)
		require.NoError(t, db.CreateDRTest(ctx, drTest))

		latest, err := db.GetLatestDRTestByRunbookID(ctx, runbook.ID)
		require.NoError(t, err)
		assert.NotNil(t, latest)
	})

	t.Run("GetEnabledReportSchedules", func(t *testing.T) {
		_, err := db.GetEnabledReportSchedules(ctx)
		require.NoError(t, err)
	})

	t.Run("DeleteDRTest", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "del-dr-test-agent")
		sched := models.NewSchedule(agent.ID, "Del DR Test Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		runbook := models.NewDRRunbook(org.ID, "Del DR Runbook")
		runbook.ScheduleID = &sched.ID
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		drTest := models.NewDRTest(runbook.ID)
		require.NoError(t, db.CreateDRTest(ctx, drTest))

		err := db.DeleteDRTest(ctx, drTest.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteDRTestSchedule", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "del-dr-ts-agent")
		sched := models.NewSchedule(agent.ID, "Del DR TS Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		runbook := models.NewDRRunbook(org.ID, "Del DR TS Runbook")
		runbook.ScheduleID = &sched.ID
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		ts := models.NewDRTestSchedule(runbook.ID, "0 0 * * 0")
		require.NoError(t, db.CreateDRTestSchedule(ctx, ts))

		err := db.DeleteDRTestSchedule(ctx, ts.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteDRRunbook", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "del-dr-rb-agent")
		sched := models.NewSchedule(agent.ID, "Del DR RB Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		runbook := models.NewDRRunbook(org.ID, "Del DR Runbook RB")
		runbook.ScheduleID = &sched.ID
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		err := db.DeleteDRRunbook(ctx, runbook.ID)
		require.NoError(t, err)
	})

	t.Run("DeletePolicy", func(t *testing.T) {
		policy := models.NewPolicy(org.ID, "Delete Policy")
		require.NoError(t, db.CreatePolicy(ctx, policy))

		err := db.DeletePolicy(ctx, policy.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteAlertRule", func(t *testing.T) {
		rule := models.NewAlertRule(org.ID, "Delete Rule", models.AlertTypeBackupSLA, models.AlertRuleConfig{})
		require.NoError(t, db.CreateAlertRule(ctx, rule))

		err := db.DeleteAlertRule(ctx, rule.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteReportSchedule", func(t *testing.T) {
		reportSched := models.NewReportSchedule(org.ID, "Delete Report", models.ReportFrequencyDaily, []string{"a@b.com"})
		require.NoError(t, db.CreateReportSchedule(ctx, reportSched))

		err := db.DeleteReportSchedule(ctx, reportSched.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteNotificationChannel", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Delete Channel", models.ChannelTypeWebhook, []byte("{}"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		err := db.DeleteNotificationChannel(ctx, channel.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteTag", func(t *testing.T) {
		tag := models.NewTag(org.ID, "del-tag-"+uuid.New().String()[:8], "#FFFF00")
		require.NoError(t, db.CreateTag(ctx, tag))

		err := db.DeleteTag(ctx, tag.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteSSOGroupMapping", func(t *testing.T) {
		mapping := models.NewSSOGroupMapping(org.ID, "del-group-"+uuid.New().String()[:8], models.OrgRoleMember)
		require.NoError(t, db.CreateSSOGroupMapping(ctx, mapping))

		err := db.DeleteSSOGroupMapping(ctx, mapping.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteStoragePricing", func(t *testing.T) {
		pricing := models.NewStoragePricing(org.ID, "azure")
		require.NoError(t, db.CreateStoragePricing(ctx, pricing))

		err := db.DeleteStoragePricing(ctx, pricing.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteCostAlert", func(t *testing.T) {
		costAlert := models.NewCostAlert(org.ID, "Delete Cost Alert", 50.0)
		require.NoError(t, db.CreateCostAlert(ctx, costAlert))

		err := db.DeleteCostAlert(ctx, costAlert.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteAgentGroup", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Delete Group", "desc", "#000000")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		err := db.DeleteAgentGroup(ctx, group.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteMembership", func(t *testing.T) {
		delOrg := createTestOrg(t, db, "Del Memb Org", "del-memb-"+uuid.New().String()[:8])
		user := createTestUser(t, db, delOrg.ID, "del-memb@test.com", "Del Memb")

		err := db.DeleteMembership(ctx, user.ID, delOrg.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteSchedule", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "del-sched-agent")
		sched := models.NewSchedule(agent.ID, "Del Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		err := db.DeleteSchedule(ctx, sched.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteBackup_SoftDelete", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "del-bkp-agent")
		sched := models.NewSchedule(agent.ID, "Del Bkp Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))
		repo := createTestRepo(t, db, org.ID, "del-bkp-repo")

		b := models.NewBackup(sched.ID, agent.ID, &repo.ID)
		b.Complete("snap-del-bkp", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		err := db.DeleteBackup(ctx, b.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteAgent", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "del-agent-test")
		err := db.DeleteAgent(ctx, agent.ID)
		require.NoError(t, err)
	})

	t.Run("DeleteRepository", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "del-repo-test")
		err := db.DeleteRepository(ctx, repo.ID)
		require.NoError(t, err)
	})

	t.Run("GetBackupSuccessRates", func(t *testing.T) {
		rate7d, rate30d, err := db.GetBackupSuccessRates(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, rate7d)
		assert.NotNil(t, rate30d)
	})

	t.Run("GetStorageGrowthTrend", func(t *testing.T) {
		trend, err := db.GetStorageGrowthTrend(ctx, org.ID, 30)
		require.NoError(t, err)
		assert.NotNil(t, trend)
	})

	t.Run("GetBackupDurationTrend", func(t *testing.T) {
		trend, err := db.GetBackupDurationTrend(ctx, org.ID, 30)
		require.NoError(t, err)
		assert.NotNil(t, trend)
	})

	t.Run("GetDailyBackupStats", func(t *testing.T) {
		stats, err := db.GetDailyBackupStats(ctx, org.ID, 30)
		require.NoError(t, err)
		assert.NotNil(t, stats)
	})

	t.Run("GetPendingInvitationsByEmail", func(t *testing.T) {
		inviter := createTestUser(t, db, org.ID, "inviter-pe@test.com", "Inviter")
		inv := models.NewOrgInvitation(org.ID, "pending-email@test.com", models.OrgRoleMember, "token-pe-"+uuid.New().String()[:8], inviter.ID, time.Now().Add(24*time.Hour))
		require.NoError(t, db.CreateInvitation(ctx, inv))

		invitations, err := db.GetPendingInvitationsByEmail(ctx, "pending-email@test.com")
		require.NoError(t, err)
		assert.NotEmpty(t, invitations)
	})

	t.Run("GetEnabledEmailChannelsByOrgID", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Email Channel", models.ChannelTypeEmail, []byte("{}"))
		channel.Enabled = true
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		channels, err := db.GetEnabledEmailChannelsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, channels)
	})

	t.Run("UpdateNotificationChannel", func(t *testing.T) {
		channel := models.NewNotificationChannel(org.ID, "Update Channel", models.ChannelTypeSlack, []byte("{}"))
		require.NoError(t, db.CreateNotificationChannel(ctx, channel))

		channel.Name = "Updated Channel"
		channel.Enabled = false
		err := db.UpdateNotificationChannel(ctx, channel)
		require.NoError(t, err)
	})

	t.Run("UpdateAlert", func(t *testing.T) {
		alert := models.NewAlert(org.ID, models.AlertTypeStorageUsage, models.AlertSeverityWarning, "Update Alert", "msg")
		require.NoError(t, db.CreateAlert(ctx, alert))

		alert.Status = models.AlertStatusAcknowledged
		err := db.UpdateAlert(ctx, alert)
		require.NoError(t, err)
	})

	t.Run("UpdateAlertRule", func(t *testing.T) {
		rule := models.NewAlertRule(org.ID, "Update Rule", models.AlertTypeBackupSLA, models.AlertRuleConfig{})
		require.NoError(t, db.CreateAlertRule(ctx, rule))

		rule.Name = "Updated Rule"
		rule.Enabled = false
		err := db.UpdateAlertRule(ctx, rule)
		require.NoError(t, err)
	})

	t.Run("UpdateReplicationStatus", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "repl-upd-agent")
		schedObj := models.NewSchedule(agent.ID, "Repl Upd Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))

		srcRepo := createTestRepo(t, db, org.ID, "repl-upd-src")
		tgtRepo := createTestRepo(t, db, org.ID, "repl-upd-tgt")

		rs, err := db.GetOrCreateReplicationStatus(ctx, schedObj.ID, srcRepo.ID, tgtRepo.ID)
		require.NoError(t, err)

		rs.Status = "synced"
		snapID := "snap-repl"
		rs.LastSnapshotID = &snapID
		now := time.Now()
		rs.LastSyncAt = &now
		err = db.UpdateReplicationStatus(ctx, rs)
		require.NoError(t, err)
	})

	t.Run("UpdateBackupScript", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "bs-upd-agent")
		schedObj := models.NewSchedule(agent.ID, "BS Upd Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))

		script := models.NewBackupScript(schedObj.ID, models.BackupScriptTypePreBackup, "echo old")
		require.NoError(t, db.CreateBackupScript(ctx, script))

		script.Script = "echo new"
		err := db.UpdateBackupScript(ctx, script)
		require.NoError(t, err)
	})

	t.Run("DeleteBackupScript", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "bs-del-agent")
		schedObj := models.NewSchedule(agent.ID, "BS Del Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))

		script := models.NewBackupScript(schedObj.ID, models.BackupScriptTypePostAlways, "echo post")
		require.NoError(t, db.CreateBackupScript(ctx, script))

		err := db.DeleteBackupScript(ctx, script.ID)
		require.NoError(t, err)
	})

	t.Run("UpdateExcludePattern", func(t *testing.T) {
		ep := models.NewExcludePattern(org.ID, "Upd EP", "desc", "temp", []string{"*.bak"})
		require.NoError(t, db.CreateExcludePattern(ctx, ep))

		ep.Name = "Updated EP"
		ep.Patterns = []string{"*.bak", "*.tmp"}
		err := db.UpdateExcludePattern(ctx, ep)
		require.NoError(t, err)
	})

	t.Run("CreateMetricsHistory", func(t *testing.T) {
		m := models.NewMetricsHistory(org.ID)
		m.BackupCount = 10
		m.BackupSuccessCount = 9
		m.BackupFailedCount = 1
		err := db.CreateMetricsHistory(ctx, m)
		require.NoError(t, err)
	})

	t.Run("UpdateBackup", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "upd-bkp-agent")
		schedObj := models.NewSchedule(agent.ID, "Upd Bkp Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))
		repo := createTestRepo(t, db, org.ID, "upd-bkp-repo")

		b := models.NewBackup(schedObj.ID, agent.ID, &repo.ID)
		require.NoError(t, db.CreateBackup(ctx, b))

		b.Complete("snap-upd", 2000, 10, 3)
		err := db.UpdateBackup(ctx, b)
		require.NoError(t, err)
	})

	t.Run("UpdateRestore", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "upd-restore-agent")
		repo := createTestRepo(t, db, org.ID, "upd-restore-repo")
		restore := models.NewRestore(agent.ID, repo.ID, "snap-upd-restore", "/restore", []string{"/a"}, []string{"/b"})
		require.NoError(t, db.CreateRestore(ctx, restore))

		now := time.Now()
		restore.Status = models.RestoreStatusCompleted
		restore.CompletedAt = &now
		err := db.UpdateRestore(ctx, restore)
		require.NoError(t, err)
	})

	t.Run("UpdateUser", func(t *testing.T) {
		user := createTestUser(t, db, org.ID, "upd-user@test.com", "Upd User")
		user.Name = "Updated User"
		err := db.UpdateUser(ctx, user)
		require.NoError(t, err)
	})

	t.Run("UpdateRepository", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "upd-repo")
		repo.Name = "Updated Repo"
		err := db.UpdateRepository(ctx, repo)
		require.NoError(t, err)
	})

	t.Run("CostEstimateHistory", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "cost-hist-repo")
		est := models.NewCostEstimateRecord(org.ID, repo.ID)
		est.MonthlyCost = 10.50
		est.StorageSizeBytes = 107374182400
		require.NoError(t, db.CreateCostEstimate(ctx, est))

		history, err := db.GetCostEstimateHistory(ctx, repo.ID, 30)
		require.NoError(t, err)
		assert.NotEmpty(t, history)
	})

	t.Run("GetAgentsWithGroupsByOrgID", func(t *testing.T) {
		agents, err := db.GetAgentsWithGroupsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, agents)
	})

	t.Run("GetAgentsByGroupID", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Agents In Group", "test", "#FF0000")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		agent := createTestAgent(t, db, org.ID, "in-group-agent")
		require.NoError(t, db.AddAgentToGroup(ctx, group.ID, agent.ID))

		agents, err := db.GetAgentsByGroupID(ctx, group.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, agents)
	})

	t.Run("RemoveAgentFromGroup", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Remove Agent Group", "test", "#FF0000")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		agent := createTestAgent(t, db, org.ID, "remove-from-group")
		require.NoError(t, db.AddAgentToGroup(ctx, group.ID, agent.ID))

		err := db.RemoveAgentFromGroup(ctx, group.ID, agent.ID)
		require.NoError(t, err)
	})

	t.Run("GetGroupsByAgentID", func(t *testing.T) {
		group := models.NewAgentGroup(org.ID, "Get Groups Agent", "test", "#0000FF")
		require.NoError(t, db.CreateAgentGroup(ctx, group))

		agent := createTestAgent(t, db, org.ID, "get-groups-agent")
		require.NoError(t, db.AddAgentToGroup(ctx, group.ID, agent.ID))

		groups, err := db.GetGroupsByAgentID(ctx, agent.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, groups)
	})

	t.Run("GetEnabledAlertRulesByOrgID", func(t *testing.T) {
		_, err := db.GetEnabledAlertRulesByOrgID(ctx, org.ID)
		require.NoError(t, err)
	})

	t.Run("GetStorageStatsByRepositoryID", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "ss-by-repo")
		ss := models.NewStorageStats(repo.ID)
		ss.TotalSize = 500
		require.NoError(t, db.CreateStorageStats(ctx, ss))

		stats, err := db.GetStorageStatsByRepositoryID(ctx, repo.ID, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, stats)
	})

	t.Run("GetLatestStorageStats", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "ss-latest")
		ss := models.NewStorageStats(repo.ID)
		ss.TotalSize = 800
		require.NoError(t, db.CreateStorageStats(ctx, ss))

		latest, err := db.GetLatestStorageStats(ctx, repo.ID)
		require.NoError(t, err)
		assert.NotNil(t, latest)
	})

	t.Run("GetBackupsByTagIDs", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "bkp-tagids-agent")
		schedObj := models.NewSchedule(agent.ID, "BkpTagIDs Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))
		repo := createTestRepo(t, db, org.ID, "bkp-tagids-repo")

		b := models.NewBackup(schedObj.ID, agent.ID, &repo.ID)
		b.Complete("snap-tagids", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		tag := models.NewTag(org.ID, "bkp-tagids-"+uuid.New().String()[:8], "#FF0000")
		require.NoError(t, db.CreateTag(ctx, tag))
		require.NoError(t, db.AssignTagToBackup(ctx, b.ID, tag.ID))

		backups, err := db.GetBackupsByTagIDs(ctx, []uuid.UUID{tag.ID})
		require.NoError(t, err)
		assert.NotEmpty(t, backups)
	})

	t.Run("GetBackupIDsByTagID", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "bkpids-tag-agent")
		schedObj := models.NewSchedule(agent.ID, "BkpIDsTag Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))
		repo := createTestRepo(t, db, org.ID, "bkpids-tag-repo")

		b := models.NewBackup(schedObj.ID, agent.ID, &repo.ID)
		b.Complete("snap-bkpids", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		tag := models.NewTag(org.ID, "bkpids-tag-"+uuid.New().String()[:8], "#FF0000")
		require.NoError(t, db.CreateTag(ctx, tag))
		require.NoError(t, db.AssignTagToBackup(ctx, b.ID, tag.ID))

		ids, err := db.GetBackupIDsByTagID(ctx, tag.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, ids)
	})

	t.Run("RemoveTagFromBackup", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "rm-tag-bkp-agent")
		schedObj := models.NewSchedule(agent.ID, "RmTagBkp Sched", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, schedObj))
		repo := createTestRepo(t, db, org.ID, "rm-tag-bkp-repo")

		b := models.NewBackup(schedObj.ID, agent.ID, &repo.ID)
		b.Complete("snap-rm-tag", 1000, 5, 2)
		require.NoError(t, db.CreateBackup(ctx, b))

		tag := models.NewTag(org.ID, "rm-tag-bkp-"+uuid.New().String()[:8], "#FF0000")
		require.NoError(t, db.CreateTag(ctx, tag))
		require.NoError(t, db.AssignTagToBackup(ctx, b.ID, tag.ID))

		err := db.RemoveTagFromBackup(ctx, b.ID, tag.ID)
		require.NoError(t, err)
	})

	t.Run("GetDRTestsByOrgID", func(t *testing.T) {
		tests, err := db.GetDRTestsByOrgID(ctx, org.ID)
		require.NoError(t, err)
		assert.NotNil(t, tests)
	})

	t.Run("UpdateOnboardingSkip", func(t *testing.T) {
		skipOrg := createTestOrg(t, db, "Skip Onboard Org", "skip-onb-"+uuid.New().String()[:8])
		progress, err := db.GetOrCreateOnboardingProgress(ctx, skipOrg.ID)
		require.NoError(t, err)
		assert.NotNil(t, progress)

		err = db.SkipOnboarding(ctx, skipOrg.ID)
		require.NoError(t, err)
	})

	t.Run("CreateStorageStats", func(t *testing.T) {
		repo := createTestRepo(t, db, org.ID, "ss-create-test")
		ss := models.NewStorageStats(repo.ID)
		ss.TotalSize = 100
		ss.TotalFileCount = 10
		err := db.CreateStorageStats(ctx, ss)
		require.NoError(t, err)
	})
}

// TestStore_ScheduleAdvancedFields tests schedule creation/update with all optional fields.
func TestStore_ScheduleAdvancedFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Sched Adv Org", "sched-adv-"+uuid.New().String()[:8])

	t.Run("CreateScheduleWithBackupWindow", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "sched-bw-agent")
		sched := models.NewSchedule(agent.ID, "BW Schedule", "0 0 * * *", []string{"/data"})
		sched.BackupWindow = &models.BackupWindow{Start: "02:00", End: "06:00"}
		sched.Excludes = []string{"*.tmp", "cache/"}
		sched.ExcludedHours = []int{8, 9, 10}
		bwLimit := 1024
		sched.BandwidthLimitKB = &bwLimit
		compression := "max"
		sched.CompressionLevel = &compression
		sched.OnMountUnavailable = models.MountBehaviorSkip

		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.NotNil(t, got.BackupWindow)
		assert.Equal(t, models.MountBehaviorSkip, got.OnMountUnavailable)
		assert.Equal(t, []string{"*.tmp", "cache/"}, got.Excludes)
		assert.Equal(t, []int{8, 9, 10}, got.ExcludedHours)
	})

	t.Run("CreateScheduleWithEmptyMountBehavior", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "sched-empty-mb-agent")
		sched := models.NewSchedule(agent.ID, "Empty MB Schedule", "0 0 * * *", []string{"/data"})
		sched.OnMountUnavailable = "" // triggers mountBehavior == "" fallback

		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, models.MountBehaviorFail, got.OnMountUnavailable)
	})

	t.Run("CreateScheduleWithRepositories", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "sched-repos-agent")
		repo := createTestRepo(t, db, org.ID, "sched-repos-repo")
		sched := models.NewSchedule(agent.ID, "Repos Schedule", "0 0 * * *", []string{"/data"})
		sched.Repositories = []models.ScheduleRepository{
			*models.NewScheduleRepository(uuid.Nil, repo.ID, 0),
		}

		err := db.CreateSchedule(ctx, sched)
		require.NoError(t, err)

		repos, err := db.GetScheduleRepositories(ctx, sched.ID)
		require.NoError(t, err)
		assert.Len(t, repos, 1)
	})

	t.Run("UpdateScheduleWithBackupWindow", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "sched-bw-upd-agent")
		sched := models.NewSchedule(agent.ID, "BW Upd Schedule", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		sched.BackupWindow = &models.BackupWindow{Start: "22:00", End: "05:00"}
		sched.OnMountUnavailable = models.MountBehaviorSkip
		sched.ExcludedHours = []int{12, 13, 14}
		err := db.UpdateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.NotNil(t, got.BackupWindow)
		assert.Equal(t, models.MountBehaviorSkip, got.OnMountUnavailable)
	})

	t.Run("UpdateScheduleEmptyMountBehavior", func(t *testing.T) {
		agent := createTestAgent(t, db, org.ID, "sched-mb-upd-agent")
		sched := models.NewSchedule(agent.ID, "MB Upd Schedule", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		sched.OnMountUnavailable = ""
		err := db.UpdateSchedule(ctx, sched)
		require.NoError(t, err)

		got, err := db.GetScheduleByID(ctx, sched.ID)
		require.NoError(t, err)
		assert.Equal(t, models.MountBehaviorFail, got.OnMountUnavailable)
	})
}

// TestStore_PolicyAdvancedFields tests policy creation with all optional fields.
func TestStore_PolicyAdvancedFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Policy Adv Org", "policy-adv-"+uuid.New().String()[:8])

	t.Run("CreatePolicyWithAllFields", func(t *testing.T) {
		policy := models.NewPolicy(org.ID, "Full Policy")
		policy.Description = "A comprehensive policy"
		policy.Paths = []string{"/data", "/home"}
		policy.Excludes = []string{"*.tmp"}
		policy.CronExpression = "0 2 * * *"
		policy.BackupWindow = &models.BackupWindow{Start: "00:00", End: "04:00"}
		policy.ExcludedHours = []int{9, 10, 11}
		bwLimit := 512
		policy.BandwidthLimitKB = &bwLimit

		err := db.CreatePolicy(ctx, policy)
		require.NoError(t, err)

		got, err := db.GetPolicyByID(ctx, policy.ID)
		require.NoError(t, err)
		assert.Equal(t, "Full Policy", got.Name)
		assert.Equal(t, []string{"/data", "/home"}, got.Paths)
	})

	t.Run("UpdatePolicyAllFields", func(t *testing.T) {
		policy := models.NewPolicy(org.ID, "Upd Policy")
		require.NoError(t, db.CreatePolicy(ctx, policy))

		policy.Name = "Updated Policy"
		policy.Description = "Updated desc"
		policy.Paths = []string{"/newpath"}
		policy.Excludes = []string{"*.bak"}
		policy.CronExpression = "0 3 * * *"
		policy.BackupWindow = &models.BackupWindow{Start: "01:00", End: "05:00"}
		bwLimit := 1024
		policy.BandwidthLimitKB = &bwLimit

		err := db.UpdatePolicy(ctx, policy)
		require.NoError(t, err)

		got, err := db.GetPolicyByID(ctx, policy.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Policy", got.Name)
	})
}

// TestStore_DRRunbookAdvanced tests DR runbook creation with steps and contacts.
func TestStore_DRRunbookAdvanced(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "DR Adv Org", "dr-adv-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "dr-adv-agent")
	sched := models.NewSchedule(agent.ID, "DR Adv Sched", "0 0 * * *", []string{"/data"})
	require.NoError(t, db.CreateSchedule(ctx, sched))

	t.Run("CreateRunbookWithStepsAndContacts", func(t *testing.T) {
		runbook := models.NewDRRunbook(org.ID, "Full DR Runbook")
		runbook.ScheduleID = &sched.ID
		runbook.Description = "Complete DR plan"
		runbook.Steps = []models.DRRunbookStep{
			{Order: 1, Title: "Stop Services", Description: "Stop all running services", Type: models.DRRunbookStepTypeManual},
			{Order: 2, Title: "Restore Data", Description: "Restore from latest backup", Type: models.DRRunbookStepTypeRestore},
			{Order: 3, Title: "Verify", Description: "Verify data integrity", Type: models.DRRunbookStepTypeVerify},
		}
		runbook.Contacts = []models.DRRunbookContact{
			{Name: "SRE Team", Email: "sre@example.com", Phone: "+1-555-0100", Role: "Primary"},
			{Name: "DBA Team", Email: "dba@example.com", Role: "Secondary"},
		}
		runbook.CredentialsLocation = "vault://secrets/dr-creds"
		rto := 120
		rpo := 30
		runbook.RecoveryTimeObjectiveMins = &rto
		runbook.RecoveryPointObjectiveMins = &rpo

		err := db.CreateDRRunbook(ctx, runbook)
		require.NoError(t, err)

		got, err := db.GetDRRunbookByID(ctx, runbook.ID)
		require.NoError(t, err)
		assert.Equal(t, "Full DR Runbook", got.Name)
		assert.Len(t, got.Steps, 3)
		assert.Len(t, got.Contacts, 2)
	})
}

// TestStore_VerificationAdvanced tests verification lifecycle with all fields.
func TestStore_VerificationAdvanced(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Verif Adv Org", "verif-adv-"+uuid.New().String()[:8])
	repo := createTestRepo(t, db, org.ID, "verif-adv-repo")

	t.Run("CreateVerificationWithDetails", func(t *testing.T) {
		v := models.NewVerification(repo.ID, models.VerificationTypeCheckReadData)
		v.SnapshotID = "detailed-snap-123"
		v.Details = &models.VerificationDetails{
			FilesRestored: 5,
			BytesRestored: 1024,
		}

		err := db.CreateVerification(ctx, v)
		require.NoError(t, err)

		got, err := db.GetVerificationByID(ctx, v.ID)
		require.NoError(t, err)
		assert.Equal(t, "detailed-snap-123", got.SnapshotID)
		assert.NotNil(t, got.Details)
	})

	t.Run("UpdateVerificationWithDetails", func(t *testing.T) {
		v := models.NewVerification(repo.ID, models.VerificationTypeCheck)
		v.SnapshotID = "upd-snap-456"
		require.NoError(t, db.CreateVerification(ctx, v))

		now := time.Now()
		dur := int64(1500)
		v.Status = models.VerificationStatusPassed
		v.CompletedAt = &now
		v.DurationMs = &dur
		v.SnapshotID = "upd-snap-789" // Change SnapshotID
		v.Details = &models.VerificationDetails{
			FilesRestored:  10,
			BytesRestored:  2048,
			ReadDataSubset: "5%",
		}

		err := db.UpdateVerification(ctx, v)
		require.NoError(t, err)
	})

	t.Run("VerificationWithReadDataSubset", func(t *testing.T) {
		vs := models.NewVerificationSchedule(repo.ID, models.VerificationTypeCheckReadData, "0 7 * * *")
		vs.ReadDataSubset = "5%"
		require.NoError(t, db.CreateVerificationSchedule(ctx, vs))

		got, err := db.GetVerificationScheduleByID(ctx, vs.ID)
		require.NoError(t, err)
		assert.Equal(t, "5%", got.ReadDataSubset)

		// Update with new subset
		vs.ReadDataSubset = "10%"
		err = db.UpdateVerificationSchedule(ctx, vs)
		require.NoError(t, err)
	})
}

// TestStore_RestoreWithPaths tests restore creation with include/exclude paths.
func TestStore_RestoreWithPaths(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Restore Path Org", "restore-path-"+uuid.New().String()[:8])
	agent := createTestAgent(t, db, org.ID, "restore-path-agent")
	repo := createTestRepo(t, db, org.ID, "restore-path-repo")

	t.Run("RestoreWithIncludeExcludePaths", func(t *testing.T) {
		restore := models.NewRestore(agent.ID, repo.ID, "snap-paths", "/target",
			[]string{"/include/a", "/include/b"},
			[]string{"/exclude/c"})
		err := db.CreateRestore(ctx, restore)
		require.NoError(t, err)

		got, err := db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"/include/a", "/include/b"}, got.IncludePaths)
		assert.Equal(t, []string{"/exclude/c"}, got.ExcludePaths)
	})

	t.Run("RestoreUpdateLifecycle", func(t *testing.T) {
		restore := models.NewRestore(agent.ID, repo.ID, "snap-lifecycle", "/target2", nil, nil)
		require.NoError(t, db.CreateRestore(ctx, restore))

		now := time.Now()
		restore.Status = models.RestoreStatusRunning
		restore.StartedAt = &now
		err := db.UpdateRestore(ctx, restore)
		require.NoError(t, err)

		completed := time.Now()
		restore.Status = models.RestoreStatusCompleted
		restore.CompletedAt = &completed
		err = db.UpdateRestore(ctx, restore)
		require.NoError(t, err)

		got, err := db.GetRestoreByID(ctx, restore.ID)
		require.NoError(t, err)
		assert.Equal(t, models.RestoreStatusCompleted, got.Status)
	})
}

// TestStore_MigrateIdempotent tests that running Migrate twice works (covers "already applied" path).
func TestStore_MigrateIdempotent(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Migrate already ran in setupTestDB, run again to cover the "already applied" branch
	err := db.Migrate(ctx)
	require.NoError(t, err)

	version, err := db.CurrentVersion(ctx)
	require.NoError(t, err)
	assert.True(t, version >= 1)
}

// TestStore_ExecTxRollback tests ExecTx when the function returns an error (covers rollback path).
func TestStore_ExecTxRollback(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	testErr := fmt.Errorf("intentional test error")
	err := db.ExecTx(ctx, func(tx pgx.Tx) error {
		return testErr
	})
	require.Error(t, err)
	assert.Equal(t, testErr, err)
}

// TestStore_ReportHistoryScanWithData tests scanReportHistory with non-nil ReportData via GetReportHistoryByOrgID.
func TestStore_ReportHistoryScanWithData(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Report Data Org", "report-data-"+uuid.New().String()[:8])

	t.Run("CreateReportHistoryWithReportData", func(t *testing.T) {
		now := time.Now()
		periodStart := now.Add(-24 * time.Hour)
		rh := models.NewReportHistory(org.ID, nil, "daily", periodStart, now, []string{"admin@example.com"})
		rh.ReportData = &models.ReportData{
			BackupSummary: models.BackupSummary{
				TotalBackups:      100,
				SuccessfulBackups: 95,
				FailedBackups:     5,
				SuccessRate:       0.95,
			},
			StorageSummary: models.StorageSummary{
				TotalRawSize:    1024 * 1024 * 1024,
				RepositoryCount: 3,
			},
			AgentSummary: models.AgentSummary{
				TotalAgents:  5,
				ActiveAgents: 4,
			},
			AlertSummary: models.AlertSummary{
				TotalAlerts:    10,
				CriticalAlerts: 2,
			},
		}
		rh.MarkSent()

		err := db.CreateReportHistory(ctx, rh)
		require.NoError(t, err)

		// Fetch by org to trigger scanReportHistory with data
		history, err := db.GetReportHistoryByOrgID(ctx, org.ID, 10)
		require.NoError(t, err)
		require.Len(t, history, 1)
		assert.NotNil(t, history[0].ReportData)
		assert.Equal(t, 100, history[0].ReportData.BackupSummary.TotalBackups)
	})
}

// TestStore_ForeignKeyErrors tests that operations with invalid foreign keys return errors.
func TestStore_ForeignKeyErrors(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	fakeID := uuid.New()

	t.Run("CreateScheduleInvalidAgent", func(t *testing.T) {
		sched := models.NewSchedule(fakeID, "Bad Schedule", "0 0 * * *", []string{"/data"})
		err := db.CreateSchedule(ctx, sched)
		require.Error(t, err)
	})

	t.Run("CreateBackupInvalidSchedule", func(t *testing.T) {
		backup := models.NewBackup(fakeID, fakeID, nil)
		err := db.CreateBackup(ctx, backup)
		require.Error(t, err)
	})

	t.Run("CreateRestoreInvalidAgent", func(t *testing.T) {
		restore := models.NewRestore(fakeID, fakeID, "snap-1", "/target", nil, nil)
		err := db.CreateRestore(ctx, restore)
		require.Error(t, err)
	})

	t.Run("CreateAlertRuleInvalidOrg", func(t *testing.T) {
		rule := models.NewAlertRule(fakeID, "Bad Rule", models.AlertTypeBackupSLA, models.AlertRuleConfig{})
		err := db.CreateAlertRule(ctx, rule)
		require.Error(t, err)
	})

	t.Run("CreateNotificationChannelInvalidOrg", func(t *testing.T) {
		ch := models.NewNotificationChannel(fakeID, "Bad Channel", models.ChannelTypeEmail, []byte("cfg"))
		err := db.CreateNotificationChannel(ctx, ch)
		require.Error(t, err)
	})

	t.Run("CreatePolicyInvalidOrg", func(t *testing.T) {
		policy := models.NewPolicy(fakeID, "Bad Policy")
		err := db.CreatePolicy(ctx, policy)
		require.Error(t, err)
	})

	t.Run("CreateAgentInvalidOrg", func(t *testing.T) {
		agent := models.NewAgent(fakeID, "bad-agent", "hash")
		err := db.CreateAgent(ctx, agent)
		require.Error(t, err)
	})

	t.Run("CreateRepositoryInvalidOrg", func(t *testing.T) {
		repo := models.NewRepository(fakeID, "bad-repo", models.RepositoryTypeLocal, nil)
		err := db.CreateRepository(ctx, repo)
		require.Error(t, err)
	})

	t.Run("CreateStorageStatsInvalidRepo", func(t *testing.T) {
		ss := models.NewStorageStats(fakeID)
		err := db.CreateStorageStats(ctx, ss)
		require.Error(t, err)
	})

	t.Run("CreateTagInvalidOrg", func(t *testing.T) {
		tag := models.NewTag(fakeID, "bad-tag", "#000000")
		err := db.CreateTag(ctx, tag)
		require.Error(t, err)
	})

	t.Run("CreateDRRunbookInvalidOrg", func(t *testing.T) {
		runbook := models.NewDRRunbook(fakeID, "Bad Runbook")
		err := db.CreateDRRunbook(ctx, runbook)
		require.Error(t, err)
	})

	t.Run("CreateDRTestInvalidRunbook", func(t *testing.T) {
		test := models.NewDRTest(fakeID)
		err := db.CreateDRTest(ctx, test)
		require.Error(t, err)
	})

	t.Run("CreateVerificationInvalidRepo", func(t *testing.T) {
		v := models.NewVerification(fakeID, models.VerificationTypeCheck)
		err := db.CreateVerification(ctx, v)
		require.Error(t, err)
	})

	t.Run("CreateMembershipInvalidUser", func(t *testing.T) {
		org := createTestOrg(t, db, "FK Err Org", "fk-err-"+uuid.New().String()[:8])
		membership := models.NewOrgMembership(fakeID, org.ID, models.OrgRoleMember)
		err := db.CreateMembership(ctx, membership)
		require.Error(t, err)
	})

	t.Run("CreateInvitationInvalidOrg", func(t *testing.T) {
		inv := models.NewOrgInvitation(fakeID, "test@example.com", models.OrgRoleMember, uuid.New().String(), fakeID, time.Now().Add(72*time.Hour))
		err := db.CreateInvitation(ctx, inv)
		require.Error(t, err)
	})

	t.Run("CreateNotificationPreferenceInvalidUser", func(t *testing.T) {
		pref := models.NewNotificationPreference(fakeID, fakeID, models.EventBackupSuccess)
		err := db.CreateNotificationPreference(ctx, pref)
		require.Error(t, err)
	})

	t.Run("CreateNotificationLogInvalidChannel", func(t *testing.T) {
		org := createTestOrg(t, db, "NL Err Org", "nl-err-"+uuid.New().String()[:8])
		log := models.NewNotificationLog(org.ID, &fakeID, "backup_success", "user@test.com", "Test")
		err := db.CreateNotificationLog(ctx, log)
		require.Error(t, err)
	})

	t.Run("CreateRepositoryKeyInvalidRepo", func(t *testing.T) {
		rk := models.NewRepositoryKey(fakeID, []byte("encrypted-key"), false, nil)
		err := db.CreateRepositoryKey(ctx, rk)
		require.Error(t, err)
	})

	t.Run("CreateAuditLogInvalidOrg", func(t *testing.T) {
		al := models.NewAuditLog(fakeID, models.AuditActionCreate, "test", models.AuditResultSuccess)
		err := db.CreateAuditLog(ctx, al)
		require.Error(t, err)
	})

	t.Run("CreateMaintenanceWindowInvalidOrg", func(t *testing.T) {
		mw := models.NewMaintenanceWindow(fakeID, "Bad Window", time.Now(), time.Now().Add(time.Hour))
		err := db.CreateMaintenanceWindow(ctx, mw)
		require.Error(t, err)
	})

	t.Run("CreateExcludePatternInvalidOrg", func(t *testing.T) {
		ep := models.NewExcludePattern(fakeID, "Bad Pattern", "test", "general", []string{"*.tmp"})
		err := db.CreateExcludePattern(ctx, ep)
		require.Error(t, err)
	})

	t.Run("CreateBackupScriptInvalidSchedule", func(t *testing.T) {
		bs := models.NewBackupScript(fakeID, models.BackupScriptTypePreBackup, "echo test")
		err := db.CreateBackupScript(ctx, bs)
		require.Error(t, err)
	})

	t.Run("CreateScheduleRepositoryInvalidSchedule", func(t *testing.T) {
		sr := models.NewScheduleRepository(fakeID, fakeID, 0)
		err := db.CreateScheduleRepository(ctx, sr)
		require.Error(t, err)
	})

	t.Run("CreateVerificationScheduleInvalidRepo", func(t *testing.T) {
		vs := models.NewVerificationSchedule(fakeID, models.VerificationTypeCheck, "0 0 * * *")
		err := db.CreateVerificationSchedule(ctx, vs)
		require.Error(t, err)
	})

	t.Run("CreateDRTestScheduleInvalidRunbook", func(t *testing.T) {
		dts := models.NewDRTestSchedule(fakeID, "0 0 * * *")
		err := db.CreateDRTestSchedule(ctx, dts)
		require.Error(t, err)
	})

	t.Run("CreateReportScheduleInvalidOrg", func(t *testing.T) {
		rs := models.NewReportSchedule(fakeID, "Bad Report", models.ReportFrequencyDaily, []string{"admin@test.com"})
		err := db.CreateReportSchedule(ctx, rs)
		require.Error(t, err)
	})

	t.Run("CreateReportHistoryInvalidOrg", func(t *testing.T) {
		rh := models.NewReportHistory(fakeID, nil, "daily", time.Now().Add(-24*time.Hour), time.Now(), []string{"admin@test.com"})
		err := db.CreateReportHistory(ctx, rh)
		require.Error(t, err)
	})

	t.Run("CreateAgentHealthHistoryInvalidAgent", func(t *testing.T) {
		h := models.NewAgentHealthHistory(fakeID, fakeID, models.HealthStatusHealthy, nil, nil)
		err := db.CreateAgentHealthHistory(ctx, h)
		require.Error(t, err)
	})

	t.Run("AssignTagToBackupInvalidTag", func(t *testing.T) {
		err := db.AssignTagToBackup(ctx, fakeID, fakeID)
		require.Error(t, err)
	})

	t.Run("AssignTagToSnapshotInvalidTag", func(t *testing.T) {
		err := db.AssignTagToSnapshot(ctx, "fake-snap", fakeID)
		require.Error(t, err)
	})

	t.Run("CreateSSOGroupMappingInvalidOrg", func(t *testing.T) {
		mapping := models.NewSSOGroupMapping(fakeID, "bad-group", models.OrgRoleMember)
		err := db.CreateSSOGroupMapping(ctx, mapping)
		require.Error(t, err)
	})

	t.Run("CreateStoragePricingInvalidOrg", func(t *testing.T) {
		sp := models.NewStoragePricing(fakeID, "s3")
		err := db.CreateStoragePricing(ctx, sp)
		require.Error(t, err)
	})

	t.Run("CreateCostAlertInvalidOrg", func(t *testing.T) {
		ca := models.NewCostAlert(fakeID, "cost-high", 100.0)
		err := db.CreateCostAlert(ctx, ca)
		require.Error(t, err)
	})

	t.Run("CreateAgentGroupInvalidOrg", func(t *testing.T) {
		ag := models.NewAgentGroup(fakeID, "Bad Group", "desc", "#000")
		err := db.CreateAgentGroup(ctx, ag)
		require.Error(t, err)
	})

	t.Run("AddAgentToGroupInvalidGroup", func(t *testing.T) {
		err := db.AddAgentToGroup(ctx, fakeID, fakeID)
		require.Error(t, err)
	})

	t.Run("CreateMetricsHistoryInvalidOrg", func(t *testing.T) {
		mh := models.NewMetricsHistory(fakeID)
		err := db.CreateMetricsHistory(ctx, mh)
		require.Error(t, err)
	})

	t.Run("CreateSnapshotCommentInvalidOrg", func(t *testing.T) {
		sc := models.NewSnapshotComment(fakeID, "snap-123", fakeID, "comment")
		err := db.CreateSnapshotComment(ctx, sc)
		require.Error(t, err)
	})

	t.Run("SetScheduleRepositoriesInvalidRepo", func(t *testing.T) {
		org := createTestOrg(t, db, "SSR Err Org", "ssr-err-"+uuid.New().String()[:8])
		agent := createTestAgent(t, db, org.ID, "ssr-err-agent")
		sched := models.NewSchedule(agent.ID, "SSR Schedule", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		repos := []models.ScheduleRepository{
			*models.NewScheduleRepository(sched.ID, fakeID, 0), // invalid repo
		}
		err := db.SetScheduleRepositories(ctx, sched.ID, repos)
		require.Error(t, err)
	})

	t.Run("UpdateScheduleInvalidPolicy", func(t *testing.T) {
		org := createTestOrg(t, db, "US Err Org", "us-err-"+uuid.New().String()[:8])
		agent := createTestAgent(t, db, org.ID, "us-err-agent")
		sched := models.NewSchedule(agent.ID, "US Schedule", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		sched.PolicyID = &fakeID // invalid FK - policy_id IS in SET clause
		err := db.UpdateSchedule(ctx, sched)
		require.Error(t, err)
	})

	t.Run("UpdateDRRunbookInvalidSchedule", func(t *testing.T) {
		org := createTestOrg(t, db, "UDR Err Org", "udr-err-"+uuid.New().String()[:8])
		runbook := models.NewDRRunbook(org.ID, "UDR Runbook")
		require.NoError(t, db.CreateDRRunbook(ctx, runbook))

		runbook.ScheduleID = &fakeID // invalid FK - schedule_id IS in SET clause
		err := db.UpdateDRRunbook(ctx, runbook)
		require.Error(t, err)
	})

	t.Run("CreateAlertInvalidOrg", func(t *testing.T) {
		alert := models.NewAlert(fakeID, models.AlertTypeBackupSLA, models.AlertSeverityCritical, "Bad Alert", "Bad message")
		err := db.CreateAlert(ctx, alert)
		require.Error(t, err)
	})

	t.Run("CreateOnboardingProgressInvalidOrg", func(t *testing.T) {
		_, err := db.GetOrCreateOnboardingProgress(ctx, fakeID)
		_ = err
	})

	t.Run("CreateCostEstimateInvalidOrg", func(t *testing.T) {
		ce := models.NewCostEstimateRecord(fakeID, fakeID)
		err := db.CreateCostEstimate(ctx, ce)
		require.Error(t, err)
	})

	t.Run("DuplicateOrganizationSlug", func(t *testing.T) {
		slug := "dup-slug-" + uuid.New().String()[:8]
		org1 := models.NewOrganization("Dup Org 1", slug)
		require.NoError(t, db.CreateOrganization(ctx, org1))

		org2 := models.NewOrganization("Dup Org 2", slug)
		err := db.CreateOrganization(ctx, org2)
		require.Error(t, err)
	})

	t.Run("DuplicateUserOIDCSubject", func(t *testing.T) {
		org := createTestOrg(t, db, "DupU Org", "dup-u-"+uuid.New().String()[:8])
		subject := "dup-oidc-" + uuid.New().String()
		user1 := models.NewUser(org.ID, subject, "user1@test.com", "User 1", models.UserRoleAdmin)
		require.NoError(t, db.CreateUser(ctx, user1))

		user2 := models.NewUser(org.ID, subject, "user2@test.com", "User 2", models.UserRoleAdmin)
		err := db.CreateUser(ctx, user2)
		require.Error(t, err)
	})

	t.Run("DuplicateAgentHostname", func(t *testing.T) {
		org := createTestOrg(t, db, "DupA Org", "dup-a-"+uuid.New().String()[:8])
		agent1 := createTestAgent(t, db, org.ID, "dup-host")
		_ = agent1

		agent2 := models.NewAgent(org.ID, "dup-host", "hash2")
		err := db.CreateAgent(ctx, agent2)
		require.Error(t, err)
	})

	t.Run("DuplicateRepositoryName", func(t *testing.T) {
		org := createTestOrg(t, db, "DupR Org", "dup-r-"+uuid.New().String()[:8])
		repo1 := createTestRepo(t, db, org.ID, "dup-repo")
		_ = repo1

		repo2 := models.NewRepository(org.ID, "dup-repo", models.RepositoryTypeLocal, nil)
		err := db.CreateRepository(ctx, repo2)
		require.Error(t, err)
	})

	t.Run("DuplicateTagName", func(t *testing.T) {
		org := createTestOrg(t, db, "DupT Org", "dup-t-"+uuid.New().String()[:8])
		tag1 := models.NewTag(org.ID, "dup-tag", "#000")
		require.NoError(t, db.CreateTag(ctx, tag1))

		tag2 := models.NewTag(org.ID, "dup-tag", "#fff")
		err := db.CreateTag(ctx, tag2)
		require.Error(t, err)
	})

	t.Run("UpdateScheduleRepositoriesError", func(t *testing.T) {
		// Test the error path in CreateScheduleRepository when called from SetScheduleRepositories
		org := createTestOrg(t, db, "CSR Err Org", "csr-err-"+uuid.New().String()[:8])
		agent := createTestAgent(t, db, org.ID, "csr-err-agent")
		sched := models.NewSchedule(agent.ID, "CSR Schedule", "0 0 * * *", []string{"/data"})
		require.NoError(t, db.CreateSchedule(ctx, sched))

		sr := models.NewScheduleRepository(sched.ID, fakeID, 0)
		err := db.CreateScheduleRepository(ctx, sr)
		require.Error(t, err)
	})
}

// TestStore_CurrentVersionBeforeMigrate tests CurrentVersion on a fresh DB where
// the schema_migrations table does not exist, covering the "does not exist" error branch.
func TestStore_CurrentVersionBeforeMigrate(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("keldris_test_fresh"),
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

	// Connect WITHOUT running Migrate - schema_migrations table does not exist
	database, err := New(ctx, cfg, logger)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	// CurrentVersion should return 0 via the "does not exist" error branch
	version, err := database.CurrentVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, version)
}

// TestStore_EmptyInputEarlyReturns tests functions that return early on empty input slices.
func TestStore_EmptyInputEarlyReturns(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	t.Run("GetBackupsByTagIDs_empty", func(t *testing.T) {
		backups, err := db.GetBackupsByTagIDs(ctx, []uuid.UUID{})
		require.NoError(t, err)
		assert.Nil(t, backups)
	})

	t.Run("GetSSOGroupMappingsByGroupNames_empty", func(t *testing.T) {
		mappings, err := db.GetSSOGroupMappingsByGroupNames(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, mappings)
	})
}

// TestStore_ReportScheduleEmptyRecipients tests creating and updating report schedules
// with empty recipients to cover the recipientsBytes == nil branches.
func TestStore_ReportScheduleEmptyRecipients(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "Empty Recip Org", "empty-recip-"+uuid.New().String()[:8])

	// Create with empty recipients - covers CreateReportSchedule nil recipients branch
	sched := models.NewReportSchedule(org.ID, "No Recipients", models.ReportFrequencyDaily, nil)
	err := db.CreateReportSchedule(ctx, sched)
	require.NoError(t, err)

	got, err := db.GetReportScheduleByID(ctx, sched.ID)
	require.NoError(t, err)
	assert.Equal(t, sched.ID, got.ID)

	// Update with empty recipients - covers UpdateReportSchedule nil recipients branch
	sched.Recipients = nil
	sched.Name = "Still No Recipients"
	err = db.UpdateReportSchedule(ctx, sched)
	require.NoError(t, err)

	got2, err := db.GetReportScheduleByID(ctx, sched.ID)
	require.NoError(t, err)
	assert.Equal(t, "Still No Recipients", got2.Name)
}

// TestStore_ReportHistoryEmptyRecipients tests creating report history
// with empty recipients to cover the nil recipientsBytes branch.
func TestStore_ReportHistoryEmptyRecipients(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	org := createTestOrg(t, db, "RH Empty Org", "rh-empty-"+uuid.New().String()[:8])

	now := time.Now()
	history := models.NewReportHistory(org.ID, nil, "daily", now.Add(-24*time.Hour), now, nil)
	err := db.CreateReportHistory(ctx, history)
	require.NoError(t, err)

	results, err := db.GetReportHistoryByOrgID(ctx, org.ID, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

// TestStore_CanceledContextErrors tests that DB methods return errors when called with a canceled context.
// This exercises the error-return paths in query, exec, and scan operations.
func TestStore_CanceledContextErrors(t *testing.T) {
	db := setupTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	id := uuid.New()
	orgID := uuid.New()
	now := time.Now()

	t.Run("OrganizationMethods", func(t *testing.T) {
		_, err := db.GetAllOrganizations(ctx)
		assert.Error(t, err)

		_, err = db.GetOrganizationByID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetOrganizationBySlug(ctx, "test")
		assert.Error(t, err)

		err = db.UpdateOrganization(ctx, &models.Organization{ID: id})
		assert.Error(t, err)

		err = db.DeleteOrganization(ctx, id)
		assert.Error(t, err)
	})

	t.Run("UserMethods", func(t *testing.T) {
		_, err := db.GetUserByID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetUserByOIDCSubject(ctx, "sub")
		assert.Error(t, err)

		_, err = db.ListUsers(ctx, orgID)
		assert.Error(t, err)

		err = db.CreateUser(ctx, &models.User{ID: id, OrgID: orgID, OIDCSubject: "sub", Email: "a@b.com"})
		assert.Error(t, err)

		err = db.UpdateUser(ctx, &models.User{ID: id})
		assert.Error(t, err)

		err = db.DeleteUser(ctx, id)
		assert.Error(t, err)
	})

	t.Run("AgentMethods", func(t *testing.T) {
		_, err := db.GetAgentsByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetAgentByID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetAgentByAPIKeyHash(ctx, "hash")
		assert.Error(t, err)

		err = db.CreateAgent(ctx, &models.Agent{ID: id, OrgID: orgID, Hostname: "h"})
		assert.Error(t, err)

		err = db.UpdateAgent(ctx, &models.Agent{ID: id, Hostname: "h"})
		assert.Error(t, err)

		err = db.DeleteAgent(ctx, id)
		assert.Error(t, err)

		err = db.UpdateAgentAPIKeyHash(ctx, id, "hash")
		assert.Error(t, err)

		err = db.RevokeAgentAPIKey(ctx, id)
		assert.Error(t, err)

		_, err = db.GetFleetHealthSummary(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetAgentStats(ctx, id)
		assert.Error(t, err)

		_, err = db.GetAllAgents(ctx)
		assert.Error(t, err)

		_, err = db.GetOrgIDByAgentID(ctx, id)
		assert.Error(t, err)
	})

	t.Run("RepositoryMethods", func(t *testing.T) {
		_, err := db.GetRepositoriesByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetRepositoryByID(ctx, id)
		assert.Error(t, err)

		err = db.UpdateRepository(ctx, &models.Repository{ID: id})
		assert.Error(t, err)

		err = db.DeleteRepository(ctx, id)
		assert.Error(t, err)
	})

	t.Run("ScheduleMethods", func(t *testing.T) {
		_, err := db.GetSchedulesByAgentID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetScheduleByID(ctx, id)
		assert.Error(t, err)

		err = db.DeleteSchedule(ctx, id)
		assert.Error(t, err)

		_, err = db.GetAllSchedules(ctx)
		assert.Error(t, err)

		_, err = db.GetEnabledSchedules(ctx)
		assert.Error(t, err)

		_, err = db.GetEnabledSchedulesByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetOrgIDByScheduleID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetSchedulesByAgentGroupID(ctx, id)
		assert.Error(t, err)
	})

	t.Run("BackupMethods", func(t *testing.T) {
		_, err := db.GetBackupsByScheduleID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetBackupsByAgentID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetBackupByID(ctx, id)
		assert.Error(t, err)

		err = db.UpdateBackup(ctx, &models.Backup{ID: id})
		assert.Error(t, err)

		err = db.DeleteBackup(ctx, id)
		assert.Error(t, err)

		_, err = db.GetLatestBackupByScheduleID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetBackupsByOrgIDSince(ctx, orgID, now)
		assert.Error(t, err)

		total, running, failed, err := db.GetBackupCountsByOrgID(ctx, orgID)
		assert.Error(t, err)
		assert.Equal(t, 0, total)
		assert.Equal(t, 0, running)
		assert.Equal(t, 0, failed)
	})

	t.Run("PolicyMethods", func(t *testing.T) {
		_, err := db.GetPoliciesByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetPolicyByID(ctx, id)
		assert.Error(t, err)

		err = db.DeletePolicy(ctx, id)
		assert.Error(t, err)

		_, err = db.GetSchedulesByPolicyID(ctx, id)
		assert.Error(t, err)
	})

	t.Run("AlertMethods", func(t *testing.T) {
		_, err := db.GetAlertsByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetActiveAlertsByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetActiveAlertCountByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetAlertByID(ctx, id)
		assert.Error(t, err)

		err = db.UpdateAlert(ctx, &models.Alert{ID: id})
		assert.Error(t, err)

		err = db.ResolveAlertsByResource(ctx, models.ResourceTypeAgent, id)
		assert.Error(t, err)

		_, err = db.GetAlertRulesByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetEnabledAlertRulesByOrgID(ctx, orgID)
		assert.Error(t, err)

		err = db.UpdateAlertRule(ctx, &models.AlertRule{ID: id})
		assert.Error(t, err)

		err = db.DeleteAlertRule(ctx, id)
		assert.Error(t, err)
	})

	t.Run("RestoreMethods", func(t *testing.T) {
		_, err := db.GetRestoresByAgentID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetRestoreByID(ctx, id)
		assert.Error(t, err)

		err = db.UpdateRestore(ctx, &models.Restore{ID: id})
		assert.Error(t, err)

		err = db.DeleteRestore(ctx, id)
		assert.Error(t, err)
	})

	t.Run("MembershipMethods", func(t *testing.T) {
		_, err := db.GetMembershipsByUserID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetMembershipsByOrgID(ctx, orgID)
		assert.Error(t, err)

		err = db.UpdateMembership(ctx, &models.OrgMembership{ID: id})
		assert.Error(t, err)

		err = db.DeleteMembership(ctx, id, orgID)
		assert.Error(t, err)

		_, err = db.GetUserOrganizations(ctx, id)
		assert.Error(t, err)

		err = db.UpdateMembershipRole(ctx, id, models.OrgRoleAdmin)
		assert.Error(t, err)
	})

	t.Run("InvitationMethods", func(t *testing.T) {
		_, err := db.GetPendingInvitationsByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetPendingInvitationsByEmail(ctx, "a@b.com")
		assert.Error(t, err)

		err = db.AcceptInvitation(ctx, id)
		assert.Error(t, err)

		err = db.DeleteInvitation(ctx, id)
		assert.Error(t, err)
	})

	t.Run("NotificationMethods", func(t *testing.T) {
		err := db.UpdateNotificationChannel(ctx, &models.NotificationChannel{ID: id})
		assert.Error(t, err)

		err = db.DeleteNotificationChannel(ctx, id)
		assert.Error(t, err)

		err = db.UpdateNotificationPreference(ctx, &models.NotificationPreference{ID: id})
		assert.Error(t, err)

		err = db.DeleteNotificationPreference(ctx, id)
		assert.Error(t, err)

		err = db.UpdateNotificationLog(ctx, &models.NotificationLog{ID: id})
		assert.Error(t, err)
	})

	t.Run("RepositoryKeyMethods", func(t *testing.T) {
		err := db.UpdateRepositoryKeyEscrow(ctx, id, true, []byte("key"))
		assert.Error(t, err)

		err = db.DeleteRepositoryKey(ctx, id)
		assert.Error(t, err)

		_, err = db.GetRepositoryKeysWithEscrowByOrgID(ctx, orgID)
		assert.Error(t, err)
	})

	t.Run("StorageStatsMethods", func(t *testing.T) {
		_, err := db.GetLatestStorageStats(ctx, id)
		assert.Error(t, err)

		_, err = db.GetStorageStatsSummary(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetStorageGrowth(ctx, id, 30)
		assert.Error(t, err)

		_, err = db.GetAllStorageGrowth(ctx, orgID, 30)
		assert.Error(t, err)

		_, err = db.GetLatestStatsForAllRepos(ctx, orgID)
		assert.Error(t, err)
	})

	t.Run("VerificationMethods", func(t *testing.T) {
		_, err := db.GetEnabledVerificationSchedules(ctx)
		assert.Error(t, err)

		_, err = db.GetVerificationSchedulesByRepoID(ctx, id)
		assert.Error(t, err)

		err = db.DeleteVerificationSchedule(ctx, id)
		assert.Error(t, err)

		_, err = db.GetVerificationsByRepoID(ctx, id)
		assert.Error(t, err)

		err = db.UpdateVerification(ctx, &models.Verification{ID: id})
		assert.Error(t, err)

		err = db.DeleteVerification(ctx, id)
		assert.Error(t, err)

		_, err = db.GetConsecutiveFailedVerifications(ctx, id)
		assert.Error(t, err)
	})

	t.Run("MaintenanceMethods", func(t *testing.T) {
		_, err := db.ListMaintenanceWindowsByOrg(ctx, orgID)
		assert.Error(t, err)

		_, err = db.ListActiveMaintenanceWindows(ctx, orgID, now)
		assert.Error(t, err)

		_, err = db.ListUpcomingMaintenanceWindows(ctx, orgID, now, 10)
		assert.Error(t, err)

		_, err = db.ListPendingMaintenanceNotifications(ctx)
		assert.Error(t, err)

		err = db.UpdateMaintenanceWindow(ctx, &models.MaintenanceWindow{ID: id})
		assert.Error(t, err)

		err = db.DeleteMaintenanceWindow(ctx, id)
		assert.Error(t, err)

		err = db.MarkMaintenanceNotificationSent(ctx, id)
		assert.Error(t, err)
	})

	t.Run("ExcludePatternMethods", func(t *testing.T) {
		_, err := db.GetExcludePatternsByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetBuiltinExcludePatterns(ctx)
		assert.Error(t, err)

		_, err = db.GetExcludePatternsByCategory(ctx, orgID, "logs")
		assert.Error(t, err)

		err = db.UpdateExcludePattern(ctx, &models.ExcludePattern{ID: id, Patterns: []string{"*.log"}})
		assert.Error(t, err)

		err = db.DeleteExcludePattern(ctx, id)
		assert.Error(t, err)
	})

	t.Run("SnapshotCommentMethods", func(t *testing.T) {
		_, err := db.GetSnapshotCommentsBySnapshotID(ctx, "snap", orgID)
		assert.Error(t, err)

		err = db.DeleteSnapshotComment(ctx, id)
		assert.Error(t, err)
	})

	t.Run("DRMethods", func(t *testing.T) {
		_, err := db.GetDRRunbooksByOrgID(ctx, orgID)
		assert.Error(t, err)

		err = db.DeleteDRRunbook(ctx, id)
		assert.Error(t, err)

		_, err = db.GetDRTestsByRunbookID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetDRTestsByOrgID(ctx, orgID)
		assert.Error(t, err)

		err = db.UpdateDRTest(ctx, &models.DRTest{ID: id})
		assert.Error(t, err)

		err = db.DeleteDRTest(ctx, id)
		assert.Error(t, err)

		_, err = db.GetDRTestSchedulesByRunbookID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetEnabledDRTestSchedules(ctx)
		assert.Error(t, err)

		err = db.UpdateDRTestSchedule(ctx, &models.DRTestSchedule{ID: id})
		assert.Error(t, err)

		err = db.DeleteDRTestSchedule(ctx, id)
		assert.Error(t, err)

		_, err = db.GetDRStatus(ctx, orgID)
		assert.Error(t, err)
	})

	t.Run("TagMethods", func(t *testing.T) {
		_, err := db.GetTagsByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetTagByNameAndOrgID(ctx, "name", orgID)
		assert.Error(t, err)

		err = db.UpdateTag(ctx, &models.Tag{ID: id})
		assert.Error(t, err)

		err = db.DeleteTag(ctx, id)
		assert.Error(t, err)

		_, err = db.GetTagsByBackupID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetBackupIDsByTagID(ctx, id)
		assert.Error(t, err)

		err = db.RemoveTagFromBackup(ctx, id, id)
		assert.Error(t, err)

		_, err = db.GetTagsBySnapshotID(ctx, "snap")
		assert.Error(t, err)

		err = db.RemoveTagFromSnapshot(ctx, "snap", id)
		assert.Error(t, err)
	})

	t.Run("DashboardMethods", func(t *testing.T) {
		_, err := db.GetDashboardStats(ctx, orgID)
		assert.Error(t, err)

		_, _, err = db.GetBackupSuccessRates(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetStorageGrowthTrend(ctx, orgID, 30)
		assert.Error(t, err)

		_, err = db.GetBackupDurationTrend(ctx, orgID, 30)
		assert.Error(t, err)

		_, err = db.GetDailyBackupStats(ctx, orgID, 30)
		assert.Error(t, err)
	})

	t.Run("ReportScheduleMethods", func(t *testing.T) {
		_, err := db.GetEnabledReportSchedules(ctx)
		assert.Error(t, err)

		_, err = db.GetReportSchedulesByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetReportScheduleByID(ctx, id)
		assert.Error(t, err)

		err = db.UpdateReportScheduleLastSent(ctx, id, now)
		assert.Error(t, err)

		err = db.DeleteReportSchedule(ctx, id)
		assert.Error(t, err)

		_, err = db.GetReportHistoryByOrgID(ctx, orgID, 10)
		assert.Error(t, err)
	})

	t.Run("BackupDateRangeMethods", func(t *testing.T) {
		_, err := db.GetBackupsByOrgIDAndDateRange(ctx, orgID, now, now)
		assert.Error(t, err)

		_, err = db.GetAlertsByOrgIDAndDateRange(ctx, orgID, now, now)
		assert.Error(t, err)
	})

	t.Run("AgentGroupMethods", func(t *testing.T) {
		err := db.UpdateAgentGroup(ctx, &models.AgentGroup{ID: id})
		assert.Error(t, err)

		err = db.DeleteAgentGroup(ctx, id)
		assert.Error(t, err)

		_, err = db.GetAgentGroupMembers(ctx, id)
		assert.Error(t, err)

		err = db.RemoveAgentFromGroup(ctx, id, id)
		assert.Error(t, err)

		_, err = db.GetAgentsWithGroupsByOrgID(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetAgentsByGroupID(ctx, id)
		assert.Error(t, err)
	})

	t.Run("OnboardingMethods", func(t *testing.T) {
		err := db.SkipOnboarding(ctx, orgID)
		assert.Error(t, err)
	})

	t.Run("CostMethods", func(t *testing.T) {
		err := db.UpdateStoragePricing(ctx, &models.StoragePricing{ID: id})
		assert.Error(t, err)

		err = db.DeleteStoragePricing(ctx, id)
		assert.Error(t, err)

		_, err = db.GetLatestCostEstimates(ctx, orgID)
		assert.Error(t, err)

		_, err = db.GetCostEstimateHistory(ctx, id, 30)
		assert.Error(t, err)

		_, err = db.GetCostAlertsByOrgID(ctx, orgID)
		assert.Error(t, err)

		err = db.UpdateCostAlert(ctx, &models.CostAlert{ID: id})
		assert.Error(t, err)

		err = db.DeleteCostAlert(ctx, id)
		assert.Error(t, err)

		err = db.UpdateCostAlertTriggered(ctx, id)
		assert.Error(t, err)

		_, err = db.GetEnabledCostAlerts(ctx, orgID)
		assert.Error(t, err)
	})

	t.Run("SSOMethods", func(t *testing.T) {
		err := db.UpdateSSOGroupMapping(ctx, &models.SSOGroupMapping{ID: id})
		assert.Error(t, err)

		err = db.DeleteSSOGroupMapping(ctx, id)
		assert.Error(t, err)

		err = db.UpsertUserSSOGroups(ctx, id, []string{"g"})
		assert.Error(t, err)

		_, _, err = db.GetOrganizationSSOSettings(ctx, orgID)
		assert.Error(t, err)

		err = db.UpdateOrganizationSSOSettings(ctx, orgID, nil, false)
		assert.Error(t, err)
	})

	t.Run("BackupScriptMethods", func(t *testing.T) {
		_, err := db.GetBackupScriptsByScheduleID(ctx, id)
		assert.Error(t, err)

		_, err = db.GetEnabledBackupScriptsByScheduleID(ctx, id)
		assert.Error(t, err)

		err = db.UpdateBackupScript(ctx, &models.BackupScript{ID: id})
		assert.Error(t, err)

		err = db.DeleteBackupScript(ctx, id)
		assert.Error(t, err)
	})

	t.Run("ReplicationMethods", func(t *testing.T) {
		err := db.UpdateReplicationStatus(ctx, &models.ReplicationStatus{ID: id})
		assert.Error(t, err)
	})

	t.Run("ScheduleRepositoryMethods", func(t *testing.T) {
		_, err := db.GetScheduleRepositories(ctx, id)
		assert.Error(t, err)

		err = db.DeleteScheduleRepositories(ctx, id)
		assert.Error(t, err)
	})

	t.Run("HealthHistoryMethods", func(t *testing.T) {
		_, err := db.GetAgentHealthHistory(ctx, id, 10)
		assert.Error(t, err)
	})

	t.Run("ExecTx", func(t *testing.T) {
		err := db.ExecTx(ctx, func(tx pgx.Tx) error {
			return nil
		})
		assert.Error(t, err)
	})

	t.Run("Migrate", func(t *testing.T) {
		err := db.Migrate(ctx)
		assert.Error(t, err)
	})

	t.Run("CurrentVersion", func(t *testing.T) {
		_, err := db.CurrentVersion(ctx)
		assert.Error(t, err)
	})

	t.Run("Ping", func(t *testing.T) {
		err := db.Ping(ctx)
		assert.Error(t, err)
	})
}
