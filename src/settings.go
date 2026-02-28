package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
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
	NtfyWarningSuffix  string  // Suffix appended to NtfyTopic for warning notifications (default: "-warning")
	FilterConfigPath   *string // Path to YAML file with transaction filter rules (optional)
	CategoriesPath     string  // Path to JSON file for persistent category store (default: "categories.json")
	CachePath          string  // Path to JSON file for transaction cache (default: "transaction_cache.json")
	DataDir            string  // Base directory for persistent data files (default: "" = current directory)
}

// NewSettings creates a new Settings instance from environment variables
func NewSettings(env_file string) (*Settings, error) {
	// Try to load .env file, but don't error if it doesn't exist
	if err := godotenv.Load(env_file); err != nil {
		log.Info().Str("env_file", env_file).Str("error", err.Error()).Msg("No .env file found, using environment variables")
	}

	dataDir := os.Getenv("DATA_DIR")

	settings := &Settings{
		SimplefinBridgeURL: os.Getenv("SIMPLEFIN_BRIDGE_URL"),
		OpenRouterURL:      os.Getenv("OPENROUTER_URL"),
		OpenRouterAPIKey:   os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:    os.Getenv("OPENROUTER_MODEL"),
		NtfyServer:         "https://ntfy.sh",
		NtfyWarningSuffix:  "-warning", // Default suffix for warning notifications
		DataDir:            dataDir,
		CategoriesPath:     "categories.json",
		CachePath:          "transaction_cache.json",
	}

	// If DATA_DIR is set, resolve default file paths relative to it
	if dataDir != "" {
		settings.CategoriesPath = filepath.Join(dataDir, "categories.json")
		settings.CachePath = filepath.Join(dataDir, "transaction_cache.json")
		log.Debug().Str("data_dir", dataDir).Msg("Using DATA_DIR for default file paths")
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
	// Optional filter config path
	if filterConfigPath := os.Getenv("FILTER_CONFIG_PATH"); filterConfigPath != "" {
		settings.FilterConfigPath = &filterConfigPath
	}
	// Explicit CATEGORIES_PATH overrides DATA_DIR default
	if categoriesPath := os.Getenv("CATEGORIES_PATH"); categoriesPath != "" {
		settings.CategoriesPath = categoriesPath
	}
	// Explicit CACHE_PATH overrides DATA_DIR default
	if cachePath := os.Getenv("CACHE_PATH"); cachePath != "" {
		settings.CachePath = cachePath
	}

	return settings, nil
}

// LoadFilterConfig loads transaction filter rules from a YAML file
func LoadFilterConfig(configPath string) (*FilterConfig, error) {
	// Read the YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML
	var config FilterConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Validate and set defaults for specific exclusions
	for i := range config.SpecificExclusions {
		excl := &config.SpecificExclusions[i]

		// Validate date format
		if _, err := time.Parse("2006-01-02", excl.Date); err != nil {
			return nil, fmt.Errorf("invalid date format for specific exclusion %d (pattern: %q): %q (expected YYYY-MM-DD)", i+1, excl.Pattern, excl.Date)
		}

		// Default match type to substring if empty
		if excl.MatchType == "" {
			excl.MatchType = MatchTypeSubstring
		}
	}

	log.Debug().
		Int("rule_count", len(config.ExcludedTransactions)).
		Int("specific_exclusion_count", len(config.SpecificExclusions)).
		Str("config_path", configPath).
		Msg("Loaded filter configuration")

	return &config, nil
}
