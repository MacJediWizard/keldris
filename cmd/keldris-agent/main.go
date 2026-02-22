// Package main is the entrypoint for the Keldris agent CLI.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	agentclient "github.com/MacJediWizard/keldris/internal/agent"
	"github.com/MacJediWizard/keldris/internal/agent"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/health"
	"github.com/MacJediWizard/keldris/internal/diagnostics"
	"github.com/MacJediWizard/keldris/internal/httpclient"
	"github.com/MacJediWizard/keldris/internal/diagnostics"
	"github.com/MacJediWizard/keldris/internal/support"
	"github.com/MacJediWizard/keldris/internal/updater"
	"github.com/robfig/cron/v3"
	"github.com/MacJediWizard/keldris/internal/support"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"runtime"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/rs/zerolog"
	"github.com/MacJediWizard/keldris/internal/updater"
	"github.com/spf13/cobra"
)

// Build-time variables set via ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// checkUpdateOnStartup checks for updates if auto-check is enabled in config.
func checkUpdateOnStartup() {
	cfg, err := config.LoadDefault()
	if err != nil || !cfg.AutoCheckUpdate {
		return
	}

	u := updater.NewWithProxy(Version, cfg.GetProxyConfig())
	u := updater.New(Version)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		// Silently ignore errors - this is a background check
		return
	}

	fmt.Printf("\n[Update available] Version %s is available (you have %s)\n", info.LatestVersion, Version)
	fmt.Println("Run 'keldris-agent update' to install it.")
	fmt.Println()
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "keldris-agent",
		Short: "Keldris backup agent - Keeper of your data",
		Long: `Keldris Agent is a backup agent that connects to a Keldris server
to perform automated backups using Restic.

Run 'keldris-agent register' to connect to a server.`,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Skip auto-check for certain commands
			if cmd.Name() == "update" || cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "diagnostics" {
			if cmd.Name() == "update" || cmd.Name() == "version" || cmd.Name() == "help" {
				return
			}
			checkUpdateOnStartup()
		},
	}

	rootCmd.AddCommand(
		newVersionCmd(),
		newRegisterCmd(),
		newConfigCmd(),
		newStatusCmd(),
		newBackupCmd(),
		newRestoreCmd(),
		newUpdateCmd(),
		newMountsCmd(),
		newStartCmd(),
		newSnapshotMountCmd(),
		newSupportBundleCmd(),
		newDiagnosticsCmd(),
		newQueueCmd(),
	)

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Keldris Agent %s\n", Version)
			fmt.Printf("  Commit:     %s\n", Commit)
			fmt.Printf("  Built:      %s\n", BuildDate)
			fmt.Printf("  Go version: %s\n", runtime.Version())
			fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}

func newRegisterCmd() *cobra.Command {
	var serverURL string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register this agent with a Keldris server",
		Long: `Register this agent with a Keldris server.

You will be prompted for an API key. To generate an API key,
log into your Keldris server and navigate to Settings > API Keys.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegister(serverURL)
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", "", "Keldris server URL (required)")
	_ = cmd.MarkFlagRequired("server")

	return cmd
}

func runRegister(serverURL string) error {
	// Validate URL
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("server URL must use http or https scheme")
	}

	// Prompt for API key
	fmt.Print("Enter API key: ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Load existing config or create new
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Update config
	cfg.ServerURL = strings.TrimSuffix(serverURL, "/")
	cfg.APIKey = apiKey

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil && cfg.Hostname == "" {
		cfg.Hostname = hostname
	}

	// Save config
	if err := cfg.SaveDefault(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	configPath, _ := config.DefaultConfigPath()
	fmt.Printf("Configuration saved to %s\n", configPath)
	fmt.Printf("Server: %s\n", cfg.ServerURL)
	fmt.Println("Registration complete. Run 'keldris-agent status' to verify connection.")

	return nil
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage agent configuration",
	}

	cmd.AddCommand(
		newConfigShowCmd(),
		newConfigSetServerCmd(),
		newConfigSetAutoUpdateCmd(),
		newConfigSetProxyCmd(),
		newConfigClearProxyCmd(),
		newConfigTestProxyCmd(),
		newConfigSetMaxQueueSizeCmd(),
	)

	return cmd
}

func newConfigSetProxyCmd() *cobra.Command {
	var httpProxy, httpsProxy, noProxy, socks5Proxy string

	cmd := &cobra.Command{
		Use:   "set-proxy",
		Short: "Configure proxy settings",
		Long: `Configure proxy settings for agent network connections.

Examples:
  # Set HTTP proxy
  keldris-agent config set-proxy --http http://proxy:8080

  # Set HTTPS proxy (often the same as HTTP proxy)
  keldris-agent config set-proxy --https http://proxy:8080

  # Set SOCKS5 proxy
  keldris-agent config set-proxy --socks5 socks5://user:pass@proxy:1080

  # Set hosts to bypass
  keldris-agent config set-proxy --no-proxy "localhost,127.0.0.1,.internal.com"

  # Set all at once
  keldris-agent config set-proxy --http http://proxy:8080 --https http://proxy:8080 --no-proxy localhost`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if httpProxy == "" && httpsProxy == "" && socks5Proxy == "" && noProxy == "" {
				return fmt.Errorf("at least one proxy option is required")
			}

			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Initialize proxy config if nil
			if cfg.Proxy == nil {
				cfg.Proxy = &config.ProxyConfig{}
			}

			// Update only provided values
			if httpProxy != "" {
				cfg.Proxy.HTTPProxy = httpProxy
			}
			if httpsProxy != "" {
				cfg.Proxy.HTTPSProxy = httpsProxy
			}
			if noProxy != "" {
				cfg.Proxy.NoProxy = noProxy
			}
			if socks5Proxy != "" {
				cfg.Proxy.SOCKS5Proxy = socks5Proxy
			}

			if err := cfg.SaveDefault(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Println("Proxy configuration updated:")
			fmt.Printf("  %s\n", httpclient.ProxyInfo(cfg.Proxy))
			return nil
		},
	}

	cmd.Flags().StringVar(&httpProxy, "http", "", "HTTP proxy URL (e.g., http://proxy:8080)")
	cmd.Flags().StringVar(&httpsProxy, "https", "", "HTTPS proxy URL (e.g., http://proxy:8080)")
	cmd.Flags().StringVar(&noProxy, "no-proxy", "", "Comma-separated hosts to bypass proxy")
	cmd.Flags().StringVar(&socks5Proxy, "socks5", "", "SOCKS5 proxy URL (e.g., socks5://user:pass@proxy:1080)")

	return cmd
}

func newConfigClearProxyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear-proxy",
		Short: "Remove all proxy settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			cfg.Proxy = nil

			if err := cfg.SaveDefault(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Println("Proxy configuration cleared.")
			return nil
		},
	}
}

func newConfigTestProxyCmd() *cobra.Command {
	var testURL string

	cmd := &cobra.Command{
		Use:   "test-proxy",
		Short: "Test proxy configuration",
		Long: `Test the proxy configuration by making a request to a test URL.

By default, tests connectivity to https://www.google.com. Use --url to specify
a different test endpoint.

Examples:
  # Test with default URL
  keldris-agent config test-proxy

  # Test with custom URL
  keldris-agent config test-proxy --url https://api.github.com

  # Test against the configured server
  keldris-agent config test-proxy --url $(keldris-agent config show | grep "Server URL" | cut -d: -f2-)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			proxyConfig := cfg.GetProxyConfig()
			if proxyConfig == nil || !proxyConfig.HasProxy() {
				fmt.Println("No proxy configured.")
				fmt.Println("Use 'keldris-agent config set-proxy' to configure a proxy.")
				return nil
			}

			fmt.Printf("Proxy: %s\n", httpclient.ProxyInfo(proxyConfig))
			fmt.Printf("Testing connection to: %s\n", testURL)
			fmt.Print("Connecting... ")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := httpclient.TestProxy(ctx, proxyConfig, testURL); err != nil {
				fmt.Println("FAILED")
				return fmt.Errorf("proxy test failed: %w", err)
			}

			fmt.Println("OK")
			fmt.Println("Proxy connection successful!")
			return nil
		},
	}

	cmd.Flags().StringVar(&testURL, "url", "https://www.google.com", "URL to test connectivity")

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			configPath, _ := config.DefaultConfigPath()
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Println()

			if !cfg.IsConfigured() {
				fmt.Println("Agent is not configured. Run 'keldris-agent register' to set up.")
				return nil
			}

			fmt.Printf("Server URL:        %s\n", cfg.ServerURL)
			fmt.Printf("API Key:           %s\n", maskAPIKey(cfg.APIKey))
			if cfg.AgentID != "" {
				fmt.Printf("Agent ID:          %s\n", cfg.AgentID)
			}
			if cfg.Hostname != "" {
				fmt.Printf("Hostname:          %s\n", cfg.Hostname)
			}
			fmt.Printf("Auto-check update: %v\n", cfg.AutoCheckUpdate)
			fmt.Printf("Proxy:             %s\n", httpclient.ProxyInfo(cfg.GetProxyConfig()))
			fmt.Printf("Max queue size:    %d\n", cfg.GetMaxQueueSize())

			return nil
		},
	}
}

func newConfigSetMaxQueueSizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-max-queue-size <size>",
		Short: "Set the maximum offline backup queue size",
		Long: `Set the maximum number of backups that can be queued locally when offline.

When this limit is reached, new scheduled backups will be skipped until
connectivity is restored and the queue is synced.

Default: 100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var size int
			if _, err := fmt.Sscanf(args[0], "%d", &size); err != nil {
				return fmt.Errorf("invalid size: must be a number")
			}
			if size < 1 || size > 10000 {
				return fmt.Errorf("size must be between 1 and 10000")
			}

			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			cfg.MaxQueueSize = size

			if err := cfg.SaveDefault(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Auto-check update: %v\n", cfg.AutoCheckUpdate)

			fmt.Printf("Max queue size set to: %d\n", size)
			return nil
		},
	}
}

func newConfigSetServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-server <url>",
		Short: "Set the server URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverURL := args[0]

			// Validate URL
			parsed, err := url.Parse(serverURL)
			if err != nil {
				return fmt.Errorf("invalid server URL: %w", err)
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				return fmt.Errorf("server URL must use http or https scheme")
			}

			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			cfg.ServerURL = strings.TrimSuffix(serverURL, "/")

			if err := cfg.SaveDefault(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Printf("Server URL set to: %s\n", cfg.ServerURL)
			return nil
		},
	}
}

func newConfigSetAutoUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-auto-update <true|false>",
		Short: "Enable or disable automatic update checks on startup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value := strings.ToLower(args[0])
			var enabled bool
			switch value {
			case "true", "1", "yes", "on":
				enabled = true
			case "false", "0", "no", "off":
				enabled = false
			default:
				return fmt.Errorf("invalid value: use true or false")
			}

			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			cfg.AutoCheckUpdate = enabled

			if err := cfg.SaveDefault(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			if enabled {
				fmt.Println("Auto-check update: enabled")
			} else {
				fmt.Println("Auto-check update: disabled")
			}
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show agent status and server connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if !cfg.IsConfigured() {
				fmt.Println("Status: Not configured")
				fmt.Println("Run 'keldris-agent register' to connect to a server.")
				return nil
			}

			fmt.Printf("Server:   %s\n", cfg.ServerURL)
			fmt.Printf("Hostname: %s\n", cfg.Hostname)
			if cfg.Proxy != nil && cfg.Proxy.HasProxy() {
				fmt.Printf("Proxy:    %s\n", httpclient.ProxyInfo(cfg.Proxy))
			}
			fmt.Println()

			// Ping the server
			fmt.Print("Checking server connection... ")

			client, err := httpclient.NewWithConfig(cfg, 10*time.Second)
			if err != nil {
				fmt.Println("FAILED")
				return fmt.Errorf("create http client: %w", err)
			}
			healthURL := cfg.ServerURL + "/health"

			resp, err := client.Get(healthURL)
			if err != nil {
				fmt.Println("FAILED")
				return fmt.Errorf("connect to server: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Println("OK")
				fmt.Println("Connection: Online")
			} else {
				fmt.Printf("FAILED (HTTP %d)\n", resp.StatusCode)
				fmt.Println("Connection: Error")
				return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
			}

			return nil
		},
	}
}

