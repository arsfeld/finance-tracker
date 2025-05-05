package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Constants
const (
	twoDaysInSeconds = 2 * 24 * 60 * 60
)

// RunConfig holds all configuration parameters for the run function
type RunConfig struct {
	Notifications        []string
	DisableNotifications bool
	DisableCache         bool
	Verbose              bool
	DateRange            string
	StartDate            string
	EndDate              string
	Force                bool
	EnvFile              string
	Version              string
	MaxRetries           int
	RetryDelay           int
	BillingDay           int
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
  finance_tracker                    # Analyze current month's transactions (billing day 15)
  finance_tracker --billing-day 1    # Analyze current month's transactions (billing day 1)
  finance_tracker --date-range last_month  # Analyze last month's transactions
  finance_tracker --notifications ntfy     # Send notifications via ntfy
  finance_tracker --disable-cache          # Force fresh analysis without caching
  finance_tracker --max-retries 5          # Set maximum number of retries for LLM calls
  finance_tracker --retry-delay 2          # Set initial retry delay in seconds`, GetVersion()),
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
			maxRetries, _ := cmd.Flags().GetInt("max-retries")
			retryDelay, _ := cmd.Flags().GetInt("retry-delay")
			billingDay, _ := cmd.Flags().GetInt("billing-day")

			return run(RunConfig{
				Notifications:        notifications,
				DisableNotifications: disableNotifications,
				DisableCache:         disableCache,
				Verbose:              verbose,
				DateRange:            dateRange,
				StartDate:            startDate,
				EndDate:              endDate,
				Force:                force,
				EnvFile:              env_file,
				Version:              GetVersion(),
				MaxRetries:           maxRetries,
				RetryDelay:           retryDelay,
				BillingDay:           billingDay,
			})
		},
	}

	rootCmd.Flags().StringSliceP("notifications", "n", []string{"email", "ntfy"}, "Notification types to send")
	rootCmd.Flags().Bool("disable-notifications", false, "Disable all notifications")
	rootCmd.Flags().Bool("disable-cache", false, "Disable database caching")
	rootCmd.Flags().Bool("verbose", false, "Enable verbose logging")
	rootCmd.Flags().String("date-range", string(DateRangeTypeCurrentMonth), "Date range type")
	rootCmd.Flags().String("start-date", "", "Start date for custom range (YYYY-MM-DD)")
	rootCmd.Flags().String("end-date", "", "End date for custom range (YYYY-MM-DD)")
	rootCmd.Flags().Bool("force", false, "Force analysis even if database is up to date")
	rootCmd.Flags().String("env-file", ".env", "Path to environment file")
	rootCmd.Flags().Bool("version", false, "Show version information")
	rootCmd.Flags().Int("max-retries", 5, "Maximum number of retries for LLM calls")
	rootCmd.Flags().Int("retry-delay", 2, "Initial retry delay in seconds")
	rootCmd.Flags().Int("billing-day", 15, "Day of the month for the billing cycle start (1-28)")
	rootCmd.SetVersionTemplate(GetVersion() + "\n")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Error executing root command")
	}
}

