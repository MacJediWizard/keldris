package vms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// newTestLogger returns a zerolog.Logger that discards all output.
func newTestLogger() zerolog.Logger {
	return zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.NoColor = true
	})).Level(zerolog.Disabled)
}

// proxmoxAPIHandler returns an http.Handler that simulates a Proxmox VE API.
// The provided overrides map lets callers override specific paths with custom handlers.
// If a path appears in overrides, the override replaces the default handler for that path.
func proxmoxAPIHandler(overrides map[string]http.HandlerFunc) http.Handler {
	// Build defaults map
	defaults := map[string]http.HandlerFunc{
		"/api2/json/version": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": map[string]string{
					"version": "8.1.3",
					"release": "1",
					"repoid":  "abc123",
				},
			})
		},
		"/api2/json/nodes/pve1/qemu": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					{"vmid": 100, "name": "ubuntu-server", "status": "running", "cpus": 4, "maxmem": 8589934592, "maxdisk": 34359738368},
					{"vmid": 101, "name": "windows-desktop", "status": "stopped", "cpus": 2, "maxmem": 4294967296, "maxdisk": 53687091200},
				},
			})
		},
		"/api2/json/nodes/pve1/lxc": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					{"vmid": 200, "name": "nginx-proxy", "status": "running", "cpus": 1, "maxmem": 536870912, "maxdisk": 8589934592},
				},
			})
		},
		"/api2/json/nodes/pve1/vzdump": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			writeJSON(w, map[string]interface{}{
				"data": "UPID:pve1:00001234:12345678:12345678:vzdump:100:root@pam:",
			})
		},
		"/api2/json/nodes/pve1/tasks/": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": map[string]interface{}{
					"status":     "stopped",
					"exitstatus": "OK",
					"starttime":  float64(time.Now().Add(-5 * time.Minute).Unix()),
					"endtime":    float64(time.Now().Unix()),
					"type":       "vzdump",
				},
			})
		},
		"/api2/json/nodes/pve1/storage/local/content": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]interface{}{
				"data": []map[string]interface{}{
					{"volid": "local:backup/vzdump-qemu-100-2024_01_15-12_00_00.vma.zst", "format": "vma.zst", "size": 1073741824, "ctime": 1705312800, "content": "backup", "vmid": 100},
					{"volid": "local:backup/vzdump-qemu-100-2024_01_14-12_00_00.vma.zst", "format": "vma.zst", "size": 1073741000, "ctime": 1705226400, "content": "backup", "vmid": 100},
					{"volid": "local:iso/ubuntu.iso", "format": "iso", "size": 4000000000, "ctime": 1700000000, "content": "iso", "vmid": 0},
				},
			})
		},
	}

	// Apply overrides (replace defaults or add new paths)
	for path, handler := range overrides {
		defaults[path] = handler
	}

	// Register all handlers
	mux := http.NewServeMux()
	for path, handler := range defaults {
		mux.HandleFunc(path, handler)
	}

	return mux
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ---- Tests ----

func TestProxmoxClient_GetVersion(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	version, err := client.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion: unexpected error: %v", err)
	}

	if version.Version != "8.1.3" {
		t.Errorf("Version = %q, want %q", version.Version, "8.1.3")
	}
	if version.Release != "1" {
		t.Errorf("Release = %q, want %q", version.Release, "1")
	}
}

