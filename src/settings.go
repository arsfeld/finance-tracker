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
	NtfyTopicWarning   *string // Added field for warning notifications
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
	// TODO: Add loading for NtfyTopicWarning if needed from env
	if ntfyTopicWarning := os.Getenv("NTFY_TOPIC_WARNING"); ntfyTopicWarning != "" {
		settings.NtfyTopicWarning = &ntfyTopicWarning
	}

	return settings, nil
} 