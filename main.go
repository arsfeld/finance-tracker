package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Global verbose flag and logger
var verboseMode bool
var logrusLogger = logrus.New()

// initLogger initializes the Logrus logger with the appropriate level
func initLogger(verbose bool) {
	verboseMode = verbose
	if verbose {
		logrusLogger.SetLevel(logrus.DebugLevel)
	} else {
		logrusLogger.SetLevel(logrus.InfoLevel)
	}
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

// Balance represents a monetary value that can be unmarshaled from either string or float64
type Balance float64

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

// Types
type NotificationType string

const (
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeNtfy  NotificationType = "ntfy"
)

type DateRangeType string

const (
	DateRangeTypeCurrentMonth DateRangeType = "current_month"
	DateRangeTypeLastMonth    DateRangeType = "last_month"
	DateRangeTypeLast3Months  DateRangeType = "last_3_months"
	DateRangeTypeCurrentYear  DateRangeType = "current_year"
	DateRangeTypeLastYear     DateRangeType = "last_year"
	DateRangeTypeCustom       DateRangeType = "custom"
)

type Organization struct {
	SfinURL string  `json:"sfin-url"`
	Domain  *string `json:"domain,omitempty"`
	Name    *string `json:"name,omitempty"`
	URL     *string `json:"url,omitempty"`
	ID      *string `json:"id,omitempty"`
}

type Transaction struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Amount       Balance  `json:"amount"`
	Posted       int64    `json:"posted"`
	TransactedAt *int64   `json:"transacted_at,omitempty"`
	Pending      *bool    `json:"pending,omitempty"`
	Extra        *map[string]interface{} `json:"extra,omitempty"`
}

type Account struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Balance         Balance       `json:"balance"`
	BalanceDate     int64         `json:"balance-date"`
	Org             Organization  `json:"org"`
	Transactions    []Transaction `json:"transactions,omitempty"`
	Currency        *string       `json:"currency,omitempty"`
	AvailableBalance *Balance     `json:"available-balance,omitempty"`
	Holdings        []interface{} `json:"holdings,omitempty"`
}

type AccountsResponse struct {
	Accounts     []Account  `json:"accounts"`
	Errors       []string   `json:"errors,omitempty"`
	XAPIMessage  []string   `json:"x-api-message,omitempty"`
}

type Settings struct {
	SimplefinBridgeURL string  `json:"simplefin_bridge_url"`
	TwilioAccountSid   *string `json:"twilio_account_sid,omitempty"`
	TwilioAuthToken    *string `json:"twilio_auth_token,omitempty"`
	TwilioFromPhone    *string `json:"twilio_from_phone,omitempty"`
	TwilioToPhones     *string `json:"twilio_to_phones,omitempty"`
	OpenRouterURL      string  `json:"openrouter_url"`
	OpenRouterAPIKey   string  `json:"openrouter_api_key"`
	OpenRouterModel    string  `json:"openrouter_model"`
	MailerURL          *string `json:"mailer_url,omitempty"`
	MailerFrom         *string `json:"mailer_from,omitempty"`
	MailerTo           *string `json:"mailer_to,omitempty"`
	NtfyServer         string  `json:"ntfy_server"`
	NtfyTopic          *string `json:"ntfy_topic,omitempty"`
}

type Cache struct {
	Accounts                map[string]map[string]interface{} `json:"accounts,omitempty"`
	LastSuccessfulMessage   *int64                           `json:"last_successful_message,omitempty"`
}

type TrackerError struct {
	Message string
}

func (e *TrackerError) Error() string {
	return e.Message
}

