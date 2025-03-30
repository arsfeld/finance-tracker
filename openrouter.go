package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
	Error   *Error   `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func getLLMResponse(settings *Settings, prompt string) (string, error) {
	reqBody := OpenRouterRequest{
		Model: settings.OpenRouterModel,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", settings.OpenRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.OpenRouterAPIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Create a comprehensive debug message
	debugFields := logrus.Fields{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     resp.Header,
	}

	// Read the response body for logging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logrusLogger.WithFields(debugFields).WithError(err).Debug("OpenRouter response (error reading body)")
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	debugFields["body"] = string(bodyBytes)

	// Log the comprehensive debug message
	logrusLogger.WithFields(debugFields).Debug("OpenRouter comprehensive response")

	// Create a new reader with the body bytes for further processing
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openRouterResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	logrusLogger.WithField("response", openRouterResp).Debug("OpenRouter response")

	// Check for error in the response
	if openRouterResp.Error != nil {
		return "", fmt.Errorf("OpenRouter API error: %s (code: %d)", openRouterResp.Error.Message, openRouterResp.Error.Code)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenRouter")
	}

	return openRouterResp.Choices[0].Message.Content, nil
}

func formatTransactions(transactions []Transaction) string {
	var result string
	result += "| Description | Amount | Date |\n"
	result += "|------------|---------|------|\n"

	for _, txn := range transactions {
		timestamp := txn.TransactedAt
		if timestamp == nil {
			timestamp = &txn.Posted
		}
		date := time.Unix(*timestamp, 0).Format("2006-01-02")
		result += fmt.Sprintf("| %s | %.2f | %s |\n", txn.Description, txn.Amount, date)
	}

	return result
}

func formatAccounts(accounts []Account) string {
	var result string
	result += "| Account | Balance | Last Synced |\n"
	result += "|------------|---------|------|\n"
	
	for _, account := range accounts {
		result += fmt.Sprintf("| %s | %.2f | %s |\n", account.Name, account.Balance, time.Unix(account.BalanceDate, 0).Format("2006-01-02"))
	}

	return result
}

func generateAnalysisPrompt(accounts []Account, transactions []Transaction, startDate, endDate time.Time) string {
	transactionsFormatted := formatTransactions(transactions)
	accountsFormatted := formatAccounts(accounts)
	return fmt.Sprintf(`## Financial Transaction Analysis
Billing Period: %s to %s

I need a structured analysis of the provided financial transactions. Please create a concise report (max 150 words total) with the following sections:

### Summary
Provide a human-friendly overview of spending patterns during this period. Be specific about trends and notable observations.

### Analysis Breakdown
1. **Total Expenses**: ${{total}} (Sum of all purchases, excluding payments, credits, and refunds)
2. **Major Categories**: List the top 4-5 spending categories with their totals
   - Category 1: ${{amount}}
   - Category 2: ${{amount}}
   - ...
3. **Largest Expenses**: 
   - ${{expense 1}}: ${{amount}} at ${{merchant}} on ${{date}}
   - ${{expense 2}}: ${{amount}} at ${{merchant}} on ${{date}}
   - ${{expense 3}}: ${{amount}} at ${{merchant}} on ${{date}}
4. **Account Status**:
   - ${{account name}}: Balance ${{amount}}, Last synced ${{date}}
   - ...

Notes:
- Consider only outgoing expenses in your analysis (ignore incoming payments, credits, refunds)
- Format all monetary values consistently (e.g., $1,234.56)
- If a category has no transactions, indicate 'No spending in this category'

Accounts Information: 
%s

Transactions: 
%s`, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), accountsFormatted, transactionsFormatted)
} 