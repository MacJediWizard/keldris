package backends

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
)

// RestBackend represents a Restic REST server backend.
type RestBackend struct {
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Type returns the repository type.
func (b *RestBackend) Type() models.RepositoryType {
	return models.RepositoryTypeRest
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *RestBackend) ToResticConfig(password string) ResticConfig {
	repository := b.URL

	// Handle authentication in URL
	if b.Username != "" && b.Password != "" {
		if u, err := url.Parse(b.URL); err == nil {
			u.User = url.UserPassword(b.Username, b.Password)
			repository = u.String()
		}
	}

	// Ensure rest: prefix
	if !isRestURL(repository) {
		repository = "rest:" + repository
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
	}
}

// Validate checks if the configuration is valid.
func (b *RestBackend) Validate() error {
	if b.URL == "" {
		return errors.New("rest backend: url is required")
	}
	parsedURL := b.URL
	if isRestURL(parsedURL) {
		parsedURL = parsedURL[5:] // Remove "rest:" prefix
	}
	if _, err := url.Parse(parsedURL); err != nil {
		return fmt.Errorf("rest backend: invalid url: %w", err)
	}
	return nil
}

// isRestURL checks if the URL has the rest: prefix.
func isRestURL(s string) bool {
	return len(s) > 5 && s[:5] == "rest:"
}

// TestConnection tests the REST backend connection by sending a HEAD request.
func (b *RestBackend) TestConnection() error {
	if err := b.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build the URL
	testURL := b.URL
	if isRestURL(testURL) {
		testURL = testURL[5:] // Remove "rest:" prefix
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		return fmt.Errorf("rest backend: failed to create request: %w", err)
	}

	// Add basic auth if credentials provided
	if b.Username != "" && b.Password != "" {
		req.SetBasicAuth(b.Username, b.Password)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("rest backend: failed to connect: %w", err)
	}
	defer resp.Body.Close()

	// REST server should respond with 200 or 401 (if auth required)
	if resp.StatusCode == http.StatusUnauthorized && (b.Username == "" || b.Password == "") {
		return errors.New("rest backend: authentication required")
	}

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("rest backend: server returned status %d", resp.StatusCode)
	}

	return nil
}
