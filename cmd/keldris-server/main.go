// Package main is the entrypoint for the Keldris server.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Version = "dev"

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Str("version", Version).Msg("Starting Keldris server")

	// TODO: Load configuration
	// TODO: Initialize database
	// TODO: Setup OIDC provider
	// TODO: Initialize Gin router
	// TODO: Start HTTP server

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Keldris server - implement me!")
}