func newBackupCmd() *cobra.Command {
	var now bool
	var scheduleName string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Run a backup operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("agent not configured: %w", err)
			}

			if !now {
				return cmd.Help()
			}

			return runBackup(cfg, scheduleName)
		},
	}

	cmd.Flags().BoolVar(&now, "now", false, "Run backup immediately")
	cmd.Flags().StringVar(&scheduleName, "schedule", "", "Schedule name to run (default: first available)")

	return cmd
}

func runBackup(cfg *config.AgentConfig, scheduleName string) error {
	client := agentclient.NewClient(cfg.ServerURL, cfg.APIKey)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	fmt.Println("Fetching backup schedules from server...")
	schedules, err := client.GetSchedules()
	if err != nil {
		return fmt.Errorf("fetch schedules: %w", err)
	}

	if len(schedules) == 0 {
		fmt.Println("No backup schedules configured for this agent.")
		return nil
	}

	// Find the schedule to run
	var sched *agentclient.ScheduleConfig
	if scheduleName != "" {
		for i := range schedules {
			if schedules[i].Name == scheduleName {
				sched = &schedules[i]
				break
			}
		}
		if sched == nil {
			fmt.Printf("Schedule %q not found. Available schedules:\n", scheduleName)
			for _, s := range schedules {
				fmt.Printf("  - %s\n", s.Name)
			}
			return fmt.Errorf("schedule %q not found", scheduleName)
		}
	} else {
		sched = &schedules[0]
	}

	fmt.Printf("Schedule: %s\n", sched.Name)
	fmt.Printf("Paths:    %s\n", strings.Join(sched.Paths, ", "))
	fmt.Printf("Repo:     %s\n", sched.Repository)
	fmt.Println()

	// Build restic config
	resticCfg := backends.ResticConfig{
		Repository: sched.Repository,
		Password:   sched.RepositoryPassword,
		Env:        sched.RepositoryEnv,
	}

	// Run the backup
	restic := backup.NewRestic(logger)
	hostname, _ := os.Hostname()
	tags := []string{
		"agent:" + cfg.AgentID,
		"schedule:" + sched.ID.String(),
		"host:" + hostname,
	}

	fmt.Println("Starting Restic backup...")
	startedAt := time.Now()

	stats, err := restic.Backup(context.Background(), resticCfg, sched.Paths, sched.Excludes, tags)

	completedAt := time.Now()

	// Report result to server
	report := &agentclient.BackupReport{
		ScheduleID:   sched.ID,
		RepositoryID: sched.RepositoryID,
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
	}

	if err != nil {
		fmt.Printf("Backup failed: %v\n", err)
		report.Status = "failed"
		errMsg := err.Error()
		report.ErrorMessage = &errMsg
	} else {
		fmt.Printf("Backup completed successfully!\n")
		fmt.Printf("  Snapshot:     %s\n", stats.SnapshotID)
		fmt.Printf("  Files new:    %d\n", stats.FilesNew)
		fmt.Printf("  Files changed: %d\n", stats.FilesChanged)
		fmt.Printf("  Size added:   %d bytes\n", stats.SizeBytes)
		fmt.Printf("  Duration:     %s\n", stats.Duration.Round(time.Second))

		report.Status = "completed"
		report.SnapshotID = stats.SnapshotID
		report.SizeBytes = &stats.SizeBytes
		report.FilesNew = &stats.FilesNew
		report.FilesChanged = &stats.FilesChanged
	}

	fmt.Print("Reporting to server... ")
	if reportErr := client.ReportBackup(report); reportErr != nil {
		fmt.Printf("failed: %v\n", reportErr)
	} else {
		fmt.Println("done")
	}

	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	return nil
}

func newRestoreCmd() *cobra.Command {
	var latest bool
	var snapshotID string
	var targetPath string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore from a backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("agent not configured: %w", err)
			}

			if !latest && snapshotID == "" {
				return cmd.Help()
			}

			return runRestore(cfg, latest, snapshotID, targetPath)
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Restore from the latest backup")
	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Restore from a specific snapshot ID")
	cmd.Flags().StringVar(&targetPath, "target", "/", "Target path for restore (default: /)")
		},
	}

	cmd.Flags().StringVar(&serverURL, "server", "", "Keldris server URL (required)")
	_ = cmd.MarkFlagRequired("server")

	return cmd
}

func runRegister(serverURL string) error {
	// Validate URL
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("server URL must use http or https scheme")
	}

	// Prompt for API key
	fmt.Print("Enter API key: ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Load existing config or create new
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Update config
	cfg.ServerURL = strings.TrimSuffix(serverURL, "/")
	cfg.APIKey = apiKey

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil && cfg.Hostname == "" {
		cfg.Hostname = hostname
	}

	// Save config
	if err := cfg.SaveDefault(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	configPath, _ := config.DefaultConfigPath()
	fmt.Printf("Configuration saved to %s\n", configPath)
	fmt.Printf("Server: %s\n", cfg.ServerURL)
	fmt.Println("Registration complete. Run 'keldris-agent status' to verify connection.")

	return nil
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage agent configuration",
	}

	cmd.AddCommand(
		newConfigShowCmd(),
		newConfigSetServerCmd(),
	)

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			configPath, _ := config.DefaultConfigPath()
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Println()

			if !cfg.IsConfigured() {
				fmt.Println("Agent is not configured. Run 'keldris-agent register' to set up.")
				return nil
			}

			fmt.Printf("Server URL: %s\n", cfg.ServerURL)
			fmt.Printf("API Key:    %s\n", maskAPIKey(cfg.APIKey))
			if cfg.AgentID != "" {
				fmt.Printf("Agent ID:   %s\n", cfg.AgentID)
			}
			if cfg.Hostname != "" {
				fmt.Printf("Hostname:   %s\n", cfg.Hostname)
			}

			return nil
		},
	}
}

func newConfigSetServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-server <url>",
		Short: "Set the server URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverURL := args[0]

			// Validate URL
			parsed, err := url.Parse(serverURL)
			if err != nil {
				return fmt.Errorf("invalid server URL: %w", err)
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				return fmt.Errorf("server URL must use http or https scheme")
			}

			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			cfg.ServerURL = strings.TrimSuffix(serverURL, "/")

			if err := cfg.SaveDefault(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Printf("Server URL set to: %s\n", cfg.ServerURL)
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show agent status and server connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if !cfg.IsConfigured() {
				fmt.Println("Status: Not configured")
				fmt.Println("Run 'keldris-agent register' to connect to a server.")
				return nil
			}

			fmt.Printf("Server:   %s\n", cfg.ServerURL)
			fmt.Printf("Hostname: %s\n", cfg.Hostname)
			fmt.Println()

			// Ping the server
			fmt.Print("Checking server connection... ")

			client := &http.Client{Timeout: 10 * time.Second}
			healthURL := cfg.ServerURL + "/health"

			resp, err := client.Get(healthURL)
			if err != nil {
				fmt.Println("FAILED")
				return fmt.Errorf("connect to server: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Println("OK")
				fmt.Println("Connection: Online")
			} else {
				fmt.Printf("FAILED (HTTP %d)\n", resp.StatusCode)
				fmt.Println("Connection: Error")
			}

			return nil
		},
	}
}

func newBackupCmd() *cobra.Command {
	var now bool

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Run a backup operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("agent not configured: %w", err)
			}

			if now {
				fmt.Println("Starting backup...")
				fmt.Printf("Server: %s\n", cfg.ServerURL)
				fmt.Println()
				// Placeholder - actual Restic integration will come later
				fmt.Println("[Placeholder] Backup would run here")
				fmt.Println("Backup functionality will be implemented with Restic integration.")
				return nil
			}

			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&now, "now", false, "Run backup immediately")

	return cmd
}

func newRestoreCmd() *cobra.Command {
	var latest bool
	var snapshotID string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore from a backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("agent not configured: %w", err)
			}

			if latest {
				fmt.Println("Restoring from latest backup...")
				fmt.Printf("Server: %s\n", cfg.ServerURL)
				fmt.Println()
				// Placeholder - actual Restic integration will come later
				fmt.Println("[Placeholder] Restore would run here")
				fmt.Println("Restore functionality will be implemented with Restic integration.")
				return nil
			}

			if snapshotID != "" {
				fmt.Printf("Restoring from snapshot %s...\n", snapshotID)
				fmt.Printf("Server: %s\n", cfg.ServerURL)
				fmt.Println()
				// Placeholder
				fmt.Println("[Placeholder] Restore would run here")
				fmt.Println("Restore functionality will be implemented with Restic integration.")
				return nil
			}

			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&latest, "latest", false, "Restore from the latest backup")
	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Restore from a specific snapshot ID")

	return cmd
}

func runRestore(cfg *config.AgentConfig, latest bool, snapshotID, targetPath string) error {
	client := agentclient.NewClient(cfg.ServerURL, cfg.APIKey)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// If --latest, look up the most recent snapshot
	if latest {
		fmt.Println("Looking up latest snapshot...")
		snapshots, err := client.GetSnapshots()
		if err != nil {
			return fmt.Errorf("fetch snapshots: %w", err)
		}
		if len(snapshots) == 0 {
			return fmt.Errorf("no snapshots found for this agent")
		}
		// Snapshots are ordered by created_at DESC from the server
		snapshotID = snapshots[0].SnapshotID
		fmt.Printf("Latest snapshot: %s\n", snapshotID)
	}

	if snapshotID == "" {
		return fmt.Errorf("no snapshot ID specified")
	}

	// Get schedule info to find repo credentials
	fmt.Println("Fetching repository credentials...")
	schedules, err := client.GetSchedules()
	if err != nil {
		return fmt.Errorf("fetch schedules: %w", err)
	}
	if len(schedules) == 0 {
		return fmt.Errorf("no schedules configured; cannot determine repository credentials")
	}

	// Use the first schedule's repo (the snapshot should be in this repo)
	sched := schedules[0]

	resticCfg := backends.ResticConfig{
		Repository: sched.Repository,
		Password:   sched.RepositoryPassword,
		Env:        sched.RepositoryEnv,
	}

	restic := backup.NewRestic(logger)
	opts := backup.RestoreOptions{
		TargetPath: targetPath,
	}

	fmt.Printf("Restoring snapshot %s to %s...\n", snapshotID, targetPath)

	if err := restic.Restore(context.Background(), resticCfg, snapshotID, opts); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Println("Restore completed successfully!")
	return nil
}

func newStartCmd() *cobra.Command {
	var heartbeatInterval time.Duration

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the agent daemon",
		Long: `Start the Keldris agent as a long-running daemon process.

The daemon will:
  - Send periodic heartbeats with system metrics to the server
  - Execute scheduled backups based on cron expressions
  - Report backup results to the server`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("agent not configured: %w", err)
			}

			return runDaemon(cfg, heartbeatInterval)
		},
	}

	cmd.Flags().DurationVar(&heartbeatInterval, "heartbeat-interval", 60*time.Second, "Heartbeat interval")

	return cmd
}