func TestProxmoxClient_GetVersion_ServerError(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(map[string]http.HandlerFunc{
		"/api2/json/version": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"authentication failure"}`, http.StatusUnauthorized)
		},
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	_, err := client.GetVersion(context.Background())
	if err == nil {
		t.Fatal("GetVersion: expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %q, want it to contain 401", err.Error())
	}
}

func TestProxmoxClient_GetVersion_Unreachable(t *testing.T) {
	// Point at a server that is not listening.
	config := &ProxmoxConfig{
		Host:        "127.0.0.1",
		Port:        1, // unlikely to be listening
		Node:        "pve1",
		Username:    "root@pam",
		TokenID:     "tok",
		TokenSecret: "sec",
		VerifySSL:   false,
	}
	client := NewProxmoxClient(config, newTestLogger())
	client.httpClient.Timeout = 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.GetVersion(ctx)
	if err == nil {
		t.Fatal("GetVersion: expected error for unreachable server, got nil")
	}
}

func TestProxmoxClient_TestConnection(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	if err := client.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}
}

func TestProxmoxClient_ListVMs(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	vms, err := client.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs: unexpected error: %v", err)
	}

	if len(vms) != 2 {
		t.Fatalf("ListVMs: got %d VMs, want 2", len(vms))
	}

	tests := []struct {
		idx    int
		vmid   int
		name   string
		typ    string
		node   string
		status string
	}{
		{0, 100, "ubuntu-server", "qemu", "pve1", "running"},
		{1, 101, "windows-desktop", "qemu", "pve1", "stopped"},
	}
	for _, tt := range tests {
		vm := vms[tt.idx]
		if vm.VMID != tt.vmid {
			t.Errorf("vm[%d].VMID = %d, want %d", tt.idx, vm.VMID, tt.vmid)
		}
		if vm.Name != tt.name {
			t.Errorf("vm[%d].Name = %q, want %q", tt.idx, vm.Name, tt.name)
		}
		if vm.Type != tt.typ {
			t.Errorf("vm[%d].Type = %q, want %q", tt.idx, vm.Type, tt.typ)
		}
		if vm.Node != tt.node {
			t.Errorf("vm[%d].Node = %q, want %q", tt.idx, vm.Node, tt.node)
		}
		if vm.Status != tt.status {
			t.Errorf("vm[%d].Status = %q, want %q", tt.idx, vm.Status, tt.status)
		}
	}
}

func TestProxmoxClient_ListContainers(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	containers, err := client.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers: unexpected error: %v", err)
	}

	if len(containers) != 1 {
		t.Fatalf("ListContainers: got %d containers, want 1", len(containers))
	}

	c := containers[0]
	if c.VMID != 200 {
		t.Errorf("VMID = %d, want 200", c.VMID)
	}
	if c.Type != "lxc" {
		t.Errorf("Type = %q, want %q", c.Type, "lxc")
	}
	if c.Node != "pve1" {
		t.Errorf("Node = %q, want %q", c.Node, "pve1")
	}
}

func TestProxmoxClient_ListAll(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	all, err := client.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: unexpected error: %v", err)
	}

	// 2 VMs + 1 container = 3
	if len(all) != 3 {
		t.Fatalf("ListAll: got %d, want 3", len(all))
	}

	// Verify we get both types
	types := map[string]int{}
	for _, vm := range all {
		types[vm.Type]++
	}
	if types["qemu"] != 2 {
		t.Errorf("qemu count = %d, want 2", types["qemu"])
	}
	if types["lxc"] != 1 {
		t.Errorf("lxc count = %d, want 1", types["lxc"])
	}
}

func TestProxmoxClient_ListAll_PartialFailure(t *testing.T) {
	// VMs endpoint fails, but containers succeed
	ts := httptest.NewServer(proxmoxAPIHandler(map[string]http.HandlerFunc{
		"/api2/json/nodes/pve1/qemu": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		},
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	all, err := client.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: unexpected error: %v", err)
	}

	// Should still get the 1 container
	if len(all) != 1 {
		t.Fatalf("ListAll: got %d, want 1 (containers only)", len(all))
	}
	if all[0].Type != "lxc" {
		t.Errorf("Type = %q, want %q", all[0].Type, "lxc")
	}
}

func TestProxmoxClient_StartBackup(t *testing.T) {
	var capturedBody string
	ts := httptest.NewServer(proxmoxAPIHandler(map[string]http.HandlerFunc{
		"/api2/json/nodes/pve1/vzdump": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			capturedBody = string(body)
			writeJSON(w, map[string]interface{}{
				"data": "UPID:pve1:00001234:12345678:12345678:vzdump:100:root@pam:",
			})
		},
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	tests := []struct {
		name       string
		opts       VzdumpOptions
		wantVMID   int
		wantParam  string // a parameter expected in the form body
		wantAbsent string // a parameter that should NOT be in the form body
	}{
		{
			name: "basic_snapshot",
			opts: VzdumpOptions{
				VMID:     100,
				Type:     "qemu",
				Mode:     "snapshot",
				Compress: "zstd",
			},
			wantVMID:   100,
			wantParam:  "mode=snapshot",
			wantAbsent: "vmstate",
		},
		{
			name: "with_storage",
			opts: VzdumpOptions{
				VMID:     101,
				Type:     "qemu",
				Mode:     "stop",
				Compress: "gzip",
				Storage:  "local-lvm",
			},
			wantVMID:  101,
			wantParam: "storage=local-lvm",
		},
		{
			name: "include_ram_qemu_snapshot",
			opts: VzdumpOptions{
				VMID:       100,
				Type:       "qemu",
				Mode:       "snapshot",
				Compress:   "zstd",
				IncludeRAM: true,
			},
			wantVMID:  100,
			wantParam: "vmstate=1",
		},
		{
			name: "include_ram_lxc_ignored",
			opts: VzdumpOptions{
				VMID:       200,
				Type:       "lxc",
				Mode:       "snapshot",
				Compress:   "zstd",
				IncludeRAM: true,
			},
			wantVMID:   200,
			wantAbsent: "vmstate",
		},
		{
			name: "include_ram_stop_mode_ignored",
			opts: VzdumpOptions{
				VMID:       100,
				Type:       "qemu",
				Mode:       "stop",
				Compress:   "zstd",
				IncludeRAM: true,
			},
			wantVMID:   100,
			wantAbsent: "vmstate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := client.StartBackup(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("StartBackup: unexpected error: %v", err)
			}

			if job.VMID != tt.wantVMID {
				t.Errorf("job.VMID = %d, want %d", job.VMID, tt.wantVMID)
			}
			if job.Status != "running" {
				t.Errorf("job.Status = %q, want %q", job.Status, "running")
			}
			if job.UPID == "" {
				t.Error("job.UPID is empty")
			}
			if tt.wantParam != "" && !strings.Contains(capturedBody, tt.wantParam) {
				t.Errorf("request body %q does not contain %q", capturedBody, tt.wantParam)
			}
			if tt.wantAbsent != "" && strings.Contains(capturedBody, tt.wantAbsent) {
				t.Errorf("request body %q unexpectedly contains %q", capturedBody, tt.wantAbsent)
			}
		})
	}
}

func TestProxmoxClient_GetTaskStatus(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	job, err := client.GetTaskStatus(context.Background(), "UPID:pve1:00001234:12345678:12345678:vzdump:100:root@pam:")
	if err != nil {
		t.Fatalf("GetTaskStatus: unexpected error: %v", err)
	}

	if job.Status != "stopped" {
		t.Errorf("Status = %q, want %q", job.Status, "stopped")
	}
	if job.ExitCode != "OK" {
		t.Errorf("ExitCode = %q, want %q", job.ExitCode, "OK")
	}
	if job.EndTime == nil {
		t.Error("EndTime is nil, expected non-nil")
	}
}

func TestProxmoxClient_ListBackupFiles(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	backups, err := client.ListBackupFiles(context.Background(), "local", 100)
	if err != nil {
		t.Fatalf("ListBackupFiles: unexpected error: %v", err)
	}

	// Should get 2 backup files for VMID 100, filtering out the iso
	if len(backups) != 2 {
		t.Fatalf("got %d backups, want 2", len(backups))
	}

	for _, b := range backups {
		if b.Content != "backup" {
			t.Errorf("Content = %q, want %q", b.Content, "backup")
		}
		if b.VMID != 100 {
			t.Errorf("VMID = %d, want 100", b.VMID)
		}
	}
}

func TestProxmoxClient_ListBackupFiles_NoMatch(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	backups, err := client.ListBackupFiles(context.Background(), "local", 999)
	if err != nil {
		t.Fatalf("ListBackupFiles: unexpected error: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("got %d backups, want 0 for non-existent VMID", len(backups))
	}
}

func TestProxmoxClient_DeleteBackup(t *testing.T) {
	var capturedPath string
	var capturedMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		writeJSON(w, map[string]interface{}{"data": nil})
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	t.Run("valid_volid", func(t *testing.T) {
		err := client.DeleteBackup(context.Background(), "local:backup/vzdump-qemu-100.vma.zst")
		if err != nil {
			t.Fatalf("DeleteBackup: unexpected error: %v", err)
		}
		if capturedMethod != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", capturedMethod)
		}
		if !strings.Contains(capturedPath, "/storage/local/content/") {
			t.Errorf("path = %q, want it to contain /storage/local/content/", capturedPath)
		}
	})

	t.Run("invalid_volid_no_colon", func(t *testing.T) {
		err := client.DeleteBackup(context.Background(), "invalid-volid-no-colon")
		if err == nil {
			t.Fatal("expected error for invalid volid, got nil")
		}
		if !strings.Contains(err.Error(), "invalid volid format") {
			t.Errorf("error = %q, want it to contain 'invalid volid format'", err.Error())
		}
	})
}

func TestProxmoxClient_DownloadBackup_InvalidVolid(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	_, err := client.DownloadBackup(context.Background(), "no-colon-volid", t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid volid, got nil")
	}
	if !strings.Contains(err.Error(), "invalid volid format") {
		t.Errorf("error = %q, want it to contain 'invalid volid format'", err.Error())
	}
}

func TestProxmoxClient_AuthHeader(t *testing.T) {
	config := &ProxmoxConfig{
		Username:    "root@pam",
		TokenID:     "my-token",
		TokenSecret: "abc-123-secret",
	}
	client := NewProxmoxClient(config, newTestLogger())

	header := client.authHeader()
	want := "PVEAPIToken=root@pam!my-token=abc-123-secret"
	if header != want {
		t.Errorf("authHeader() = %q, want %q", header, want)
	}
}

func TestProxmoxClient_BaseURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{"default_port", "pve.example.com", 8006, "https://pve.example.com:8006/api2/json"},
		{"custom_port", "10.0.0.1", 443, "https://10.0.0.1:443/api2/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ProxmoxConfig{Host: tt.host, Port: tt.port}
			client := NewProxmoxClient(config, newTestLogger())
			got := client.baseURL()
			if got != tt.want {
				t.Errorf("baseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProxmoxClient_ParseResponse_ProxmoxError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"data":  nil,
			"error": "permission denied",
		})
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	_, err := client.GetVersion(context.Background())
	if err == nil {
		t.Fatal("expected error for proxmox error response, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %q, want it to contain 'permission denied'", err.Error())
	}
}

func TestProxmoxClient_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(5 * time.Second)
		writeJSON(w, map[string]interface{}{"data": nil})
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.GetVersion(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// ---- ToModelInfo / ToModelVMInfo tests ----

func TestToModelVMInfo(t *testing.T) {
	vm := ProxmoxVM{
		VMID:    100,
		Name:    "test-vm",
		Type:    "qemu",
		Status:  "running",
		Node:    "pve1",
		CPUs:    4,
		MaxMem:  8589934592,
		MaxDisk: 34359738368,
	}

	info := vm.ToModelVMInfo()

	if info.VMID != 100 {
		t.Errorf("VMID = %d, want 100", info.VMID)
	}
	if info.Name != "test-vm" {
		t.Errorf("Name = %q, want %q", info.Name, "test-vm")
	}
	if info.Type != "qemu" {
		t.Errorf("Type = %q, want %q", info.Type, "qemu")
	}
	if info.CPUs != 4 {
		t.Errorf("CPUs = %d, want 4", info.CPUs)
	}
}

func TestToModelInfo(t *testing.T) {
	tests := []struct {
		name         string
		vms          []ProxmoxVM
		wantVMCount  int
		wantLXC      int
		wantTotal    int
		wantAvail    bool
	}{
		{
			name: "mixed_vms_and_containers",
			vms: []ProxmoxVM{
				{VMID: 100, Name: "vm1", Type: "qemu", Status: "running"},
				{VMID: 101, Name: "vm2", Type: "qemu", Status: "stopped"},
				{VMID: 200, Name: "ct1", Type: "lxc", Status: "running"},
			},
			wantVMCount: 2,
			wantLXC:     1,
			wantTotal:   3,
			wantAvail:   true,
		},
		{
			name:        "empty_list",
			vms:         []ProxmoxVM{},
			wantVMCount: 0,
			wantLXC:     0,
			wantTotal:   0,
			wantAvail:   true,
		},
		{
			name: "containers_only",
			vms: []ProxmoxVM{
				{VMID: 200, Name: "ct1", Type: "lxc"},
				{VMID: 201, Name: "ct2", Type: "lxc"},
			},
			wantVMCount: 0,
			wantLXC:     2,
			wantTotal:   2,
			wantAvail:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ToModelInfo(tt.vms, "pve.local", "pve1", "8.1.3", "conn-123")

			if info.Available != tt.wantAvail {
				t.Errorf("Available = %v, want %v", info.Available, tt.wantAvail)
			}
			if info.VMCount != tt.wantVMCount {
				t.Errorf("VMCount = %d, want %d", info.VMCount, tt.wantVMCount)
			}
			if info.LXCCount != tt.wantLXC {
				t.Errorf("LXCCount = %d, want %d", info.LXCCount, tt.wantLXC)
			}
			if len(info.VMs) != tt.wantTotal {
				t.Errorf("len(VMs) = %d, want %d", len(info.VMs), tt.wantTotal)
			}
			if info.Host != "pve.local" {
				t.Errorf("Host = %q, want %q", info.Host, "pve.local")
			}
			if info.Node != "pve1" {
				t.Errorf("Node = %q, want %q", info.Node, "pve1")
			}
			if info.Version != "8.1.3" {
				t.Errorf("Version = %q, want %q", info.Version, "8.1.3")
			}
			if info.ConnectionID != "conn-123" {
				t.Errorf("ConnectionID = %q, want %q", info.ConnectionID, "conn-123")
			}
			if info.DetectedAt == nil {
				t.Error("DetectedAt is nil, expected non-nil")
			}
		})
	}
}

// ---- Discovery tests ----

func TestProxmoxDiscovery_Discover(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(nil))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	discovery := NewProxmoxDiscovery(newTestLogger())

	result := discovery.Discover(context.Background(), client, "conn-abc")

	if !result.Success {
		t.Fatalf("Discover: expected success, got error: %s", result.ErrorMessage)
	}
	if result.ProxmoxInfo == nil {
		t.Fatal("ProxmoxInfo is nil")
	}
	if !result.ProxmoxInfo.Available {
		t.Error("Available = false, want true")
	}
	if result.ProxmoxInfo.VMCount != 2 {
		t.Errorf("VMCount = %d, want 2", result.ProxmoxInfo.VMCount)
	}
	if result.ProxmoxInfo.LXCCount != 1 {
		t.Errorf("LXCCount = %d, want 1", result.ProxmoxInfo.LXCCount)
	}
	if result.ProxmoxInfo.ConnectionID != "conn-abc" {
		t.Errorf("ConnectionID = %q, want %q", result.ProxmoxInfo.ConnectionID, "conn-abc")
	}
}

func TestProxmoxDiscovery_Discover_VersionFailure(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(map[string]http.HandlerFunc{
		"/api2/json/version": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		},
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	discovery := NewProxmoxDiscovery(newTestLogger())

	result := discovery.Discover(context.Background(), client, "conn-abc")

	if result.Success {
		t.Fatal("expected failure when version endpoint fails")
	}
	if result.ProxmoxInfo == nil {
		t.Fatal("ProxmoxInfo should not be nil even on failure")
	}
	if result.ProxmoxInfo.Available {
		t.Error("Available should be false when version fails")
	}
	if result.ErrorMessage == "" {
		t.Error("ErrorMessage should not be empty")
	}
}

func TestProxmoxDiscovery_Discover_ListFailure(t *testing.T) {
	ts := httptest.NewServer(proxmoxAPIHandler(map[string]http.HandlerFunc{
		"/api2/json/nodes/pve1/qemu": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "node error", http.StatusInternalServerError)
		},
		"/api2/json/nodes/pve1/lxc": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "node error", http.StatusInternalServerError)
		},
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)
	discovery := NewProxmoxDiscovery(newTestLogger())

	result := discovery.Discover(context.Background(), client, "conn-abc")

	// ListAll returns empty lists on partial failure rather than error,
	// so discovery should still succeed with 0 VMs.
	if !result.Success {
		t.Fatalf("expected success (with 0 VMs), got error: %s", result.ErrorMessage)
	}
	if result.ProxmoxInfo.VMCount != 0 {
		t.Errorf("VMCount = %d, want 0", result.ProxmoxInfo.VMCount)
	}
}

// ---- Backup path construction tests ----

func TestBackupPathConstruction(t *testing.T) {
	// Test the VM backup directory structure
	tempDir := t.TempDir()
	vmDir := filepath.Join(tempDir, fmt.Sprintf("vm-%d", 100))

	if err := os.MkdirAll(vmDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Verify the directory was created
	info, err := os.Stat(vmDir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}

	// Verify the path format
	want := filepath.Join(tempDir, "vm-100")
	if vmDir != want {
		t.Errorf("vmDir = %q, want %q", vmDir, want)
	}
}

// ---- Backup result aggregation tests ----

func TestBackupResult_Aggregation(t *testing.T) {
	result := &BackupResult{
		Success:     true,
		BackupPaths: []string{},
		VMResults:   []VMBackupResult{},
	}

	// Simulate adding successful VM results
	successResult := VMBackupResult{
		VMID:       100,
		Name:       "vm1",
		Type:       "qemu",
		Success:    true,
		BackupPath: "/tmp/backups/vm-100/backup.vma.zst",
		Size:       1073741824,
		Duration:   5 * time.Minute,
	}
	result.VMResults = append(result.VMResults, successResult)
	result.VMsBackedUp++
	result.TotalSize += successResult.Size
	result.BackupPaths = append(result.BackupPaths, successResult.BackupPath)

	// Simulate adding a failed VM result
	failResult := VMBackupResult{
		VMID:    101,
		Name:    "vm2",
		Type:    "qemu",
		Success: false,
		Error:   "vzdump failed with exit code: ERROR",
	}
	result.VMResults = append(result.VMResults, failResult)
	result.Success = false
	result.ErrorMessage = failResult.Error

	if result.VMsBackedUp != 1 {
		t.Errorf("VMsBackedUp = %d, want 1", result.VMsBackedUp)
	}
	if result.TotalSize != 1073741824 {
		t.Errorf("TotalSize = %d, want 1073741824", result.TotalSize)
	}
	if len(result.BackupPaths) != 1 {
		t.Errorf("BackupPaths count = %d, want 1", len(result.BackupPaths))
	}
	if result.Success {
		t.Error("Success should be false when any VM fails")
	}
	if len(result.VMResults) != 2 {
		t.Errorf("VMResults count = %d, want 2", len(result.VMResults))
	}
}

// ---- VzdumpOptions tests ----

func TestVzdumpOptions_FormEncoding(t *testing.T) {
	tests := []struct {
		name       string
		opts       VzdumpOptions
		wantParams map[string]string
		wantAbsent []string
	}{
		{
			name: "basic",
			opts: VzdumpOptions{
				VMID:     100,
				Type:     "qemu",
				Mode:     "snapshot",
				Compress: "zstd",
			},
			wantParams: map[string]string{
				"vmid":     "100",
				"mode":     "snapshot",
				"compress": "zstd",
			},
			wantAbsent: []string{"storage", "vmstate"},
		},
		{
			name: "with_storage",
			opts: VzdumpOptions{
				VMID:     100,
				Type:     "qemu",
				Mode:     "snapshot",
				Compress: "zstd",
				Storage:  "ceph-pool",
			},
			wantParams: map[string]string{
				"storage": "ceph-pool",
			},
		},
		{
			name: "vmstate_only_for_qemu_snapshot",
			opts: VzdumpOptions{
				VMID:       100,
				Type:       "qemu",
				Mode:       "snapshot",
				Compress:   "zstd",
				IncludeRAM: true,
			},
			wantParams: map[string]string{
				"vmstate": "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("vmid", strconv.Itoa(tt.opts.VMID))
			params.Set("mode", tt.opts.Mode)
			params.Set("compress", tt.opts.Compress)
			if tt.opts.Storage != "" {
				params.Set("storage", tt.opts.Storage)
			}
			if tt.opts.IncludeRAM && tt.opts.Type == "qemu" && tt.opts.Mode == "snapshot" {
				params.Set("vmstate", "1")
			}

			encoded := params.Encode()
			for key, val := range tt.wantParams {
				want := key + "=" + val
				if !strings.Contains(encoded, want) {
					t.Errorf("encoded = %q, missing %q", encoded, want)
				}
			}
			for _, key := range tt.wantAbsent {
				if strings.Contains(encoded, key+"=") {
					t.Errorf("encoded = %q, should not contain %q", encoded, key)
				}
			}
		})
	}
}

// ---- WaitForTask timeout test ----

func TestProxmoxClient_WaitForTask_ContextCancel(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Always return "running" to force the loop to continue
		writeJSON(w, map[string]interface{}{
			"data": map[string]interface{}{
				"status":    "running",
				"starttime": float64(time.Now().Unix()),
			},
		})
	}))
	defer ts.Close()

	client := newTestClientHTTP(ts)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := client.WaitForTask(ctx, "UPID:test", 1*time.Minute)
	if err == nil {
		t.Fatal("expected error for context cancellation, got nil")
	}
}

// ---- CleanupTempFiles tests ----

func TestCleanupTempFiles(t *testing.T) {
	service := NewProxmoxBackupService(newTestLogger())

	// Create temp files
	dir := t.TempDir()
	f1 := filepath.Join(dir, "backup1.vma")
	f2 := filepath.Join(dir, "backup2.vma")
	os.WriteFile(f1, []byte("data1"), 0644)
	os.WriteFile(f2, []byte("data2"), 0644)

	service.CleanupTempFiles([]string{f1, f2})

	for _, f := range []string{f1, f2} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("file %s should have been removed", f)
		}
	}
}

func TestCleanupTempFiles_NonexistentPaths(t *testing.T) {
	service := NewProxmoxBackupService(newTestLogger())
	// Should not panic on non-existent paths
	service.CleanupTempFiles([]string{"/nonexistent/path1", "/nonexistent/path2"})
}

// ---- Helper: newTestClientHTTP creates a client that talks plain HTTP to the test server ----

func newTestClientHTTP(ts *httptest.Server) *ProxmoxClient {
	u, _ := url.Parse(ts.URL)
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())

	config := &ProxmoxConfig{
		Host:        host,
		Port:        port,
		Node:        "pve1",
		Username:    "root@pam",
		TokenID:     "test-token",
		TokenSecret: "secret-value",
		VerifySSL:   false,
	}

	client := &ProxmoxClient{
		config:     config,
		httpClient: ts.Client(),
		logger:     newTestLogger(),
	}

	// Override baseURL to use http:// instead of https://
	origBaseURL := client.baseURL
	_ = origBaseURL
	// We need to monkey-patch the baseURL method. Since Go doesn't support that
	// directly, we'll wrap the HTTP client with a transport that rewrites URLs.
	client.httpClient.Transport = &rewriteTransport{
		base:    ts.Client().Transport,
		testURL: ts.URL,
	}

	return client
}

// rewriteTransport intercepts requests and rewrites the URL to point at the test server.
type rewriteTransport struct {
	base    http.RoundTripper
	testURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	testURL, _ := url.Parse(t.testURL)

	// Rewrite the request URL to point at our test server
	req.URL.Scheme = testURL.Scheme
	req.URL.Host = testURL.Host

	transport := t.base
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(req)
}
