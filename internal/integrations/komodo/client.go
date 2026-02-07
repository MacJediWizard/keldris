package komodo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/httpclient"
	"github.com/rs/zerolog"
)

// Client provides methods to interact with the Komodo API
type Client struct {
	baseURL    string
	apiKey     string
	username   string
	password   string
	httpClient *http.Client
	logger     zerolog.Logger
}

// ClientConfig holds configuration for creating a new Komodo client
type ClientConfig struct {
	BaseURL     string
	APIKey      string
	Username    string
	Password    string
	Timeout     time.Duration
	ProxyConfig *config.ProxyConfig
}

// NewClient creates a new Komodo API client
func NewClient(cfg ClientConfig, logger zerolog.Logger) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("komodo client: base URL is required")
	}

	// Validate URL
	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("komodo client: invalid URL: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client, err := httpclient.New(httpclient.Options{
		Timeout:     timeout,
		ProxyConfig: cfg.ProxyConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("komodo client: create http client: %w", err)
	}

	return &Client{
		baseURL:    parsedURL.String(),
		apiKey:     cfg.APIKey,
		username:   cfg.Username,
		password:   cfg.Password,
		httpClient: client,
		logger:     logger.With().Str("component", "komodo_client").Logger(),
	}, nil
}

// TestConnection verifies the connection to Komodo
func (c *Client) TestConnection(ctx context.Context) error {
	req, err := c.newRequest(ctx, "GET", "/api/health", nil)
	if err != nil {
		return fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("komodo: connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("komodo: authentication failed")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("komodo: health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		// Health endpoint may return plain text
		c.logger.Debug().Msg("health endpoint returned non-JSON response, connection successful")
		return nil
	}

	c.logger.Debug().
		Str("status", health.Status).
		Str("version", health.Version).
		Msg("komodo connection successful")

	return nil
}

// ListStacks retrieves all stacks from Komodo
func (c *Client) ListStacks(ctx context.Context) ([]APIStack, error) {
	req, err := c.newRequest(ctx, "GET", "/api/stacks", nil)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to list stacks: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var result ListStacksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Try decoding as array directly
		resp.Body = io.NopCloser(bytes.NewReader([]byte{}))
		var stacks []APIStack
		if err2 := json.NewDecoder(resp.Body).Decode(&stacks); err2 != nil {
			return nil, fmt.Errorf("komodo: failed to decode stacks response: %w", err)
		}
		return stacks, nil
	}

	return result.Stacks, nil
}

// GetStack retrieves a single stack by ID
func (c *Client) GetStack(ctx context.Context, stackID string) (*APIStack, error) {
	req, err := c.newRequest(ctx, "GET", fmt.Sprintf("/api/stacks/%s", stackID), nil)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to get stack: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var stack APIStack
	if err := json.NewDecoder(resp.Body).Decode(&stack); err != nil {
		return nil, fmt.Errorf("komodo: failed to decode stack: %w", err)
	}

	return &stack, nil
}

// ListContainers retrieves all containers from Komodo
func (c *Client) ListContainers(ctx context.Context) ([]APIContainer, error) {
	req, err := c.newRequest(ctx, "GET", "/api/containers", nil)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to list containers: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var result ListContainersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Try decoding as array directly
		var containers []APIContainer
		resp.Body = io.NopCloser(bytes.NewReader([]byte{}))
		if err2 := json.NewDecoder(resp.Body).Decode(&containers); err2 != nil {
			return nil, fmt.Errorf("komodo: failed to decode containers response: %w", err)
		}
		return containers, nil
	}

	return result.Containers, nil
}

// GetContainer retrieves a single container by ID
func (c *Client) GetContainer(ctx context.Context, containerID string) (*APIContainer, error) {
	req, err := c.newRequest(ctx, "GET", fmt.Sprintf("/api/containers/%s", containerID), nil)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to get container: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var container APIContainer
	if err := json.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, fmt.Errorf("komodo: failed to decode container: %w", err)
	}

	return &container, nil
}

// ListContainersByStack retrieves all containers for a specific stack
func (c *Client) ListContainersByStack(ctx context.Context, stackID string) ([]APIContainer, error) {
	req, err := c.newRequest(ctx, "GET", fmt.Sprintf("/api/stacks/%s/containers", stackID), nil)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to list stack containers: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var result ListContainersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		var containers []APIContainer
		resp.Body = io.NopCloser(bytes.NewReader([]byte{}))
		if err2 := json.NewDecoder(resp.Body).Decode(&containers); err2 != nil {
			return nil, fmt.Errorf("komodo: failed to decode containers response: %w", err)
		}
		return containers, nil
	}

	return result.Containers, nil
}

// ListServers retrieves all servers from Komodo
func (c *Client) ListServers(ctx context.Context) ([]APIServer, error) {
	req, err := c.newRequest(ctx, "GET", "/api/servers", nil)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("komodo: failed to list servers: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var result ListServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		var servers []APIServer
		resp.Body = io.NopCloser(bytes.NewReader([]byte{}))
		if err2 := json.NewDecoder(resp.Body).Decode(&servers); err2 != nil {
			return nil, fmt.Errorf("komodo: failed to decode servers response: %w", err)
		}
		return servers, nil
	}

	return result.Servers, nil
}

// UpdateBackupStatus sends a backup status update to Komodo
func (c *Client) UpdateBackupStatus(ctx context.Context, status *StatusUpdateRequest) error {
	body, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("komodo: failed to marshal status update: %w", err)
	}

	req, err := c.newRequest(ctx, "POST", "/api/integrations/keldris/status", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("komodo: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("komodo: failed to send status update: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		// Status updates may not be supported by all Komodo versions
		c.logger.Warn().Err(err).Msg("failed to update backup status in Komodo")
		return nil
	}

	c.logger.Debug().
		Str("container_id", status.ContainerID).
		Str("stack_id", status.StackID).
		Str("status", status.BackupStatus).
		Msg("backup status updated in Komodo")

	return nil
}

// newRequest creates a new HTTP request with authentication
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	fullURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}

	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add authentication
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("X-API-Key", c.apiKey)
	} else if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Keldris-Backup/1.0")

	return req, nil
}

// checkResponse checks if the HTTP response indicates an error
func (c *Client) checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return &apiErr
	}

	return fmt.Errorf("komodo: request failed with status %d: %s", resp.StatusCode, string(body))
}