func runDaemon(cfg *config.AgentConfig, heartbeatInterval time.Duration) error {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	client := agentclient.NewClient(cfg.ServerURL, cfg.APIKey)
	collector := health.NewCollector(cfg.ServerURL, "restic")

	fmt.Printf("Keldris Agent %s starting...\n", Version)
	fmt.Printf("Server: %s\n", cfg.ServerURL)
	fmt.Printf("Heartbeat interval: %s\n", heartbeatInterval)

	// Verify restic is available before starting
	if _, err := exec.LookPath("restic"); err != nil {
		logger.Warn().Msg("restic binary not found in PATH; backups will fail until restic is installed")
		fmt.Println("WARNING: restic not found in PATH")
	} else {
		fmt.Println("Restic: available")
	}
	fmt.Println()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Send initial heartbeat
	sendHeartbeat(client, collector, &logger)

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Set up cron scheduler for backups
	cronScheduler := cron.New()

	// Fetch initial schedules and register them
	refreshSchedules(cronScheduler, client, cfg, &logger)

	cronScheduler.Start()
	defer cronScheduler.Stop()

	// Refresh schedules periodically (every 5 minutes)
	scheduleRefreshTicker := time.NewTicker(5 * time.Minute)
	defer scheduleRefreshTicker.Stop()

	fmt.Println("Agent daemon running. Press Ctrl+C to stop.")

	for {
		select {
		case <-heartbeatTicker.C:
			sendHeartbeat(client, collector, &logger)
		case <-scheduleRefreshTicker.C:
			refreshSchedules(cronScheduler, client, cfg, &logger)
		case sig := <-sigChan:
			fmt.Printf("\nReceived %s, shutting down...\n", sig)
			return nil
		}
	}
}

// refreshSchedules fetches schedules from the server and updates the cron scheduler.
func refreshSchedules(c *cron.Cron, client *agentclient.Client, cfg *config.AgentConfig, logger *zerolog.Logger) {
	schedules, err := client.GetSchedules()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to fetch schedules")
		return
	}

	// Remove all existing cron entries and re-register
	for _, entry := range c.Entries() {
		c.Remove(entry.ID)
	}

	for _, sched := range schedules {
		s := sched // capture loop variable
		_, err := c.AddFunc(s.CronExpression, func() {
			logger.Info().Str("schedule", s.Name).Msg("cron triggered backup")
			if err := runBackup(cfg, s.Name); err != nil {
				logger.Error().Err(err).Str("schedule", s.Name).Msg("scheduled backup failed")
			}
		})
		if err != nil {
			logger.Error().Err(err).Str("schedule", s.Name).Str("cron", s.CronExpression).Msg("invalid cron expression")
			continue
		}
		logger.Info().Str("schedule", s.Name).Str("cron", s.CronExpression).Msg("registered backup schedule")
	}

	logger.Info().Int("count", len(schedules)).Msg("schedules refreshed")
}

func sendHeartbeat(client *agentclient.Client, collector *health.Collector, logger *zerolog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Collect real system metrics
	metrics, err := collector.Collect(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to collect system metrics")
	}

	// Get OS info
	osInfo := health.GetOSInfo()

	req := &agentclient.HeartbeatRequest{
		Status: "healthy",
		OSInfo: &agentclient.OSInfo{
			OS:       osInfo["os"],
			Arch:     osInfo["arch"],
			Hostname: osInfo["hostname"],
			Version:  osInfo["version"],
		},
	}

	if metrics != nil {
		req.Metrics = &agentclient.HeartbeatMetrics{
			CPUUsage:        metrics.CPUUsage,
			MemoryUsage:     metrics.MemoryUsage,
			DiskUsage:       metrics.DiskUsage,
			DiskFreeBytes:   metrics.DiskFreeBytes,
			DiskTotalBytes:  metrics.DiskTotalBytes,
			NetworkUp:       metrics.NetworkUp,
			UptimeSeconds:   metrics.UptimeSeconds,
			ResticVersion:   metrics.ResticVersion,
			ResticAvailable: metrics.ResticAvailable,
		}
	}

	if err := client.SendHeartbeat(req); err != nil {
		logger.Warn().Err(err).Msg("heartbeat failed")
	} else {
		logger.Debug().
			Float64("cpu", req.Metrics.CPUUsage).
			Float64("mem", req.Metrics.MemoryUsage).
			Float64("disk", req.Metrics.DiskUsage).
			Msg("heartbeat sent")
	}
}

func newUpdateCmd() *cobra.Command {
	var checkOnly bool
	var force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install agent updates",
		Long: `Check for new versions of the Keldris agent and optionally install them.

By default, this command will check for updates and prompt before installing.
Use --check to only check without installing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(checkOnly, force)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for updates without installing")
	cmd.Flags().BoolVar(&force, "force", false, "Install update without confirmation")

	return cmd
}

func runUpdate(checkOnly, force bool) error {
	cfg, _ := config.LoadDefault()
	var proxyConfig *config.ProxyConfig
	if cfg != nil {
		proxyConfig = cfg.GetProxyConfig()
	}

	u := updater.NewWithProxy(Version, proxyConfig)
	u := updater.New(Version)

	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		if errors.Is(err, updater.ErrNoUpdateAvailable) {
			fmt.Println("You are running the latest version.")
			return nil
		}
		return fmt.Errorf("check for updates: %w", err)
	}

	fmt.Printf("\nNew version available: %s\n", info.LatestVersion)
	if info.ReleaseNotes != "" {
		fmt.Println("\nRelease notes:")
		fmt.Println(info.ReleaseNotes)
	}
	fmt.Println()

	if checkOnly {
		fmt.Println("Run 'keldris-agent update' to install this update.")
		return nil
	}

	if !force {
		fmt.Print("Install this update? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	fmt.Printf("Downloading %s...\n", info.AssetName)

	downloadCtx, downloadCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer downloadCancel()

	tmpPath, err := u.Download(downloadCtx, info, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rDownloading: %.1f%% (%d / %d bytes)", pct, downloaded, total)
		}
	})
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer os.Remove(tmpPath)

	fmt.Println("\n\nInstalling update...")

	if err := u.Apply(tmpPath); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}

	fmt.Printf("Successfully updated to %s!\n", info.LatestVersion)
	fmt.Println("Restarting agent...")

	if err := u.Restart(); err != nil {
		// If restart fails, just exit - the update was still successful
		fmt.Printf("Note: Could not restart automatically: %v\n", err)
		fmt.Println("Please restart the agent manually.")
	}

	return nil
}

func newMountsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mounts",
		Short: "List detected network mounts",
		Long: `Detects and displays all network mounts (NFS, SMB, CIFS) on this system.

Shows mount path, type, remote location, and current accessibility status.
Network mounts can be included in backup schedules, and the agent will
report mount availability to the server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			nd := backup.NewNetworkDrives(logger)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mounts, err := nd.DetectMounts(ctx)
			if err != nil {
				return fmt.Errorf("detect mounts: %w", err)
			}

			if len(mounts) == 0 {
				fmt.Println("No network mounts detected")
				return nil
			}

			fmt.Printf("%-40s %-8s %-40s %-12s\n", "PATH", "TYPE", "REMOTE", "STATUS")
			fmt.Println(strings.Repeat("-", 104))
			for _, m := range mounts {
				fmt.Printf("%-40s %-8s %-40s %-12s\n",
					m.Path, m.Type, m.Remote, m.Status)
			}

			return nil
		},
	}
}

