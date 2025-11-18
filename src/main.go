package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// RunConfig holds all configuration parameters for the run function
type RunConfig struct {
	Notifications        []string
	DisableNotifications bool
	Verbose              bool
	DateRange            string
	StartDate            string
	EndDate              string
	EnvFile              string
	Version              string
	MaxRetries           int
	RetryDelay           int
	BillingDay           int
	AllAccounts          bool
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "finance_tracker",
		Short: "Track your finances with AI-powered analysis",
		Long: fmt.Sprintf(`Finance Tracker is a powerful tool that analyzes your financial transactions using AI.
It connects to your SimpleFin account to fetch transactions and uses OpenAI's LLM to provide
insightful analysis of your spending patterns.

The tool supports multiple notification channels and can analyze transactions for various
time periods, providing detailed breakdowns of your spending habits.

By default, only credit card accounts are analyzed. Use --all-accounts to include all account types.

Version: %s

Example usage:
  finance_tracker                    # Analyze 3 billing cycles (default: cycles based on day 15)
  finance_tracker --billing-day 1    # Analyze 3 billing cycles starting from day 1
  finance_tracker --date-range current_month  # Analyze only current billing cycle
  finance_tracker --date-range last_month     # Analyze only previous billing cycle
  finance_tracker --all-accounts              # Include all account types (not just credit cards)
  finance_tracker --notifications ntfy        # Send notifications via ntfy
  finance_tracker --max-retries 5             # Set maximum number of retries for LLM calls
  finance_tracker --retry-delay 2             # Set initial retry delay in seconds`, GetVersion()),
		RunE: func(cmd *cobra.Command, args []string) error {
			notifications, _ := cmd.Flags().GetStringSlice("notifications")
			disableNotifications, _ := cmd.Flags().GetBool("disable-notifications")
			verbose, _ := cmd.Flags().GetBool("verbose")
			dateRange, _ := cmd.Flags().GetString("date-range")
			startDate, _ := cmd.Flags().GetString("start-date")
			endDate, _ := cmd.Flags().GetString("end-date")
			env_file, _ := cmd.Flags().GetString("env-file")
			maxRetries, _ := cmd.Flags().GetInt("max-retries")
			retryDelay, _ := cmd.Flags().GetInt("retry-delay")
			billingDay, _ := cmd.Flags().GetInt("billing-day")
			allAccounts, _ := cmd.Flags().GetBool("all-accounts")

			return run(RunConfig{
				Notifications:        notifications,
				DisableNotifications: disableNotifications,
				Verbose:              verbose,
				DateRange:            dateRange,
				StartDate:            startDate,
				EndDate:              endDate,
				EnvFile:              env_file,
				Version:              GetVersion(),
				MaxRetries:           maxRetries,
				RetryDelay:           retryDelay,
				BillingDay:           billingDay,
				AllAccounts:          allAccounts,
			})
		},
	}

	rootCmd.Flags().StringSliceP("notifications", "n", []string{"email", "ntfy"}, "Notification types to send")
	rootCmd.Flags().Bool("disable-notifications", false, "Disable all notifications")
	rootCmd.Flags().Bool("verbose", false, "Enable verbose logging")
	rootCmd.Flags().String("date-range", string(DateRangeTypeCurrentAndLastMonth), "Date range type (default: 3 billing cycles)")
	rootCmd.Flags().String("start-date", "", "Start date for custom range (YYYY-MM-DD)")
	rootCmd.Flags().String("end-date", "", "End date for custom range (YYYY-MM-DD)")
	rootCmd.Flags().String("env-file", ".env", "Path to environment file")
	rootCmd.Flags().Bool("version", false, "Show version information")
	rootCmd.Flags().Int("max-retries", 5, "Maximum number of retries for LLM calls")
	rootCmd.Flags().Int("retry-delay", 2, "Initial retry delay in seconds")
	rootCmd.Flags().Int("billing-day", 15, "Day of the month for the billing cycle start (1-28)")
	rootCmd.Flags().Bool("all-accounts", false, "Include all account types (default: credit cards only)")
	rootCmd.SetVersionTemplate(GetVersion() + "\n")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Error executing root command")
	}
}

// isCreditCard determines if an account is a credit card based on available data
func isCreditCard(account Account) bool {
	// First, check if the "extra" field contains type information
	// Note: The SimpleFin API spec doesn't standardize this, but some providers may include it
	// This would need to be verified with actual data from your SimpleFin provider

	// For now, use name-based heuristics as the primary method
	nameLower := strings.ToLower(account.Name)

	// Common credit card indicators
	creditCardKeywords := []string{
		"credit",
		"card",
		"visa",
		"mastercard",
		"amex",
		"american express",
		"discover",
		"rewards",
	}

	for _, keyword := range creditCardKeywords {
		if strings.Contains(nameLower, keyword) {
			return true
		}
	}

	return false
}

// matchesRule checks if a transaction description matches a filter rule
func matchesRule(description string, rule FilterRule) bool {
	descLower := strings.ToLower(description)
	patternLower := strings.ToLower(rule.Pattern)

	switch rule.MatchType {
	case MatchTypeSubstring:
		return strings.Contains(descLower, patternLower)
	case MatchTypePrefix:
		return strings.HasPrefix(descLower, patternLower)
	case MatchTypeSuffix:
		return strings.HasSuffix(descLower, patternLower)
	default:
		log.Warn().
			Str("match_type", string(rule.MatchType)).
			Msg("Unknown match type, treating as substring")
		return strings.Contains(descLower, patternLower)
	}
}

