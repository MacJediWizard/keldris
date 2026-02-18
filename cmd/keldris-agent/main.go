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
	"runtime"
	"strings"
	"syscall"
	"time"

	agentclient "github.com/MacJediWizard/keldris/internal/agent"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/health"
	"github.com/MacJediWizard/keldris/internal/updater"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
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

			fmt.Printf("Server URL:        %s\n", cfg.ServerURL)
			fmt.Printf("API Key:           %s\n", maskAPIKey(cfg.APIKey))
			if cfg.AgentID != "" {
				fmt.Printf("Agent ID:          %s\n", cfg.AgentID)
			}
			if cfg.Hostname != "" {
				fmt.Printf("Hostname:          %s\n", cfg.Hostname)
			}
			fmt.Printf("Auto-check update: %v\n", cfg.AutoCheckUpdate)

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
