package categorization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"finance_tracker/src/internal/models"
)

// llmEngine implements the LLMEngine interface
type llmEngine struct {
	repo         LLMRepository
	costManager  CostManager
	llmClient    LLMClient // Interface for LLM API calls
	models       []models.LLMModel
	batchConfig  models.BatchConfig
}

// LLMClient interface for making LLM API calls
type LLMClient interface {
	// CategorizeTransactionsBatch sends a batch of transactions to LLM for categorization
	CategorizeTransactionsBatch(ctx context.Context, transactions []*models.Transaction, categories []*models.Category, model *models.LLMModel) (*LLMBatchResponse, error)
	
	// EstimateTokens estimates the number of tokens for a request
	EstimateTokens(transactions []*models.Transaction, categories []*models.Category) (inputTokens, outputTokens int, err error)
}

// LLMBatchResponse represents the response from LLM categorization
type LLMBatchResponse struct {
	Results      []LLMCategorizationResult `json:"results"`
	InputTokens  int                       `json:"input_tokens"`
	OutputTokens int                       `json:"output_tokens"`
	Model        string                    `json:"model"`
	ProcessingTime time.Duration           `json:"processing_time"`
}

// LLMCategorizationResult represents a single categorization result from LLM
type LLMCategorizationResult struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	CategoryID    *int      `json:"category_id"`
	CategoryName  *string   `json:"category_name"`
	Confidence    float64   `json:"confidence"`
	Reasoning     string    `json:"reasoning"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
}

// NewLLMEngine creates a new LLM-based categorization engine
func NewLLMEngine(repo LLMRepository, costManager CostManager, llmClient LLMClient, batchConfig models.BatchConfig) LLMEngine {
	return &llmEngine{
		repo:        repo,
		costManager: costManager,
		llmClient:   llmClient,
		models:      models.DefaultLLMModels,
		batchConfig: batchConfig,
	}
}

// CategorizeByLLM categorizes transactions using LLM in batch
func (e *llmEngine) CategorizeByLLM(ctx context.Context, transactions []*models.Transaction) ([]*models.CategorizationResult, error) {
	if len(transactions) == 0 {
		return []*models.CategorizationResult{}, nil
	}

	startTime := time.Now()
	organizationID := transactions[0].OrganizationID

	// Check budget before processing
	model, err := e.GetBestModel(ctx, organizationID, "cost_optimized")
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	estimatedCost, err := e.EstimateCost(ctx, transactions, model)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate cost: %w", err)
	}

	if err := e.costManager.CheckBudget(ctx, organizationID, estimatedCost); err != nil {
		return nil, fmt.Errorf("budget check failed: %w", err)
	}

	// Get categories for context
	categories, err := e.getCategoriesForOrganization(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	// Process in batches if needed
	var allResults []*models.CategorizationResult
	batchSize := e.batchConfig.MaxBatchSize
	
	for i := 0; i < len(transactions); i += batchSize {
		end := i + batchSize
		if end > len(transactions) {
			end = len(transactions)
		}
		
		batch := transactions[i:end]
		batchResults, err := e.processBatch(ctx, batch, categories, model)
		if err != nil {
			return nil, fmt.Errorf("failed to process batch %d: %w", i/batchSize+1, err)
		}
		
		allResults = append(allResults, batchResults...)
	}

	// Record actual processing time
	totalProcessingTime := time.Since(startTime)
	
	// Calculate actual cost and record batch
	totalInputTokens := 0
	totalOutputTokens := 0
	successCount := 0
	totalConfidence := 0.0
	
	for _, result := range allResults {
		if result.CategoryID != nil {
			successCount++
			totalConfidence += result.Confidence
		}
	}
	
	// Estimate tokens based on results (simplified)
	totalInputTokens, totalOutputTokens, _ = e.llmClient.EstimateTokens(transactions, categories)
	
	actualCost := e.calculateCost(totalInputTokens, totalOutputTokens, model)
	
	// Record the batch
	batch := &models.LLMCategorizationBatch{
		ID:               uuid.New(),
		OrganizationID:   organizationID,
		TransactionCount: len(transactions),
		InputTokens:      totalInputTokens,
		OutputTokens:     totalOutputTokens,
		TotalCost:        actualCost,
		ModelUsed:        model.Name,
		BatchType:        "categorization",
		ProcessingTimeMs: func() *int { i := int(totalProcessingTime.Milliseconds()); return &i }(),
		CreatedAt:        time.Now(),
	}
	
	if successCount > 0 {
		successRate := float64(successCount) / float64(len(transactions))
		avgConfidence := totalConfidence / float64(successCount)
		batch.SuccessRate = &successRate
		batch.AvgConfidence = &avgConfidence
	}
	
	if err := e.RecordBatch(ctx, batch); err != nil {
		// Log error but don't fail the categorization
		fmt.Printf("Failed to record batch: %v\n", err)
	}
	
	// Record cost
	if err := e.costManager.RecordCost(ctx, organizationID, actualCost, len(transactions)); err != nil {
		// Log error but don't fail the categorization
		fmt.Printf("Failed to record cost: %v\n", err)
	}

	return allResults, nil
}

// processBatch processes a single batch of transactions
func (e *llmEngine) processBatch(ctx context.Context, transactions []*models.Transaction, categories []*models.Category, model *models.LLMModel) ([]*models.CategorizationResult, error) {
	batchStartTime := time.Now()
	
	// Call LLM API
	response, err := e.llmClient.CategorizeTransactionsBatch(ctx, transactions, categories, model)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}
	
	// Convert LLM results to categorization results
	results := make([]*models.CategorizationResult, len(transactions))
	
	// Create map for quick lookup of LLM results by transaction ID
	llmResultMap := make(map[uuid.UUID]*LLMCategorizationResult)
	for i := range response.Results {
		llmResultMap[response.Results[i].TransactionID] = &response.Results[i]
	}
	
	processingTimeMs := time.Since(batchStartTime).Milliseconds()
	costPerTransaction := e.calculateCost(response.InputTokens, response.OutputTokens, model) / float64(len(transactions))
	
	for i, transaction := range transactions {
		llmResult, found := llmResultMap[transaction.ID]
		
		if !found || !llmResult.Success {
			// LLM failed to categorize this transaction
			errorMsg := "LLM failed to categorize transaction"
			if found && llmResult.Error != "" {
				errorMsg = llmResult.Error
			}
			
			results[i] = &models.CategorizationResult{
				CategoryID:       nil,
				Confidence:       0.0,
				Method:           models.CategorizationMethodLLMBatch,
				ProcessingTimeMs: processingTimeMs,
				CostEstimate:     costPerTransaction,
				Explanation:      errorMsg,
			}
			continue
		}
		
		// Find category ID by name if not provided directly
		var categoryID *int
		if llmResult.CategoryID != nil {
			categoryID = llmResult.CategoryID
		} else if llmResult.CategoryName != nil {
			for _, category := range categories {
				if strings.EqualFold(category.Name, *llmResult.CategoryName) {
					categoryID = &category.ID
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
		
		explanation := "LLM categorization"
		if llmResult.Reasoning != "" {
			explanation = llmResult.Reasoning
		}
		
		results[i] = &models.CategorizationResult{
			CategoryID:       categoryID,
			Confidence:       confidence,
			Method:           models.CategorizationMethodLLMBatch,
			ProcessingTimeMs: processingTimeMs,
			CostEstimate:     costPerTransaction,
			Explanation:      explanation,
		}
	}
	
	return results, nil
}

// GetBestModel returns the best model based on cost/accuracy tradeoff
func (e *llmEngine) GetBestModel(ctx context.Context, organizationID uuid.UUID, strategy string) (*models.LLMModel, error) {
	switch strategy {
	case "cost_optimized":
		// Return the cheapest model
		var cheapest *models.LLMModel
		for i := range e.models {
			if cheapest == nil || e.models[i].CostPer1K < cheapest.CostPer1K {
				cheapest = &e.models[i]
			}
		}
		return cheapest, nil
		
	case "accuracy_optimized":
		// Return the most accurate model
		var mostAccurate *models.LLMModel
		for i := range e.models {
			if mostAccurate == nil || e.models[i].Accuracy > mostAccurate.Accuracy {
				mostAccurate = &e.models[i]
			}
		}
		return mostAccurate, nil
		
	case "balanced":
		// Return the best value (accuracy/cost ratio)
		var bestValue *models.LLMModel
		var bestRatio float64
		
		for i := range e.models {
			ratio := e.models[i].Accuracy / e.models[i].CostPer1K
			if bestValue == nil || ratio > bestRatio {
				bestValue = &e.models[i]
				bestRatio = ratio
			}
		}
		return bestValue, nil
		
	default:
		// Return default model
		for i := range e.models {
			if e.models[i].IsDefault {
				return &e.models[i], nil
			}
		}
		return &e.models[0], nil
	}
}

// EstimateCost estimates the cost for categorizing transactions
func (e *llmEngine) EstimateCost(ctx context.Context, transactions []*models.Transaction, model *models.LLMModel) (float64, error) {
	// Get categories for context (needed for token estimation)
	if len(transactions) == 0 {
		return 0.0, nil
	}
	
	organizationID := transactions[0].OrganizationID
	categories, err := e.getCategoriesForOrganization(ctx, organizationID)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get categories: %w", err)
	}
	
	inputTokens, outputTokens, err := e.llmClient.EstimateTokens(transactions, categories)
	if err != nil {
		return 0.0, fmt.Errorf("failed to estimate tokens: %w", err)
	}
	
	return e.calculateCost(inputTokens, outputTokens, model), nil
}

// calculateCost calculates the cost based on token usage and model pricing
func (e *llmEngine) calculateCost(inputTokens, outputTokens int, model *models.LLMModel) float64 {
	totalTokens := inputTokens + outputTokens
	return (float64(totalTokens) / 1000.0) * model.CostPer1K
}

// RecordBatch records a completed batch for cost tracking
func (e *llmEngine) RecordBatch(ctx context.Context, batch *models.LLMCategorizationBatch) error {
	return e.repo.CreateBatch(ctx, batch)
}

// getCategoriesForOrganization gets categories for an organization (placeholder)
func (e *llmEngine) getCategoriesForOrganization(ctx context.Context, organizationID uuid.UUID) ([]*models.Category, error) {
	// This would need to be implemented to get categories from the category repository
	// For now, return default categories
	return []*models.Category{
		{ID: 1, Name: "Food & Dining", OrganizationID: organizationID},
		{ID: 2, Name: "Transportation", OrganizationID: organizationID},
		{ID: 3, Name: "Shopping", OrganizationID: organizationID},
		{ID: 4, Name: "Entertainment", OrganizationID: organizationID},
		{ID: 5, Name: "Bills & Utilities", OrganizationID: organizationID},
		{ID: 6, Name: "Healthcare", OrganizationID: organizationID},
		{ID: 7, Name: "Education", OrganizationID: organizationID},
		{ID: 8, Name: "Travel", OrganizationID: organizationID},
		{ID: 9, Name: "Other", OrganizationID: organizationID},
	}, nil
}

// buildCategorizationPrompt builds the prompt for LLM categorization
func (e *llmEngine) buildCategorizationPrompt(transactions []*models.Transaction, categories []*models.Category) string {
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
	prompt.WriteString("- transaction_id: the transaction ID\n")
	prompt.WriteString("- category_id: the most appropriate category ID\n")
	prompt.WriteString("- confidence: your confidence level (0.0 to 1.0)\n")
	prompt.WriteString("- reasoning: brief explanation of your choice\n\n")
	
	prompt.WriteString("Respond with a JSON array of these objects, one for each transaction.")
	
	return prompt.String()
}