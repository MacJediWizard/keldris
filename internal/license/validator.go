package license

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	// GracePeriod is how long paid tiers keep running when the license server is unreachable.
	GracePeriod = 30 * 24 * time.Hour

	// ValidationInterval is how often the validator checks with the license server.
	ValidationInterval = 24 * time.Hour

	// RetryInterval is how often to retry when the license server is unreachable.
	RetryInterval = 1 * time.Hour

	// HeartbeatInterval is how often telemetry is sent (all tiers).
	HeartbeatInterval = 6 * time.Hour
)

// SettingsStore provides persistence for server settings.
type SettingsStore interface {
	GetServerSetting(ctx context.Context, key string) (string, error)
	SetServerSetting(ctx context.Context, key, value string) error
}

// MetricsProvider returns current server metrics for telemetry.
type MetricsProvider interface {
	AgentCount(ctx context.Context) (int, error)
	UserCount(ctx context.Context) (int, error)
}

// OrgCounter provides organization count for telemetry.
type OrgCounter interface {
	OrgCount(ctx context.Context) (int, error)
}

// OIDCConfiguredChecker reports whether OIDC is currently configured.
type OIDCConfiguredChecker interface {
	IsConfigured() bool
}

// Validator manages phone-home communication with the license server.
// It handles instance registration, heartbeat, license activation/validation,
// and grace period management.
type Validator struct {
	mu                  sync.RWMutex
	license             *License
	entitlement         *Entitlement
	entitlementToken    string
	licenseKey          string
	licenseKeySource    string // "env", "database", or "none" — cached on set
	serverURL           string
	instanceID          string
	serverVersion       string
	startedAt           time.Time
	killed              bool
	store               SettingsStore
	metrics             MetricsProvider
	orgCounter          OrgCounter
	oidcProvider        OIDCConfiguredChecker
	publicKey           ed25519.PublicKey
	logger              zerolog.Logger
	lastValidation      time.Time
	graceStartedAt      *time.Time
	featureUsage        *FeatureUsageTracker
	featureRefreshToken string
	lastHeartbeatAt     time.Time
	integrityHash       string
	stopCh              chan struct{}
	stopOnce            sync.Once
	validationRunning   bool
}

// ValidatorConfig holds configuration for the validator.
type ValidatorConfig struct {
	LicenseKey    string
	ServerURL     string
	ServerVersion string
	Store         SettingsStore
	Metrics       MetricsProvider
	OrgCounter    OrgCounter
	PublicKey     ed25519.PublicKey
	Logger        zerolog.Logger
}

// NewValidator creates a new license validator.
func NewValidator(cfg ValidatorConfig) *Validator {
	return &Validator{
		license:       FreeLicense(),
		licenseKey:    cfg.LicenseKey,
		serverURL:     cfg.ServerURL,
		serverVersion: cfg.ServerVersion,
		startedAt:     time.Now(),
		store:         cfg.Store,
		metrics:       cfg.Metrics,
		orgCounter:    cfg.OrgCounter,
		publicKey:     cfg.PublicKey,
		logger:        cfg.Logger.With().Str("component", "license_validator").Logger(),
		featureUsage:  NewFeatureUsageTracker(),
		stopCh:        make(chan struct{}),
	}
}

// GetLicense returns the current validated license (thread-safe).
func (v *Validator) GetLicense() *License {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.license
}

// GetEntitlement returns the current parsed entitlement (thread-safe).
func (v *Validator) GetEntitlement() *Entitlement {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.entitlement
}

// GetFeatureUsageTracker returns the feature usage tracker for middleware.
func (v *Validator) GetFeatureUsageTracker() *FeatureUsageTracker {
	return v.featureUsage
}

// IsKilled returns whether the instance has been remotely killed.
func (v *Validator) IsKilled() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.killed
}

// HasValidRefreshToken returns true if the validator has a fresh refresh token
// from a recent heartbeat. The token is valid for 12 hours (allows one missed
// 6-hour heartbeat cycle before expiry).
func (v *Validator) HasValidRefreshToken() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.featureRefreshToken != "" && time.Since(v.lastHeartbeatAt) < 12*time.Hour
}

