package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
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

// callOpenRouter sends a request to the OpenRouter API and returns the raw content and model name
func callOpenRouter(settings *Settings, request OpenRouterRequest) (content string, model string, err error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, settings.OpenRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.OpenRouterAPIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 360 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error making request: %w", err)
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
		return "", "", fmt.Errorf("error reading response body: %w", err)
	}
	event.Str("body", string(bodyBytes))

	// Log the comprehensive debug message
	event.Msg("OpenRouter comprehensive response")

	// Create a new reader with the body bytes for further processing
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openRouterResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		return "", "", fmt.Errorf("error decoding response: %w", err)
	}

	log.Info().Str("model", openRouterResp.Model).Str("provider", openRouterResp.Provider).Msg(" └ OpenRouter response")

	// Check for error in the response
	if openRouterResp.Error != nil {
		return "", "", fmt.Errorf("OpenRouter API error: %s (code: %d)", openRouterResp.Error.Message, openRouterResp.Error.Code)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", "", fmt.Errorf("no response from OpenRouter")
	}

	responseContent := openRouterResp.Choices[0].Message.Content
	if responseContent == "" {
		return "", "", fmt.Errorf("received empty response from LLM")
	}

	return responseContent, openRouterResp.Model, nil
}

// getLLMResponse sends a prompt to the OpenRouter API and returns the analysis response
func getLLMResponse(settings *Settings, prompt string, isComplexAnalysis bool) (string, error) {
	models := strings.Split(settings.OpenRouterModel, ",")

	log.Debug().Msgf("Using models in order: %v", models)

	reqBody := OpenRouterRequest{
		Models:      models,
		Temperature: 0.4,
		Messages: []Message{
			{
				Role: "system",
				Content: `You are an expert financial analyst specializing in personal finance and spending pattern analysis for families.
You are analyzing spending for a Canadian family of 4: 2 adults, 2 daughters (born 2021 and 2024, currently ~4 and ~1 years old).
Your role is to provide clear, actionable insights from transaction data. Focus on identifying trends,
highlighting what's improving and what needs attention, and providing family-relevant suggestions.
Be concise, specific, and use the pre-calculated data provided — do not recalculate totals or percentages.`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Reasoning: Reasoning{
			Exclude: !isComplexAnalysis,
		},
	}

	content, model, err := callOpenRouter(settings, reqBody)
	if err != nil {
		return "", err
	}

	// Add model information as a small note at the bottom
	content = fmt.Sprintf("%s\n\n---\n*Generated by %s*", content, model)

	return content, nil
}

// getLLMResponseJSON sends a prompt to the OpenRouter API expecting a JSON response.
// Uses low temperature and disabled reasoning for deterministic categorization.
func getLLMResponseJSON(settings *Settings, systemPrompt string, userPrompt string) (string, error) {
	models := strings.Split(settings.OpenRouterModel, ",")

	reqBody := OpenRouterRequest{
		Models:      models,
		Temperature: 0.1,
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Reasoning: Reasoning{
			Exclude: true,
		},
	}

	content, _, err := callOpenRouter(settings, reqBody)
	if err != nil {
		return "", err
	}

	// Strip markdown code fences if present
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

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

// calculateBillingPeriods generates N billing periods ending at endDate, working backwards from billingDay.
// The last period (current) may be incomplete. The last 3 are marked as focus periods.
func calculateBillingPeriods(startDate, endDate time.Time, billingDay int) []BillingPeriod {
	// Find the current cycle start
	currentYear, currentMonth, _ := endDate.Date()
	var currentCycleStart time.Time
	if endDate.Day() >= billingDay {
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC)
	} else {
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	}

	// Build periods from newest to oldest
	var periods []BillingPeriod
	cycleStart := currentCycleStart

	for cycleStart.After(startDate) || cycleStart.Equal(startDate) {
		var periodEnd time.Time
		var isComplete bool

		if cycleStart.Equal(currentCycleStart) {
			// Current (most recent) period — ends at endDate, incomplete
			periodEnd = endDate
			isComplete = false
		} else {
			// Completed period — ends the day before the next cycle starts
			nextCycle := cycleStart.AddDate(0, 1, 0)
			periodEnd = nextCycle.Add(-24 * time.Hour)
			isComplete = true
		}

		periodStart := cycleStart
		if periodStart.Before(startDate) {
			periodStart = startDate
		}

		label := fmt.Sprintf("%s %d - %s %d", periodStart.Format("Jan"), periodStart.Day(), periodEnd.Format("Jan"), periodEnd.Day())

		periods = append([]BillingPeriod{{
			Label:      label,
			Start:      periodStart,
			End:        periodEnd,
			IsComplete: isComplete,
			IsFocus:    false, // set below
		}}, periods...)

		// Move to previous cycle
		cycleStart = cycleStart.AddDate(0, -1, 0)
	}

	// Mark the last 3 periods (or all if fewer) as focus
	for i := len(periods) - 1; i >= 0 && i >= len(periods)-3; i-- {
		periods[i].IsFocus = true
	}

	return periods
}

// calculatePeriodTotals calculates expense totals for each billing period.
func calculatePeriodTotals(transactions []Transaction, periods []BillingPeriod) []float64 {
	totals := make([]float64, len(periods))

	for _, txn := range transactions {
		timestamp := txn.TransactedAt
		if timestamp == nil {
			timestamp = &txn.Posted
		}
		txnDate := time.Unix(*timestamp, 0)

		for i, p := range periods {
			if !txnDate.Before(p.Start) && !txnDate.After(p.End) {
				totals[i] += -float64(txn.Amount) // Convert to positive
				break
			}
		}
	}

	return totals
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
func generateAnalysisPrompt(accounts []Account, transactions []Transaction, startDate, endDate time.Time, dateRangeType DateRangeType, billingDay int, filterResult *FilterResult, categories map[string]string) string {
	var transactionsFormatted string
	if categories != nil {
		transactionsFormatted = formatTransactionsWithCategories(transactions, categories)
	} else {
		transactionsFormatted = formatTransactions(transactions)
	}
	accountsFormatted := formatAccounts(accounts)

	totalExpenses := calculateTotalExpenses(transactions)

	// Determine if this is a multi-month analysis
	isMultiMonth := dateRangeType == DateRangeTypeCurrentAndLastMonth

	if isMultiMonth {
		return generateMultiPeriodPrompt(accounts, transactions, startDate, endDate, billingDay, filterResult, categories, accountsFormatted, transactionsFormatted, totalExpenses)
	}

	return generateSinglePeriodPrompt(accounts, transactions, startDate, endDate, filterResult, categories, accountsFormatted, transactionsFormatted, totalExpenses)
}

// generateSinglePeriodPrompt creates the prompt for single-period analysis
func generateSinglePeriodPrompt(accounts []Account, transactions []Transaction, startDate, endDate time.Time, filterResult *FilterResult, categories map[string]string, accountsFormatted, transactionsFormatted string, totalExpenses float64) string {
	calendarDays := int(endDate.Sub(startDate).Hours() / 24)
	transactionDays := countTransactionDays(transactions, startDate, endDate)

	dailyBurnRate := 0.0
	if transactionDays > 0 {
		dailyBurnRate = totalExpenses / float64(transactionDays)
	}
	monthlyProjection := dailyBurnRate * 30

	periodDescription := fmt.Sprintf("Billing Period: %s to %s (%d calendar days, %d transaction days)\nTotal Expenses: $%.2f\nDaily Burn Rate: $%.2f/day (based on transaction days)\nMonthly Projection: $%.2f (at current rate)", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), calendarDays, transactionDays, totalExpenses, dailyBurnRate, monthlyProjection)

	categoryBreakdownSection := ""
	if categories != nil {
		categoryBreakdownSection = buildSinglePeriodBreakdown(transactions, categories)
	}

	categoryDescription := "List the top 4-5 spending categories with their totals for this period"
	if categories != nil {
		categoryDescription = "Use the pre-calculated category totals below — do NOT re-categorize"
	}

	topExpensesFormatted := formatTopExpenses(transactions)
	filteredSection := buildFilteredSection(filterResult)

	return fmt.Sprintf(`## Financial Transaction Analysis — Family of 4

Household: 2 adults, 2 children (born 2021 and 2024, currently ~4 and ~1 years old).

%s
%s

Please create a concise report (~250 words) with:

### Summary
Provide a human-friendly overview of spending patterns during this period.

### Analysis Breakdown
1. **Total Expenses**: $%.2f
2. **Major Categories**: %s
3. **Top 10 Largest Expenses**:
%s4. **Key Insights**: 1-2 actionable insights referencing the burn rate and projection above.

Notes:
- Consider only outgoing expenses (ignore income/credits/refunds)
- Format all monetary values consistently (e.g., $1,234.56)
- Use CAD ($) for all amounts

Accounts Information:
%s

All Transactions:
%s
%s`, periodDescription, categoryBreakdownSection, totalExpenses, categoryDescription, topExpensesFormatted, accountsFormatted, transactionsFormatted, filteredSection)
}

// generateMultiPeriodPrompt creates the enhanced prompt for multi-period (5 billing cycles) analysis
func generateMultiPeriodPrompt(accounts []Account, transactions []Transaction, startDate, endDate time.Time, billingDay int, filterResult *FilterResult, categories map[string]string, accountsFormatted, transactionsFormatted string, totalExpenses float64) string {
	periods := calculateBillingPeriods(startDate, endDate, billingDay)
	periodTotals := calculatePeriodTotals(transactions, periods)

	// Build period summary table
	var periodSummary strings.Builder
	periodSummary.WriteString(fmt.Sprintf("Multi-Cycle Analysis (%d Billing Periods):\n", len(periods)))

	// Calculate per-period stats
	var completedBurnRates []float64
	for i, p := range periods {
		calDays := int(p.End.Sub(p.Start).Hours()/24) + 1
		txnDays := countTransactionDays(transactions, p.Start, p.End)
		burnRate := 0.0
		if txnDays > 0 {
			burnRate = periodTotals[i] / float64(txnDays)
		}

		status := "completed"
		if !p.IsComplete {
			status = "in progress"
		}

		focusMarker := ""
		if p.IsFocus {
			focusMarker = " [FOCUS]"
		}

		// Calculate change vs previous period
		changeStr := ""
		if i > 0 && periodTotals[i-1] > 0 {
			pctChange := ((periodTotals[i] - periodTotals[i-1]) / periodTotals[i-1]) * 100
			changeStr = fmt.Sprintf(" - Change: %+.1f%% (%s)", pctChange, formatChange(pctChange))
		}

		periodSummary.WriteString(fmt.Sprintf("- Period %d: %s (%d cal/%d txn days) - $%.2f [%s] - Burn: $%.2f/day%s%s\n",
			i+1, p.Label, calDays, txnDays, periodTotals[i], status, burnRate, changeStr, focusMarker))

		if p.IsComplete {
			completedBurnRates = append(completedBurnRates, burnRate)
		}
	}

	// Calculate averages
	fivePeriodAvg := 0.0
	if len(periodTotals) > 0 {
		sum := 0.0
		for _, t := range periodTotals {
			sum += t
		}
		fivePeriodAvg = sum / float64(len(periodTotals))
	}

	threePeriodAvg := 0.0
	focusCount := 0
	for i, p := range periods {
		if p.IsFocus {
			threePeriodAvg += periodTotals[i]
			focusCount++
		}
	}
	if focusCount > 0 {
		threePeriodAvg = threePeriodAvg / float64(focusCount)
	}

	avgCompletedBurnRate := 0.0
	if len(completedBurnRates) > 0 {
		sum := 0.0
		for _, r := range completedBurnRates {
			sum += r
		}
		avgCompletedBurnRate = sum / float64(len(completedBurnRates))
	}

	periodSummary.WriteString(fmt.Sprintf("- Grand Total: $%.2f\n", totalExpenses))
	periodSummary.WriteString(fmt.Sprintf("- %d-Period Average Monthly Spend: $%.2f\n", len(periods), fivePeriodAvg))
	periodSummary.WriteString(fmt.Sprintf("- 3-Period Average (recent trend): $%.2f\n", threePeriodAvg))
	periodSummary.WriteString(fmt.Sprintf("- Average Burn Rate (completed cycles): $%.2f/day\n", avgCompletedBurnRate))
	periodSummary.WriteString(fmt.Sprintf("- Monthly Projection: $%.2f (based on completed cycles)\n", avgCompletedBurnRate*30))

	// Build category breakdown for all periods
	categoryBreakdownSection := ""
	if categories != nil {
		categoryBreakdownSection = buildMultiPeriodBreakdown(transactions, categories, startDate, endDate, billingDay)
	}

	// Build trend highlights
	trendHighlights := ""
	if categories != nil {
		trendHighlights = buildTrendHighlights(transactions, categories, periods)
	}

	// Get top expenses from focus periods only
	var focusTransactions []Transaction
	for _, txn := range transactions {
		timestamp := txn.TransactedAt
		if timestamp == nil {
			timestamp = &txn.Posted
		}
		txnDate := time.Unix(*timestamp, 0)
		for _, p := range periods {
			if p.IsFocus && !txnDate.Before(p.Start) && !txnDate.After(p.End) {
				focusTransactions = append(focusTransactions, txn)
				break
			}
		}
	}
	topExpensesFormatted := formatTopExpenses(focusTransactions)

	filteredSection := buildFilteredSection(filterResult)

	return fmt.Sprintf(`## Financial Transaction Analysis — Family of 4

Household: 2 adults, 2 children (born 2021 and 2024, currently ~4 and ~1 years old).
Consider age-appropriate spending patterns (childcare, diapers, kids' activities, family dining, etc.)

%s
%s
%s

### Instructions

Analyze this family's spending. Write a concise report (~250 words) with:

1. **Summary**: Overview of the last 3 billing cycles. What's the overall trajectory?

2. **What's Improving**: Categories or habits trending in a positive direction.
   Use the pre-calculated trends — don't recalculate.

3. **What Needs Attention**: Categories growing faster than average, unusual
   spikes, or potential areas to cut back. Be specific with dollar amounts.

4. **Top Expenses**: The 10 largest individual charges across the last 3 periods:
%s
5. **Family-Specific Insights**: Note anything relevant to a family with young
   children (childcare costs, kids' activities, family-size grocery spending, etc.)

6. **Actionable Suggestions**: 2-3 concrete things to try next billing cycle,
   based on the data. Reference specific categories and amounts.

Notes:
- All calculations are pre-computed. Use them directly.
- Focus narrative on the LAST 3 periods only (the FOCUS periods).
- Earlier periods are provided for trend context only.
- The current period (last one) is partial/in-progress.
- Use CAD ($) for all amounts.
- Consider only outgoing expenses (ignore income/credits/refunds).

Accounts Information:
%s

All Transactions:
%s
%s`, periodSummary.String(), categoryBreakdownSection, trendHighlights, topExpensesFormatted, accountsFormatted, transactionsFormatted, filteredSection)
}

// buildFilteredSection creates the filtered transactions section for the prompt
func buildFilteredSection(filterResult *FilterResult) string {
	if filterResult == nil || filterResult.TotalFiltered == 0 {
		return ""
	}

	merchantMap := make(map[string]float64)
	for _, tx := range filterResult.FilteredTransactions {
		merchantMap[tx.Description] += float64(tx.Amount)
	}

	merchantSummary := ""
	for merchant, amount := range merchantMap {
		merchantSummary += fmt.Sprintf("   - %s: $%.2f\n", merchant, -amount)
	}

	return fmt.Sprintf(`
Filtered Transactions (Excluded from Analysis):
- Total Filtered: %d transactions
- Total Amount: $%.2f
- Top Merchants:
%s
Note: These transactions were filtered per user configuration and are NOT included in the analysis above.

`, filterResult.TotalFiltered, -float64(filterResult.TotalAmount), merchantSummary)
}

// formatTransactionsWithCategories formats transactions as a markdown table with a Category column
func formatTransactionsWithCategories(transactions []Transaction, categories map[string]string) string {
	if categories == nil {
		return formatTransactions(transactions)
	}

	var result string
	result += "| Description | Amount | Date | Category |\n"
	result += "|------------|---------|------|----------|\n"

	for _, txn := range transactions {
		timestamp := txn.TransactedAt
		if timestamp == nil {
			timestamp = &txn.Posted
		}
		date := time.Unix(*timestamp, 0).Format("2006-01-02")
		category := categories[txn.Description]
		if category == "" {
			category = "Uncategorized"
		}
		result += fmt.Sprintf("| %s | %.2f | %s | %s |\n", txn.Description, txn.Amount, date, category)
	}

	return result
}

// generateCategorizationPrompt builds a prompt for categorizing uncategorized merchant descriptions.
// Existing categories are provided as context examples for consistency.
func generateCategorizationPrompt(uncategorized []string, existingStore CategoryStore) string {
	var sb strings.Builder

	sb.WriteString("Categorize the following merchant/transaction descriptions into spending categories.\n\n")

	// Provide existing categories as context
	if len(existingStore) > 0 {
		sb.WriteString("Here are the existing categories and their merchants (use these as a guide for consistency):\n")
		// Sort category names for deterministic output
		catNames := make([]string, 0, len(existingStore))
		for cat := range existingStore {
			catNames = append(catNames, cat)
		}
		sort.Strings(catNames)

		for _, cat := range catNames {
			merchants := existingStore[cat]
			if len(merchants) > 5 {
				merchants = merchants[:5]
			}
			sb.WriteString(fmt.Sprintf("- %s: [%s]\n", cat, strings.Join(merchants, ", ")))
		}
		sb.WriteString("\nAssign each new merchant to an existing category above if it fits. Create a new category only if none fit.\n\n")
	}

	sb.WriteString("Merchant descriptions to categorize:\n")
	for _, desc := range uncategorized {
		sb.WriteString(fmt.Sprintf("- %s\n", desc))
	}

	sb.WriteString("\nRespond with valid JSON only in this exact format:\n")
	sb.WriteString(`{"categories": [{"description": "MERCHANT NAME", "category": "Category Name"}, ...]}`)
	sb.WriteString("\n")

	return sb.String()
}

// categorizeTransactions orchestrates transaction categorization using a persistent store and LLM.
// Returns a map of description -> category for all provided transactions.
func categorizeTransactions(settings *Settings, transactions []Transaction, categoriesPath string) (map[string]string, error) {
	// Load existing category store
	store, err := LoadCategories(categoriesPath)
	if err != nil {
		return nil, fmt.Errorf("error loading categories: %w", err)
	}

	// Build deduplicated set of descriptions and look up existing categories
	result := make(map[string]string)
	var uncategorized []string
	seen := make(map[string]bool)

	for _, tx := range transactions {
		desc := tx.Description
		if seen[desc] {
			continue
		}
		seen[desc] = true

		if cat, found := LookupCategory(store, desc); found {
			result[desc] = cat
		} else {
			uncategorized = append(uncategorized, desc)
		}
	}

	log.Info().
		Int("known", len(result)).
		Int("uncategorized", len(uncategorized)).
		Msg("🏷️  Category lookup results")

	// If everything is already categorized, skip the LLM call
	if len(uncategorized) == 0 {
		log.Info().Msg("🏷️  All merchants already categorized, skipping LLM call")
		return result, nil
	}

	// Generate categorization prompt with existing categories as context
	prompt := generateCategorizationPrompt(uncategorized, store)
	log.Debug().Str("prompt", prompt).Msg("Generated categorization prompt")

	// Call LLM for categorization
	log.Info().Int("count", len(uncategorized)).Msg("🏷️  Categorizing new merchants with LLM...")
	systemPrompt := "You are a financial transaction categorization system. Respond with valid JSON only. Categorize merchants into clear spending categories like Dining, Groceries, Transportation, Entertainment, Subscriptions, Shopping, Healthcare, Utilities, etc."
	jsonResp, err := getLLMResponseJSON(settings, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("error getting categorization from LLM: %w", err)
	}

	// Parse the response
	var catResp CategorizationResponse
	if err := json.Unmarshal([]byte(jsonResp), &catResp); err != nil {
		return nil, fmt.Errorf("error parsing categorization response: %w (response: %s)", err, jsonResp)
	}

	// Merge new categorizations into the store
	for _, tc := range catResp.Categories {
		result[tc.Description] = tc.Category
		AddToCategory(store, tc.Category, tc.Description)
	}

	// Save updated store
	if err := SaveCategories(categoriesPath, store); err != nil {
		log.Warn().Err(err).Msg("Failed to save categories file")
	} else {
		log.Info().Str("path", categoriesPath).Msg("🏷️  Saved updated categories")
	}

	return result, nil
}

// buildCategoryBreakdown pre-calculates per-category spending totals for the analysis prompt.
func buildCategoryBreakdown(transactions []Transaction, categories map[string]string, isMultiMonth bool, startDate, endDate time.Time, billingDay int) string {
	if categories == nil || len(categories) == 0 {
		return ""
	}

	if !isMultiMonth {
		return buildSinglePeriodBreakdown(transactions, categories)
	}

	return buildMultiPeriodBreakdown(transactions, categories, startDate, endDate, billingDay)
}

// buildSinglePeriodBreakdown creates a simple category spending summary for single-period analysis
func buildSinglePeriodBreakdown(transactions []Transaction, categories map[string]string) string {
	type catStats struct {
		total float64
		count int
	}
	stats := make(map[string]*catStats)

	for _, tx := range transactions {
		cat := categories[tx.Description]
		if cat == "" {
			cat = "Uncategorized"
		}
		if stats[cat] == nil {
			stats[cat] = &catStats{}
		}
		stats[cat].total += -float64(tx.Amount) // Convert to positive
		stats[cat].count++
	}

	// Sort by total descending
	type catEntry struct {
		name  string
		total float64
		count int
	}
	var entries []catEntry
	for name, s := range stats {
		entries = append(entries, catEntry{name, s.total, s.count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].total > entries[j].total
	})

	var sb strings.Builder
	sb.WriteString("\nPre-Calculated Category Spending:\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("- %s: $%.2f (%d transactions)\n", e.name, e.total, e.count))
	}

	return sb.String()
}

// buildMultiPeriodBreakdown creates a per-period category spending table for multi-month analysis (N periods)
func buildMultiPeriodBreakdown(transactions []Transaction, categories map[string]string, startDate, endDate time.Time, billingDay int) string {
	periods := calculateBillingPeriods(startDate, endDate, billingDay)
	if len(periods) == 0 {
		return ""
	}

	// Accumulate per-category per-period totals
	stats := make(map[string][]float64)
	allCats := make(map[string]bool)

	for _, tx := range transactions {
		cat := categories[tx.Description]
		if cat == "" {
			cat = "Uncategorized"
		}
		allCats[cat] = true
		if stats[cat] == nil {
			stats[cat] = make([]float64, len(periods))
		}

		timestamp := tx.TransactedAt
		if timestamp == nil {
			timestamp = &tx.Posted
		}
		txnDate := time.Unix(*timestamp, 0)
		amount := -float64(tx.Amount) // Convert to positive

		for i, p := range periods {
			if !txnDate.Before(p.Start) && !txnDate.After(p.End) {
				stats[cat][i] += amount
				break
			}
		}
	}

	// Sort categories by grand total descending
	type catEntry struct {
		name   string
		totals []float64
		grand  float64
	}
	var entries []catEntry
	for name := range allCats {
		s := stats[name]
		grand := 0.0
		for _, v := range s {
			grand += v
		}
		entries = append(entries, catEntry{name, s, grand})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].grand > entries[j].grand
	})

	// Build table header
	var sb strings.Builder
	sb.WriteString("\nPre-Calculated Category Spending by Period:\n\n")
	sb.WriteString("| Category |")
	for i, p := range periods {
		label := fmt.Sprintf(" P%d (%s-%s)", i+1, p.Start.Format("Jan"), p.End.Format("Jan"))
		if !p.IsComplete {
			label += "*"
		}
		sb.WriteString(label + " |")
	}
	sb.WriteString(" Trend (last 2 complete) |\n")

	sb.WriteString("|----------|")
	for range periods {
		sb.WriteString("--------|")
	}
	sb.WriteString("--------|\n")

	// Build rows
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("| %s |", e.name))
		for _, v := range e.totals {
			sb.WriteString(fmt.Sprintf(" $%.2f |", v))
		}

		// Calculate trend from the last two completed periods
		trend := "—"
		completedIndices := []int{}
		for i, p := range periods {
			if p.IsComplete {
				completedIndices = append(completedIndices, i)
			}
		}
		if len(completedIndices) >= 2 {
			prev := e.totals[completedIndices[len(completedIndices)-2]]
			curr := e.totals[completedIndices[len(completedIndices)-1]]
			if prev > 0 {
				pctChange := ((curr - prev) / prev) * 100
				trend = fmt.Sprintf("%+.1f%%", pctChange)
			}
		}
		sb.WriteString(fmt.Sprintf(" %s |\n", trend))
	}

	return sb.String()
}

