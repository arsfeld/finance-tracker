package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/models"
)

// AIService provides advanced AI capabilities including RAG and batch categorization
type AIService struct {
	client        *config.Client
	openRouterKey string
	openRouterURL string
	models        []string
}

// NewAIService creates a new AI service
func NewAIService(client *config.Client, openRouterKey, openRouterURL string, models []string) *AIService {
	if openRouterURL == "" {
		openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
	}
	if len(models) == 0 {
		models = []string{
			"anthropic/claude-3.5-sonnet",
			"openai/gpt-4o-mini",
			"google/gemini-pro-1.5",
		}
	}
	
	return &AIService{
		client:        client,
		openRouterKey: openRouterKey,
		openRouterURL: openRouterURL,
		models:        models,
	}
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the AI chat API
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ChatResponse represents the AI API response
type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// FinancialContext represents the context for RAG-based responses
type FinancialContext struct {
	Transactions       []models.Transaction  `json:"transactions"`
	Categories         []models.Category     `json:"categories"`
	Accounts          []models.Account      `json:"accounts"`
	RecentSpending    map[string]float64    `json:"recent_spending"`
	MonthlyTrends     map[string]float64    `json:"monthly_trends"`
	TopMerchants      []string              `json:"top_merchants"`
	UserQuestion      string                `json:"user_question"`
	AnalysisPeriod    string                `json:"analysis_period"`
}

// BatchCategorizationRequest represents a request for batch categorization
type BatchCategorizationRequest struct {
	TransactionIDs     []uuid.UUID `json:"transaction_ids"`
	ModelPreference    string      `json:"model_preference"` // "speed", "cost", "accuracy"
	ForceRecategorize  bool        `json:"force_recategorize"`
	UseRAG            bool        `json:"use_rag"`
}

// BatchCategorizationResponse represents the response from batch categorization
type BatchCategorizationResponse struct {
	JobID              uuid.UUID                    `json:"job_id"`
	TransactionCount   int                          `json:"transaction_count"`
	EstimatedCost      float64                      `json:"estimated_cost"`
	Model              string                       `json:"model"`
	Results            []CategorizationResult       `json:"results,omitempty"`
	ProcessingTimeMs   int64                        `json:"processing_time_ms,omitempty"`
}

// CategorizationResult represents a single categorization result
type CategorizationResult struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	CategoryID    *int      `json:"category_id"`
	CategoryName  *string   `json:"category_name"`
	Confidence    float64   `json:"confidence"`
	Reasoning     string    `json:"reasoning"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
}

// ChatWithRAG performs AI chat with Retrieval Augmented Generation
func (s *AIService) ChatWithRAG(ctx context.Context, orgID uuid.UUID, userMessage string, conversationHistory []ChatMessage) (*ChatResponse, error) {
	// Retrieve relevant financial context
	context, err := s.getFinancialContext(ctx, orgID, userMessage)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get financial context, proceeding without RAG")
		// Fallback to simple chat without context
		return s.simpleChat(userMessage, conversationHistory)
	}

	// Construct RAG-enhanced prompt
	systemPrompt := s.buildRAGSystemPrompt(context)
	
	// Prepare messages with system context
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	
	// Add conversation history (last 10 messages to stay within token limits)
	historyCount := len(conversationHistory)
	if historyCount > 10 {
		conversationHistory = conversationHistory[historyCount-10:]
	}
	messages = append(messages, conversationHistory...)
	
	// Add current user message
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Select best model for the task
	model := s.selectModelForTask("chat")
	
	return s.callLLMAPI(model, messages, 0.7, 2000)
}

// BatchCategorizeTransactions performs intelligent batch categorization
func (s *AIService) BatchCategorizeTransactions(ctx context.Context, orgID uuid.UUID, request BatchCategorizationRequest) (*BatchCategorizationResponse, error) {
	// Get transactions to categorize
	transactions, err := s.getTransactionsForCategorization(ctx, orgID, request.TransactionIDs, request.ForceRecategorize)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	if len(transactions) == 0 {
		return &BatchCategorizationResponse{
			TransactionCount: 0,
			EstimatedCost:   0,
			Model:           s.selectModelForTask(request.ModelPreference),
		}, nil
	}

	// Get categories for this organization
	categories, err := s.getOrganizationCategories(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	// Select optimal model based on preference
	model := s.selectModelForTask(request.ModelPreference)
	
	// Estimate cost
	estimatedCost := s.estimateCategorizationCost(transactions, model)
	
	response := &BatchCategorizationResponse{
		JobID:            uuid.New(),
		TransactionCount: len(transactions),
		EstimatedCost:   estimatedCost,
		Model:           model,
	}

	// If this is just an estimate, return early
	if len(request.TransactionIDs) == 0 && !request.ForceRecategorize {
		return response, nil
	}

	// Perform actual categorization
	startTime := time.Now()
	results, err := s.categorizeTransactionsBatch(ctx, transactions, categories, model, request.UseRAG)
	if err != nil {
		return nil, fmt.Errorf("failed to categorize transactions: %w", err)
	}

	response.Results = results
	response.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	return response, nil
}

// getFinancialContext retrieves relevant financial data for RAG
func (s *AIService) getFinancialContext(ctx context.Context, orgID uuid.UUID, userQuestion string) (*FinancialContext, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	context := &FinancialContext{
		UserQuestion:   userQuestion,
		AnalysisPeriod: "last 90 days",
	}

	// Get recent transactions (last 90 days)
	since := time.Now().AddDate(0, 0, -90)
	var dbTransactions []struct {
		ID          uuid.UUID `json:"id"`
		Amount      float64   `json:"amount"`
		Description string    `json:"description"`
		Date        string    `json:"date"`
		CategoryID  *int      `json:"category_id"`
		AccountID   uuid.UUID `json:"account_id"`
	}

	_, err := s.client.Service.
		From("transactions").
		Select("id, amount, description, date, category_id, account_id", "", false).
		Eq("organization_id", orgID.String()).
		Gte("date", since.Format("2006-01-02")).
		Order("date", nil).
		Limit(500, "").
		ExecuteTo(&dbTransactions)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Convert to models
	context.Transactions = make([]models.Transaction, len(dbTransactions))
	for i, tx := range dbTransactions {
		date, _ := time.Parse("2006-01-02", tx.Date)
		context.Transactions[i] = models.Transaction{
			ID:            tx.ID,
			Amount:        tx.Amount,
			Description:   &tx.Description,
			Date:          date,
			CategoryID:    tx.CategoryID,
			BankAccountID: tx.AccountID,
		}
	}

	// Analyze spending patterns
	context.RecentSpending = s.analyzeSpendingByCategory(context.Transactions)
	context.MonthlyTrends = s.analyzeMonthlyTrends(context.Transactions)
	context.TopMerchants = s.extractTopMerchants(context.Transactions)

	return context, nil
}

// buildRAGSystemPrompt creates a system prompt with financial context
func (s *AIService) buildRAGSystemPrompt(context *FinancialContext) string {
	prompt := `You are Finaro's AI Financial Assistant, an expert in personal finance analysis and advice.

**Your Role:**
- Provide personalized financial insights based on the user's actual transaction data
- Answer questions about spending patterns, budgets, and financial health
- Suggest actionable improvements and optimizations
- Use data-driven analysis while being conversational and helpful

**Available Financial Data:**
`

	if len(context.Transactions) > 0 {
		prompt += fmt.Sprintf("- %d recent transactions from the last %s\n", len(context.Transactions), context.AnalysisPeriod)
		
		// Add spending breakdown
		if len(context.RecentSpending) > 0 {
			prompt += "- Recent spending by category:\n"
			for category, amount := range context.RecentSpending {
				prompt += fmt.Sprintf("  • %s: $%.2f\n", category, amount)
			}
		}

		// Add top merchants
		if len(context.TopMerchants) > 0 {
			prompt += "- Frequently used merchants: " + strings.Join(context.TopMerchants[:min(5, len(context.TopMerchants))], ", ") + "\n"
		}

		// Add trends
		if len(context.MonthlyTrends) > 0 {
			prompt += "- Monthly spending trends available\n"
		}
	}

	prompt += `
**Guidelines:**
- Base your responses on the actual financial data provided
- Be specific with amounts, dates, and patterns when relevant
- Provide actionable insights and recommendations
- If asked about data not available in the context, clearly state limitations
- Use a friendly, helpful tone while maintaining professionalism
- Include relevant financial tips and best practices
- Format monetary amounts clearly (e.g., $1,234.56)

**Response Format:**
- Keep responses concise but informative (aim for 100-300 words)
- Use bullet points or numbered lists for clarity when appropriate
- Include specific data points when making observations
- End with a helpful suggestion or next step when relevant
`

	return prompt
}

// selectModelForTask chooses the optimal model based on task requirements
func (s *AIService) selectModelForTask(preference string) string {
	if len(s.models) == 0 {
		return "anthropic/claude-3.5-sonnet"
	}

	switch preference {
	case "speed":
		// Prefer faster models
		for _, model := range []string{"openai/gpt-4o-mini", "google/gemini-pro-1.5"} {
			if s.hasModel(model) {
				return model
			}
		}
	case "cost":
		// Prefer cheaper models
		for _, model := range []string{"openai/gpt-4o-mini", "anthropic/claude-3-haiku"} {
			if s.hasModel(model) {
				return model
			}
		}
	case "accuracy":
		// Prefer most capable models
		for _, model := range []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o"} {
			if s.hasModel(model) {
				return model
			}
		}
	}

	// Default to first available model
	return s.models[0]
}

// Helper functions

func (s *AIService) hasModel(model string) bool {
	for _, m := range s.models {
		if m == model {
			return true
		}
	}
	return false
}

func (s *AIService) simpleChat(userMessage string, history []ChatMessage) (*ChatResponse, error) {
	messages := append(history, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})
	
	model := s.selectModelForTask("balance")
	return s.callLLMAPI(model, messages, 0.7, 1000)
}

// CallLLMAPI calls the LLM API with the given parameters (public method for categorization engine)
func (s *AIService) CallLLMAPI(model string, messages []ChatMessage, temperature float64, maxTokens int) (*ChatResponse, error) {
	return s.callLLMAPI(model, messages, temperature, maxTokens)
}

func (s *AIService) callLLMAPI(model string, messages []ChatMessage, temperature float64, maxTokens int) (*ChatResponse, error) {
	request := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.openRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.openRouterKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s", response.Error.Message)
	}

	return &response, nil
}

func (s *AIService) analyzeSpendingByCategory(transactions []models.Transaction) map[string]float64 {
	spending := make(map[string]float64)
	
	for _, tx := range transactions {
		if tx.Amount < 0 { // Only expenses
			category := "Uncategorized"
			if tx.CategoryID != nil {
				// Would need to resolve category name here
				category = fmt.Sprintf("Category_%d", *tx.CategoryID)
			}
			spending[category] += -tx.Amount
		}
	}
	
	return spending
}

func (s *AIService) analyzeMonthlyTrends(transactions []models.Transaction) map[string]float64 {
	trends := make(map[string]float64)
	
	for _, tx := range transactions {
		if tx.Amount < 0 { // Only expenses
			month := tx.Date.Format("2006-01")
			trends[month] += -tx.Amount
		}
	}
	
	return trends
}

func (s *AIService) extractTopMerchants(transactions []models.Transaction) []string {
	merchantCount := make(map[string]int)
	
	for _, tx := range transactions {
		if tx.Amount < 0 { // Only expenses
			description := ""
			if tx.Description != nil {
				description = *tx.Description
			}
			merchant := s.extractMerchantName(description)
			merchantCount[merchant]++
		}
	}
	
	// Sort by frequency
	type merchantFreq struct {
		name  string
		count int
	}
	
	var merchants []merchantFreq
	for name, count := range merchantCount {
		merchants = append(merchants, merchantFreq{name, count})
	}
	
	sort.Slice(merchants, func(i, j int) bool {
		return merchants[i].count > merchants[j].count
	})
	
	var result []string
	for i, m := range merchants {
		if i >= 10 { // Top 10
			break
		}
		result = append(result, m.name)
	}
	
	return result
}

func (s *AIService) extractMerchantName(description string) string {
	// Simple merchant extraction - could be enhanced with ML
	parts := strings.Fields(description)
	if len(parts) > 0 {
		return parts[0]
	}
	return description
}

func (s *AIService) getTransactionsForCategorization(ctx context.Context, orgID uuid.UUID, transactionIDs []uuid.UUID, forceRecategorize bool) ([]models.Transaction, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var dbTransactions []struct {
		ID          uuid.UUID `json:"id"`
		Amount      float64   `json:"amount"`
		Description string    `json:"description"`
		Date        string    `json:"date"`
		CategoryID  *int      `json:"category_id"`
		AccountID   uuid.UUID `json:"account_id"`
		Merchant    *string   `json:"merchant_name"`
	}

	query := s.client.Service.
		From("transactions").
		Select("id, amount, description, date, category_id, account_id, merchant_name", "", false).
		Eq("organization_id", orgID.String())

	// If specific transaction IDs are provided
	if len(transactionIDs) > 0 {
		var idStrings []string
		for _, id := range transactionIDs {
			idStrings = append(idStrings, id.String())
		}
		query = query.In("id", idStrings)
	} else {
		// Get uncategorized transactions if no specific IDs provided
		if !forceRecategorize {
			query = query.Is("category_id", "null")
		}
		// Limit to last 3 months for batch processing
		since := time.Now().AddDate(0, -3, 0)
		query = query.Gte("date", since.Format("2006-01-02"))
	}

	_, err := query.Order("date", nil).Limit(1000, "").ExecuteTo(&dbTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Convert to models
	transactions := make([]models.Transaction, len(dbTransactions))
	for i, tx := range dbTransactions {
		date, _ := time.Parse("2006-01-02", tx.Date)
		transactions[i] = models.Transaction{
			ID:              tx.ID,
			Amount:          tx.Amount,
			Description:     &tx.Description,
			Date:           date,
			CategoryID:     tx.CategoryID,
			BankAccountID:  tx.AccountID,
			MerchantName:   tx.Merchant,
			OrganizationID: orgID,
		}
	}

	return transactions, nil
}

func (s *AIService) getOrganizationCategories(ctx context.Context, orgID uuid.UUID) ([]models.Category, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var dbCategories []struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Color        *string `json:"color"`
		Icon         *string `json:"icon"`
	}

	_, err := s.client.Service.
		From("categories").
		Select("id, name, color, icon", "", false).
		Eq("organization_id", orgID.String()).
		Order("name", nil).
		ExecuteTo(&dbCategories)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}

	// Convert to models
	categories := make([]models.Category, len(dbCategories))
	for i, cat := range dbCategories {
		categories[i] = models.Category{
			ID:             cat.ID,
			Name:           cat.Name,
			Color:          cat.Color,
			Icon:           cat.Icon,
			OrganizationID: orgID,
		}
	}

	// If no categories exist, create default ones
	if len(categories) == 0 {
		defaultCategories := []string{
			"Food & Dining",
			"Transportation",
			"Shopping",
			"Entertainment",
			"Bills & Utilities",
			"Healthcare",
			"Education",
			"Travel",
			"Other",
		}
		
		for _, categoryName := range defaultCategories {
			categoryID := len(categories) + 1
			
			// Insert into database
			var insertResult []struct {
				ID int `json:"id"`
			}
			
			_, err := s.client.Service.
				From("categories").
				Insert(map[string]interface{}{
					"name":            categoryName,
					"organization_id": orgID,
				}, false, "", "", "").
				ExecuteTo(&insertResult)
				
			if err != nil {
				log.Warn().Err(err).Str("category", categoryName).Msg("Failed to create default category")
				continue
			}
			
			if len(insertResult) > 0 {
				categoryID = insertResult[0].ID
			}
			
			category := models.Category{
				ID:             categoryID,
				Name:           categoryName,
				OrganizationID: orgID,
			}
			
			categories = append(categories, category)
		}
	}

	return categories, nil
}

func (s *AIService) estimateCategorizationCost(transactions []models.Transaction, model string) float64 {
	// Simple cost estimation based on transaction count and model
	baseTokensPerTransaction := 50
	totalTokens := len(transactions) * baseTokensPerTransaction
	
	// Cost per 1K tokens (rough estimates)
	costPer1K := map[string]float64{
		"openai/gpt-4o-mini":        0.0001,
		"anthropic/claude-3-haiku":  0.0003,
		"google/gemini-pro-1.5":     0.0005,
		"anthropic/claude-3.5-sonnet": 0.003,
		"openai/gpt-4o":             0.005,
	}
	
	cost, exists := costPer1K[model]
	if !exists {
		cost = 0.001 // Default
	}
	
	return float64(totalTokens) / 1000.0 * cost
}

func (s *AIService) categorizeTransactionsBatch(ctx context.Context, transactions []models.Transaction, categories []models.Category, model string, useRAG bool) ([]CategorizationResult, error) {
	if len(transactions) == 0 {
		return []CategorizationResult{}, nil
	}

	// Build categorization prompt
	prompt := s.buildCategorizationPrompt(transactions, categories, useRAG)
	
	// Prepare messages for LLM
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "You are an expert financial transaction categorization system. Analyze transactions and return valid JSON responses only.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call LLM API
	response, err := s.callLLMAPI(model, messages, 0.3, 2000)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse JSON response
	var llmResults []struct {
		TransactionID string  `json:"transaction_id"`
		CategoryID    *int    `json:"category_id"`
		CategoryName  *string `json:"category_name"`
		Confidence    float64 `json:"confidence"`
		Reasoning     string  `json:"reasoning"`
	}

	resContent := response.Choices[0].Message.Content
	// Try to extract JSON from response (sometimes LLMs wrap JSON in markdown)
	if strings.Contains(resContent, "```json") {
		start := strings.Index(resContent, "```json") + 7
		end := strings.Index(resContent[start:], "```")
		if end > 0 {
			resContent = resContent[start : start+end]
		}
	} else if strings.Contains(resContent, "```") {
		start := strings.Index(resContent, "```") + 3
		end := strings.Index(resContent[start:], "```")
		if end > 0 {
			resContent = resContent[start : start+end]
		}
	}

	if err := json.Unmarshal([]byte(resContent), &llmResults); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response JSON: %w, response: %s", err, resContent)
	}

	// Convert to CategorizationResult format
	results := make([]CategorizationResult, len(transactions))
	llmResultMap := make(map[string]*struct {
		TransactionID string  `json:"transaction_id"`
		CategoryID    *int    `json:"category_id"`
		CategoryName  *string `json:"category_name"`
		Confidence    float64 `json:"confidence"`
		Reasoning     string  `json:"reasoning"`
	})

	for i := range llmResults {
		llmResultMap[llmResults[i].TransactionID] = &llmResults[i]
	}

	for i, transaction := range transactions {
		llmResult, found := llmResultMap[transaction.ID.String()]
		
		if !found {
			results[i] = CategorizationResult{
				TransactionID: transaction.ID,
				CategoryID:    nil,
				Confidence:    0.0,
				Reasoning:     "No result from LLM",
				Success:       false,
				Error:         "Transaction not found in LLM response",
			}
			continue
		}

		// Find category ID by name if not provided directly
		var categoryID *int
		var categoryName *string
		
		if llmResult.CategoryID != nil {
			categoryID = llmResult.CategoryID
			// Find category name
			for _, category := range categories {
				if category.ID == *llmResult.CategoryID {
					categoryName = &category.Name
					break
				}
			}
		} else if llmResult.CategoryName != nil {
			// Find category ID by name
			for _, category := range categories {
				if strings.EqualFold(category.Name, *llmResult.CategoryName) {
					categoryID = &category.ID
					categoryName = llmResult.CategoryName
					break
				}
			}
		}

		confidence := llmResult.Confidence
		if confidence > 1.0 {
			confidence = 1.0
		}
		if confidence < 0.0 {
			confidence = 0.0
		}

		results[i] = CategorizationResult{
			TransactionID: transaction.ID,
			CategoryID:    categoryID,
			CategoryName:  categoryName,
			Confidence:    confidence,
			Reasoning:     llmResult.Reasoning,
			Success:       categoryID != nil,
			Error:         func() string {
				if categoryID == nil {
					return "Could not find matching category"
				}
				return ""
			}(),
		}
	}

	return results, nil
}