// SetIntegrityHash sets the build integrity hash for heartbeat reporting.
func (v *Validator) SetIntegrityHash(hash string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.integrityHash = hash
}

// SetOIDCProvider sets the OIDC provider used to report oidc_configured in heartbeats.
func (v *Validator) SetOIDCProvider(provider OIDCConfiguredChecker) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.oidcProvider = provider
}

// isOIDCConfigured returns whether an OIDC provider is set and configured.
func (v *Validator) isOIDCConfigured() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.oidcProvider != nil && v.oidcProvider.IsConfigured()
}

// SetLicenseKey stores a license key in the database and triggers activation.
func (v *Validator) SetLicenseKey(ctx context.Context, key string) error {
	if err := v.store.SetServerSetting(ctx, "license_key", key); err != nil {
		return fmt.Errorf("store license key: %w", err)
	}
	v.mu.Lock()
	v.licenseKey = key
	v.licenseKeySource = "database"
	v.killed = false
	shouldStartLoop := !v.validationRunning
	v.mu.Unlock()

	v.activateLicense(ctx)

	// Start validation loop if not already running
	if shouldStartLoop {
		v.mu.Lock()
		v.validationRunning = true
		v.mu.Unlock()
		go v.validationLoop()
	}

	return nil
}

// ClearLicenseKey removes the stored license key and reverts to free tier.
func (v *Validator) ClearLicenseKey(ctx context.Context) error {
	if err := v.store.SetServerSetting(ctx, "license_key", ""); err != nil {
		return fmt.Errorf("clear license key: %w", err)
	}
	v.mu.Lock()
	v.licenseKey = ""
	v.licenseKeySource = "none"
	v.license = FreeLicense()
	v.entitlement = nil
	v.entitlementToken = ""
	v.killed = false
	v.mu.Unlock()
	return nil
}

// GetLicenseKey returns the current license key.
func (v *Validator) GetLicenseKey() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.licenseKey
}

// Start initializes the validator: loads/creates instance ID, registers with the
// license server, and starts background goroutines for heartbeat and validation.
func (v *Validator) Start(ctx context.Context) error {
	// Load or create instance ID
	instanceID, err := v.store.GetServerSetting(ctx, "instance_id")
	if err != nil || instanceID == "" {
		instanceID = uuid.New().String()
		if err := v.store.SetServerSetting(ctx, "instance_id", instanceID); err != nil {
			return fmt.Errorf("save instance ID: %w", err)
		}
		v.logger.Info().Str("instance_id", instanceID).Msg("generated new instance ID")
	}
	v.instanceID = instanceID

	// Load license key from DB if not set via env
	if v.licenseKey == "" {
		if dbKey, err := v.store.GetServerSetting(ctx, "license_key"); err == nil && dbKey != "" {
			v.licenseKey = dbKey
			v.licenseKeySource = "database"
			v.logger.Info().Msg("loaded license key from database")
		} else {
			v.licenseKeySource = "none"
		}
	} else {
		v.licenseKeySource = "env"
	}

	// Parse license key if provided
	if v.licenseKey != "" {
		v.logger.Info().Str("instance_id", instanceID).Msg("license validator started (paid tier)")
	} else {
		v.logger.Info().Str("instance_id", instanceID).Msg("license validator started (free tier)")
	}

	// Register instance (all tiers)
	v.registerInstance(ctx)

	// If paid tier, activate license
	if v.licenseKey != "" {
		v.activateLicense(ctx)
	}

	// Send initial heartbeat immediately so the license server has full
	// metrics (org count, feature usage, etc.) without waiting 24 hours.
	v.sendHeartbeat(ctx)

	// Start background goroutines
	go v.heartbeatLoop()
	if v.licenseKey != "" {
		v.validationRunning = true
		go v.validationLoop()
	}

	return nil
}

// Stop shuts down the validator and deactivates the license if applicable.
func (v *Validator) Stop(ctx context.Context) {
	v.stopOnce.Do(func() {
		close(v.stopCh)
	})

	v.mu.RLock()
	key := v.licenseKey
	v.mu.RUnlock()
	if key != "" {
		v.deactivateLicense(ctx)
	}
}

