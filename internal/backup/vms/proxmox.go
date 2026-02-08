// Package vms provides VM backup functionality for various hypervisors.
package vms

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// ProxmoxClient handles communication with the Proxmox VE API.
type ProxmoxClient struct {
	config     *ProxmoxConfig
	httpClient *http.Client
	logger     zerolog.Logger
}

// ProxmoxConfig contains connection settings for Proxmox VE.
type ProxmoxConfig struct {
	Host        string
	Port        int
	Node        string
	Username    string
	TokenID     string
	TokenSecret string
	VerifySSL   bool
}

// ProxmoxVM represents a VM or LXC container from Proxmox.
type ProxmoxVM struct {
	VMID      int    `json:"vmid"`
	Name      string `json:"name"`
	Type      string `json:"type"`   // qemu or lxc
	Status    string `json:"status"` // running, stopped, paused
	Node      string `json:"node"`
	CPUs      int    `json:"cpus"`
	MaxMem    int64  `json:"maxmem"`
	MaxDisk   int64  `json:"maxdisk"`
	Uptime    int64  `json:"uptime,omitempty"`
	NetIn     int64  `json:"netin,omitempty"`
	NetOut    int64  `json:"netout,omitempty"`
	DiskRead  int64  `json:"diskread,omitempty"`
	DiskWrite int64  `json:"diskwrite,omitempty"`
}

// BackupJob represents an active vzdump backup task.
type BackupJob struct {
	UPID      string    `json:"upid"`
	Node      string    `json:"node"`
	VMID      int       `json:"vmid"`
	Type      string    `json:"type"` // qemu or lxc
	Status    string    `json:"status"`
	StartTime time.Time `json:"starttime"`
	EndTime   *time.Time `json:"endtime,omitempty"`
	ExitCode  string    `json:"exitstatus,omitempty"`
}

// BackupFile represents a backup file stored on Proxmox.
type BackupFile struct {
	Volid   string `json:"volid"`
	Format  string `json:"format"`
	Size    int64  `json:"size"`
	Ctime   int64  `json:"ctime"`
	Content string `json:"content"`
	VMID    int    `json:"vmid"`
}

// ProxmoxVersion contains version information from the API.
type ProxmoxVersion struct {
	Version string `json:"version"`
	Release string `json:"release"`
	RepoID  string `json:"repoid"`
}

// proxmoxResponse wraps the standard Proxmox API response format.
type proxmoxResponse struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error,omitempty"`
}

// NewProxmoxClient creates a new Proxmox API client.
func NewProxmoxClient(config *ProxmoxConfig, logger zerolog.Logger) *ProxmoxClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.VerifySSL,
		},
	}

	return &ProxmoxClient{
		config: config,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		logger: logger.With().Str("component", "proxmox_client").Logger(),
	}
}

// NewProxmoxClientFromConnection creates a client from a ProxmoxConnection model.
func NewProxmoxClientFromConnection(conn *models.ProxmoxConnection, tokenSecret string, logger zerolog.Logger) *ProxmoxClient {
	config := &ProxmoxConfig{
		Host:        conn.Host,
		Port:        conn.Port,
		Node:        conn.Node,
		Username:    conn.Username,
		TokenID:     conn.TokenID,
		TokenSecret: tokenSecret,
		VerifySSL:   conn.VerifySSL,
	}
	return NewProxmoxClient(config, logger)
}

// baseURL returns the base API URL.
func (c *ProxmoxClient) baseURL() string {
	return fmt.Sprintf("https://%s:%d/api2/json", c.config.Host, c.config.Port)
}

// authHeader returns the Authorization header value.
func (c *ProxmoxClient) authHeader() string {
	return fmt.Sprintf("PVEAPIToken=%s!%s=%s", c.config.Username, c.config.TokenID, c.config.TokenSecret)
}

// doRequest performs an HTTP request to the Proxmox API.
func (c *ProxmoxClient) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := c.baseURL() + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader())
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

// parseResponse parses a Proxmox API response.
func (c *ProxmoxClient) parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error %d: %s", resp.StatusCode, string(body))
	}

	var pveResp proxmoxResponse
	if err := json.NewDecoder(resp.Body).Decode(&pveResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if pveResp.Error != "" {
		return fmt.Errorf("proxmox error: %s", pveResp.Error)
	}

	if v != nil && len(pveResp.Data) > 0 {
		if err := json.Unmarshal(pveResp.Data, v); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}

	return nil
}

