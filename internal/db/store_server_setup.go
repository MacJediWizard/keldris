package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// Server Setup methods

// GetServerSetup returns the current server setup state.
func (db *DB) GetServerSetup(ctx context.Context) (*models.ServerSetup, error) {
	var setup models.ServerSetup
	var completedSteps []string
	var currentStepStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, setup_completed, setup_completed_at, setup_completed_by,
		       current_step, completed_steps, created_at, updated_at
		FROM server_setup
		WHERE id = 1
	`).Scan(
		&setup.ID, &setup.SetupCompleted, &setup.SetupCompletedAt, &setup.SetupCompletedBy,
		&currentStepStr, &completedSteps, &setup.CreatedAt, &setup.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get server setup: %w", err)
	}

	setup.CurrentStep = models.ServerSetupStep(currentStepStr)
	setup.CompletedSteps = make([]models.ServerSetupStep, len(completedSteps))
	for i, s := range completedSteps {
		setup.CompletedSteps[i] = models.ServerSetupStep(s)
	}

	return &setup, nil
}

// IsSetupComplete returns true if server setup has been completed.
func (db *DB) IsSetupComplete(ctx context.Context) (bool, error) {
	var completed bool
	err := db.Pool.QueryRow(ctx, `
		SELECT setup_completed FROM server_setup WHERE id = 1
	`).Scan(&completed)
	if err != nil {
		return false, fmt.Errorf("check setup complete: %w", err)
	}
	return completed, nil
}

// UpdateServerSetup updates the current setup state.
func (db *DB) UpdateServerSetup(ctx context.Context, setup *models.ServerSetup) error {
	completedSteps := make([]string, len(setup.CompletedSteps))
	for i, s := range setup.CompletedSteps {
		completedSteps[i] = string(s)
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE server_setup
		SET setup_completed = $1, setup_completed_at = $2, setup_completed_by = $3,
		    current_step = $4, completed_steps = $5
		WHERE id = 1
	`, setup.SetupCompleted, setup.SetupCompletedAt, setup.SetupCompletedBy,
		string(setup.CurrentStep), completedSteps)
	if err != nil {
		return fmt.Errorf("update server setup: %w", err)
	}
	return nil
}

// CompleteSetupStep marks a step as completed and advances to the next step.
func (db *DB) CompleteSetupStep(ctx context.Context, step models.ServerSetupStep) error {
	setup, err := db.GetServerSetup(ctx)
	if err != nil {
		return err
	}

	// Add step to completed steps if not already there
	found := false
	for _, s := range setup.CompletedSteps {
		if s == step {
			found = true
			break
		}
	}
	if !found {
		setup.CompletedSteps = append(setup.CompletedSteps, step)
	}

	// Advance to next step
	setup.CurrentStep = setup.NextStep()

	return db.UpdateServerSetup(ctx, setup)
}

// FinalizeSetup marks setup as complete and locks the wizard.
func (db *DB) FinalizeSetup(ctx context.Context, userID *uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE server_setup
		SET setup_completed = true, setup_completed_at = $1, setup_completed_by = $2, current_step = 'complete'
		WHERE id = 1
	`, now, userID)
	if err != nil {
		return fmt.Errorf("finalize setup: %w", err)
	}
	return nil
}

// HasAnySuperuser returns true if at least one superuser exists.
func (db *DB) HasAnySuperuser(ctx context.Context) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE is_superuser = true
	`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count superusers: %w", err)
	}
	return count > 0, nil
}

