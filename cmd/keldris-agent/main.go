// Package main is the entrypoint for the Keldris agent CLI.
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
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

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "keldris-agent",
		Short: "Keldris backup agent - Keeper of your data",
		Long: `Keldris Agent is a backup agent that connects to a Keldris server
to perform automated backups using Restic.

Run 'keldris-agent register' to connect to a server.`,
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		newVersionCmd(),
		newRegisterCmd(),
		newConfigCmd(),
		newStatusCmd(),
		newBackupCmd(),
		newRestoreCmd(),
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

// maskAPIKey returns a masked version of the API key for display.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
