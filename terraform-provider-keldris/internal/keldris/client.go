// Package keldris provides the API client for Keldris backup management.
package keldris

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the Keldris API client.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Keldris API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Agent represents a Keldris backup agent.
type Agent struct {
	ID        string     `json:"id"`
	OrgID     string     `json:"org_id"`
	Hostname  string     `json:"hostname"`
	Status    string     `json:"status"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CreateAgentRequest is the request for creating an agent.
type CreateAgentRequest struct {
	Hostname string `json:"hostname"`
}

// CreateAgentResponse is the response for creating an agent.
type CreateAgentResponse struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	APIKey   string `json:"api_key"`
}

// CreateAgent creates a new agent.
func (c *Client) CreateAgent(ctx context.Context, req *CreateAgentRequest) (*CreateAgentResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/api/v1/agents", req)
	if err != nil {
		return nil, err
	}

	var resp CreateAgentResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}

// GetAgent retrieves an agent by ID.
func (c *Client) GetAgent(ctx context.Context, id string) (*Agent, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/agents/"+id, nil)
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := json.Unmarshal(respBody, &agent); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &agent, nil
}

// DeleteAgent deletes an agent by ID.
func (c *Client) DeleteAgent(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/v1/agents/"+id, nil)
	return err
}

// ListAgents lists all agents.
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/agents", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Agents []Agent `json:"agents"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.Agents, nil
}

// Repository represents a backup repository.
type Repository struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRepositoryRequest is the request for creating a repository.
type CreateRepositoryRequest struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Config        map[string]interface{} `json:"config"`
	EscrowEnabled bool                   `json:"escrow_enabled"`
}

// CreateRepositoryResponse is the response for creating a repository.
type CreateRepositoryResponse struct {
	Repository Repository `json:"repository"`
	Password   string     `json:"password"`
}

// CreateRepository creates a new repository.
func (c *Client) CreateRepository(ctx context.Context, req *CreateRepositoryRequest) (*CreateRepositoryResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/api/v1/repositories", req)
	if err != nil {
		return nil, err
	}

	var resp CreateRepositoryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}

// GetRepository retrieves a repository by ID.
func (c *Client) GetRepository(ctx context.Context, id string) (*Repository, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/repositories/"+id, nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := json.Unmarshal(respBody, &repo); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &repo, nil
}

// UpdateRepositoryRequest is the request for updating a repository.
type UpdateRepositoryRequest struct {
	Name   string                 `json:"name,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpdateRepository updates an existing repository.
func (c *Client) UpdateRepository(ctx context.Context, id string, req *UpdateRepositoryRequest) (*Repository, error) {
	respBody, err := c.doRequest(ctx, http.MethodPut, "/api/v1/repositories/"+id, req)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := json.Unmarshal(respBody, &repo); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &repo, nil
}

// DeleteRepository deletes a repository by ID.
func (c *Client) DeleteRepository(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/v1/repositories/"+id, nil)
	return err
}

// ListRepositories lists all repositories.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/repositories", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Repositories []Repository `json:"repositories"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.Repositories, nil
}

// RetentionPolicy defines backup retention settings.
type RetentionPolicy struct {
	KeepLast    int `json:"keep_last,omitempty"`
	KeepHourly  int `json:"keep_hourly,omitempty"`
	KeepDaily   int `json:"keep_daily,omitempty"`
	KeepWeekly  int `json:"keep_weekly,omitempty"`
	KeepMonthly int `json:"keep_monthly,omitempty"`
	KeepYearly  int `json:"keep_yearly,omitempty"`
}

// BackupWindow defines when backups are allowed.
type BackupWindow struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// ScheduleRepository represents a repository association in a schedule.
type ScheduleRepository struct {
	RepositoryID string `json:"repository_id"`
	Priority     int    `json:"priority"`
	Enabled      bool   `json:"enabled"`
}