// buildTrendHighlights identifies category trends across focus periods and formats them for the prompt.
func buildTrendHighlights(transactions []Transaction, categories map[string]string, periods []BillingPeriod) string {
	if len(periods) < 2 {
		return ""
	}

	// Calculate per-category per-period totals
	stats := make(map[string][]float64)
	for _, tx := range transactions {
		cat := categories[tx.Description]
		if cat == "" {
			cat = "Uncategorized"
		}
		if stats[cat] == nil {
			stats[cat] = make([]float64, len(periods))
		}

		timestamp := tx.TransactedAt
		if timestamp == nil {
			timestamp = &tx.Posted
		}
		txnDate := time.Unix(*timestamp, 0)

		for i, p := range periods {
			if !txnDate.Before(p.Start) && !txnDate.After(p.End) {
				stats[cat][i] += -float64(tx.Amount)
				break
			}
		}
	}

	// Identify focus period indices
	var focusIndices []int
	for i, p := range periods {
		if p.IsFocus {
			focusIndices = append(focusIndices, i)
		}
	}

	// Identify old (non-focus) and recent (focus) indices
	var oldIndices []int
	for i, p := range periods {
		if !p.IsFocus {
			oldIndices = append(oldIndices, i)
		}
	}

	var trendingUp, trendingDown, stable, newCats, disappearedCats []string

	for cat, totals := range stats {
		// Check for new/disappeared categories
		oldTotal := 0.0
		for _, idx := range oldIndices {
			oldTotal += totals[idx]
		}
		recentTotal := 0.0
		for _, idx := range focusIndices {
			recentTotal += totals[idx]
		}

		if oldTotal == 0 && recentTotal > 0 {
			newCats = append(newCats, fmt.Sprintf("%s ($%.0f)", cat, recentTotal))
			continue
		}
		if oldTotal > 0 && recentTotal == 0 {
			disappearedCats = append(disappearedCats, cat)
			continue
		}

		// Analyze trend across focus periods (need at least 2 completed focus periods)
		completedFocus := []int{}
		for _, idx := range focusIndices {
			if periods[idx].IsComplete {
				completedFocus = append(completedFocus, idx)
			}
		}

		if len(completedFocus) >= 2 {
			// Check consecutive increases/decreases
			allUp := true
			allDown := true
			for j := 1; j < len(completedFocus); j++ {
				prev := totals[completedFocus[j-1]]
				curr := totals[completedFocus[j]]
				if prev == 0 {
					allUp = false
					allDown = false
					break
				}
				change := ((curr - prev) / prev) * 100
				if change <= 10 {
					allUp = false
				}
				if change >= -10 {
					allDown = false
				}
			}

			// Check stability (within ±5% across completed focus periods)
			isStable := true
			if len(completedFocus) >= 2 {
				base := totals[completedFocus[0]]
				if base > 0 {
					for _, idx := range completedFocus[1:] {
						change := ((totals[idx] - base) / base) * 100
						if change > 5 || change < -5 {
							isStable = false
							break
						}
					}
				} else {
					isStable = false
				}
			}

			lastTwo := completedFocus[len(completedFocus)-2:]
			prev := totals[lastTwo[0]]
			curr := totals[lastTwo[1]]
			pctStr := ""
			if prev > 0 {
				pct := ((curr - prev) / prev) * 100
				pctStr = fmt.Sprintf(" %+.0f%%", pct)
			}

			if allUp {
				trendingUp = append(trendingUp, fmt.Sprintf("%s%s", cat, pctStr))
			} else if allDown {
				trendingDown = append(trendingDown, fmt.Sprintf("%s%s", cat, pctStr))
			} else if isStable {
				avgAmount := 0.0
				for _, idx := range completedFocus {
					avgAmount += totals[idx]
				}
				avgAmount /= float64(len(completedFocus))
				stable = append(stable, fmt.Sprintf("%s ($%.0f/mo avg)", cat, avgAmount))
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("\nPre-Calculated Trend Highlights:\n")

	if len(trendingUp) > 0 {
		sb.WriteString("- Categories trending UP (focus periods): " + strings.Join(trendingUp, ", ") + "\n")
	}
	if len(trendingDown) > 0 {
		sb.WriteString("- Categories trending DOWN (focus periods): " + strings.Join(trendingDown, ", ") + "\n")
	}
	if len(stable) > 0 {
		sb.WriteString("- Stable categories: " + strings.Join(stable, ", ") + "\n")
	}
	if len(newCats) > 0 {
		sb.WriteString("- New categories this period: " + strings.Join(newCats, ", ") + "\n")
	}
	if len(disappearedCats) > 0 {
		sb.WriteString("- Disappeared categories: " + strings.Join(disappearedCats, ", ") + "\n")
	}

	return sb.String()
}
