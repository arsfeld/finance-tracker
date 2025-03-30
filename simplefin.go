package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
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
func getTransactionsForPeriod(settings *Settings, startDate, endDate time.Time) ([]Account, error) {
	startTS := startDate.Unix()
	endTS := endDate.Unix()

	url := fmt.Sprintf("%s/accounts?start-date=%d&end-date=%d", settings.SimplefinBridgeURL, startTS, endTS)
	logger.WithField("url", url).Debug("Fetching transactions from SimpleFin bridge")

	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Debug("API request failed")
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accountsResponse AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountsResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	logger.WithField("account_count", len(accountsResponse.Accounts)).Debug("Successfully decoded response")

	// Log account details for debugging
	for _, account := range accountsResponse.Accounts {
		logFields := logrus.Fields{
			"id":           account.ID,
			"name":         account.Name,
			"balance":      account.Balance.String(),
			"balance_date": time.Unix(account.BalanceDate, 0).Format("2006-01-02 15:04:05"),
		}
		
		if account.Currency != nil {
			logFields["currency"] = *account.Currency
		}
		
		logger.WithFields(logFields).Debug("Account details")
		
		if account.AvailableBalance != nil {
			logger.WithField("available_balance", account.AvailableBalance.String()).Debug("Available Balance")
		}

		// Log transaction details
		logger.WithField("transaction_count", len(account.Transactions)).Debug("Transactions")
		if len(account.Transactions) > 0 {
			for _, tx := range account.Transactions {
				logger.WithFields(logrus.Fields{
					"id":          tx.ID,
					"amount":      tx.Amount.String(),
					"status":      getTransactionStatus(tx),
					"posted":      time.Unix(tx.Posted, 0).Format("2006-01-02 15:04:05"),
					"transacted":  getTransactionTime(tx),
					"description": tx.Description,
				}).Debug("Transaction details")
			}
		} else {
			logger.Debug("No transactions found")
		}
	}

	// Log any errors or messages from the API
	if len(accountsResponse.Errors) > 0 {
		for _, errMsg := range accountsResponse.Errors {
			logger.Warnf("API Error: %s", errMsg)
			if err := sendNtfyNotification(settings, fmt.Sprintf("API Error: %s", errMsg), "warning"); err != nil {
				logger.WithError(err).Debug("Error sending notification")
				logger.Errorf("Error sending notification: %v", err)
			}
		}
	}

	if len(accountsResponse.XAPIMessage) > 0 {
		for _, msg := range accountsResponse.XAPIMessage {
			logger.WithField("message", msg).Debug("API Message")
		}
	}

	// Filter out accounts with zero balance
	var filteredAccounts []Account
	for _, account := range accountsResponse.Accounts {
		if float64(account.Balance) != 0 {
			logger.WithFields(logrus.Fields{
				"account_id": account.ID,
				"balance":    float64(account.Balance),
			}).Debug("Included account with non-zero balance")
			filteredAccounts = append(filteredAccounts, account)
		} else {
			logger.WithField("account_id", account.ID).Debug("Filtered out account with zero balance")
		}
	}
	logger.WithField("filtered_account_count", len(filteredAccounts)).Debug("Filtered accounts with non-zero balance")

	return filteredAccounts, nil
}
