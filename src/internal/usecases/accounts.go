package usecases

import (
	"context"

	"github.com/google/uuid"
	"finance_tracker/src/internal/models"
)

// AccountService defines the interface for account data operations
type BankAccountService interface {
	ListAccounts(ctx context.Context, orgID uuid.UUID) ([]models.Account, error)
	GetAccount(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) (*models.Account, error)
}

// AccountUseCase handles account business logic
type AccountUseCase struct {
	accountService BankAccountService
}

// NewAccountUseCase creates a new account use case
func NewAccountUseCase(accountService BankAccountService) *AccountUseCase {
	return &AccountUseCase{
		accountService: accountService,
	}
}

// ListAccounts returns all accounts for an organization
func (uc *AccountUseCase) ListAccounts(ctx context.Context, orgID uuid.UUID) ([]models.Account, error) {
	return uc.accountService.ListAccounts(ctx, orgID)
}

// GetAccount returns a specific account
func (uc *AccountUseCase) GetAccount(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) (*models.Account, error) {
	return uc.accountService.GetAccount(ctx, orgID, accountID)
}