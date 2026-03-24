package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/config"
	"finance_tracker/internal/store"
)

type ChatHandler struct {
	cfg           *config.Config
	txnStore      *store.TransactionStore
	acctStore     *store.AccountStore
	catStore      *store.CategoryStore
	analysisStore *store.AnalysisStore
	budgetStore   *store.BudgetStore
	events        *EventHub
}

func NewChatHandler(
	cfg *config.Config,
	txns *store.TransactionStore,
	accts *store.AccountStore,
	cats *store.CategoryStore,
	analyses *store.AnalysisStore,
	budgets *store.BudgetStore,
	events *EventHub,
) *ChatHandler {
	return &ChatHandler{
		cfg:           cfg,
		txnStore:      txns,
		acctStore:     accts,
		catStore:      cats,
		analysisStore: analyses,
		budgetStore:   budgets,
		events:        events,
	}
}

// Chat request/response types matching OpenAI/OpenRouter format.

type chatRequest struct {
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type llmRequest struct {
	Models      []string      `json:"models"`
	Messages    []chatMessage `json:"messages"`
	Tools       []toolDef     `json:"tools"`
	Temperature float64       `json:"temperature"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function toolDefFunc `json:"function"`
}

type toolDefFunc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type llmResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

var chatTools = []toolDef{
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "search_transactions",
			Description: "Search transactions by description, date range, or category. Returns matching transactions.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search":   map[string]string{"type": "string", "description": "Text to search in transaction descriptions"},
					"category": map[string]string{"type": "string", "description": "Filter by category name"},
					"days":     map[string]string{"type": "integer", "description": "Number of days back to search (default: 30)"},
					"limit":    map[string]string{"type": "integer", "description": "Max results to return (default: 20)"},
				},
			},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "get_spending_summary",
			Description: "Get total spending broken down by category for a time period.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"days": map[string]string{"type": "integer", "description": "Number of days back to summarize (default: 30)"},
				},
			},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "set_transaction_category",
			Description: "Set or change the category of a transaction. Use apply_to_merchant=true to apply to all transactions from the same merchant.",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"transaction_id", "category"},
				"properties": map[string]interface{}{
					"transaction_id":    map[string]string{"type": "string", "description": "The transaction ID"},
					"category":          map[string]string{"type": "string", "description": "The category to assign"},
					"apply_to_merchant": map[string]string{"type": "boolean", "description": "Apply to all transactions from this merchant (default: true)"},
				},
			},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "bulk_set_categories",
			Description: "Set categories for multiple merchants at once. Each entry maps a merchant description to a category.",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"categories"},
				"properties": map[string]interface{}{
					"categories": map[string]interface{}{
						"type":        "array",
						"description": "List of merchant-to-category mappings",
						"items": map[string]interface{}{
							"type":     "object",
							"required": []string{"merchant", "category"},
							"properties": map[string]interface{}{
								"merchant": map[string]string{"type": "string", "description": "Merchant description (exact match)"},
								"category": map[string]string{"type": "string", "description": "Category to assign"},
							},
						},
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "get_accounts",
			Description: "List all financial accounts with their balances.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "get_latest_analysis",
			Description: "Get the most recent spending analysis report.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "get_budget_status",
			Description: "Get all budgets with current spending progress for the current billing period, including unbudgeted categories.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "set_budget",
			Description: "Create or update a monthly spending budget for a category. Always confirm with the user before calling this.",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"category", "amount"},
				"properties": map[string]interface{}{
					"category": map[string]string{"type": "string", "description": "The category name (e.g. Groceries, Dining)"},
					"amount":   map[string]string{"type": "number", "description": "The monthly budget amount in dollars (must be > 0)"},
				},
			},
		},
	},
	{
		Type: "function",
		Function: toolDefFunc{
			Name:        "delete_budget",
			Description: "Remove a budget for a category. Always confirm with the user before calling this.",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"category"},
				"properties": map[string]interface{}{
					"category": map[string]string{"type": "string", "description": "The category name to remove the budget for"},
				},
			},
		},
	},
}

const chatSystemPrompt = `You are a helpful financial assistant for a personal finance tracker. You can search transactions, view spending summaries, change transaction categories, retrieve analysis reports, and manage budgets.

When the user asks about their spending, use the tools to look up real data before answering.
When categorizing transactions, use common categories like: Groceries, Dining, Transportation, Gas, Entertainment, Shopping, Subscriptions, Healthcare, Utilities, Insurance, Housing, Childcare, Education, Clothing, Personal Care, Fitness, Travel, Gifts, Coffee, Parking, Fees, Other.
Be concise and helpful. Format monetary values as $X.XX in CAD.
When bulk categorizing, group similar merchants together and confirm with the user what you plan to do before executing.

You can also view and manage the user's budgets:
- Use get_budget_status to see current budgets and spending progress
- Use set_budget to create or update a budget for a category
- Use delete_budget to remove a budget
Always describe what you plan to do and ask for confirmation before modifying or deleting budgets.`

func (h *ChatHandler) Handle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2 MB limit
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON")
		return
	}

	if h.cfg.OpenRouterURL == "" || h.cfg.OpenRouterAPIKey == "" || h.cfg.OpenRouterModel == "" {
		WriteError(w, http.StatusServiceUnavailable, "NOT_CONFIGURED", "OpenRouter not configured in .env")
		return
	}

	// Prepend system prompt.
	messages := append([]chatMessage{{Role: "system", Content: chatSystemPrompt}}, req.Messages...)

	// Tool-calling loop: call LLM, execute tools, repeat until we get a text response.
	for i := 0; i < 10; i++ {
		resp, err := h.callLLM(messages)
		if err != nil {
			log.Error().Err(err).Msg("Chat LLM call failed")
			WriteError(w, http.StatusInternalServerError, "LLM_ERROR", err.Error())
			return
		}

		if len(resp.Choices) == 0 {
			WriteError(w, http.StatusInternalServerError, "LLM_ERROR", "No response from LLM")
			return
		}

		msg := resp.Choices[0].Message
		msg.Role = "assistant"

		// If no tool calls, return the final text response.
		if len(msg.ToolCalls) == 0 {
			WriteData(w, map[string]interface{}{
				"message": msg.Content,
				"model":   resp.Model,
			})
			return
		}

		// Add assistant message with tool calls to history.
		messages = append(messages, msg)

		// Execute each tool call and add results.
		for _, tc := range msg.ToolCalls {
			result := h.executeTool(r.Context(), tc.Function.Name, tc.Function.Arguments)
			messages = append(messages, chatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	WriteError(w, http.StatusInternalServerError, "LOOP_LIMIT", "Too many tool call iterations")
}

func (h *ChatHandler) callLLM(messages []chatMessage) (*llmResponse, error) {
	models := strings.Split(h.cfg.OpenRouterModel, ",")
	for i := range models {
		models[i] = strings.TrimSpace(models[i])
	}

	body := llmRequest{
		Models:      models,
		Messages:    messages,
		Tools:       chatTools,
		Temperature: 0.3,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, h.cfg.OpenRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.OpenRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(respBytes))
	}

	var llmResp llmResponse
	if err := json.Unmarshal(respBytes, &llmResp); err != nil {
		return nil, fmt.Errorf("decode LLM response: %w", err)
	}

	if llmResp.Error != nil {
		return nil, fmt.Errorf("LLM error: %s", llmResp.Error.Message)
	}

	return &llmResp, nil
}

func (h *ChatHandler) executeTool(ctx context.Context, name, argsJSON string) string {
	log.Info().Str("tool", name).Str("args", argsJSON).Msg("Executing chat tool")

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %s", err)
	}

	switch name {
	case "search_transactions":
		return h.toolSearchTransactions(ctx, args)
	case "get_spending_summary":
		return h.toolSpendingSummary(ctx, args)
	case "set_transaction_category":
		return h.toolSetCategory(ctx, args)
	case "bulk_set_categories":
		return h.toolBulkSetCategories(ctx, args)
	case "get_accounts":
		return h.toolGetAccounts(ctx)
	case "get_latest_analysis":
		return h.toolGetLatestAnalysis(ctx)
	case "get_budget_status":
		return h.toolGetBudgetStatus(ctx)
	case "set_budget":
		return h.toolSetBudget(ctx, args)
	case "delete_budget":
		return h.toolDeleteBudget(ctx, args)
	default:
		return fmt.Sprintf("Unknown tool: %s", name)
	}
}

func (h *ChatHandler) toolSearchTransactions(ctx context.Context, args map[string]interface{}) string {
	days := intArg(args, "days", 30)
	limit := intArg(args, "limit", 20)
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)

	f := store.TransactionFilter{
		StartDate:       start.Unix(),
		EndDate:         end.Unix(),
		Search:          strArg(args, "search"),
		Category:        strArg(args, "category"),
		IncludePositive: true,
		Page:            1,
		Limit:           limit,
		SortBy:          "posted",
		SortDir:         "desc",
	}

	txns, total, err := h.txnStore.List(ctx, f)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d transactions (showing %d):\n\n", total, len(txns)))
	for _, t := range txns {
		date := time.Unix(t.Posted, 0).Format("Jan 2")
		cat := t.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		b.WriteString(fmt.Sprintf("- [%s] %s | $%.2f | %s | ID: %s\n", date, t.Description, math.Abs(t.Amount), cat, t.ID))
	}
	return b.String()
}

func (h *ChatHandler) toolSpendingSummary(ctx context.Context, args map[string]interface{}) string {
	days := intArg(args, "days", 30)
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)

	totals, err := h.txnStore.CountByCategory(ctx, start.Unix(), end.Unix())
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	type entry struct {
		name  string
		total float64
	}
	var sorted []entry
	var grand float64
	for k, v := range totals {
		sorted = append(sorted, entry{k, v})
		grand += v
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].total > sorted[j].total })

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Spending summary for last %d days (total: $%.2f):\n\n", days, grand))
	for _, e := range sorted {
		pct := 0.0
		if grand > 0 {
			pct = (e.total / grand) * 100
		}
		b.WriteString(fmt.Sprintf("- %s: $%.2f (%.0f%%)\n", e.name, e.total, pct))
	}
	return b.String()
}

func (h *ChatHandler) toolSetCategory(ctx context.Context, args map[string]interface{}) string {
	txnID := strArg(args, "transaction_id")
	category := strArg(args, "category")
	applyToMerchant := boolArg(args, "apply_to_merchant", true)

	if txnID == "" || category == "" {
		return "Error: transaction_id and category are required"
	}

	if applyToMerchant {
		txn, err := h.txnStore.GetByID(ctx, txnID)
		if err != nil || txn == nil {
			return "Error: transaction not found"
		}
		if err := h.catStore.SetOverrideAndMerchant(ctx, txnID, txn.Description, category); err != nil {
			return fmt.Sprintf("Error: %s", err)
		}
		h.events.Broadcast("categories_updated", `{"status":"ok"}`)
		return fmt.Sprintf("Set category '%s' for '%s' and all matching transactions.", category, txn.Description)
	}

	if err := h.catStore.SetOverride(ctx, txnID, category); err != nil {
		return fmt.Sprintf("Error: %s", err)
	}
	h.events.Broadcast("categories_updated", `{"status":"ok"}`)
	return fmt.Sprintf("Set category '%s' for transaction %s.", category, txnID)
}

func (h *ChatHandler) toolBulkSetCategories(ctx context.Context, args map[string]interface{}) string {
	categoriesRaw, ok := args["categories"]
	if !ok {
		return "Error: categories array is required"
	}

	catList, ok := categoriesRaw.([]interface{})
	if !ok {
		return "Error: categories must be an array"
	}

	var results []string
	for _, item := range catList {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		merchant := strArg(m, "merchant")
		category := strArg(m, "category")
		if merchant == "" || category == "" {
			continue
		}

		if err := h.catStore.Set(ctx, merchant, category, "user"); err != nil {
			results = append(results, fmt.Sprintf("Failed: %s -> %s: %s", merchant, category, err))
		} else {
			results = append(results, fmt.Sprintf("OK: %s -> %s", merchant, category))
		}
	}

	h.events.Broadcast("categories_updated", `{"status":"ok"}`)
	return fmt.Sprintf("Bulk categorization complete:\n%s", strings.Join(results, "\n"))
}

func (h *ChatHandler) toolGetAccounts(ctx context.Context) string {
	accounts, err := h.acctStore.List(ctx)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d accounts:\n\n", len(accounts)))
	for _, a := range accounts {
		included := "included"
		if !a.IsIncluded {
			included = "excluded"
		}
		b.WriteString(fmt.Sprintf("- %s: $%.2f (%s, %s)\n", a.Name, a.Balance, a.Currency, included))
	}
	return b.String()
}

func (h *ChatHandler) toolGetLatestAnalysis(ctx context.Context) string {
	a, err := h.analysisStore.GetLatest(ctx)
	if err != nil || a == nil {
		return "No analysis reports found. Try running an analysis first."
	}
	return fmt.Sprintf("Latest analysis from %s:\n\n%s", a.CreatedAt, a.ResponseText)
}

func (h *ChatHandler) toolGetBudgetStatus(ctx context.Context) string {
	start, end := billing.CurrentBillingPeriod(h.cfg.BillingDay)

	budgets, err := h.budgetStore.GetAll(ctx)
	if err != nil {
		return fmt.Sprintf("Error loading budgets: %s", err)
	}

	spending, err := h.txnStore.CountByCategory(ctx, start.Unix(), end.Unix())
	if err != nil {
		return fmt.Sprintf("Error loading spending: %s", err)
	}

	budgetMap := make(map[string]float64)
	for _, b := range budgets {
		budgetMap[strings.ToLower(b.Category)] = b.Amount
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Budget status for %s %d (billing period: %s to %s):\n\n",
		start.Month().String()[:3], start.Year(),
		start.Format("Jan 2"), end.Format("Jan 2")))

	if len(budgets) == 0 && len(spending) == 0 {
		b.WriteString("No budgets set and no spending data for this period.\n")
		return b.String()
	}

	matched := make(map[string]bool)
	if len(budgets) > 0 {
		b.WriteString("Budgeted categories:\n")
		for _, budget := range budgets {
			key := strings.ToLower(budget.Category)
			spent := spending[budget.Category]
			// Case-insensitive match
			if spent == 0 {
				for cat, val := range spending {
					if strings.EqualFold(cat, budget.Category) {
						spent = val
						break
					}
				}
			}
			matched[key] = true
			pct := 0.0
			if budget.Amount > 0 {
				pct = (spent / budget.Amount) * 100
			}
			status := "on track"
			if pct >= 100 {
				status = fmt.Sprintf("OVER by $%.2f", spent-budget.Amount)
			} else if pct >= 75 {
				status = "approaching limit"
			}
			b.WriteString(fmt.Sprintf("- %s: $%.2f spent / $%.2f budget (%.0f%%) — %s\n",
				budget.Category, spent, budget.Amount, pct, status))
		}
	}

	// Unbudgeted categories
	var unbudgeted []string
	for cat, spent := range spending {
		if !matched[strings.ToLower(cat)] {
			unbudgeted = append(unbudgeted, fmt.Sprintf("- %s: $%.2f spent (no budget)", cat, spent))
		}
	}
	if len(unbudgeted) > 0 {
		sort.Strings(unbudgeted)
		b.WriteString("\nUnbudgeted categories:\n")
		for _, line := range unbudgeted {
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

func (h *ChatHandler) toolSetBudget(ctx context.Context, args map[string]interface{}) string {
	category := strings.TrimSpace(strArg(args, "category"))

	amountRaw, ok := args["amount"]
	if !ok {
		return "Error: amount is required"
	}
	amount, ok := amountRaw.(float64)
	if !ok {
		return "Error: amount must be a number"
	}

	if msg := validateBudgetInput(category, amount); msg != "" {
		return "Error: " + msg
	}

	if err := h.budgetStore.Upsert(ctx, category, amount); err != nil {
		return fmt.Sprintf("Error saving budget: %s", err)
	}

	h.events.Broadcast("budgets_updated", `{"status":"ok"}`)
	return fmt.Sprintf("Budget set: %s = $%.2f per month.", category, amount)
}

func (h *ChatHandler) toolDeleteBudget(ctx context.Context, args map[string]interface{}) string {
	category := strings.TrimSpace(strArg(args, "category"))
	if category == "" {
		return "Error: category is required"
	}

	deleted, err := h.budgetStore.Delete(ctx, category)
	if err != nil {
		return fmt.Sprintf("Error deleting budget: %s", err)
	}
	if !deleted {
		return fmt.Sprintf("No budget found for category '%s'.", category)
	}

	h.events.Broadcast("budgets_updated", `{"status":"ok"}`)
	return fmt.Sprintf("Budget for '%s' has been removed.", category)
}

// Arg helpers.

func strArg(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intArg(m map[string]interface{}, key string, fallback int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case string:
			if i, err := strconv.Atoi(n); err == nil {
				return i
			}
		}
	}
	return fallback
}

func boolArg(m map[string]interface{}, key string, fallback bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return fallback
}