// Restart reinitializes the stop channel so the validator can accept new
// background goroutines after a Stop(). Must be called before SetLicenseKey()
// when reactivating after deactivation.
func (v *Validator) Restart() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.stopCh = make(chan struct{})
	v.stopOnce = sync.Once{}
	v.validationRunning = false
}

// GetServerURL returns the license server URL.
func (v *Validator) GetServerURL() string {
	return v.serverURL
}

// SetLicense updates the current license (used after successful validation or initial parse).
func (v *Validator) SetLicense(lic *License) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.license = lic
}

// LicenseKeySource returns where the license key was loaded from.
func (v *Validator) LicenseKeySource() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.licenseKeySource != "" {
		return v.licenseKeySource
	}
	if v.licenseKey == "" {
		return "none"
	}
	return "env"
}

func (v *Validator) registerInstance(ctx context.Context) {
	var agentCount, userCount int
	if v.metrics != nil {
		agentCount, _ = v.metrics.AgentCount(ctx)
		userCount, _ = v.metrics.UserCount(ctx)
	}

	var orgCount int
	if v.orgCounter != nil {
		orgCount, _ = v.orgCounter.OrgCount(ctx)
	}

	hostname, _ := os.Hostname()
	lic := v.GetLicense()

	body := map[string]interface{}{
		"instance_id":    v.instanceID,
		"product":        "keldris",
		"hostname":       hostname,
		"server_version": v.serverVersion,
		"tier":           string(lic.Tier),
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"metrics": map[string]interface{}{
			"agent_count": agentCount,
			"user_count":  userCount,
			"org_count":   orgCount,
		},
	}

	if err := v.postJSON(ctx, "/api/v1/instances/register", body); err != nil {
		v.logger.Warn().Err(err).Msg("failed to register instance with license server")
		return
	}

	v.logger.Info().Str("hostname", hostname).Msg("registered with license server")
}

func (v *Validator) activateLicense(ctx context.Context) {
	hostname, _ := os.Hostname()
	body := map[string]interface{}{
		"license_key":    v.licenseKey,
		"instance_id":    v.instanceID,
		"product":        "keldris",
		"hostname":       hostname,
		"server_version": v.serverVersion,
	}

	resp, err := v.postJSONWithResponse(ctx, "/api/v1/licenses/activate", body)
	if err != nil {
		v.logger.Warn().Err(err).Msg("license server unreachable, verifying key locally")
		v.verifyKeyLocally()
		return
	}

	status, _ := resp["status"].(string)
	switch status {
	case "active":
		v.logger.Info().Msg("license activated successfully")
		v.mu.Lock()
		v.lastValidation = time.Now()
		v.graceStartedAt = nil
		v.mu.Unlock()

		// Update license with server-provided limits
		v.updateLicenseFromResponse(resp)

		// Store entitlement token
		v.storeEntitlementFromResponse(resp)

	case "revoked":
		v.logger.Warn().Msg("license has been revoked - downgrading to Free")
		v.SetLicense(FreeLicense())
		v.clearEntitlement()
	case "expired":
		v.logger.Warn().Msg("license has expired - downgrading to Free")
		v.SetLicense(FreeLicense())
		v.clearEntitlement()
	case "limit_reached":
		v.logger.Warn().Msg("maximum activations reached for this license")
	default:
		v.logger.Warn().Str("status", status).Msg("unexpected activation response")
	}
}

func (v *Validator) deactivateLicense(ctx context.Context) {
	body := map[string]interface{}{
		"license_key": v.licenseKey,
		"instance_id": v.instanceID,
		"product":     "keldris",
	}

	if err := v.postJSON(ctx, "/api/v1/licenses/deactivate", body); err != nil {
		v.logger.Warn().Err(err).Msg("failed to deactivate license on shutdown")
		return
	}

	v.logger.Info().Msg("license deactivated (shutdown)")
}

func (v *Validator) heartbeatLoop() {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			v.sendHeartbeat(ctx)
			cancel()
		}
	}
}

