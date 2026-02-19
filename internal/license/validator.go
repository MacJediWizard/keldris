package license

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	HeartbeatInterval = 24 * time.Hour
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

// Validator manages phone-home communication with the license server.
// It handles instance registration, heartbeat, license activation/validation,
// and grace period management.
type Validator struct {
	mu              sync.RWMutex
	license         *License
	licenseKey      string
	serverURL       string
	instanceID      string
	serverVersion   string
	store           SettingsStore
	metrics         MetricsProvider
	logger          zerolog.Logger
	lastValidation  time.Time
	graceStartedAt  *time.Time
	stopCh          chan struct{}
}

// ValidatorConfig holds configuration for the validator.
type ValidatorConfig struct {
	LicenseKey    string
	ServerURL     string
	ServerVersion string
	Store         SettingsStore
	Metrics       MetricsProvider
	Logger        zerolog.Logger
}

// NewValidator creates a new license validator.
func NewValidator(cfg ValidatorConfig) *Validator {
	return &Validator{
		license:       FreeLicense(),
		licenseKey:    cfg.LicenseKey,
		serverURL:     cfg.ServerURL,
		serverVersion: cfg.ServerVersion,
		store:         cfg.Store,
		metrics:       cfg.Metrics,
		logger:        cfg.Logger.With().Str("component", "license_validator").Logger(),
		stopCh:        make(chan struct{}),
	}
}

// GetLicense returns the current validated license (thread-safe).
func (v *Validator) GetLicense() *License {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.license
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

	// Parse license key if provided
	if v.licenseKey != "" {
		// The license was already parsed and validated in main.go
		// The validator just needs the key for phone-home
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

	// Start background goroutines
	go v.heartbeatLoop()
	if v.licenseKey != "" {
		go v.validationLoop()
	}

	return nil
}

// Stop shuts down the validator and deactivates the license if applicable.
func (v *Validator) Stop(ctx context.Context) {
	close(v.stopCh)

	if v.licenseKey != "" {
		v.deactivateLicense(ctx)
	}
}

// SetLicense updates the current license (used after successful validation or initial parse).
func (v *Validator) SetLicense(lic *License) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.license = lic
}

func (v *Validator) registerInstance(ctx context.Context) {
	var agentCount, userCount int
	if v.metrics != nil {
		agentCount, _ = v.metrics.AgentCount(ctx)
		userCount, _ = v.metrics.UserCount(ctx)
	}

	lic := v.GetLicense()

	body := map[string]interface{}{
		"instance_id":    v.instanceID,
		"product":        "keldris",
		"hostname":       "",
		"server_version": v.serverVersion,
		"tier":           string(lic.Tier),
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"metrics": map[string]interface{}{
			"agent_count": agentCount,
			"user_count":  userCount,
		},
	}

	if err := v.postJSON(ctx, "/api/v1/instances/register", body); err != nil {
		v.logger.Warn().Err(err).Msg("failed to register instance with license server")
		return
	}

	v.logger.Info().Msg("registered with license server")
}

func (v *Validator) activateLicense(ctx context.Context) {
	body := map[string]interface{}{
		"license_key": v.licenseKey,
		"instance_id": v.instanceID,
		"product":     "keldris",
		"hostname":    "",
		"server_version": v.serverVersion,
	}

	resp, err := v.postJSONWithResponse(ctx, "/api/v1/licenses/activate", body)
	if err != nil {
		v.logger.Warn().Err(err).Msg("failed to activate license")
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
	case "revoked":
		v.logger.Warn().Msg("license has been revoked - downgrading to Free")
		v.SetLicense(FreeLicense())
	case "expired":
		v.logger.Warn().Msg("license has expired - downgrading to Free")
		v.SetLicense(FreeLicense())
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

	body := map[string]interface{}{
		"instance_id": v.instanceID,
		"product":     "keldris",
		"metrics": map[string]interface{}{
			"agent_count": agentCount,
			"user_count":  userCount,
		},
	}

	if err := v.postJSON(ctx, "/api/v1/instances/heartbeat", body); err != nil {
		v.logger.Debug().Err(err).Msg("heartbeat failed")
		return
	}

	v.logger.Debug().Msg("heartbeat sent")
}

func (v *Validator) validationLoop() {
	ticker := time.NewTicker(ValidationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.stopCh:
			return
		case <-ticker.C:
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
		// License server unreachable - enter grace period
		v.mu.Lock()
		if v.graceStartedAt == nil {
			now := time.Now()
			v.graceStartedAt = &now
			v.logger.Warn().Msg("license server unreachable - starting grace period")
		}

		graceRemaining := GracePeriod - time.Since(*v.graceStartedAt)
		if graceRemaining <= 0 {
			v.logger.Error().Msg("grace period expired - downgrading to Free")
			v.license = FreeLicense()
		} else {
			v.logger.Warn().Dur("remaining", graceRemaining).Msg("in grace period")
		}
		v.mu.Unlock()
		return
	}

	status, _ := resp["status"].(string)
	switch status {
	case "valid":
		v.mu.Lock()
		v.lastValidation = time.Now()
		v.graceStartedAt = nil
		v.mu.Unlock()

		// Check for tier/expiry updates
		if tier, ok := resp["tier"].(string); ok {
			currentLic := v.GetLicense()
			if LicenseTier(tier) != currentLic.Tier {
				v.logger.Info().
					Str("old_tier", string(currentLic.Tier)).
					Str("new_tier", tier).
					Msg("license tier updated")
				v.SetLicense(&License{
					Tier:       LicenseTier(tier),
					CustomerID: currentLic.CustomerID,
					ExpiresAt:  currentLic.ExpiresAt,
					IssuedAt:   currentLic.IssuedAt,
					Limits:     GetLimits(LicenseTier(tier)),
				})
			}
		}

		v.logger.Debug().Msg("license validated successfully")

	case "revoked":
		v.logger.Warn().Msg("license has been revoked - downgrading to Free")
		v.SetLicense(FreeLicense())

	case "expired":
		v.logger.Warn().Msg("license has expired - downgrading to Free")
		v.SetLicense(FreeLicense())

	default:
		v.logger.Warn().Str("status", status).Msg("unexpected validation response")
	}
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
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}
