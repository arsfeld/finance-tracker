package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"finance_tracker/src/internal/config"
	"finance_tracker/src/web"
)

// WebConfig holds configuration for the web server
type WebConfig struct {
	Port        string
	EnvFile     string
	Environment string
}

// webCmd represents the web server command
func webCmd() *cobra.Command {
	var cfg WebConfig

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Run the web server",
		Long:  `Start the WalletMind web server with health check and API endpoints.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebServer(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.Port, "port", "p", "8080", "Port to run the server on")
	cmd.Flags().StringVar(&cfg.EnvFile, "env-file", ".env", "Path to environment file")
	cmd.Flags().StringVar(&cfg.Environment, "environment", "development", "Environment (development/production)")

	return cmd
}

func runWebServer(cfg WebConfig) error {
	// Initialize logger
	initLogger(cfg.Environment == "development")

	log.Info().Str("version", GetVersion()).Msg("🚀 Starting WalletMind Web Server")

	// Load settings
	_, err := NewSettings(cfg.EnvFile)
	if err != nil {
		return fmt.Errorf("error loading settings: %w", err)
	}

	// Initialize Supabase client
	supabaseClient, err := config.NewSupabaseClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize Supabase client - authentication will be disabled")
		// Continue running without Supabase for now, but authentication will not work
	}

	// Create and start server
	serverConfig := web.ServerConfig{
		Port:          cfg.Port,
		Environment:   cfg.Environment,
		Client:        supabaseClient,
		IsDevelopment: cfg.Environment == "development",
	}

	server := web.NewServer(serverConfig)
	return server.Start()
}