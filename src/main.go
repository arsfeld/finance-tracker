package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// initLogger initializes the Zerolog logger with the appropriate level
func initLogger(verbose bool) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// Balance represents a monetary value that can be unmarshaled from either string or float64
type Balance float64

// UnmarshalJSON implements the json.Unmarshaler interface for Balance
func (b *Balance) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first (since API specifies numeric string)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		// If successful, try to convert to float64
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("error parsing balance string '%s': %w", s, err)
		}
		*b = Balance(f)
		return nil
	}

	// If string unmarshal fails, try float64 (for backward compatibility)
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("error unmarshaling balance: %w", err)
	}
	*b = Balance(f)
	return nil
}

// String returns a string representation of the balance
func (b Balance) String() string {
	return fmt.Sprintf("%.2f", float64(b))
}

// Constants
const (
	twoDaysInSeconds = 2 * 24 * 60 * 60
)

// NotificationType defines the type of notification
type NotificationType string

// Available notification types
const (
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeNtfy  NotificationType = "ntfy"
)

// DateRangeType defines the type of date range for analysis
type DateRangeType string

// Available date range types
const (
	DateRangeTypeCurrentMonth DateRangeType = "current_month"
	DateRangeTypeLastMonth    DateRangeType = "last_month"
	DateRangeTypeLast3Months  DateRangeType = "last_3_months"
	DateRangeTypeCurrentYear  DateRangeType = "current_year"
	DateRangeTypeLastYear     DateRangeType = "last_year"
	DateRangeTypeCustom       DateRangeType = "custom"
)

// Organization represents a financial institution or organization
type Organization struct {
	SfinURL string  `json:"sfin-url"`
	Domain  *string `json:"domain,omitempty"`
	Name    *string `json:"name,omitempty"`
	URL     *string `json:"url,omitempty"`
	ID      *string `json:"id,omitempty"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID           string                  `json:"id"`
	Description  string                  `json:"description"`
	Amount       Balance                 `json:"amount"`
	Posted       int64                   `json:"posted"`
	TransactedAt *int64                  `json:"transacted_at,omitempty"`
	Pending      *bool                   `json:"pending,omitempty"`
	Extra        *map[string]interface{} `json:"extra,omitempty"`
}

// Account represents a financial account
type Account struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Balance          Balance       `json:"balance"`
	BalanceDate      int64         `json:"balance-date"`
	Org              Organization  `json:"org"`
	Transactions     []Transaction `json:"transactions,omitempty"`
	Currency         *string       `json:"currency,omitempty"`
	AvailableBalance *Balance      `json:"available-balance,omitempty"`
	Holdings         []interface{} `json:"holdings,omitempty"`
}

// AccountsResponse represents the response from the SimpleFin API
type AccountsResponse struct {
	Accounts    []Account `json:"accounts"`
	Errors      []string  `json:"errors,omitempty"`
	XAPIMessage []string  `json:"x-api-message,omitempty"`
}

// Settings represents the application settings
type Settings struct {
	SimplefinBridgeURL string  `json:"simplefin_bridge_url"`
	OpenRouterURL      string  `json:"openrouter_url"`
	OpenRouterAPIKey   string  `json:"openrouter_api_key"`
	OpenRouterModel    string  `json:"openrouter_model"`
	MailerURL          *string `json:"mailer_url,omitempty"`
	MailerFrom         *string `json:"mailer_from,omitempty"`
	MailerTo           *string `json:"mailer_to,omitempty"`
	NtfyServer         string  `json:"ntfy_server"`
	NtfyTopic          *string `json:"ntfy_topic,omitempty"`
	NtfyTopicWarning   *string `json:"ntfy_topic_warning,omitempty"`
}

// Cache represents the cache for the application
type Cache struct {
	Version               int64              `json:"version"`
	Accounts              map[string]Account `json:"accounts,omitempty"`
	LastSuccessfulMessage *int64             `json:"last_successful_message,omitempty"`
}

// NewSettings creates a new Settings instance from environment variables
func NewSettings(env_file string) (*Settings, error) {
	// Try to load .env file, but don't error if it doesn't exist
	if err := godotenv.Load(".env", env_file); err != nil {
		log.Debug().Msg("No .env file found, using environment variables")
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

	return settings, nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "finance_tracker",
		Short: "Track your finances with AI-powered analysis",
		Long: fmt.Sprintf(`Finance Tracker is a powerful tool that analyzes your financial transactions using AI.
It connects to your SimpleFin account to fetch transactions and uses OpenAI's LLM to provide
insightful analysis of your spending patterns.

The tool supports multiple notification channels and includes a caching mechanism to prevent
duplicate notifications. It can analyze transactions for various time periods and provides
detailed breakdowns of your spending habits.

Version: %s

Example usage:
  finance_tracker                    # Analyze current month's transactions
  finance_tracker --date-range last_month  # Analyze last month's transactions
  finance_tracker --notifications ntfy     # Send notifications via ntfy
  finance_tracker --disable-cache          # Force fresh analysis without caching`, GetVersion()),
		RunE: func(cmd *cobra.Command, args []string) error {
			notifications, _ := cmd.Flags().GetStringSlice("notifications")
			disableNotifications, _ := cmd.Flags().GetBool("disable-notifications")
			disableCache, _ := cmd.Flags().GetBool("disable-cache")
			verbose, _ := cmd.Flags().GetBool("verbose")
			dateRange, _ := cmd.Flags().GetString("date-range")
			startDate, _ := cmd.Flags().GetString("start-date")
			endDate, _ := cmd.Flags().GetString("end-date")
			force, _ := cmd.Flags().GetBool("force")
			env_file, _ := cmd.Flags().GetString("env-file")

			return run(notifications, disableNotifications, disableCache, verbose, dateRange, startDate, endDate, force, env_file, GetVersion())
		},
	}

	rootCmd.Flags().StringSliceP("notifications", "n", []string{"sms", "email", "ntfy"}, "Notification types to send")
	rootCmd.Flags().Bool("disable-notifications", false, "Disable all notifications")
	rootCmd.Flags().Bool("disable-cache", false, "Disable caching")
	rootCmd.Flags().Bool("verbose", false, "Enable verbose logging")
	rootCmd.Flags().String("date-range", string(DateRangeTypeCurrentMonth), "Date range type")
	rootCmd.Flags().String("start-date", "", "Start date for custom range (YYYY-MM-DD)")
	rootCmd.Flags().String("end-date", "", "End date for custom range (YYYY-MM-DD)")
	rootCmd.Flags().Bool("force", false, "Force analysis even if cache is up to date")
	rootCmd.Flags().String("env-file", "", "Path to environment file")
	rootCmd.Flags().Bool("version", false, "Show version information")
	rootCmd.SetVersionTemplate(GetVersion() + "\n")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Error executing root command")
	}
}

