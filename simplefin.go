package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Helper functions for transaction logging
func getTransactionStatus(tx Transaction) string {
	if tx.Pending != nil && *tx.Pending {
		return "pending"
	}
	return "posted"
}

func getTransactionTime(tx Transaction) string {
	if tx.TransactedAt != nil {
		return time.Unix(*tx.TransactedAt, 0).Format("2006-01-02 15:04:05")
	}
	return "not available"
}

func getTransactionsForPeriod(settings *Settings, startDate, endDate time.Time) ([]Account, error) {
	startTS := startDate.Unix()
	endTS := endDate.Unix()

	url := fmt.Sprintf("%s/accounts?start-date=%d&end-date=%d", settings.SimplefinBridgeURL, startTS, endTS)
	logrusLogger.WithField("url", url).Debug("Fetching transactions from SimpleFin bridge")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
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
		logrusLogger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"body":       string(body),
		}).Debug("API request failed")
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accountsResponse AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountsResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	logrusLogger.WithField("account_count", len(accountsResponse.Accounts)).Debug("Successfully decoded response")

	// Log account details for debugging
	for _, account := range accountsResponse.Accounts {
		logrusLogger.WithFields(logrus.Fields{
			"id":           account.ID,
			"name":         account.Name,
			"currency":     account.Currency,
			"balance":      account.Balance.String(),
			"balance_date": time.Unix(account.BalanceDate, 0).Format("2006-01-02 15:04:05"),
		}).Debug("Account details")
		if account.AvailableBalance != nil {
			logrusLogger.WithField("available_balance", account.AvailableBalance.String()).Debug("Available Balance")
		}

		// Log transaction details
		logrusLogger.WithField("transaction_count", len(account.Transactions)).Debug("Transactions")
		if len(account.Transactions) > 0 {
			for _, tx := range account.Transactions {
				logrusLogger.WithFields(logrus.Fields{
					"id":           tx.ID,
					"amount":       tx.Amount.String(),
					"status":       getTransactionStatus(tx),
					"posted":       time.Unix(tx.Posted, 0).Format("2006-01-02 15:04:05"),
					"transacted":   getTransactionTime(tx),
					"description":  tx.Description,
				}).Debug("Transaction details")
			}
		} else {
			logrusLogger.Debug("No transactions found")
		}
	}

	// Log any errors or messages from the API
	if len(accountsResponse.Errors) > 0 {
		for _, err := range accountsResponse.Errors {
			fmt.Printf("API Error: %s\n", err)
			if err := sendNtfyNotification(settings, fmt.Sprintf("API Error: %s", err), "warning"); err != nil {
				logrusLogger.WithError(err).Debug("Error sending notification")
				fmt.Printf("Error sending notification: %v\n", err)
			}
		}
	}

	if len(accountsResponse.XAPIMessage) > 0 {
		for _, msg := range accountsResponse.XAPIMessage {
			logrusLogger.WithField("message", msg).Debug("API Message")
		}
	}

	// Filter out accounts with zero balance
	var filteredAccounts []Account
	for _, account := range accountsResponse.Accounts {
		if float64(account.Balance) != 0 {
			logrusLogger.WithFields(logrus.Fields{
				"account_id": account.ID,
				"balance":    float64(account.Balance),
			}).Debug("Included account with non-zero balance")
			filteredAccounts = append(filteredAccounts, account)
		} else {
			logrusLogger.WithField("account_id", account.ID).Debug("Filtered out account with zero balance")
		}
	}
	logrusLogger.WithField("filtered_account_count", len(filteredAccounts)).Debug("Filtered accounts with non-zero balance")

	return filteredAccounts, nil
} 