func NewSettings() (*Settings, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	settings := &Settings{
		SimplefinBridgeURL: os.Getenv("SIMPLEFIN_BRIDGE_URL"),
		OpenRouterURL:      os.Getenv("OPENROUTER_URL"),
		OpenRouterAPIKey:   os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:    os.Getenv("OPENROUTER_MODEL"),
		NtfyServer:         "https://ntfy.sh",
	}

	// Optional fields
	if twilioSid := os.Getenv("TWILIO_ACCOUNT_SID"); twilioSid != "" {
		settings.TwilioAccountSid = &twilioSid
	}
	if twilioToken := os.Getenv("TWILIO_AUTH_TOKEN"); twilioToken != "" {
		settings.TwilioAuthToken = &twilioToken
	}
	if twilioFrom := os.Getenv("TWILIO_FROM_PHONE"); twilioFrom != "" {
		settings.TwilioFromPhone = &twilioFrom
	}
	if twilioTo := os.Getenv("TWILIO_TO_PHONES"); twilioTo != "" {
		settings.TwilioToPhones = &twilioTo
	}
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
		Long: `Finance Tracker is a powerful tool that analyzes your financial transactions using AI.
It connects to your SimpleFin account to fetch transactions and uses OpenAI's LLM to provide
insightful analysis of your spending patterns.

The tool supports multiple notification channels and includes a caching mechanism to prevent
duplicate notifications. It can analyze transactions for various time periods and provides
detailed breakdowns of your spending habits.

Example usage:
  finance_tracker                    # Analyze current month's transactions
  finance_tracker --date-range last_month  # Analyze last month's transactions
  finance_tracker --notifications ntfy     # Send notifications via ntfy
  finance_tracker --disable-cache          # Force fresh analysis without caching`,
		RunE: func(cmd *cobra.Command, args []string) error {
			notifications, _ := cmd.Flags().GetStringSlice("notifications")
			disableNotifications, _ := cmd.Flags().GetBool("disable-notifications")
			disableCache, _ := cmd.Flags().GetBool("disable-cache")
			verbose, _ := cmd.Flags().GetBool("verbose")
			dateRange, _ := cmd.Flags().GetString("date-range")
			startDate, _ := cmd.Flags().GetString("start-date")
			endDate, _ := cmd.Flags().GetString("end-date")

			return run(notifications, disableNotifications, disableCache, verbose, dateRange, startDate, endDate)
		},
	}

	rootCmd.Flags().StringSliceP("notifications", "n", []string{"sms", "email", "ntfy"}, "Notification types to send")
	rootCmd.Flags().Bool("disable-notifications", false, "Disable all notifications")
	rootCmd.Flags().Bool("disable-cache", false, "Disable caching")
	rootCmd.Flags().Bool("verbose", false, "Enable verbose logging")
	rootCmd.Flags().String("date-range", string(DateRangeTypeCurrentMonth), "Date range type")
	rootCmd.Flags().String("start-date", "", "Start date for custom range (YYYY-MM-DD)")
	rootCmd.Flags().String("end-date", "", "End date for custom range (YYYY-MM-DD)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(
	notifications []string,
	disableNotifications bool,
	disableCache bool,
	verbose bool,
	dateRange string,
	startDate string,
	endDate string,
) error {
	// Initialize logger
	initLogger(verbose)
	logrusLogger.Debug("Starting finance tracker with verbose mode: ", verbose)

	fmt.Println("🔧 Loading configuration...")
	settings, err := NewSettings()
	if err != nil {
		return fmt.Errorf("error loading settings: %w", err)
	}
	logrusLogger.Debug("Configuration loaded successfully")

	// Parse date range
	dateRangeType := DateRangeType(dateRange)
	if dateRangeType != DateRangeTypeCurrentMonth {
		disableCache = true
		logrusLogger.Debug("Using non-current month date range, cache disabled")
	}

	// Parse custom dates if provided
	var parsedStartDate, parsedEndDate *time.Time
	if startDate != "" {
		parsed, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return fmt.Errorf("error parsing start date: %w", err)
		}
		parsedStartDate = &parsed
		logrusLogger.WithField("start_date", parsed.Format("2006-01-02")).Debug("Parsed start date")
	}
	if endDate != "" {
		parsed, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return fmt.Errorf("error parsing end date: %w", err)
		}
		parsedEndDate = &parsed
		logrusLogger.WithField("end_date", parsed.Format("2006-01-02")).Debug("Parsed end date")
	}

	// Calculate date range
	billingStart, billingEnd, err := calculateDateRange(dateRangeType, parsedStartDate, parsedEndDate)
	if err != nil {
		return fmt.Errorf("error calculating date range: %w", err)
	}
	logrusLogger.WithFields(logrus.Fields{
		"start": billingStart.Format("2006-01-02"),
		"end":   billingEnd.Format("2006-01-02"),
	}).Debug("Calculated date range")

	// Validate billing period
	if err := validateBillingPeriod(billingStart, billingEnd); err != nil {
		return fmt.Errorf("error validating billing period: %w", err)
	}
	logrusLogger.Debug("Billing period validated successfully")

	// Load cache
	cache := &Cache{}
	if !disableCache {
		if err := cache.Load(); err != nil {
			return fmt.Errorf("error loading cache: %w", err)
		}
		logrusLogger.Debug("Cache loaded successfully")
	} else {
		logrusLogger.Debug("Cache loading skipped (disabled)")
	}

	// Fetch transactions
	fmt.Println("📊 Fetching transactions...")
	accounts, err := getTransactionsForPeriod(settings, billingStart, billingEnd)
	if err != nil {
		return fmt.Errorf("error fetching transactions: %w", err)
	}
	logrusLogger.WithField("account_count", len(accounts)).Debug("Fetched accounts")

	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	// Process accounts
	fmt.Println("💳 Accounts:")
	hasUpdatedAccounts := false
	for _, account := range accounts {
		fmt.Printf("• %s (%s)\n", account.Name, account.ID)
		syncTime := time.Unix(account.BalanceDate, 0).Format("2006-01-02 15:04:05")
		fmt.Printf("  └ Last synced at: %s\n", syncTime)
		logrusLogger.WithFields(logrus.Fields{
			"account_name": account.Name,
			"account_id":   account.ID,
		}).Debug("Processing account")

		if !disableCache && cache.IsAccountUpdated(account.ID, account.BalanceDate) {
			hasUpdatedAccounts = true
			cache.UpdateAccount(account.ID, account.Balance, account.BalanceDate)
			logrusLogger.WithField("account_id", account.ID).Debug("Account updated in cache")
		} else {
			logrusLogger.WithField("account_id", account.ID).Debug("Account not updated (cache disabled or no changes)")
		}
	}

	// Early return conditions
	if !hasUpdatedAccounts {
		logrusLogger.Debug("No accounts were updated, returning early")
		fmt.Println("🔴 No updated accounts")
	}

	// Collect all transactions
	var allTransactions []Transaction
	for _, account := range accounts {
		allTransactions = append(allTransactions, account.Transactions...)
	}
	logrusLogger.WithField("transaction_count", len(allTransactions)).Debug("Collected total transactions")

	if len(allTransactions) == 0 {
		return fmt.Errorf("no transactions found")
	}

	// Check last message time
	if !disableCache && cache.LastSuccessfulMessage != nil {
		lastMsgTime := time.Unix(*cache.LastSuccessfulMessage, 0)
		if time.Since(lastMsgTime).Seconds() < float64(twoDaysInSeconds) {
			logrusLogger.WithField("last_message_time", lastMsgTime.Format("2006-01-02 15:04:05")).Debug("Last message was sent too recently")
			return fmt.Errorf("last message was sent too recently (at %s)", lastMsgTime.Format("2006-01-02 15:04:05"))
		}
		logrusLogger.Debug("Last message check passed")
	}

	// Process transactions with AI
	fmt.Println("🤖 Analyzing transactions with AI...")
	prompt := generateAnalysisPrompt(accounts, allTransactions, billingStart, billingEnd)
	logrusLogger.WithField("prompt", prompt).Debug("Generated analysis prompt")
	
	analysis, err := getLLMResponse(settings, prompt)
	if err != nil {
		return fmt.Errorf("error getting LLM response: %w", err)
	}
	logrusLogger.WithField("analysis", analysis).Debug("Received AI analysis")

	fmt.Println("\n✨ AI Summary:")
	fmt.Println(analysis)

	// Send notifications
	if !disableNotifications {
		logrusLogger.WithField("notification_channels", notifications).Debug("Sending notifications")
		if err := sendNotification(settings, analysis, "info", notifications); err != nil {
			return fmt.Errorf("error sending notifications: %w", err)
		}
		logrusLogger.Debug("Notifications sent successfully")

		// Update cache
		if !disableCache {
			cache.UpdateLastMessageTime()
			if err := cache.Save(); err != nil {
				return fmt.Errorf("error saving cache: %w", err)
			}
			logrusLogger.Debug("Cache updated with new message time")
		}
	} else {
		logrusLogger.Debug("Notifications disabled, skipping")
		fmt.Println("ℹ️ Notifications disabled")
	}

	logrusLogger.Debug("Finance tracker completed successfully")
	return nil
} 