// run is the main function that runs the finance tracker
func run(
	notifications []string,
	disableNotifications bool,
	disableCache bool,
	verbose bool,
	dateRange string,
	startDate string,
	endDate string,
	force bool,
	env_file string,
	version string,
) error {
	// Initialize logger
	initLogger(verbose)
	log.Debug().Bool("verbose", verbose).Msg("Starting finance tracker")

	log.Info().Str("notifications", strings.Join(notifications, ", ")).
		Bool("disable_notifications", disableNotifications).
		Bool("disable_cache", disableCache).
		Str("date_range", dateRange).
		Str("start_date", startDate).
		Str("end_date", endDate).
		Bool("force", force).
		Str("version", version).
		Msg("Starting finance tracker")

	log.Info().Msg("🔧 Loading configuration...")
	settings, err := NewSettings(env_file)
	if err != nil {
		return fmt.Errorf("error loading settings: %w", err)
	}

	// Log settings in a structured way
	log.Info().
		Str("simplefin_bridge_url", settings.SimplefinBridgeURL).
		Str("openrouter_url", settings.OpenRouterURL).
		Str("openrouter_model", settings.OpenRouterModel).
		Str("ntfy_server", settings.NtfyServer).
		Str("mailer_url", getStringValue(settings.MailerURL)).
		Str("mailer_from", getStringValue(settings.MailerFrom)).
		Str("mailer_to", getStringValue(settings.MailerTo)).
		Str("ntfy_topic", getStringValue(settings.NtfyTopic)).
		Str("ntfy_topic_warning", getStringValue(settings.NtfyTopicWarning)).
		Msg("Configuration loaded successfully")

	// Parse date range
	dateRangeType := DateRangeType(dateRange)
	if dateRangeType != DateRangeTypeCurrentMonth {
		disableCache = true
		log.Debug().Msg("Using non-current month date range, cache disabled")
	}

	// Parse custom dates if provided
	var parsedStartDate, parsedEndDate *time.Time
	if startDate != "" {
		parsed, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return fmt.Errorf("error parsing start date: %w", err)
		}
		parsedStartDate = &parsed
		log.Debug().Str("start_date", parsed.Format("2006-01-02")).Msg("Parsed start date")
	}
	if endDate != "" {
		parsed, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return fmt.Errorf("error parsing end date: %w", err)
		}
		parsedEndDate = &parsed
		log.Debug().Str("end_date", parsed.Format("2006-01-02")).Msg("Parsed end date")
	}

	// Calculate date range
	billingStart, billingEnd, err := calculateDateRange(dateRangeType, parsedStartDate, parsedEndDate)
	if err != nil {
		return fmt.Errorf("error calculating date range: %w", err)
	}
	log.Debug().
		Str("start", billingStart.Format("2006-01-02")).
		Str("end", billingEnd.Format("2006-01-02")).
		Msg("Calculated date range")

	// Validate billing period
	if err := validateBillingPeriod(billingStart, billingEnd); err != nil {
		return fmt.Errorf("error validating billing period: %w", err)
	}
	log.Debug().Msg("Billing period validated successfully")

	// Load cache
	log.Info().Msg("🔄 Loading cache...")
	cache := &Cache{}
	if !disableCache {
		if err := cache.Load(); err != nil {
			return fmt.Errorf("error loading cache: %w", err)
		}
		log.Debug().Msg("Cache loaded successfully")
	} else {
		log.Debug().Msg("Cache loading skipped (disabled)")
	}

	// Fetch transactions
	log.Info().Msg("📊 Fetching transactions...")
	accounts, err := getTransactionsForPeriod(settings, billingStart, billingEnd)
	if err != nil {
		return fmt.Errorf("error fetching transactions: %w", err)
	}
	log.Debug().Int("account_count", len(accounts)).Msg("Fetched accounts")

	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	// Process accounts
	log.Info().Msg("💳 Accounts:")
	hasUpdatedAccounts := false
	for _, account := range accounts {
		log.Info().Str("account_name", account.Name).Str("account_id", account.ID).Msg("•")
		syncTime := time.Unix(account.BalanceDate, 0).Format("2006-01-02 15:04:05")
		log.Info().Str("sync_time", syncTime).
			Str("balance", account.Balance.String()).
			Str("transactions", strconv.Itoa(len(account.Transactions))).
			Msg("  └")

		if !disableCache && cache.IsAccountUpdated(account.ID, account.BalanceDate) {
			hasUpdatedAccounts = true
			cache.UpdateAccount(account)
			log.Debug().Str("account_id", account.ID).Msg("Account updated in cache")
		} else {
			log.Debug().Str("account_id", account.ID).Msg("Account not updated (cache disabled or no changes)")
		}
	}

	// Early return conditions
	if !hasUpdatedAccounts && !force {
		log.Debug().Msg("No accounts were updated, returning early")
		log.Info().Msg("🔴 No updated accounts")
		return nil
	}

	// Collect all transactions
	var allTransactions []Transaction
	for _, account := range accounts {
		allTransactions = append(allTransactions, account.Transactions...)
	}
	log.Debug().Int("transaction_count", len(allTransactions)).Msg("Collected total transactions")

	if len(allTransactions) == 0 {
		return fmt.Errorf("no transactions found")
	}

	// Check last message time
	if !force && cache.LastSuccessfulMessage != nil {
		lastMsgTime := time.Unix(*cache.LastSuccessfulMessage, 0)
		if time.Since(lastMsgTime).Seconds() < float64(twoDaysInSeconds) {
			log.Debug().Str("last_message_time", lastMsgTime.Format("2006-01-02 15:04:05")).Msg("Last message was sent too recently")
			return fmt.Errorf("last message was sent too recently (at %s)", lastMsgTime.Format("2006-01-02 15:04:05"))
		}
		log.Debug().Msg("Last message check passed")
	}

	// Process transactions with AI
	log.Info().Msg("🤖 Analyzing transactions with AI...")
	prompt := generateAnalysisPrompt(accounts, allTransactions, billingStart, billingEnd)
	log.Debug().Str("prompt", prompt).Msg("Generated analysis prompt")

	analysis, err := getLLMResponse(settings, prompt)
	if err != nil {
		return fmt.Errorf("error getting LLM response: %w", err)
	}
	log.Debug().Str("analysis", analysis).Msg("Received AI analysis")

	if analysis == "" {
		return fmt.Errorf("received empty analysis from LLM")
	}

	log.Info().Msg("✨ AI Summary:")
	log.Info().Msg(analysis)

	// Send notifications
	if !disableNotifications {
		log.Debug().Strs("notification_channels", notifications).Msg("Sending notifications")
		if err := sendNotification(settings, analysis, allTransactions, "info", notifications); err != nil {
			return fmt.Errorf("error sending notifications: %w", err)
		}
		log.Debug().Msg("Notifications sent successfully")

		// Update cache
		if !disableCache {
			cache.UpdateLastMessageTime()
			if err := cache.Save(); err != nil {
				return fmt.Errorf("error saving cache: %w", err)
			}
			log.Debug().Msg("Cache updated with new message time")
		}
	} else {
		log.Debug().Msg("Notifications disabled, skipping")
		log.Info().Msg("ℹ️ Notifications disabled")
	}

	log.Debug().Msg("Finance tracker completed successfully")
	return nil
}

// Helper function to safely get string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