// TestConnection tests the connection to the Proxmox API.
func (c *ProxmoxClient) TestConnection(ctx context.Context) error {
	_, err := c.GetVersion(ctx)
	return err
}

// GetVersion retrieves the Proxmox version information.
func (c *ProxmoxClient) GetVersion(ctx context.Context) (*ProxmoxVersion, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/version", nil)
	if err != nil {
		return nil, err
	}

	var version ProxmoxVersion
	if err := c.parseResponse(resp, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// ListVMs retrieves all VMs (qemu) on the node.
func (c *ProxmoxClient) ListVMs(ctx context.Context) ([]ProxmoxVM, error) {
	path := fmt.Sprintf("/nodes/%s/qemu", c.config.Node)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var vms []ProxmoxVM
	if err := c.parseResponse(resp, &vms); err != nil {
		return nil, err
	}

	for i := range vms {
		vms[i].Type = "qemu"
		vms[i].Node = c.config.Node
	}

	return vms, nil
}

// ListContainers retrieves all LXC containers on the node.
func (c *ProxmoxClient) ListContainers(ctx context.Context) ([]ProxmoxVM, error) {
	path := fmt.Sprintf("/nodes/%s/lxc", c.config.Node)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var containers []ProxmoxVM
	if err := c.parseResponse(resp, &containers); err != nil {
		return nil, err
	}

	for i := range containers {
		containers[i].Type = "lxc"
		containers[i].Node = c.config.Node
	}

	return containers, nil
}

// ListAll retrieves all VMs and containers on the node.
func (c *ProxmoxClient) ListAll(ctx context.Context) ([]ProxmoxVM, error) {
	vms, err := c.ListVMs(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to list VMs")
		vms = []ProxmoxVM{}
	}

	containers, err := c.ListContainers(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to list containers")
		containers = []ProxmoxVM{}
	}

	return append(vms, containers...), nil
}

// VzdumpOptions contains options for vzdump backup.
type VzdumpOptions struct {
	VMID       int
	Type       string // qemu or lxc
	Mode       string // snapshot, suspend, stop
	Compress   string // 0, gzip, lzo, zstd
	Storage    string // Proxmox storage ID for backup
	IncludeRAM bool   // Include RAM state (VMs only, requires snapshot mode)
}

// StartBackup initiates a vzdump backup for a VM or container.
func (c *ProxmoxClient) StartBackup(ctx context.Context, opts VzdumpOptions) (*BackupJob, error) {
	path := fmt.Sprintf("/nodes/%s/vzdump", c.config.Node)

	params := url.Values{}
	params.Set("vmid", strconv.Itoa(opts.VMID))
	params.Set("mode", opts.Mode)
	params.Set("compress", opts.Compress)

	if opts.Storage != "" {
		params.Set("storage", opts.Storage)
	}
	if opts.IncludeRAM && opts.Type == "qemu" && opts.Mode == "snapshot" {
		params.Set("vmstate", "1")
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	var upid string
	if err := c.parseResponse(resp, &upid); err != nil {
		return nil, err
	}

	return &BackupJob{
		UPID:      upid,
		Node:      c.config.Node,
		VMID:      opts.VMID,
		Type:      opts.Type,
		Status:    "running",
		StartTime: time.Now(),
	}, nil
}

// GetTaskStatus retrieves the status of a Proxmox task.
func (c *ProxmoxClient) GetTaskStatus(ctx context.Context, upid string) (*BackupJob, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", c.config.Node, url.PathEscape(upid))
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status struct {
		Status     string  `json:"status"`
		ExitStatus string  `json:"exitstatus"`
		StartTime  float64 `json:"starttime"`
		EndTime    float64 `json:"endtime,omitempty"`
		Type       string  `json:"type"`
		ID         string  `json:"id"`
	}
	if err := c.parseResponse(resp, &status); err != nil {
		return nil, err
	}

	job := &BackupJob{
		UPID:      upid,
		Node:      c.config.Node,
		Status:    status.Status,
		ExitCode:  status.ExitStatus,
		StartTime: time.Unix(int64(status.StartTime), 0),
	}

	if status.EndTime > 0 {
		endTime := time.Unix(int64(status.EndTime), 0)
		job.EndTime = &endTime
	}

	return job, nil
}

// WaitForTask waits for a task to complete with timeout.
func (c *ProxmoxClient) WaitForTask(ctx context.Context, upid string, maxWait time.Duration) (*BackupJob, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(maxWait)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for task %s", upid)
			}

			job, err := c.GetTaskStatus(ctx, upid)
			if err != nil {
				c.logger.Warn().Err(err).Str("upid", upid).Msg("error checking task status")
				continue
			}

			if job.Status == "stopped" {
				return job, nil
			}
		}
	}
}

