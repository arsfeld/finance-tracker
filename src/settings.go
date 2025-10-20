package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Settings holds the application configuration
type Settings struct {
	SimplefinBridgeURL string
	OpenRouterURL      string
	OpenRouterAPIKey   string
	OpenRouterModel    string
	NtfyServer         string
	MailerURL          *string
	MailerFrom         *string
	MailerTo           *string
	NtfyTopic          *string
	NtfyWarningSuffix  string // Suffix appended to NtfyTopic for warning notifications (default: "-warning")
}

// NewSettings creates a new Settings instance from environment variables
func NewSettings(env_file string) (*Settings, error) {
	// Try to load .env file, but don't error if it doesn't exist
	if err := godotenv.Load(env_file); err != nil {
		log.Info().Str("env_file", env_file).Str("error", err.Error()).Msg("No .env file found, using environment variables")
	}

	settings := &Settings{
		SimplefinBridgeURL: os.Getenv("SIMPLEFIN_BRIDGE_URL"),
		OpenRouterURL:      os.Getenv("OPENROUTER_URL"),
		OpenRouterAPIKey:   os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:    os.Getenv("OPENROUTER_MODEL"),
		NtfyServer:         "https://ntfy.sh",
		NtfyWarningSuffix:  "-warning", // Default suffix for warning notifications
	}

	// Optional fields
	if mailerURL := os.Getenv("MAILER_URL"); mailerURL != "" {
		settings.MailerURL = &mailerURL
	}
	if mailerFrom := os.Getenv("MAILER_FROM"); mailerFrom != "" {
		settings.MailerFrom = &mailerFrom
	}
	if mailerTo := os.Getenv("MAILER_TO"); mailerTo != "" {
		settings.MailerTo = &mailerTo
	}
	if ntfyTopic := os.Getenv("NTFY_TOPIC"); ntfyTopic != "" {
		settings.NtfyTopic = &ntfyTopic
	}
	// Allow customizing the warning suffix (optional)
	if ntfyWarningSuffix := os.Getenv("NTFY_WARNING_SUFFIX"); ntfyWarningSuffix != "" {
		settings.NtfyWarningSuffix = ntfyWarningSuffix
	}

	return settings, nil
}