// Schedule represents a backup schedule.
type Schedule struct {
	ID               string               `json:"id"`
	AgentID          string               `json:"agent_id"`
	PolicyID         *string              `json:"policy_id,omitempty"`
	Name             string               `json:"name"`
	CronExpression   string               `json:"cron_expression"`
	Paths            []string             `json:"paths"`
	Excludes         []string             `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                 `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *BackupWindow        `json:"backup_window,omitempty"`
	ExcludedHours    []int                `json:"excluded_hours,omitempty"`
	CompressionLevel *string              `json:"compression_level,omitempty"`
	Enabled          bool                 `json:"enabled"`
	Repositories     []ScheduleRepository `json:"repositories,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// CreateScheduleRequest is the request for creating a schedule.
type CreateScheduleRequest struct {
	AgentID          string               `json:"agent_id"`
	Name             string               `json:"name"`
	CronExpression   string               `json:"cron_expression"`
	Paths            []string             `json:"paths"`
	Excludes         []string             `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                 `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *BackupWindow        `json:"backup_window,omitempty"`
	ExcludedHours    []int                `json:"excluded_hours,omitempty"`
	CompressionLevel *string              `json:"compression_level,omitempty"`
	Enabled          *bool                `json:"enabled,omitempty"`
	Repositories     []ScheduleRepository `json:"repositories"`
}

// CreateSchedule creates a new schedule.
func (c *Client) CreateSchedule(ctx context.Context, req *CreateScheduleRequest) (*Schedule, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/api/v1/schedules", req)
	if err != nil {
		return nil, err
	}

	var schedule Schedule
	if err := json.Unmarshal(respBody, &schedule); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &schedule, nil
}

// GetSchedule retrieves a schedule by ID.
func (c *Client) GetSchedule(ctx context.Context, id string) (*Schedule, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/schedules/"+id, nil)
	if err != nil {
		return nil, err
	}

	var schedule Schedule
	if err := json.Unmarshal(respBody, &schedule); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &schedule, nil
}

// UpdateScheduleRequest is the request for updating a schedule.
type UpdateScheduleRequest struct {
	Name             string               `json:"name,omitempty"`
	CronExpression   string               `json:"cron_expression,omitempty"`
	Paths            []string             `json:"paths,omitempty"`
	Excludes         []string             `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                 `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *BackupWindow        `json:"backup_window,omitempty"`
	ExcludedHours    []int                `json:"excluded_hours,omitempty"`
	CompressionLevel *string              `json:"compression_level,omitempty"`
	Enabled          *bool                `json:"enabled,omitempty"`
	Repositories     []ScheduleRepository `json:"repositories,omitempty"`
}

// UpdateSchedule updates an existing schedule.
func (c *Client) UpdateSchedule(ctx context.Context, id string, req *UpdateScheduleRequest) (*Schedule, error) {
	respBody, err := c.doRequest(ctx, http.MethodPut, "/api/v1/schedules/"+id, req)
	if err != nil {
		return nil, err
	}

	var schedule Schedule
	if err := json.Unmarshal(respBody, &schedule); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &schedule, nil
}

// DeleteSchedule deletes a schedule by ID.
func (c *Client) DeleteSchedule(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/v1/schedules/"+id, nil)
	return err
}

// ListSchedules lists all schedules.
func (c *Client) ListSchedules(ctx context.Context) ([]Schedule, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/schedules", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Schedules []Schedule `json:"schedules"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.Schedules, nil
}

// Policy represents a backup policy template.
type Policy struct {
	ID               string           `json:"id"`
	OrgID            string           `json:"org_id"`
	Name             string           `json:"name"`
	Description      string           `json:"description,omitempty"`
	Paths            []string         `json:"paths,omitempty"`
	Excludes         []string         `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int             `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int            `json:"excluded_hours,omitempty"`
	CronExpression   string           `json:"cron_expression,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// CreatePolicyRequest is the request for creating a policy.
type CreatePolicyRequest struct {
	Name             string           `json:"name"`
	Description      string           `json:"description,omitempty"`
	Paths            []string         `json:"paths,omitempty"`
	Excludes         []string         `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int             `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int            `json:"excluded_hours,omitempty"`
	CronExpression   string           `json:"cron_expression,omitempty"`
}

// CreatePolicy creates a new policy.
func (c *Client) CreatePolicy(ctx context.Context, req *CreatePolicyRequest) (*Policy, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/api/v1/policies", req)
	if err != nil {
		return nil, err
	}

	var policy Policy
	if err := json.Unmarshal(respBody, &policy); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &policy, nil
}

// GetPolicy retrieves a policy by ID.
func (c *Client) GetPolicy(ctx context.Context, id string) (*Policy, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/policies/"+id, nil)
	if err != nil {
		return nil, err
	}

	var policy Policy
	if err := json.Unmarshal(respBody, &policy); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &policy, nil
}

// UpdatePolicyRequest is the request for updating a policy.
type UpdatePolicyRequest struct {
	Name             string           `json:"name,omitempty"`
	Description      string           `json:"description,omitempty"`
	Paths            []string         `json:"paths,omitempty"`
	Excludes         []string         `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int             `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int            `json:"excluded_hours,omitempty"`
	CronExpression   string           `json:"cron_expression,omitempty"`
}

// UpdatePolicy updates an existing policy.
func (c *Client) UpdatePolicy(ctx context.Context, id string, req *UpdatePolicyRequest) (*Policy, error) {
	respBody, err := c.doRequest(ctx, http.MethodPut, "/api/v1/policies/"+id, req)
	if err != nil {
		return nil, err
	}

	var policy Policy
	if err := json.Unmarshal(respBody, &policy); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &policy, nil
}

// DeletePolicy deletes a policy by ID.
func (c *Client) DeletePolicy(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/v1/policies/"+id, nil)
	return err
}

// ListPolicies lists all policies.
func (c *Client) ListPolicies(ctx context.Context) ([]Policy, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api/v1/policies", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Policies []Policy `json:"policies"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.Policies, nil
}