// ListBackupFiles lists backup files for a VM on a storage.
func (c *ProxmoxClient) ListBackupFiles(ctx context.Context, storage string, vmid int) ([]BackupFile, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/content", c.config.Node, storage)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var allFiles []BackupFile
	if err := c.parseResponse(resp, &allFiles); err != nil {
		return nil, err
	}

	// Filter for backup files matching the VMID
	var backups []BackupFile
	for _, f := range allFiles {
		if f.Content == "backup" && f.VMID == vmid {
			backups = append(backups, f)
		}
	}

	return backups, nil
}

// DownloadBackup downloads a backup file to a local directory.
func (c *ProxmoxClient) DownloadBackup(ctx context.Context, volid string, destDir string) (string, error) {
	// Parse the volid to get storage and filename
	parts := strings.SplitN(volid, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid volid format: %s", volid)
	}
	storage := parts[0]
	filename := parts[1]

	// Use the download-url API to get a download link
	path := fmt.Sprintf("/nodes/%s/storage/%s/file-restore/download", c.config.Node, storage)
	params := url.Values{}
	params.Set("volume", volid)
	params.Set("filepath", "/")

	resp, err := c.doRequest(ctx, http.MethodGet, path+"?"+params.Encode(), nil)
	if err != nil {
		// Fallback: try direct download via vzdump path
		return c.downloadBackupDirect(ctx, storage, filename, destDir)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.downloadBackupDirect(ctx, storage, filename, destDir)
	}

	destPath := filepath.Join(destDir, filepath.Base(filename))
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create destination file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return "", fmt.Errorf("download backup: %w", err)
	}

	return destPath, nil
}

// downloadBackupDirect attempts to download the backup file directly.
func (c *ProxmoxClient) downloadBackupDirect(ctx context.Context, storage, filename, destDir string) (string, error) {
	// This would typically use PBS or a direct file transfer method
	// For now, return an error indicating manual intervention may be needed
	return "", fmt.Errorf("direct download not implemented for storage %s, file %s - consider using Proxmox Backup Server integration", storage, filename)
}

// DeleteBackup removes a backup file from Proxmox storage.
func (c *ProxmoxClient) DeleteBackup(ctx context.Context, volid string) error {
	parts := strings.SplitN(volid, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid volid format: %s", volid)
	}
	storage := parts[0]

	path := fmt.Sprintf("/nodes/%s/storage/%s/content/%s", c.config.Node, storage, url.PathEscape(volid))
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	return c.parseResponse(resp, nil)
}

// ToModelVMInfo converts a ProxmoxVM to the models.ProxmoxVMInfo type.
func (vm *ProxmoxVM) ToModelVMInfo() models.ProxmoxVMInfo {
	return models.ProxmoxVMInfo{
		VMID:    vm.VMID,
		Name:    vm.Name,
		Type:    vm.Type,
		Status:  vm.Status,
		Node:    vm.Node,
		CPUs:    vm.CPUs,
		MaxMem:  vm.MaxMem,
		MaxDisk: vm.MaxDisk,
	}
}

// ToModelInfo creates a models.ProxmoxInfo from a list of VMs.
func ToModelInfo(vms []ProxmoxVM, host, node, version, connectionID string) *models.ProxmoxInfo {
	now := time.Now()
	vmCount := 0
	lxcCount := 0
	modelVMs := make([]models.ProxmoxVMInfo, 0, len(vms))

	for _, vm := range vms {
		if vm.Type == "qemu" {
			vmCount++
		} else if vm.Type == "lxc" {
			lxcCount++
		}
		modelVMs = append(modelVMs, vm.ToModelVMInfo())
	}

	return &models.ProxmoxInfo{
		Available:    true,
		Host:         host,
		Node:         node,
		Version:      version,
		VMCount:      vmCount,
		LXCCount:     lxcCount,
		VMs:          modelVMs,
		ConnectionID: connectionID,
		DetectedAt:   &now,
	}
}
