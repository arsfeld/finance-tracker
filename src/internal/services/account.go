package services

import (
	"context"

	"github.com/google/uuid"
	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/models"
	"finance_tracker/src/providers"
)

// AccountService handles account data operations
type AccountService struct {
	client *config.Client
}

// NewAccountService creates a new account service
func NewAccountService(client *config.Client) *AccountService {
	return &AccountService{
		client: client,
	}
}

// ListAccounts returns all accounts for an organization
func (s *AccountService) ListAccounts(ctx context.Context, orgID uuid.UUID) ([]models.Account, error) {
	// TODO: Implement actual database query
	return []models.Account{}, nil
}

// GetAccount returns a specific account
func (s *AccountService) GetAccount(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) (*models.Account, error) {
	// TODO: Implement actual database query
	return nil, nil
}

// CreateAccounts creates bank accounts from provider accounts
func (s *AccountService) CreateAccounts(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID, accounts []providers.ProviderAccount) error {
	// TODO: Implement actual database operations
	return nil
}

// ListConnectionAccounts returns all bank accounts for a connection
func (s *AccountService) ListConnectionAccounts(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID) ([]models.ConnectionAccount, error) {
	// TODO: Implement actual database query using Supabase client
	if s.client.Service == nil {
		return []models.ConnectionAccount{}, nil
	}

	var accounts []models.ConnectionAccount
	_, err := s.client.Service.
		From("bank_accounts").
		Select("id, provider_account_id, name, institution, account_type, balance, currency, is_active, last_sync", "", false).
		Eq("connection_id", connectionID.String()).
		Eq("organization_id", orgID.String()).
		ExecuteTo(&accounts)

	if err != nil {
		return nil, err
	}

	// Set connection ID for all accounts
	for i := range accounts {
		accounts[i].ConnectionID = connectionID
	}

	return accounts, nil
}

// UpdateAccountStatus updates the active status of a bank account
func (s *AccountService) UpdateAccountStatus(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID, isActive bool) error {
	// TODO: Implement actual database update using Supabase client
	if s.client.Service == nil {
		return nil
	}

	updateData := map[string]interface{}{
		"is_active": isActive,
	}

	var updatedAccounts []models.ConnectionAccount
	_, err := s.client.Service.
		From("bank_accounts").
		Update(updateData, "*", "").
		Eq("id", accountID.String()).
		Eq("organization_id", orgID.String()).
		ExecuteTo(&updatedAccounts)

	return err
}