// HasAnyOrganization returns true if at least one organization exists.
func (db *DB) HasAnyOrganization(ctx context.Context) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM organizations
	`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count organizations: %w", err)
	}
	return count > 0, nil
}

// CreateSuperuserWithPassword creates the first superuser during setup with password auth.
// This creates both the user and the organization, and sets up org membership.
func (db *DB) CreateSuperuserWithPassword(ctx context.Context, email, password, name string) (*models.User, *models.Organization, error) {
	// Check if any superusers already exist
	hasSuperuser, err := db.HasAnySuperuser(ctx)
	if err != nil {
		return nil, nil, err
	}
	if hasSuperuser {
		return nil, nil, fmt.Errorf("superuser already exists")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	// Start a transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Create the default organization for setup
	org := &models.Organization{
		ID:        uuid.New(),
		Name:      "Default Organization",
		Slug:      "default",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (slug) DO UPDATE SET updated_at = EXCLUDED.updated_at
		RETURNING id
	`, org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("create organization: %w", err)
	}

	// Get the org ID (in case it already existed)
	err = tx.QueryRow(ctx, `SELECT id FROM organizations WHERE slug = 'default'`).Scan(&org.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("get organization: %w", err)
	}

	// Create the superuser
	user := &models.User{
		ID:          uuid.New(),
		OrgID:       org.ID,
		OIDCSubject: "local:" + email, // Local auth marker
		Email:       email,
		Name:        name,
		Role:        models.UserRoleAdmin,
		Status:      models.UserStatusActive,
		IsSuperuser: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, org_id, oidc_subject, email, name, role, status, is_superuser, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, user.ID, user.OrgID, user.OIDCSubject, user.Email, user.Name,
		string(user.Role), string(user.Status), user.IsSuperuser, string(hashedPassword),
		user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("create superuser: %w", err)
	}

	// Create org membership
	membershipID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO org_memberships (id, org_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, membershipID, org.ID, user.ID, "owner", now, now)
	if err != nil {
		return nil, nil, fmt.Errorf("create membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	db.logger.Info().
		Str("email", email).
		Str("user_id", user.ID.String()).
		Str("org_id", org.ID.String()).
		Msg("created initial superuser during setup")

	return user, org, nil
}

// VerifyPassword checks if the provided password matches the stored hash.
func (db *DB) VerifyPassword(ctx context.Context, email, password string) (*models.User, error) {
	var user models.User
	var passwordHash string
	var roleStr, statusStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, status, is_superuser, password_hash, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1) AND password_hash IS NOT NULL
	`, email).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email, &user.Name,
		&roleStr, &statusStr, &user.IsSuperuser, &passwordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found or no password set")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	user.Role = models.UserRole(roleStr)
	user.Status = models.UserStatus(statusStr)

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return &user, nil
}

// License Key methods

// GetActiveLicense returns the current active license key.
func (db *DB) GetActiveLicense(ctx context.Context) (*models.LicenseKey, error) {
	var license models.LicenseKey
	var typeStr, statusStr string
	var featuresBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, license_key, license_type, status, max_agents, max_repositories, max_storage_gb,
		       features, issued_at, expires_at, activated_at, activated_by, company_name, contact_email,
		       created_at, updated_at
		FROM license_keys
		WHERE status = 'active'
		ORDER BY activated_at DESC NULLS LAST
		LIMIT 1
	`).Scan(
		&license.ID, &license.LicenseKey, &typeStr, &statusStr,
		&license.MaxAgents, &license.MaxRepositories, &license.MaxStorageGB,
		&featuresBytes, &license.IssuedAt, &license.ExpiresAt,
		&license.ActivatedAt, &license.ActivatedBy, &license.CompanyName, &license.ContactEmail,
		&license.CreatedAt, &license.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active license: %w", err)
	}

	license.LicenseType = models.LicenseType(typeStr)
	license.Status = models.LicenseStatus(statusStr)
	license.Features = featuresBytes

	return &license, nil
}

