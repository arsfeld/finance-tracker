package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	// Server
	ListenAddr  string
	DataDir     string
	DBPath      string
	FrontendDir string
	ViteDevURL  string // If set, proxy non-API requests to Vite dev server (e.g. "http://localhost:5173")

	// SimpleFin
	SimplefinBridgeURL string

	// OpenRouter
	OpenRouterURL    string
	OpenRouterAPIKey string
	OpenRouterModel  string

	// Notifications - Email
	MailerURL  string
	MailerFrom string
	MailerTo   string

	// Notifications - Ntfy
	NtfyServer        string
	NtfyTopic         string
	NtfyWarningSuffix string

	// Billing
	BillingDay int // Day of month (1-28) for billing cycle start

	// Scheduler
	SyncSchedule string

	// Env file path (for initial loading only)
	EnvFile string
}

// Load reads configuration from environment variables, with optional .env file.
func Load(envFile string) (*Config, error) {
	if err := godotenv.Load(envFile); err != nil {
		log.Info().Str("env_file", envFile).Msg("No .env file found, using environment variables")
	}

	dataDir := getEnvOr("DATA_DIR", "")
	dbPath := getEnvOr("DB_PATH", "finance_tracker.db")
	if dataDir != "" && !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(dataDir, dbPath)
	}

	cfg := &Config{
		ListenAddr:  getEnvOr("LISTEN_ADDR", ":8080"),
		DataDir:     dataDir,
		DBPath:      dbPath,
		FrontendDir: getEnvOr("FRONTEND_DIR", "web/dist"),
		ViteDevURL:  os.Getenv("VITE_DEV_URL"),
		EnvFile:     envFile,

		SimplefinBridgeURL: os.Getenv("SIMPLEFIN_BRIDGE_URL"),

		OpenRouterURL:    os.Getenv("OPENROUTER_URL"),
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:  os.Getenv("OPENROUTER_MODEL"),

		MailerURL:  os.Getenv("MAILER_URL"),
		MailerFrom: os.Getenv("MAILER_FROM"),
		MailerTo:   os.Getenv("MAILER_TO"),

		NtfyServer:        getEnvOr("NTFY_SERVER", "https://ntfy.sh"),
		NtfyTopic:         os.Getenv("NTFY_TOPIC"),
		NtfyWarningSuffix: getEnvOr("NTFY_WARNING_SUFFIX", "-warning"),

		BillingDay:   getEnvInt("BILLING_DAY", 15),
		SyncSchedule: getEnvOr("SYNC_SCHEDULE", "0 0 */6 * * *"),
	}

	return cfg, nil
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