// maskAPIKey returns a masked version of the API key for display.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func newSnapshotMountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot-mount",
		Short: "Mount and browse backup snapshots",
		Long: `Mount backup snapshots as FUSE filesystems for browsing.

This allows you to browse the contents of a backup snapshot as if it were
a regular directory, making it easy to find and restore individual files.

Note: FUSE support must be available on the system (macFUSE on macOS,
FUSE on Linux).`,
	}

	cmd.AddCommand(
		newSnapshotMountStartCmd(),
		newSnapshotMountStopCmd(),
		newSnapshotMountListCmd(),
	)

	return cmd
}

func newSupportBundleCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "support-bundle",
		Short: "Generate a support bundle for troubleshooting",
		Long: `Generate a diagnostic bundle containing sanitized logs, configuration,
and system information for troubleshooting.

The bundle automatically removes sensitive information like API keys,
passwords, and credentials. Review the contents before sharing if you
have concerns about sensitive data.

The generated zip file can be shared with support or attached to
GitHub issues to help diagnose problems.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSupportBundle(outputPath)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for the bundle (default: current directory)")

	return cmd
}

func newSnapshotMountStartCmd() *cobra.Command {
	var (
		snapshotID     string
		repositoryPath string
		password       string
		timeout        time.Duration
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Mount a snapshot for browsing",
		Long: `Mount a backup snapshot as a read-only FUSE filesystem.

The snapshot will be mounted at a path under /tmp/keldris-mounts/<uuid>.
After the timeout period, the mount will be automatically unmounted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if snapshotID == "" {
				return fmt.Errorf("snapshot ID is required (--snapshot)")
			}
			if repositoryPath == "" {
				return fmt.Errorf("repository path is required (--repo)")
			}
			if password == "" {
				// Prompt for password
				fmt.Print("Repository password: ")
				reader := bufio.NewReader(os.Stdin)
				pw, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read password: %w", err)
				}
				password = strings.TrimSpace(pw)
			}

			// Create mount manager
			mountBasePath := filepath.Join(os.TempDir(), "keldris-mounts")
			if err := os.MkdirAll(mountBasePath, 0755); err != nil {
				return fmt.Errorf("create mount base directory: %w", err)
			}

			logger := zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()
			manager := backup.NewMountManager(mountBasePath, logger)

			// Configure restic
			cfg := backends.ResticConfig{
				Repository: repositoryPath,
				Password:   password,
			}

			mountID := uuid.New()
			ctx := context.Background()

			fmt.Printf("Mounting snapshot %s...\n", snapshotID)

			info, err := manager.Mount(ctx, mountID, cfg, snapshotID, timeout)
			if err != nil {
				return fmt.Errorf("mount snapshot: %w", err)
			}

			fmt.Printf("\nSnapshot mounted successfully!\n")
			fmt.Printf("Mount ID:   %s\n", info.ID)
			fmt.Printf("Mount Path: %s\n", info.MountPath)
			fmt.Printf("Expires:    %s\n", info.ExpiresAt.Format(time.RFC3339))
			fmt.Println()
			fmt.Printf("Browse your files at: %s\n", info.MountPath)
			fmt.Println()
			fmt.Println("Press Ctrl+C to unmount, or run 'keldris-agent snapshot-mount stop --id <mount-id>'")

			// Wait for interrupt signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\nUnmounting...")
			if err := manager.Unmount(ctx, info.ID); err != nil {
				return fmt.Errorf("unmount: %w", err)
			}
			fmt.Println("Unmounted successfully.")

			return nil
		},
	}

	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Snapshot ID to mount (required)")
	cmd.Flags().StringVar(&repositoryPath, "repo", "", "Repository path (required)")
	cmd.Flags().StringVar(&password, "password", "", "Repository password (will prompt if not provided)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "Auto-unmount timeout (default 30m)")

	return cmd
}

func newSnapshotMountStopCmd() *cobra.Command {
	var mountID string

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Unmount a mounted snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if mountID == "" {
				return fmt.Errorf("mount ID is required (--id)")
			}

			id, err := uuid.Parse(mountID)
			if err != nil {
				return fmt.Errorf("invalid mount ID: %w", err)
			}

			mountBasePath := filepath.Join(os.TempDir(), "keldris-mounts")
			logger := zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()
			manager := backup.NewMountManager(mountBasePath, logger)

			ctx := context.Background()
			if err := manager.Unmount(ctx, id); err != nil {
				return fmt.Errorf("unmount: %w", err)
			}

			fmt.Println("Unmounted successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&mountID, "id", "", "Mount ID to unmount (required)")

	return cmd
}

func newSnapshotMountListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active snapshot mounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			mountBasePath := filepath.Join(os.TempDir(), "keldris-mounts")
			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			manager := backup.NewMountManager(mountBasePath, logger)

			mounts := manager.ListMounts()

			if len(mounts) == 0 {
				fmt.Println("No active snapshot mounts")
				return nil
			}

			fmt.Printf("%-36s %-20s %-50s %-25s\n", "MOUNT ID", "SNAPSHOT", "PATH", "EXPIRES")
			fmt.Println(strings.Repeat("-", 135))
			for _, m := range mounts {
				fmt.Printf("%-36s %-20s %-50s %-25s\n",
					m.ID.String(),
					m.SnapshotID,
					m.MountPath,
					m.ExpiresAt.Format(time.RFC3339))
			}

			return nil
		},
	}
}

func newDiagnosticsCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Run self-test diagnostics",
		Long: `Run diagnostic checks to verify the agent is properly configured and operational.

Checks performed:
  - Server connectivity: Tests connection to the Keldris server
  - API key validation: Verifies the API key is valid
  - Restic binary: Checks that restic is installed and working
  - Disk space: Verifies adequate free space
  - Config permissions: Checks file permissions on config directory

Use --json for machine-readable output suitable for automation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiagnostics(jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")

	return cmd
}

func newQueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Manage offline backup queue",
		Long: `Manage backups that were queued while the server was unreachable.

When the agent cannot reach the server, scheduled backups are queued locally.
Once connectivity is restored, queued backups are automatically synced to
the server.`,
	}

	cmd.AddCommand(
		newQueueStatusCmd(),
		newQueueListCmd(),
		newQueueSyncCmd(),
		newQueueClearCmd(),
	)

	return cmd
}

func runDiagnostics(jsonOutput bool) error {
	cfg, err := config.LoadDefault()
	if err != nil {
		if jsonOutput {
			fmt.Printf(`{"error": "failed to load config: %s"}`, err.Error())
			fmt.Println()
		} else {
			fmt.Printf("Warning: Could not load config: %v\n", err)
		}
		cfg = &config.AgentConfig{}
	}

	runner := diagnostics.NewRunner(cfg, Version)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	if jsonOutput {
		data, err := result.ToJSON()
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		if !result.Summary.AllPass {
			return fmt.Errorf("diagnostics failed: %d check(s) failed", result.Summary.Failed)
		}
		return nil
	}

	// Human-readable output
	fmt.Println("Keldris Agent Diagnostics")
	fmt.Println("========================")
	fmt.Printf("Version:   %s\n", result.AgentVersion)
	fmt.Printf("Hostname:  %s\n", result.Hostname)
	fmt.Printf("OS/Arch:   %s/%s\n", result.OS, result.Arch)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Println()

	// Print each check result
	for _, check := range result.Checks {
		var statusIcon string
		switch check.Status {
		case diagnostics.StatusPass:
			statusIcon = "✓"
		case diagnostics.StatusFail:
			statusIcon = "✗"
		case diagnostics.StatusWarn:
			statusIcon = "!"
		case diagnostics.StatusSkip:
			statusIcon = "-"
		}

		fmt.Printf("[%s] %s\n", statusIcon, formatCheckName(check.Name))
		if check.Message != "" {
			fmt.Printf("    %s\n", check.Message)
		}
	}

	fmt.Println()
	fmt.Println("Summary")
	fmt.Println("-------")
	fmt.Printf("Total:   %d\n", result.Summary.Total)
	fmt.Printf("Passed:  %d\n", result.Summary.Passed)
	fmt.Printf("Failed:  %d\n", result.Summary.Failed)
	fmt.Printf("Warned:  %d\n", result.Summary.Warned)
	fmt.Printf("Skipped: %d\n", result.Summary.Skipped)
	fmt.Println()

	if result.Summary.AllPass {
		fmt.Println("All checks passed!")
		return nil
	}

	return fmt.Errorf("diagnostics failed: %d check(s) failed", result.Summary.Failed)
}

// formatCheckName converts a check name from snake_case to Title Case.
func formatCheckName(name string) string {
	words := strings.Split(name, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func newQueueStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show offline backup queue status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			configDir, err := config.DefaultConfigDir()
			if err != nil {
				return fmt.Errorf("get config dir: %w", err)
			}

			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			store, err := agent.NewSQLiteStore(configDir, logger)
			if err != nil {
				return fmt.Errorf("open queue database: %w", err)
			}
			defer store.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			status, err := store.GetQueueStatus(ctx)
			if err != nil {
				return fmt.Errorf("get queue status: %w", err)
			}

			maxQueueSize := cfg.GetMaxQueueSize()

			fmt.Println("Offline Backup Queue Status")
			fmt.Println(strings.Repeat("-", 40))
			fmt.Printf("Pending:        %d\n", status.PendingCount)
			fmt.Printf("Synced:         %d\n", status.SyncedCount)
			fmt.Printf("Failed:         %d\n", status.FailedCount)
			fmt.Printf("Total entries:  %d\n", status.TotalQueued)
			fmt.Printf("Max queue size: %d\n", maxQueueSize)
			fmt.Println()

			if status.OldestQueuedAt != nil {
				fmt.Printf("Oldest pending: %s\n", status.OldestQueuedAt.Format(time.RFC3339))
			}

			// Check server connectivity
			if cfg.IsConfigured() {
				fmt.Print("\nServer status:  ")
				client := &http.Client{Timeout: 5 * time.Second}
				resp, checkErr := client.Get(cfg.ServerURL + "/health")
				if checkErr != nil {
					fmt.Println("OFFLINE")
				} else {
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						fmt.Println("ONLINE")
					} else {
						fmt.Printf("ERROR (HTTP %d)\n", resp.StatusCode)
					}
				}
			}

			return nil
		},
	}
}

func newQueueListCmd() *cobra.Command {
	var showAll bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List queued backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir, err := config.DefaultConfigDir()
			if err != nil {
				return fmt.Errorf("get config dir: %w", err)
			}

			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			store, err := agent.NewSQLiteStore(configDir, logger)
			if err != nil {
				return fmt.Errorf("open queue database: %w", err)
			}
			defer store.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var backups []*agent.QueuedBackup
			if showAll {
				backups, err = store.ListAllBackups(ctx)
			} else {
				backups, err = store.ListPendingBackups(ctx)
			}
			if err != nil {
				return fmt.Errorf("list backups: %w", err)
			}

			if len(backups) == 0 {
				if showAll {
					fmt.Println("No backups in queue")
				} else {
					fmt.Println("No pending backups in queue")
				}
				return nil
			}

			fmt.Printf("%-36s %-20s %-10s %-8s %-25s\n", "ID", "SCHEDULE", "STATUS", "RETRIES", "QUEUED AT")
			fmt.Println(strings.Repeat("-", 105))
			for _, b := range backups {
				name := b.ScheduleName
				if len(name) > 18 {
					name = name[:17] + "..."
				}
				fmt.Printf("%-36s %-20s %-10s %-8d %-25s\n",
					b.ID.String(),
					name,
					b.Status,
					b.RetryCount,
					b.QueuedAt.Format(time.RFC3339))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all backups (including synced/failed)")

	return cmd
}

func newQueueSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Force sync queued backups to server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if !cfg.IsConfigured() {
				return fmt.Errorf("agent not configured, run 'keldris-agent register' first")
			}

			configDir, err := config.DefaultConfigDir()
			if err != nil {
				return fmt.Errorf("get config dir: %w", err)
			}

			logger := zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()

			store, err := agent.NewSQLiteStore(configDir, logger)
			if err != nil {
				return fmt.Errorf("open queue database: %w", err)
			}
			defer store.Close()

			client := agent.NewHTTPServerClient(cfg.ServerURL, cfg.APIKey, logger)
			queueCfg := agent.DefaultQueueConfig()
			queue := agent.NewQueue(store, client, queueCfg, logger)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			fmt.Println("Syncing queued backups...")
			if err := queue.SyncNow(ctx); err != nil {
				if errors.Is(err, agent.ErrServerUnreachable) {
					fmt.Println("Server is unreachable. Backups remain queued.")
					return nil
				}
				return fmt.Errorf("sync: %w", err)
			}

			fmt.Println("Sync completed successfully.")
			return nil
		},
	}
}

func newQueueClearCmd() *cobra.Command {
	var force bool
	var status string

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear queued backups",
		Long: `Clear backups from the offline queue.

