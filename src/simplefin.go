package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// getTransactionStatus returns the status of a transaction (pending or posted)
func getTransactionStatus(tx Transaction) string {
	if tx.Pending != nil && *tx.Pending {
		return "pending"
	}
	return "posted"
}

// getTransactionTime returns the formatted time of a transaction
func getTransactionTime(tx Transaction) string {
	if tx.TransactedAt != nil {
		return time.Unix(*tx.TransactedAt, 0).Format("2006-01-02 15:04:05")
	}
	return "not available"
}

// getTransactionsForPeriod fetches transactions from the SimpleFin bridge for the specified date range
func getTransactionsForPeriod(settings *Settings, startDate, endDate time.Time) ([]Account, []string, error) {
	startTS := startDate.Unix()
	endTS := endDate.Unix()

	url := fmt.Sprintf("%s/accounts?start-date=%d&end-date=%d", settings.SimplefinBridgeURL, startTS, endTS)
	log.Debug().Str("url", url).Msg("Fetching transactions from SimpleFin bridge")

	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Debug().
			Int("status_code", resp.StatusCode).
			Str("body", string(body)).
			Msg("API request failed")
		return nil, nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accountsResponse AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountsResponse); err != nil {
		return nil, nil, fmt.Errorf("error decoding response: %w", err)
	}
	log.Debug().Int("account_count", len(accountsResponse.Accounts)).Msg("Successfully decoded response")

	// Log account details for debugging
	for _, account := range accountsResponse.Accounts {
		event := log.Debug().
			Str("id", account.ID).
			Str("name", account.Name).
			Str("balance", account.Balance.String()).
			Str("balance_date", time.Unix(account.BalanceDate, 0).Format("2006-01-02 15:04:05"))

		if account.Currency != nil {
			event.Str("currency", *account.Currency)
		}
		event.Msg("Account details")

		if account.AvailableBalance != nil {
			log.Debug().Str("available_balance", account.AvailableBalance.String()).Msg("Available Balance")
		}

		// Log transaction details
		log.Debug().Int("transaction_count", len(account.Transactions)).Msg("Transactions")
	}

	// Log any errors or messages from the API
	if len(accountsResponse.Errors) > 0 {
		for _, errMsg := range accountsResponse.Errors {
			log.Warn().Str("error", errMsg).Msg("API Error")
		}
	}

	if len(accountsResponse.XAPIMessage) > 0 {
		for _, msg := range accountsResponse.XAPIMessage {
			log.Debug().Str("message", msg).Msg("API Message")
		}
	}

	// Filter out accounts with zero balance
	var filteredAccounts []Account
	for _, account := range accountsResponse.Accounts {
		if float64(account.Balance) != 0 {
			log.Debug().
				Str("account_id", account.ID).
				Float64("balance", float64(account.Balance)).
				Msg("Included account with non-zero balance")
			filteredAccounts = append(filteredAccounts, account)
		} else {
			log.Debug().Str("account_id", account.ID).Msg("Filtered out account with zero balance")
		}
	}
	log.Debug().Int("filtered_account_count", len(filteredAccounts)).Msg("Filtered accounts with non-zero balance")

	return filteredAccounts, accountsResponse.Errors, nil
}