// filterTransactions filters out transactions based on the provided filter config
func filterTransactions(transactions []Transaction, filterConfig *FilterConfig) ([]Transaction, FilterResult) {
	if filterConfig == nil || len(filterConfig.ExcludedTransactions) == 0 {
		// No filtering configured, return all transactions
		return transactions, FilterResult{
			FilteredTransactions: []Transaction{},
			TotalFiltered:        0,
			TotalAmount:          0,
		}
	}

	var included []Transaction
	var filtered []Transaction
	var totalAmount Balance = 0

	for _, tx := range transactions {
		shouldFilter := false
		for _, rule := range filterConfig.ExcludedTransactions {
			if matchesRule(tx.Description, rule) {
				shouldFilter = true
				log.Debug().
					Str("description", tx.Description).
					Str("pattern", rule.Pattern).
					Str("match_type", string(rule.MatchType)).
					Float64("amount", float64(tx.Amount)).
					Msg("Transaction matched filter rule")
				break
			}
		}

		if shouldFilter {
			filtered = append(filtered, tx)
			totalAmount += tx.Amount
		} else {
			included = append(included, tx)
		}
	}

	result := FilterResult{
		FilteredTransactions: filtered,
		TotalFiltered:        len(filtered),
		TotalAmount:          totalAmount,
	}

	if len(filtered) > 0 {
		log.Info().
			Int("filtered_count", len(filtered)).
			Float64("total_amount", float64(totalAmount)).
			Int("remaining_count", len(included)).
			Msg("🚫 Filtered transactions based on rules")
	}

	return included, result
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

	// Load filter config if configured
	var filterConfig *FilterConfig
	if settings.FilterConfigPath != nil {
		log.Info().Str("config_path", *settings.FilterConfigPath).Msg("📋 Loading filter configuration...")
		fc, err := LoadFilterConfig(*settings.FilterConfigPath)
		if err != nil {
			log.Warn().
				Err(err).
				Str("config_path", *settings.FilterConfigPath).
				Msg("Failed to load filter config, continuing without filtering")
		} else {
			filterConfig = fc
		}
	}

	// Parse date range
	dateRangeType := DateRangeType(config.DateRange)

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

	// Filter accounts based on account type (credit cards only by default)
	if !config.AllAccounts {
		var creditCardAccounts []Account
		for _, account := range accounts {
			if isCreditCard(account) {
				creditCardAccounts = append(creditCardAccounts, account)
				log.Debug().
					Str("account_id", account.ID).
					Str("account_name", account.Name).
					Msg("Included credit card account")
			} else {
				log.Debug().
					Str("account_id", account.ID).
					Str("account_name", account.Name).
					Msg("Filtered out non-credit card account")
			}
		}

		// Warn if no credit card accounts found
		if len(creditCardAccounts) == 0 {
			log.Warn().
				Int("total_accounts", len(accounts)).
				Msg("No credit card accounts found. Use --all-accounts to include all account types.")
			return fmt.Errorf("no credit card accounts found (use --all-accounts to include all account types)")
		}

		log.Info().
			Int("credit_card_accounts", len(creditCardAccounts)).
			Int("total_accounts", len(accounts)).
			Msg("💳 Filtering to credit card accounts only")
		accounts = creditCardAccounts
	} else {
		log.Debug().Msg("Using all accounts (--all-accounts flag set)")
	}

	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	// Process accounts
	log.Info().Msg("💳 Accounts:")
	for _, account := range accounts {
		log.Info().Str("account_name", account.Name).Str("account_id", account.ID).Msg("•")
		syncTime := time.Unix(account.BalanceDate, 0).Format("2006-01-02 15:04:05")
		log.Info().Str("sync_time", syncTime).
			Str("balance", account.Balance.String()).
			Str("transactions", strconv.Itoa(len(account.Transactions))).
			Msg("└")
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

	// Apply merchant/description filtering if configured
	var filterResult FilterResult
	allTransactions, filterResult = filterTransactions(allTransactions, filterConfig)

	if len(allTransactions) == 0 {
		return fmt.Errorf("no transactions found")
	}

	// Process transactions with AI
	log.Info().Msg("🤖 Analyzing transactions with AI...")
	prompt := generateAnalysisPrompt(accounts, allTransactions, billingStart, billingEnd, dateRangeType, config.BillingDay, &filterResult)
	log.Debug().Str("prompt", prompt).Msg("Generated analysis prompt")

	// Determine if this is complex analysis requiring reasoning
	isComplexAnalysis := dateRangeType == DateRangeTypeCurrentAndLastMonth ||
		dateRangeType == DateRangeTypeLast3Months ||
		dateRangeType == DateRangeTypeCurrentYear ||
		dateRangeType == DateRangeTypeLastYear

	// Get LLM response with retry
	analysis, err := retryWithBackoff(
		func() (string, error) {
			return getLLMResponse(settings, prompt, isComplexAnalysis)
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
	} else {
		log.Debug().Msg("Notifications disabled, skipping")
		log.Info().Msg("ℹ️ Notifications disabled")
	}

	log.Debug().Msg("Finance tracker completed successfully")
	return nil
}

// getStringValue helper function is defined in settings.go