func (v *Validator) sendHeartbeat(ctx context.Context) {
	var agentCount, userCount int
	if v.metrics != nil {
		agentCount, _ = v.metrics.AgentCount(ctx)
		userCount, _ = v.metrics.UserCount(ctx)
	}

	var orgCount int
	if v.orgCounter != nil {
		orgCount, _ = v.orgCounter.OrgCount(ctx)
	}

	// Get and reset feature usage
	featureUsage := v.featureUsage.GetAndReset()

	// Compute entitlement token hash
	v.mu.RLock()
	tokenHash := ""
	if v.entitlementToken != "" {
		h := sha256.Sum256([]byte(v.entitlementToken))
		tokenHash = hex.EncodeToString(h[:])
	}
	hasValidEntitlement := v.entitlement != nil && !v.entitlement.IsExpired()
	reportedTier := string(v.license.Tier)
	v.mu.RUnlock()

	uptimeHours := time.Since(v.startedAt).Hours()

	v.mu.RLock()
	integrityHash := v.integrityHash
	v.mu.RUnlock()

	body := map[string]interface{}{
		"instance_id": v.instanceID,
		"product":     "keldris",
		"metrics": map[string]interface{}{
			"agent_count":            agentCount,
			"user_count":             userCount,
			"org_count":              orgCount,
			"oidc_configured":        v.isOIDCConfigured(),
			"feature_usage":          featureUsage,
			"entitlement_token_hash": tokenHash,
			"server_version":         v.serverVersion,
			"uptime_hours":           uptimeHours,
			"integrity_hash":         integrityHash,
		},
		"reported_tier":        reportedTier,
		"has_valid_entitlement": hasValidEntitlement,
	}

	resp, err := v.postJSONWithResponse(ctx, "/api/v1/instances/heartbeat", body)
	if err != nil {
		v.logger.Debug().Err(err).Msg("heartbeat failed")
		return
	}

	// Store refresh token from heartbeat config
	if cfgMap, ok := resp["config"].(map[string]interface{}); ok {
		if token, ok := cfgMap["feature_refresh_token"].(string); ok && token != "" {
			v.mu.Lock()
			v.featureRefreshToken = token
			v.lastHeartbeatAt = time.Now()
			v.mu.Unlock()
		}
	}

	// Handle kill switch response
	if action, ok := resp["action"].(string); ok {
		switch action {
		case "downgrade":
			v.logger.Warn().Msg("remote downgrade action received")
			v.SetLicense(FreeLicense())
			v.clearEntitlement()
		case "kill":
			v.logger.Warn().Msg("remote kill action received")
			v.mu.Lock()
			v.license = FreeLicense()
			v.entitlement = nil
			v.entitlementToken = ""
			v.killed = true
			v.mu.Unlock()
		}
	}

	v.logger.Debug().Msg("heartbeat sent")
}

func (v *Validator) validationLoop() {
	defer func() {
		v.mu.Lock()
		v.validationRunning = false
		v.mu.Unlock()
	}()

	ticker := time.NewTicker(ValidationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.stopCh:
			return
		case <-ticker.C:
			v.mu.RLock()
			key := v.licenseKey
			v.mu.RUnlock()
			if key == "" {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			v.validateLicense(ctx)
			cancel()
		}
	}
}

func (v *Validator) validateLicense(ctx context.Context) {
	body := map[string]interface{}{
		"license_key": v.licenseKey,
		"instance_id": v.instanceID,
		"product":     "keldris",
	}

	resp, err := v.postJSONWithResponse(ctx, "/api/v1/licenses/validate", body)
	if err != nil {
		v.logger.Warn().Err(err).Msg("license server unreachable, verifying key locally")
		v.verifyKeyLocally()
		return
	}

	status, _ := resp["status"].(string)
	switch status {
	case "valid":
		v.mu.Lock()
		v.lastValidation = time.Now()
		v.graceStartedAt = nil
		v.mu.Unlock()

		// Update license with server-provided limits
		v.updateLicenseFromResponse(resp)

		// Store entitlement token
		v.storeEntitlementFromResponse(resp)

		v.logger.Debug().Msg("license validated successfully")

	case "revoked":
		v.logger.Warn().Msg("license has been revoked - downgrading to Free")
		v.SetLicense(FreeLicense())
		v.clearEntitlement()

	case "expired":
		v.logger.Warn().Msg("license has expired - downgrading to Free")
		v.SetLicense(FreeLicense())
		v.clearEntitlement()

	default:
		v.logger.Warn().Str("status", status).Msg("unexpected validation response")
	}
}

