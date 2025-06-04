package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"finance_tracker/src/internal/models"
)

// TransactionService defines the interface for transaction data operations
type TransactionService interface {
	GetTransactions(ctx context.Context, orgID uuid.UUID, filter models.TransactionFilter) ([]models.APITransaction, int, error)
	GetTransaction(ctx context.Context, orgID uuid.UUID, transactionID uuid.UUID) (*models.APITransaction, error)
	UpdateTransactionCategory(ctx context.Context, orgID uuid.UUID, transactionID uuid.UUID, category string) error
	GetRecentTransactions(ctx context.Context, orgID uuid.UUID, limit int) ([]models.APITransaction, error)
}

// TransactionUseCase handles transaction business logic
type TransactionUseCase struct {
	transactionService TransactionService
}

// NewTransactionUseCase creates a new transaction use case
func NewTransactionUseCase(transactionService TransactionService) *TransactionUseCase {
	return &TransactionUseCase{
		transactionService: transactionService,
	}
}

// ListTransactionsRequest represents the request to list transactions
type ListTransactionsRequest struct {
	AccountID  *uuid.UUID
	Category   *string
	StartDate  *time.Time
	EndDate    *time.Time
	Search     *string
	Limit      int
	Offset     int
}

// ListTransactionsResponse represents the response for listing transactions
type ListTransactionsResponse struct {
	Transactions []models.APITransaction `json:"transactions"`
	Total        int                     `json:"total"`
	Limit        int                     `json:"limit"`
	Offset       int                     `json:"offset"`
}

// ListTransactions returns paginated transactions for an organization
func (uc *TransactionUseCase) ListTransactions(ctx context.Context, orgID uuid.UUID, req ListTransactionsRequest) (*ListTransactionsResponse, error) {
	// Apply defaults
	if req.Limit <= 0 || req.Limit > 200 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	filter := models.TransactionFilter{
		AccountID:  req.AccountID,
		Category:   req.Category,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		Search:     req.Search,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}

	transactions, total, err := uc.transactionService.GetTransactions(ctx, orgID, filter)
	if err != nil {
		return nil, err
	}

	return &ListTransactionsResponse{
		Transactions: transactions,
		Total:        total,
		Limit:        req.Limit,
		Offset:       req.Offset,
	}, nil
}

// GetRecentTransactions returns recent transactions for the dashboard
func (uc *TransactionUseCase) GetRecentTransactions(ctx context.Context, orgID uuid.UUID, limit int) ([]models.APITransaction, error) {
	if limit <= 0 {
		limit = 10
	}
	return uc.transactionService.GetRecentTransactions(ctx, orgID, limit)
}

// GetTransactionDetail returns detailed information about a specific transaction
func (uc *TransactionUseCase) GetTransactionDetail(ctx context.Context, orgID uuid.UUID, transactionID uuid.UUID) (*models.APITransaction, error) {
	return uc.transactionService.GetTransaction(ctx, orgID, transactionID)
}

// UpdateTransactionCategoryRequest represents the request to update a transaction category
type UpdateTransactionCategoryRequest struct {
	TransactionID uuid.UUID
	Category      string
}

// UpdateTransactionCategory updates the category of a transaction
func (uc *TransactionUseCase) UpdateTransactionCategory(ctx context.Context, orgID uuid.UUID, req UpdateTransactionCategoryRequest) error {
	if req.Category == "" {
		return &ValidationError{Field: "category", Message: "Category is required"}
	}

	return uc.transactionService.UpdateTransactionCategory(ctx, orgID, req.TransactionID, req.Category)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}