By default, only synced entries are cleared. Use --status to clear
entries with a specific status (pending, synced, failed), or --force
to clear all entries regardless of status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir, err := config.DefaultConfigDir()
			if err != nil {
				return fmt.Errorf("get config dir: %w", err)
			}

			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			store, err := agent.NewSQLiteStore(configDir, logger)
			if err != nil {
				return fmt.Errorf("open queue database: %w", err)
			}
			defer store.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if force {
				// Clear all entries
				backups, err := store.ListAllBackups(ctx)
				if err != nil {
					return fmt.Errorf("list backups: %w", err)
				}

				for _, b := range backups {
					if err := store.DeleteQueuedBackup(ctx, b.ID); err != nil {
						fmt.Printf("Failed to delete %s: %v\n", b.ID, err)
					}
				}
				fmt.Printf("Cleared %d queue entries\n", len(backups))
				return nil
			}

			// Clear by status
			if status == "" {
				status = "synced"
			}

			// Prune old synced/failed entries
			pruned, err := store.PruneOldEntries(ctx, 0)
			if err != nil {
				return fmt.Errorf("prune entries: %w", err)
			}

			fmt.Printf("Cleared %d queue entries\n", pruned)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Clear all entries regardless of status")
	cmd.Flags().StringVar(&status, "status", "synced", "Status of entries to clear (pending, synced, failed)")

	return cmd
}

func runSupportBundle(outputPath string) error {
	fmt.Println("Generating support bundle...")

	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Printf("Warning: Could not load config: %v\n", err)
	}

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)

	opts := support.DefaultBundleOptions()
	opts.IncludeAgentInfo = true
	opts.IncludeServerInfo = false

	// Try to find log directory
	configDir, _ := config.DefaultConfigDir()
	if configDir != "" {
		logDir := filepath.Join(configDir, "logs")
		if _, err := os.Stat(logDir); err == nil {
			opts.LogDir = logDir
		}
	}

	generator := support.NewGenerator(logger, opts)

	bundleData := support.BundleData{
		AgentInfo: &support.AgentInfo{
			Version:   Version,
			Commit:    Commit,
			BuildDate: BuildDate,
			AgentID:   cfg.AgentID,
			Hostname:  cfg.Hostname,
			ServerURL: cfg.ServerURL,
		},
		Config: &support.ConfigInfo{
			ServerURL:       cfg.ServerURL,
			AgentID:         cfg.AgentID,
			Hostname:        cfg.Hostname,
			AutoCheckUpdate: cfg.AutoCheckUpdate,
		},
		CustomData: make(map[string]any),
	}

	// Add system info to custom data
	bundleData.CustomData["runtime"] = map[string]any{
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"num_cpu":       runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, info, err := generator.Generate(ctx, bundleData)
	if err != nil {
		return fmt.Errorf("generate bundle: %w", err)
	}

	// Determine output path
	if outputPath == "" {
		outputPath = info.Filename
	} else {
		// If output is a directory, use default filename inside it
		stat, err := os.Stat(outputPath)
		if err == nil && stat.IsDir() {
			outputPath = filepath.Join(outputPath, info.Filename)
		}
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}

	fmt.Printf("\nSupport bundle generated: %s\n", outputPath)
	fmt.Printf("Size: %d bytes\n", info.Size)
	fmt.Println()
	fmt.Println("This bundle contains sanitized diagnostic information.")
	fmt.Println("To submit for support:")
	fmt.Println("  1. Email to support@keldris.io")
	fmt.Println("  2. Attach to a GitHub issue: https://github.com/MacJediWizard/keldris/issues")
	fmt.Println()
	fmt.Println("Please review the contents before sharing if you have concerns")
	fmt.Println("about sensitive data.")

	return nil
// maskAPIKey returns a masked version of the API key for display.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