// updateLicenseFromResponse updates the license with server-provided data.
func (v *Validator) updateLicenseFromResponse(resp map[string]interface{}) {
	currentLic := v.GetLicense()

	tier := currentLic.Tier
	if t, ok := resp["tier"].(string); ok {
		tier = LicenseTier(t)
	}

	customerID := currentLic.CustomerID
	if cid, ok := resp["customer_id"].(string); ok && cid != "" {
		customerID = cid
	}

	customerName := currentLic.CustomerName
	if cn, ok := resp["customer_name"].(string); ok && cn != "" {
		customerName = cn
	}

	company := currentLic.Company
	if co, ok := resp["company"].(string); ok && co != "" {
		company = co
	}

	expiresAt := currentLic.ExpiresAt
	if exp, ok := resp["expires_at"].(string); ok && exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil {
			expiresAt = t
		}
	}

	issuedAt := currentLic.IssuedAt
	if iss, ok := resp["issued_at"].(string); ok && iss != "" {
		if t, err := time.Parse(time.RFC3339, iss); err == nil {
			issuedAt = t
		}
	}

	limits := parseLimitsFromResponse(resp)
	if limits.MaxAgents == 0 && limits.MaxUsers == 0 && limits.MaxOrgs == 0 {
		// No limits in response, fall back to tier defaults
		limits = GetLimits(tier)
	}

	// Parse trial fields
	isTrial := currentLic.IsTrial
	if v, ok := resp["is_trial"].(bool); ok {
		isTrial = v
	}
	trialDurationDays := currentLic.TrialDurationDays
	if v, ok := resp["trial_duration_days"].(float64); ok {
		trialDurationDays = int(v)
	}
	trialStartedAt := currentLic.TrialStartedAt
	if v, ok := resp["trial_started_at"].(string); ok && v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			trialStartedAt = t
		}
	}

	newLic := &License{
		Tier:              tier,
		CustomerID:        customerID,
		CustomerName:      customerName,
		Company:           company,
		ExpiresAt:         expiresAt,
		IssuedAt:          issuedAt,
		Limits:            limits,
		IsTrial:           isTrial,
		TrialDurationDays: trialDurationDays,
		TrialStartedAt:    trialStartedAt,
	}

	if tier != currentLic.Tier || limits != currentLic.Limits ||
		customerID != currentLic.CustomerID || !expiresAt.Equal(currentLic.ExpiresAt) {
		v.SetLicense(newLic)
		if tier != currentLic.Tier {
			v.logger.Info().
				Str("old_tier", string(currentLic.Tier)).
				Str("new_tier", string(tier)).
				Msg("license tier updated")
		}
	}
}

// parseLimitsFromResponse extracts TierLimits from the response map.
func parseLimitsFromResponse(resp map[string]interface{}) TierLimits {
	limits := TierLimits{}
	limitsMap, ok := resp["limits"].(map[string]interface{})
	if !ok {
		return limits
	}

	if v, ok := limitsMap["agents"].(float64); ok {
		limits.MaxAgents = int(v)
	} else if v, ok := limitsMap["max_agents"].(float64); ok {
		limits.MaxAgents = int(v)
	}
	if v, ok := limitsMap["servers"].(float64); ok {
		limits.MaxServers = int(v)
	} else if v, ok := limitsMap["max_servers"].(float64); ok {
		limits.MaxServers = int(v)
	}
	if v, ok := limitsMap["users"].(float64); ok {
		limits.MaxUsers = int(v)
	} else if v, ok := limitsMap["max_users"].(float64); ok {
		limits.MaxUsers = int(v)
	}
	if v, ok := limitsMap["orgs"].(float64); ok {
		limits.MaxOrgs = int(v)
	} else if v, ok := limitsMap["max_orgs"].(float64); ok {
		limits.MaxOrgs = int(v)
	}
	return limits
}

