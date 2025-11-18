package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// OpenRouterRequest represents a request to the OpenRouter API
type OpenRouterRequest struct {
	Model       string    `json:"model"`
	Models      []string  `json:"models"`
	Messages    []Message `json:"messages"`
	Reasoning   Reasoning `json:"reasoning"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message represents a message in the OpenRouter API request/response
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Reasoning struct {
	Exclude bool `json:"exclude"`
}

// OpenRouterResponse represents a response from the OpenRouter API
type OpenRouterResponse struct {
	Id       string   `json:"id"`
	Choices  []Choice `json:"choices"`
	Error    *Error   `json:"error,omitempty"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
	Object   string   `json:"object"`
	Created  int64    `json:"created"`
	Usage    Usage    `json:"usage"`
}

// Choice represents a choice in the OpenRouter API response
type Choice struct {
	Message            Message `json:"message"`
	FinishReason       string  `json:"finish_reason"`
	NativeFinishReason string  `json:"native_finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Error represents an error in the OpenRouter API response
type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// shuffleModels randomly shuffles the models slice
func shuffleModels(models []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(models), func(i, j int) {
		models[i], models[j] = models[j], models[i]
	})
}

// getLLMResponse sends a prompt to the OpenRouter API and returns the response
func getLLMResponse(settings *Settings, prompt string, isComplexAnalysis bool) (string, error) {
	models := strings.Split(settings.OpenRouterModel, ",")

	log.Debug().Msgf("Using models in order: %v", models)

	// System message to prime the model with financial analyst role
	systemMessage := Message{
		Role: "system",
		Content: `You are an expert financial analyst specializing in personal finance and spending pattern analysis.
Your role is to provide clear, actionable insights from transaction data. Focus on identifying trends,
categorizing expenses accurately, and highlighting notable patterns or concerns. Be concise, specific,
and use data to support your observations.`,
	}

	reqBody := OpenRouterRequest{
		Models:      models,
		Temperature: 0.4, // Lower temperature for more consistent, factual responses
		Messages: []Message{
			systemMessage,
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Reasoning: Reasoning{
			Exclude: !isComplexAnalysis, // Enable reasoning for complex analysis (multi-month, etc.)
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, settings.OpenRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.OpenRouterAPIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 360 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Create a comprehensive debug message
	event := log.Debug().
		Int("status_code", resp.StatusCode).
		Str("status", resp.Status).
		Interface("headers", resp.Header)

	// Read the response body for logging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		event.Err(err).Msg("OpenRouter response (error reading body)")
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	event.Str("body", string(bodyBytes))

	// Log the comprehensive debug message
	event.Msg("OpenRouter comprehensive response")

	// Create a new reader with the body bytes for further processing
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openRouterResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	log.Info().Str("model", openRouterResp.Model).Str("provider", openRouterResp.Provider).Msg(" └ OpenRouter response")

	// Check for error in the response
	if openRouterResp.Error != nil {
		return "", fmt.Errorf("OpenRouter API error: %s (code: %d)", openRouterResp.Error.Message, openRouterResp.Error.Code)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenRouter")
	}

	content := openRouterResp.Choices[0].Message.Content
	if content == "" {
		return "", fmt.Errorf("received empty analysis from LLM")
	}

	// Add model information as a small note at the bottom
	content = fmt.Sprintf("%s\n\n---\n*Generated by %s*", content, openRouterResp.Model)

	return content, nil
}

// formatTransactions formats the transactions as a markdown table
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

// formatAccounts formats the accounts as a markdown table
func formatAccounts(accounts []Account) string {
	var result string
	result += "| Account | Balance | Last Synced |\n"
	result += "|------------|---------|------|\n"

	for _, account := range accounts {
		result += fmt.Sprintf("| %s | %.2f | %s |\n", account.Name, account.Balance, time.Unix(account.BalanceDate, 0).Format("2006-01-02"))
	}

	return result
}

// getTopExpenses returns the top N expenses sorted by amount (most negative first)
func getTopExpenses(transactions []Transaction, n int) []Transaction {
	if len(transactions) == 0 {
		return []Transaction{}
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]Transaction, len(transactions))
	copy(sorted, transactions)

	// Sort by amount (most negative first)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Amount < sorted[i].Amount {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Return top N
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// formatTopExpenses formats the top expenses as a bulleted list
func formatTopExpenses(transactions []Transaction) string {
	topExpenses := getTopExpenses(transactions, 10)
	var result string

	for _, txn := range topExpenses {
		timestamp := txn.TransactedAt
		if timestamp == nil {
			timestamp = &txn.Posted
		}
		date := time.Unix(*timestamp, 0).Format("Jan 2")
		result += fmt.Sprintf("   - $%.2f at %s on %s\n", -txn.Amount, txn.Description, date)
	}

	return result
}

// formatChange formats a percentage change with a descriptive label
func formatChange(percentChange float64) string {
	if percentChange > 0 {
		return "increase"
	} else if percentChange < 0 {
		return "decrease"
	}
	return "no change"
}

// calculateBillingPeriodTotals calculates expense totals for each billing period in a multi-month analysis (3 periods)
func calculateBillingPeriodTotals(transactions []Transaction, split1 time.Time, split2 time.Time) (float64, float64, float64) {
	var period1Total, period2Total, period3Total float64

	for _, txn := range transactions {
		timestamp := txn.TransactedAt
		if timestamp == nil {
			timestamp = &txn.Posted
		}
		txnDate := time.Unix(*timestamp, 0)

		if txnDate.Before(split1) {
			period1Total += -float64(txn.Amount) // Convert to positive - oldest period
		} else if txnDate.Before(split2) {
			period2Total += -float64(txn.Amount) // Convert to positive - middle period
		} else {
			period3Total += -float64(txn.Amount) // Convert to positive - current period
		}
	}

	return period1Total, period2Total, period3Total
}

// calculateTotalExpenses calculates the total expenses for all transactions
func calculateTotalExpenses(transactions []Transaction) float64 {
	var total float64
	for _, txn := range transactions {
		total += -float64(txn.Amount) // Convert to positive
	}
	return total
}

// generateAnalysisPrompt generates a prompt for the AI to analyze transactions
func generateAnalysisPrompt(accounts []Account, transactions []Transaction, startDate, endDate time.Time, dateRangeType DateRangeType, billingDay int, filterResult *FilterResult) string {
	transactionsFormatted := formatTransactions(transactions)
	accountsFormatted := formatAccounts(accounts)
	topExpensesFormatted := formatTopExpenses(transactions)

	// Calculate period details
	periodDays := int(endDate.Sub(startDate).Hours() / 24)
	totalExpenses := calculateTotalExpenses(transactions)

	// Calculate daily burn rate
	dailyBurnRate := 0.0
	if periodDays > 0 {
		dailyBurnRate = totalExpenses / float64(periodDays)
	}

	// Calculate monthly projection (assuming 30-day month)
	monthlyProjection := dailyBurnRate * 30

	// Determine if this is a multi-month analysis
	isMultiMonth := dateRangeType == DateRangeTypeCurrentAndLastMonth
	periodDescription := fmt.Sprintf("Billing Period: %s to %s (%d days)\nTotal Expenses: $%.2f\nDaily Burn Rate: $%.2f/day\nMonthly Projection: $%.2f (at current rate)", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), periodDays, totalExpenses, dailyBurnRate, monthlyProjection)

	summaryInstructions := "Provide a human-friendly overview of spending patterns during this period. Be specific about trends and notable observations."
	trendAnalysisSection := ""

	if isMultiMonth {
		// Calculate the split points between billing periods (3 periods total)
		currentYear, currentMonth, _ := endDate.Date()
		var currentCycleStart time.Time
		if endDate.Day() >= billingDay {
			currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC)
		} else {
			currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		}
		previousCycleStart := currentCycleStart.AddDate(0, -1, 0)

		// Calculate totals for each of the 3 billing periods
		period1Total, period2Total, period3Total := calculateBillingPeriodTotals(transactions, previousCycleStart, currentCycleStart)

		// Period 1 (oldest completed cycle)
		period1Start := startDate
		period1End := previousCycleStart.Add(-24 * time.Hour)
		period1Days := int(period1End.Sub(period1Start).Hours()/24) + 1

		// Period 2 (previous completed cycle)
		period2Start := previousCycleStart
		period2End := currentCycleStart.Add(-24 * time.Hour)
		period2Days := int(period2End.Sub(period2Start).Hours()/24) + 1

		// Period 3 (current incomplete cycle)
		period3Start := currentCycleStart
		period3End := endDate
		period3Days := int(period3End.Sub(period3Start).Hours()/24) + 1

		// Calculate percentage changes
		period2Change := 0.0
		if period1Total > 0 {
			period2Change = ((period2Total - period1Total) / period1Total) * 100
		}
		period3Change := 0.0
		if period2Total > 0 {
			period3Change = ((period3Total - period2Total) / period2Total) * 100
		}

		// Format cycle labels with month names
		cycle1Label := fmt.Sprintf("%s %d - %s %d", period1Start.Format("Jan"), period1Start.Day(), period1End.Format("Jan"), period1End.Day())
		cycle2Label := fmt.Sprintf("%s %d - %s %d", period2Start.Format("Jan"), period2Start.Day(), period2End.Format("Jan"), period2End.Day())
		cycle3Label := fmt.Sprintf("%s %d - %s %d", period3Start.Format("Jan"), period3Start.Day(), period3End.Format("Jan"), period3End.Day())

		// Calculate burn rates for each period
		period1BurnRate := 0.0
		if period1Days > 0 {
			period1BurnRate = period1Total / float64(period1Days)
		}
		period2BurnRate := 0.0
		if period2Days > 0 {
			period2BurnRate = period2Total / float64(period2Days)
		}
		period3BurnRate := 0.0
		if period3Days > 0 {
			period3BurnRate = period3Total / float64(period3Days)
		}

		// Calculate average burn rate from completed cycles
		avgCompletedBurnRate := 0.0
		if period1Days > 0 && period2Days > 0 {
			avgCompletedBurnRate = (period1BurnRate + period2BurnRate) / 2
		}

		// Monthly projection based on average of completed cycles
		completedMonthlyProjection := avgCompletedBurnRate * 30

		periodDescription = fmt.Sprintf(`Multi-Cycle Analysis (3 Billing Periods):
- %s: %s to %s (%d days) - $%.2f [completed] - Burn rate: $%.2f/day
- %s: %s to %s (%d days) - $%.2f [completed] - Burn rate: $%.2f/day - Change: %.1f%% (%s)
- %s: %s to %s (%d days) - $%.2f [in progress] - Burn rate: $%.2f/day - Change: %.1f%% (%s)
- Grand Total: $%.2f
- Average Burn Rate (completed cycles): $%.2f/day
- Monthly Projection: $%.2f (based on completed cycles)`,
			cycle1Label, period1Start.Format("2006-01-02"), period1End.Format("2006-01-02"), period1Days, period1Total, period1BurnRate,
			cycle2Label, period2Start.Format("2006-01-02"), period2End.Format("2006-01-02"), period2Days, period2Total, period2BurnRate, period2Change, formatChange(period2Change),
			cycle3Label, period3Start.Format("2006-01-02"), period3End.Format("2006-01-02"), period3Days, period3Total, period3BurnRate, period3Change, formatChange(period3Change),
			totalExpenses, avgCompletedBurnRate, completedMonthlyProjection)

		summaryInstructions = fmt.Sprintf("Provide a human-friendly overview of spending patterns across the 3 billing cycles (%s, %s, %s). Focus on comparing the two completed cycles and note that the current cycle is still in progress. Use the provided billing period totals for accurate comparisons.", cycle1Label, cycle2Label, cycle3Label)
		trendAnalysisSection = fmt.Sprintf(`4. **📈 Spending Trends** (use pre-calculated totals above):
   - Compare the two completed cycles (%s vs %s)
   - Note current cycle (%s) progress relative to completed cycles
   - Identify which categories changed significantly between cycles
5. `, cycle1Label, cycle2Label, cycle3Label)
	} else {
		trendAnalysisSection = "4. "
	}

	// Determine category description based on analysis type
	categoryDescription := "List the top 4-5 spending categories with their totals for the LATEST billing cycle only"
	if !isMultiMonth {
		categoryDescription = "List the top 4-5 spending categories with their totals for this period"
	}

	// Add filtered transactions section if any were filtered
	filteredSection := ""
	if filterResult != nil && filterResult.TotalFiltered > 0 {
		// Get unique merchant names from filtered transactions
		merchantMap := make(map[string]float64)
		for _, tx := range filterResult.FilteredTransactions {
			merchantMap[tx.Description] += float64(tx.Amount)
		}

		// Build merchant summary
		merchantSummary := ""
		for merchant, amount := range merchantMap {
			merchantSummary += fmt.Sprintf("   - %s: $%.2f\n", merchant, -amount)
		}

		filteredSection = fmt.Sprintf(`
Filtered Transactions (Excluded from Analysis):
- Total Filtered: %d transactions
- Total Amount: $%.2f
- Top Merchants:
%s
Note: These transactions were filtered per user configuration and are NOT included in the analysis above.

`, filterResult.TotalFiltered, -float64(filterResult.TotalAmount), merchantSummary)
	}

	return fmt.Sprintf(`## Financial Transaction Analysis
%s

I need a structured analysis of the provided financial transactions. Use emojis to make the report more engaging.
Please create a concise report (max 180 words total) with the following sections:

### Summary
%s

### Analysis Breakdown
1. **Total Expenses**: Per billing cycle totals shown above
2. **Major Categories** (latest cycle only): %s
   - Category 1: ${{amount}}
   - Category 2: ${{amount}}
   - ...
3. **Top 10 Largest Expenses** (across all periods):
%s%s**🔍 Key Insights**: Provide 1-2 actionable insights such as:
   - Reference the daily burn rate and monthly projection provided above
   - Notable patterns or anomalies worth mentioning
   - Recurring charges or subscription reminders if relevant

Notes:
- Consider only outgoing expenses in your analysis (ignore incoming payments, credits, refunds)
- Format all monetary values consistently (e.g., $1,234.56)
- Keep insights brief and actionable
- Use the pre-calculated burn rates and projections provided in the period description above
- Category totals should be for the LATEST billing cycle only (not combined across periods)
- If a category has no transactions, indicate 'No spending in this category'

Accounts Information:
%s

All Transactions:
%s
%s`, periodDescription, summaryInstructions, categoryDescription, topExpensesFormatted, trendAnalysisSection, accountsFormatted, transactionsFormatted, filteredSection)
}