// run is the main function that runs the finance tracker
func run(config RunConfig) error {
	// Initialize logger
	initLogger(config.Verbose)

	log.Info().Msg("🔧 Starting " + GetVersion())

	log.Debug().Interface("config", config).Msg("Starting finance tracker")

	log.Info().Msg("🔧 Loading configuration...")
	settings, err := NewSettings(config.EnvFile)
	if err != nil {
		return fmt.Errorf("error loading settings: %w", err)
	}

	// Log settings in a structured way
	log.Debug().Interface("settings", settings).Msg("Configuration loaded successfully")

	// Parse date range
	dateRangeType := DateRangeType(config.DateRange)
	if dateRangeType != DateRangeTypeCurrentMonth {
		config.DisableCache = true
		log.Debug().Msg("Using non-current month date range, database disabled")
	}

	// Parse custom dates if provided
	var parsedStartDate, parsedEndDate *time.Time
	if config.StartDate != "" {
		parsed, err := time.Parse("2006-01-02", config.StartDate)
		if err != nil {
			return fmt.Errorf("error parsing start date: %w", err)
		}
		parsedStartDate = &parsed
		log.Debug().Str("start_date", parsed.Format("2006-01-02")).Msg("Parsed start date")
	}
	if config.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", config.EndDate)
		if err != nil {
			return fmt.Errorf("error parsing end date: %w", err)
		}
		parsedEndDate = &parsed
		log.Debug().Str("end_date", parsed.Format("2006-01-02")).Msg("Parsed end date")
	}

	// Calculate date range
	billingStart, billingEnd, err := calculateDateRange(dateRangeType, parsedStartDate, parsedEndDate, config.BillingDay)
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

	// Load database
	log.Info().Msg("🔄 Loading database...")
	var db *DB
	if !config.DisableCache {
		db, err = NewDB()
		if err != nil {
			return fmt.Errorf("error creating database: %w", err)
		}
		defer db.Close()

		log.Debug().Msg("Database loaded successfully")
	} else {
		log.Debug().Msg("Database loading skipped (disabled)")
	}

	// Fetch transactions
	log.Info().Msg("📊 Fetching transactions...")
	accounts, apiErrors, err := getTransactionsForPeriod(settings, billingStart, billingEnd)
	if err != nil {
		return fmt.Errorf("error fetching transactions: %w", err)
	}
	log.Debug().Int("account_count", len(accounts)).Msg("Fetched accounts")

	// Handle API errors by sending warnings through configured channels
	if len(apiErrors) > 0 && !config.DisableNotifications {
		log.Warn().Strs("api_errors", apiErrors).Msg("Received API errors during transaction fetch")
		for _, apiErr := range apiErrors {
			warnMsg := fmt.Sprintf("API Error: %s", apiErr)
			_, notifyErr := sendNotification(settings, warnMsg, nil, "warning", config.Notifications)
			if notifyErr != nil {
				// Log the notification error but don't stop the main process
				log.Error().Err(notifyErr).Str("original_api_error", apiErr).Msg("Failed to send API error warning notification")
			}
		}
		log.Debug().Msg("Sent warning notifications for API errors")
	}

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
			Msg("└")

		if !config.DisableCache && db.IsAccountUpdated(account.ID, account.BalanceDate) {
			hasUpdatedAccounts = true
			if err := db.UpdateAccount(account); err != nil {
				return fmt.Errorf("error updating account in database: %w", err)
			}
			log.Debug().Str("account_id", account.ID).Msg("Account updated in database")
		} else {
			log.Debug().Str("account_id", account.ID).Msg("Account not updated (database disabled or no changes)")
		}
	}

	// Early return conditions
	if !hasUpdatedAccounts && !config.Force {
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

	// Filter out positive transactions (keep only expenses)
	var expenses []Transaction
	positiveTxnCount := 0
	for _, tx := range allTransactions {
		if tx.Amount < 0 {
			expenses = append(expenses, tx)
		} else {
			positiveTxnCount++
		}
	}
	allTransactions = expenses // Replace the original slice with the filtered one
	log.Debug().
		Int("filtered_transaction_count", len(allTransactions)).
		Int("positive_txns_ignored", positiveTxnCount).
		Msg("Filtered out positive transactions (e.g., income, payments)")

	if len(allTransactions) == 0 {
		return fmt.Errorf("no transactions found")
	}

	// Check last message time
	if !config.Force {
		lastMsgTime, err := db.GetLastMessageTime()
		if err != nil {
			return fmt.Errorf("error getting last message time: %w", err)
		}
		if lastMsgTime != nil {
			lastMsgTimeUnix := time.Unix(*lastMsgTime, 0)
			if time.Since(lastMsgTimeUnix).Seconds() < float64(twoDaysInSeconds) {
				log.Info().Str("last_message_time", lastMsgTimeUnix.Format("2006-01-02 15:04:05")).Msg("🚫 Notification skipped: Last message sent too recently.")
				return nil // Return success, but indicate skipped notification
			}
			log.Debug().Msg("Last message check passed")
		}
	}

	// Process transactions with AI
	log.Info().Msg("🤖 Analyzing transactions with AI...")
	prompt := generateAnalysisPrompt(accounts, allTransactions, billingStart, billingEnd)
	log.Debug().Str("prompt", prompt).Msg("Generated analysis prompt")

	// Get LLM response with retry
	analysis, err := retryWithBackoff(
		func() (string, error) {
			return getLLMResponse(settings, prompt)
		},
		config.MaxRetries,
		config.RetryDelay,
		"LLM request",
	)
	if err != nil {
		return fmt.Errorf("error getting LLM response: %w", err)
	}

	log.Debug().Str("analysis", analysis).Msg("Received AI analysis")

	log.Info().Msg("✨ AI Summary:")
	log.Info().Msg(analysis)

	// Send notifications
	if !config.DisableNotifications {
		log.Debug().Strs("notification_channels", config.Notifications).Msg("Sending notifications")
		successfulChannels, err := sendNotification(settings, analysis, allTransactions, "info", config.Notifications)
		if err != nil {
			return fmt.Errorf("error sending notifications: %w", err)
		}

		if len(successfulChannels) > 0 {
			log.Info().
				Str("channels", strings.Join(successfulChannels, "\n• ")).
				Msg("📱 Notifications sent successfully via:\n• " + strings.Join(successfulChannels, "\n• "))
		}
		log.Debug().Msg("Notifications sent successfully")

		// Update database
		if !config.DisableCache {
			if err := db.UpdateLastMessageTime(); err != nil {
				return fmt.Errorf("error updating last message time: %w", err)
			}
			log.Debug().Msg("Database updated with new message time")
		}
	} else {
		log.Debug().Msg("Notifications disabled, skipping")
		log.Info().Msg("ℹ️ Notifications disabled")
	}

	log.Debug().Msg("Finance tracker completed successfully")
	return nil
}

// getStringValue helper function is defined in settings.go
