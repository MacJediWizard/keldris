package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
)

// B2Backend represents a Backblaze B2 storage backend.
type B2Backend struct {
	Bucket         string `json:"bucket"`
	Prefix         string `json:"prefix,omitempty"`
	AccountID      string `json:"account_id"`
	ApplicationKey string `json:"application_key"`
}

// Type returns the repository type.
func (b *B2Backend) Type() models.RepositoryType {
	return models.RepositoryTypeB2
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *B2Backend) ToResticConfig(password string) ResticConfig {
	repository := fmt.Sprintf("b2:%s", b.Bucket)
	if b.Prefix != "" {
		repository = repository + ":" + b.Prefix
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env: map[string]string{
			"B2_ACCOUNT_ID":  b.AccountID,
			"B2_ACCOUNT_KEY": b.ApplicationKey,
		},
	}
}

// Validate checks if the configuration is valid.
func (b *B2Backend) Validate() error {
	if b.Bucket == "" {
		return errors.New("b2 backend: bucket is required")
	}
	if b.AccountID == "" {
		return errors.New("b2 backend: account_id is required")
	}
	if b.ApplicationKey == "" {
		return errors.New("b2 backend: application_key is required")
	}
	return nil
}

// b2AuthResponse represents the B2 authorization response.
type b2AuthResponse struct {
	AccountID          string `json:"accountId"`
	AuthorizationToken string `json:"authorizationToken"`
	APIURL             string `json:"apiUrl"`
	Allowed            struct {
		BucketID   string   `json:"bucketId"`
		BucketName string   `json:"bucketName"`
		Caps       []string `json:"capabilities"`
	} `json:"allowed"`
}

// TestConnection tests the B2 backend connection by attempting to authorize.
func (b *B2Backend) TestConnection() error {
	if err := b.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create request to B2 authorize_account endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	if err != nil {
		return fmt.Errorf("b2 backend: failed to create request: %w", err)
	}

	req.SetBasicAuth(b.AccountID, b.ApplicationKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("b2 backend: failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("b2 backend: authorization failed with status %d", resp.StatusCode)
	}

	var authResp b2AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("b2 backend: failed to parse response: %w", err)
	}

	// If a specific bucket is configured, verify it's accessible
	if b.Bucket != "" && authResp.Allowed.BucketName != "" {
		if authResp.Allowed.BucketName != b.Bucket {
			return fmt.Errorf("b2 backend: key does not have access to bucket %s", b.Bucket)
		}
	}

	return nil
}
