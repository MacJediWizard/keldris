// Package main is the entrypoint for the Keldris agent CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "keldris",
		Short: "Keldris backup agent - Keeper of your data",
	}

	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "version",
			Short: "Print version",
			Run:   func(cmd *cobra.Command, args []string) { fmt.Printf("Keldris Agent %s\n", Version) },
		},
		&cobra.Command{
			Use:   "register",
			Short: "Register with server",
			Run:   func(cmd *cobra.Command, args []string) { fmt.Println("TODO: implement register") },
		},
		&cobra.Command{
			Use:   "backup",
			Short: "Run backup",
			Run:   func(cmd *cobra.Command, args []string) { fmt.Println("TODO: implement backup") },
		},
		&cobra.Command{
			Use:   "restore",
			Short: "Restore from backup",
			Run:   func(cmd *cobra.Command, args []string) { fmt.Println("TODO: implement restore") },
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show status",
			Run:   func(cmd *cobra.Command, args []string) { fmt.Println("TODO: implement status") },
		},
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