// storeEntitlementFromResponse parses and stores the entitlement token from a response.
func (v *Validator) storeEntitlementFromResponse(resp map[string]interface{}) {
	token, ok := resp["entitlement_token"].(string)
	if !ok || token == "" {
		return
	}

	if len(v.publicKey) == ed25519.PublicKeySize {
		ent, err := ParseEntitlementToken(token, v.publicKey)
		if err != nil {
			v.logger.Warn().Err(err).Msg("failed to parse entitlement token")
			return
		}
		v.mu.Lock()
		v.entitlement = ent
		v.entitlementToken = token
		v.mu.Unlock()
		v.logger.Debug().Str("tier", string(ent.Tier)).Int("features", len(ent.Features)).Msg("entitlement token stored")
	} else {
		// No public key available, store token without verification
		v.mu.Lock()
		v.entitlementToken = token
		v.mu.Unlock()
	}
}

// verifyKeyLocally validates the license key using the hardcoded Ed25519 public key
// when the license server is unreachable. If the key is valid but expired, a 30-day
// grace period allows continued operation.
func (v *Validator) verifyKeyLocally() {
	v.mu.RLock()
	key := v.licenseKey
	v.mu.RUnlock()

	if key == "" || len(v.publicKey) != ed25519.PublicKeySize {
		return
	}

	lic, err := ParseLicenseKeyEd25519(key, v.publicKey)
	if err != nil {
		v.logger.Error().Err(err).Msg("local license key verification failed")
		v.SetLicense(FreeLicense())
		v.clearEntitlement()
		return
	}

	if time.Now().Before(lic.ExpiresAt) {
		// Key is valid and not expired — keep running on this tier
		v.mu.Lock()
		v.graceStartedAt = nil
		v.mu.Unlock()

		// Preserve server-provided limits if we have them, otherwise use tier defaults
		currentLic := v.GetLicense()
		if currentLic.Tier == lic.Tier && currentLic.Limits != (TierLimits{}) {
			lic.Limits = currentLic.Limits
			lic.CustomerName = currentLic.CustomerName
		}
		v.SetLicense(lic)
		v.logger.Info().Str("tier", string(lic.Tier)).Msg("license verified locally")
		return
	}

	// Key has expired — start or continue grace period
	v.mu.Lock()
	if v.graceStartedAt == nil {
		now := time.Now()
		v.graceStartedAt = &now
		v.logger.Warn().Msg("license expired - starting 30-day grace period")
	}

	graceRemaining := GracePeriod - time.Since(*v.graceStartedAt)
	if graceRemaining <= 0 {
		v.logger.Error().Msg("grace period expired - downgrading to Free")
		v.license = FreeLicense()
		v.entitlement = nil
		v.entitlementToken = ""
	} else {
		v.logger.Warn().Dur("remaining", graceRemaining).Msg("in grace period (license expired)")
	}
	v.mu.Unlock()
}

func (v *Validator) clearEntitlement() {
	v.mu.Lock()
	v.entitlement = nil
	v.entitlementToken = ""
	v.mu.Unlock()
}

func (v *Validator) postJSON(ctx context.Context, path string, body interface{}) error {
	_, err := v.postJSONWithResponse(ctx, path, body)
	return err
}

func (v *Validator) postJSONWithResponse(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	url := v.serverURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request to %s: %w", path, err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response from %s (status %d): %w", path, resp.StatusCode, err)
	}

	if resp.StatusCode >= 500 {
		return result, fmt.Errorf("server error from %s: %d", path, resp.StatusCode)
	}

	return result, nil
}

// FeatureUsageTracker records which premium features have been accessed.
type FeatureUsageTracker struct {
	mu       sync.Mutex
	features map[string]bool
}

// NewFeatureUsageTracker creates a new feature usage tracker.
func NewFeatureUsageTracker() *FeatureUsageTracker {
	return &FeatureUsageTracker{
		features: make(map[string]bool),
	}
}

// Record records that a feature was accessed.
func (t *FeatureUsageTracker) Record(feature string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.features[feature] = true
}

// GetAndReset returns all recorded features and resets the tracker.
func (t *FeatureUsageTracker) GetAndReset() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	features := make([]string, 0, len(t.features))
	for f := range t.features {
		features = append(features, f)
	}
	t.features = make(map[string]bool)
	return features
}
