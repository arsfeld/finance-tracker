package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/models"
)

// TransactionService handles transaction data operations
type TransactionService struct {
	client *config.Client
}

// NewTransactionService creates a new transaction service
func NewTransactionService(client *config.Client) *TransactionService {
	return &TransactionService{
		client: client,
	}
}

// GetTransactions returns paginated transactions for an organization
func (s *TransactionService) GetTransactions(ctx context.Context, orgID uuid.UUID, filter models.TransactionFilter) ([]models.APITransaction, int, error) {
	if s.client == nil || s.client.Service == nil {
		return []models.APITransaction{}, 0, fmt.Errorf("database client not available")
	}

	// Build the query
	query := s.client.Service.From("transactions").
		Select("*", "", false).
		Eq("organization_id", orgID.String()).
		Order("date", nil)

	// Apply filters
	if filter.AccountID != nil {
		query = query.Eq("bank_account_id", filter.AccountID.String())
	}
	
	if filter.StartDate != nil {
		query = query.Gte("date", filter.StartDate.Format("2006-01-02"))
	}
	
	if filter.EndDate != nil {
		query = query.Lte("date", filter.EndDate.Format("2006-01-02"))
	}
	
	if filter.Search != nil && *filter.Search != "" {
		// Simple text search in description
		query = query.Ilike("description", fmt.Sprintf("%%%s%%", *filter.Search))
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit, "")
	}
	if filter.Offset > 0 {
		query = query.Range(filter.Offset, filter.Offset+filter.Limit-1, "")
	}

	// Execute query
	var dbResults []struct {
		ID                  uuid.UUID  `json:"id"`
		OrganizationID      uuid.UUID  `json:"organization_id"`
		BankAccountID       uuid.UUID  `json:"bank_account_id"`
		Date                string     `json:"date"`
		Amount              float64    `json:"amount"`
		Description         *string    `json:"description"`
		ProviderTransID     *string    `json:"provider_transaction_id"`
		MerchantName        *string    `json:"merchant_name"`
		CategoryID          *int       `json:"category_id"`
		Pending             bool       `json:"pending"`
		CreatedAt           string     `json:"created_at"`
		UpdatedAt           string     `json:"updated_at"`
	}
	
	_, err := query.ExecuteTo(&dbResults)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Get total count for pagination (simplified)
	totalCount := len(dbResults)

	// Convert to API format
	transactions := make([]models.APITransaction, len(dbResults))
	for i, result := range dbResults {
		// Parse dates
		date, _ := parseDate(result.Date)
		createdAt, _ := parseTimestamp(result.CreatedAt)
		updatedAt, _ := parseTimestamp(result.UpdatedAt)
		
		// Get account name (simplified - we'll do this in batches later for optimization)
		accountName := "Unknown Account"
		
		transactions[i] = models.APITransaction{
			ID:               result.ID,
			OrganizationID:   result.OrganizationID,
			AccountID:        result.BankAccountID,
			AccountName:      accountName,
			Date:             date,
			Amount:           result.Amount,
			Description:      stringPtrToString(result.Description),
			ProviderTransID:  stringPtrToString(result.ProviderTransID),
			Merchant:         result.MerchantName,
			Pending:          result.Pending,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		}
	}

	return transactions, totalCount, nil
}

// GetTransaction returns a specific transaction
func (s *TransactionService) GetTransaction(ctx context.Context, orgID uuid.UUID, transactionID uuid.UUID) (*models.APITransaction, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var dbResult struct {
		ID                  uuid.UUID  `json:"id"`
		OrganizationID      uuid.UUID  `json:"organization_id"`
		BankAccountID       uuid.UUID  `json:"bank_account_id"`
		Date                string     `json:"date"`
		Amount              float64    `json:"amount"`
		Description         *string    `json:"description"`
		ProviderTransID     *string    `json:"provider_transaction_id"`
		MerchantName        *string    `json:"merchant_name"`
		CategoryID          *int       `json:"category_id"`
		Pending             bool       `json:"pending"`
		CreatedAt           string     `json:"created_at"`
		UpdatedAt           string     `json:"updated_at"`
	}

	_, err := s.client.Service.From("transactions").
		Select("*", "", false).
		Eq("id", transactionID.String()).
		Eq("organization_id", orgID.String()).
		Single().
		ExecuteTo(&dbResult)

	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	// Parse dates
	date, _ := parseDate(dbResult.Date)
	createdAt, _ := parseTimestamp(dbResult.CreatedAt)
	updatedAt, _ := parseTimestamp(dbResult.UpdatedAt)

	transaction := &models.APITransaction{
		ID:               dbResult.ID,
		OrganizationID:   dbResult.OrganizationID,
		AccountID:        dbResult.BankAccountID,
		AccountName:      "Unknown Account", // Simplified for now
		Date:             date,
		Amount:           dbResult.Amount,
		Description:      stringPtrToString(dbResult.Description),
		ProviderTransID:  stringPtrToString(dbResult.ProviderTransID),
		Merchant:         dbResult.MerchantName,
		Pending:          dbResult.Pending,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	return transaction, nil
}

// UpdateTransactionCategory updates the category of a transaction
func (s *TransactionService) UpdateTransactionCategory(ctx context.Context, orgID uuid.UUID, transactionID uuid.UUID, category string) error {
	if s.client == nil || s.client.Service == nil {
		return fmt.Errorf("database client not available")
	}

	// First, find the category ID by name within the organization
	var categoryResult struct {
		ID int `json:"id"`
	}

	_, err := s.client.Service.From("categories").
		Select("id", "", false).
		Eq("organization_id", orgID.String()).
		Eq("name", category).
		Single().
		ExecuteTo(&categoryResult)

	if err != nil {
		// Category doesn't exist, create it
		var newCategory struct {
			ID int `json:"id"`
		}
		
		_, err = s.client.Service.From("categories").
			Insert(map[string]interface{}{
				"organization_id": orgID.String(),
				"name":           category,
			}, false, "", "", "").
			Single().
			ExecuteTo(&newCategory)
			
		if err != nil {
			return fmt.Errorf("failed to create category: %w", err)
		}
		categoryResult.ID = newCategory.ID
	}

	// Update the transaction with the category ID
	_, _, err = s.client.Service.From("transactions").
		Update(map[string]interface{}{
			"category_id": categoryResult.ID,
		}, "", "").
		Eq("id", transactionID.String()).
		Eq("organization_id", orgID.String()).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to update transaction category: %w", err)
	}

	return nil
}

// GetRecentTransactions returns recent transactions for the dashboard
func (s *TransactionService) GetRecentTransactions(ctx context.Context, orgID uuid.UUID, limit int) ([]models.APITransaction, error) {
	if s.client == nil || s.client.Service == nil {
		return []models.APITransaction{}, fmt.Errorf("database client not available")
	}

	// Use the existing GetTransactions method with appropriate filter
	filter := models.TransactionFilter{
		Limit:  limit,
		Offset: 0,
	}

	transactions, _, err := s.GetTransactions(ctx, orgID, filter)
	if err != nil {
		return []models.APITransaction{}, fmt.Errorf("failed to get recent transactions: %w", err)
	}

	return transactions, nil
}

