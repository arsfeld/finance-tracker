package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/models"
)

// TransactionRepository implements transaction data operations for categorization jobs
type TransactionRepository struct {
	client *config.Client
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(client *config.Client) *TransactionRepository {
	return &TransactionRepository{
		client: client,
	}
}

// GetByID retrieves a single transaction by ID
func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	if r.client == nil || r.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var result []models.Transaction
	_, err := r.client.Service.From("transactions").
		Select("*", "", false).
		Eq("id", id.String()).
		Single().
		ExecuteTo(&result)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("transaction not found")
	}

	return &result[0], nil
}

// GetByIDs retrieves multiple transactions by their IDs
func (r *TransactionRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Transaction, error) {
	if r.client == nil || r.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	if len(ids) == 0 {
		return []*models.Transaction{}, nil
	}

	// Convert UUIDs to strings for the query
	stringIDs := make([]string, len(ids))
	for i, id := range ids {
		stringIDs[i] = id.String()
	}

	var transactions []models.Transaction
	_, err := r.client.Service.From("transactions").
		Select("*", "", false).
		In("id", stringIDs).
		Order("date", nil).
		ExecuteTo(&transactions)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Convert to pointer slice
	result := make([]*models.Transaction, len(transactions))
	for i := range transactions {
		result[i] = &transactions[i]
	}

	return result, nil
}

// GetByDateRange retrieves transactions within a date range for an organization
func (r *TransactionRepository) GetByDateRange(ctx context.Context, organizationID uuid.UUID, startDate, endDate time.Time) ([]*models.Transaction, error) {
	if r.client == nil || r.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var transactions []models.Transaction
	_, err := r.client.Service.From("transactions").
		Select("*", "", false).
		Eq("organization_id", organizationID.String()).
		Gte("date", startDate.Format("2006-01-02")).
		Lte("date", endDate.Format("2006-01-02")).
		Order("date", nil).
		ExecuteTo(&transactions)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by date range: %w", err)
	}

	// Convert to pointer slice
	result := make([]*models.Transaction, len(transactions))
	for i := range transactions {
		result[i] = &transactions[i]
	}

	return result, nil
}

// GetUncategorized retrieves all uncategorized transactions for an organization
func (r *TransactionRepository) GetUncategorized(ctx context.Context, organizationID uuid.UUID) ([]*models.Transaction, error) {
	if r.client == nil || r.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var transactions []models.Transaction
	_, err := r.client.Service.From("transactions").
		Select("*", "", false).
		Eq("organization_id", organizationID.String()).
		Is("category_id", "null").
		Order("date", nil).
		ExecuteTo(&transactions)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get uncategorized transactions: %w", err)
	}

	// Convert to pointer slice
	result := make([]*models.Transaction, len(transactions))
	for i := range transactions {
		result[i] = &transactions[i]
	}

	return result, nil
}

// GetRecentCategorized retrieves recently categorized transactions within a time duration
func (r *TransactionRepository) GetRecentCategorized(ctx context.Context, organizationID uuid.UUID, since time.Duration) ([]*models.Transaction, error) {
	if r.client == nil || r.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	// Calculate the cutoff time
	cutoffTime := time.Now().Add(-since)

	var transactions []models.Transaction
	_, err := r.client.Service.From("transactions").
		Select("*", "", false).
		Eq("organization_id", organizationID.String()).
		Not("category_id", "is", "null").
		Gte("updated_at", cutoffTime.Format(time.RFC3339)).
		Order("updated_at", nil).
		ExecuteTo(&transactions)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get recent categorized transactions: %w", err)
	}

	// Convert to pointer slice
	result := make([]*models.Transaction, len(transactions))
	for i := range transactions {
		result[i] = &transactions[i]
	}

	return result, nil
}

// UpdateCategorization updates a transaction's categorization with metadata
func (r *TransactionRepository) UpdateCategorization(ctx context.Context, transactionID uuid.UUID, categoryID int, metadata models.CategorizationMetadata) error {
	if r.client == nil || r.client.Service == nil {
		return fmt.Errorf("database client not available")
	}

	// Prepare update data
	updateData := map[string]interface{}{
		"category_id": categoryID,
		"updated_at":  time.Now().Format(time.RFC3339),
	}

	// Add metadata to the transaction metadata field
	// First get the current transaction to preserve existing metadata
	currentTx, err := r.GetByID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to get current transaction: %w", err)
	}

	// Merge categorization metadata with existing metadata
	if currentTx.Metadata == nil {
		currentTx.Metadata = make(map[string]interface{})
	}
	
	// Convert categorization metadata to map for JSON storage
	categorizationData := map[string]interface{}{}
	
	if metadata.ConfidenceScore != nil {
		categorizationData["confidence_score"] = *metadata.ConfidenceScore
	}
	if metadata.CategorizationMethod != nil {
		categorizationData["categorization_method"] = *metadata.CategorizationMethod
	}
	if metadata.LLMBatchID != nil {
		categorizationData["llm_batch_id"] = metadata.LLMBatchID.String()
	}
	categorizationData["user_corrected"] = metadata.UserCorrected
	
	if len(metadata.SimilarityMatches) > 0 {
		categorizationData["similarity_matches"] = metadata.SimilarityMatches
	}
	if len(metadata.RuleMatches) > 0 {
		categorizationData["rule_matches"] = metadata.RuleMatches
	}
	if metadata.ProcessingTimeMs != nil {
		categorizationData["processing_time_ms"] = *metadata.ProcessingTimeMs
	}
	if metadata.CostEstimate != nil {
		categorizationData["cost_estimate"] = *metadata.CostEstimate
	}
	if metadata.FeedbackID != nil {
		categorizationData["feedback_id"] = metadata.FeedbackID.String()
	}
	if metadata.CorrectionDate != nil {
		categorizationData["correction_date"] = metadata.CorrectionDate.Format(time.RFC3339)
	}

	// Add categorization metadata
	currentTx.Metadata["categorization"] = categorizationData

	updateData["metadata"] = currentTx.Metadata

	_, _, err = r.client.Service.From("transactions").
		Update(updateData, "", "").
		Eq("id", transactionID.String()).
		Execute()
	
	if err != nil {
		return fmt.Errorf("failed to update transaction categorization: %w", err)
	}

	return nil
}