// ActivateLicense activates a license key.
func (db *DB) ActivateLicense(ctx context.Context, licenseKey string, activatedBy *uuid.UUID) (*models.LicenseKey, error) {
	// For now, we do basic validation and store the key
	// In production, this would validate against a license server
	now := time.Now()

	// Determine license type from key format (simplified)
	licenseType := models.LicenseTypeStandard
	if strings.HasPrefix(licenseKey, "TRIAL-") {
		licenseType = models.LicenseTypeTrial
	} else if strings.HasPrefix(licenseKey, "PRO-") {
		licenseType = models.LicenseTypeProfessional
	} else if strings.HasPrefix(licenseKey, "ENT-") {
		licenseType = models.LicenseTypeEnterprise
	}

	license := &models.LicenseKey{
		ID:          uuid.New(),
		LicenseKey:  licenseKey,
		LicenseType: licenseType,
		Status:      models.LicenseStatusActive,
		IssuedAt:    now,
		ActivatedAt: &now,
		ActivatedBy: activatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set limits based on license type
	switch licenseType {
	case models.LicenseTypeTrial:
		agents := 5
		repos := 2
		storage := 50
		expires := now.AddDate(0, 0, 14) // 14 days trial
		license.MaxAgents = &agents
		license.MaxRepositories = &repos
		license.MaxStorageGB = &storage
		license.ExpiresAt = &expires
	case models.LicenseTypeProfessional:
		agents := 50
		repos := 10
		storage := 500
		license.MaxAgents = &agents
		license.MaxRepositories = &repos
		license.MaxStorageGB = &storage
	case models.LicenseTypeEnterprise:
		// No limits for enterprise
	default:
		agents := 10
		repos := 5
		storage := 100
		license.MaxAgents = &agents
		license.MaxRepositories = &repos
		license.MaxStorageGB = &storage
	}

	// Deactivate any existing licenses
	_, err := db.Pool.Exec(ctx, `
		UPDATE license_keys SET status = 'expired', updated_at = NOW()
		WHERE status = 'active'
	`)
	if err != nil {
		return nil, fmt.Errorf("deactivate existing licenses: %w", err)
	}

	// Insert the new license
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO license_keys (
			id, license_key, license_type, status, max_agents, max_repositories, max_storage_gb,
			features, issued_at, expires_at, activated_at, activated_by, company_name, contact_email,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, license.ID, license.LicenseKey, string(license.LicenseType), string(license.Status),
		license.MaxAgents, license.MaxRepositories, license.MaxStorageGB,
		license.Features, license.IssuedAt, license.ExpiresAt,
		license.ActivatedAt, license.ActivatedBy, license.CompanyName, license.ContactEmail,
		license.CreatedAt, license.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create license: %w", err)
	}

	return license, nil
}

// CreateTrialLicense creates a new 14-day trial license during initial setup.
func (db *DB) CreateTrialLicense(ctx context.Context, companyName, contactEmail string, activatedBy *uuid.UUID) (*models.LicenseKey, error) {
	// Check if a trial has already been used
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM license_keys WHERE license_type = 'trial'
	`).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("check existing trial: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("trial license has already been used")
	}

	// Generate a unique trial key
	keyBytes := make([]byte, 8)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("generate trial key: %w", err)
	}
	trialKey := "TRIAL-" + strings.ToUpper(hex.EncodeToString(keyBytes))

	now := time.Now()
	expires := now.AddDate(0, 0, 14) // 14 days

	agents := 5
	repos := 2
	storage := 50

	license := &models.LicenseKey{
		ID:              uuid.New(),
		LicenseKey:      trialKey,
		LicenseType:     models.LicenseTypeTrial,
		Status:          models.LicenseStatusActive,
		MaxAgents:       &agents,
		MaxRepositories: &repos,
		MaxStorageGB:    &storage,
		IssuedAt:        now,
		ExpiresAt:       &expires,
		ActivatedAt:     &now,
		ActivatedBy:     activatedBy,
		CompanyName:     companyName,
		ContactEmail:    contactEmail,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO license_keys (
			id, license_key, license_type, status, max_agents, max_repositories, max_storage_gb,
			features, issued_at, expires_at, activated_at, activated_by, company_name, contact_email,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, license.ID, license.LicenseKey, string(license.LicenseType), string(license.Status),
		license.MaxAgents, license.MaxRepositories, license.MaxStorageGB,
		license.Features, license.IssuedAt, license.ExpiresAt,
		license.ActivatedAt, license.ActivatedBy, license.CompanyName, license.ContactEmail,
		license.CreatedAt, license.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create trial license: %w", err)
	}

	return license, nil
}

// Server Setup Audit Log methods

// CreateServerSetupAuditLog creates a new server setup audit log entry.
func (db *DB) CreateServerSetupAuditLog(ctx context.Context, log *models.ServerSetupAuditLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO server_setup_audit_log (id, action, step, performed_by, ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, log.ID, log.Action, log.Step, log.PerformedBy, log.IPAddress, log.UserAgent, log.Details, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("create server setup audit log: %w", err)
	}
	return nil
}

// GetServerSetupAuditLogs returns server setup audit logs with pagination.
func (db *DB) GetServerSetupAuditLogs(ctx context.Context, limit, offset int) ([]*models.ServerSetupAuditLog, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM server_setup_audit_log`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count server setup audit logs: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, action, step, performed_by, ip_address, user_agent, details, created_at
		FROM server_setup_audit_log
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list server setup audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.ServerSetupAuditLog
	for rows.Next() {
		var log models.ServerSetupAuditLog
		var detailsBytes []byte
		err := rows.Scan(
			&log.ID, &log.Action, &log.Step, &log.PerformedBy,
			&log.IPAddress, &log.UserAgent, &detailsBytes, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan server setup audit log: %w", err)
		}
		log.Details = detailsBytes
		logs = append(logs, &log)
	}

	return logs, total, nil
}

// CreateFirstOrganization creates the first organization during setup.
// If an organization with the given name already exists, it returns that organization.
func (db *DB) CreateFirstOrganization(ctx context.Context, name string, createdBy uuid.UUID) (*models.Organization, error) {
	// Generate a slug from the name
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	if slug == "" {
		slug = "organization"
	}

	now := time.Now()
	org := &models.Organization{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Try to insert, or get existing if slug conflicts
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (slug) DO UPDATE SET updated_at = EXCLUDED.updated_at
		RETURNING id, name, slug, created_at, updated_at
	`, org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt).Scan(
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}

	// Update the superuser's org_id to this organization
	_, err = db.Pool.Exec(ctx, `
		UPDATE users SET org_id = $1, updated_at = NOW()
		WHERE id = $2
	`, org.ID, createdBy)
	if err != nil {
		return nil, fmt.Errorf("update user org: %w", err)
	}

	// Ensure membership exists
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO org_memberships (id, org_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, 'owner', $4, $5)
		ON CONFLICT (org_id, user_id) DO NOTHING
	`, uuid.New(), org.ID, createdBy, now, now)
	if err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}

	return org, nil
}
