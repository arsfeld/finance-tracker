package categorization

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"finance_tracker/src/internal/models"
	"finance_tracker/src/internal/services"
)

// openRouterLLMClient implements the LLMClient interface using OpenRouter API
type openRouterLLMClient struct {
	aiService *services.AIService
}

// NewOpenRouterLLMClient creates a new OpenRouter LLM client
func NewOpenRouterLLMClient(aiService *services.AIService) LLMClient {
	return &openRouterLLMClient{
		aiService: aiService,
	}
}

// CategorizeTransactionsBatch sends a batch of transactions to LLM for categorization
func (c *openRouterLLMClient) CategorizeTransactionsBatch(ctx context.Context, transactions []*models.Transaction, categories []*models.Category, model *models.LLMModel) (*LLMBatchResponse, error) {
	if len(transactions) == 0 {
		return &LLMBatchResponse{
			Results:        []LLMCategorizationResult{},
			Model:          model.Name,
			ProcessingTime: 0,
		}, nil
	}

	startTime := time.Now()

	// Build categorization prompt
	prompt := c.buildCategorizationPrompt(transactions, categories)
	
	// Prepare messages for LLM
	messages := []services.ChatMessage{
		{
			Role:    "system",
			Content: "You are an expert financial transaction categorization system. Analyze transactions and return valid JSON responses only.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call AI service
	response, err := c.aiService.CallLLMAPI(model.Name, messages, 0.3, 2000)
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

	// Convert to LLMBatchResponse format
	results := make([]LLMCategorizationResult, len(transactions))
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
			results[i] = LLMCategorizationResult{
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

		results[i] = LLMCategorizationResult{
			TransactionID: transaction.ID,
			CategoryID:    categoryID,
			CategoryName:  categoryName,
			Confidence:    confidence,
			Reasoning:     llmResult.Reasoning,
			Success:       categoryID != nil,
			Error: func() string {
				if categoryID == nil {
					return "Could not find matching category"
				}
				return ""
			}(),
		}
	}

	processingTime := time.Since(startTime)

	// Estimate token usage (simplified)
	inputTokens, outputTokens, _ := c.EstimateTokens(transactions, categories)

	return &LLMBatchResponse{
		Results:        results,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		Model:          model.Name,
		ProcessingTime: processingTime,
	}, nil
}

// EstimateTokens estimates the number of tokens for a request
func (c *openRouterLLMClient) EstimateTokens(transactions []*models.Transaction, categories []*models.Category) (inputTokens, outputTokens int, err error) {
	// Rough token estimation based on text length
	// Average token is about 4 characters
	
	// System prompt tokens
	systemPromptLength := 200 // Approximate
	inputTokens += systemPromptLength / 4
	
	// Categories tokens
	for _, category := range categories {
		inputTokens += len(category.Name) / 4
		inputTokens += 10 // For formatting
	}
	
	// Transaction tokens
	for _, tx := range transactions {
		inputTokens += len(tx.ID.String()) / 4
		inputTokens += 20 // For amount and date formatting
		
		if tx.Description != nil {
			inputTokens += len(*tx.Description) / 4
		}
		
		if tx.MerchantName != nil {
			inputTokens += len(*tx.MerchantName) / 4
		}
		
		inputTokens += 50 // For formatting and structure
	}
	
	// Output tokens (response JSON)
	outputTokens = len(transactions) * 100 // Rough estimate for JSON response per transaction
	
	return inputTokens, outputTokens, nil
}

// buildCategorizationPrompt builds the prompt for LLM categorization
func (c *openRouterLLMClient) buildCategorizationPrompt(transactions []*models.Transaction, categories []*models.Category) string {
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