// buildCategorizationPrompt builds the prompt for LLM categorization
func (s *AIService) buildCategorizationPrompt(transactions []models.Transaction, categories []models.Category, useRAG bool) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are a financial transaction categorization expert. ")
	prompt.WriteString("Your task is to categorize each transaction into the most appropriate category.\n\n")
	
	prompt.WriteString("Available Categories:\n")
	for _, category := range categories {
		prompt.WriteString(fmt.Sprintf("- %s (ID: %d)\n", category.Name, category.ID))
	}
	
	prompt.WriteString("\nTransactions to categorize:\n")
	for i, tx := range transactions {
		prompt.WriteString(fmt.Sprintf("%d. ID: %s\n", i+1, tx.ID))
		prompt.WriteString(fmt.Sprintf("   Amount: $%.2f\n", tx.Amount))
		
		if tx.MerchantName != nil {
			prompt.WriteString(fmt.Sprintf("   Merchant: %s\n", *tx.MerchantName))
		}
		
		if tx.Description != nil {
			prompt.WriteString(fmt.Sprintf("   Description: %s\n", *tx.Description))
		}
		
		prompt.WriteString(fmt.Sprintf("   Date: %s\n", tx.Date.Format("2006-01-02")))
		prompt.WriteString("\n")
	}
	
	prompt.WriteString("For each transaction, respond with a JSON object containing:\n")
	prompt.WriteString("- transaction_id: the transaction ID (string)\n")
	prompt.WriteString("- category_id: the most appropriate category ID (integer)\n")
	prompt.WriteString("- confidence: your confidence level (0.0 to 1.0)\n")
	prompt.WriteString("- reasoning: brief explanation of your choice\n\n")
	
	prompt.WriteString("IMPORTANT: Respond with ONLY a valid JSON array of these objects, one for each transaction. Do not include any other text.")
	
	return prompt.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}