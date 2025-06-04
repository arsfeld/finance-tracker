package services

import (
	"context"
	"fmt"
	"time"

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
	if s.client == nil || s.client.Service == nil {
		return []models.Account{}, fmt.Errorf("database client not available")
	}

	var dbResults []struct {
		ID             uuid.UUID  `json:"id"`
		OrganizationID uuid.UUID  `json:"organization_id"`
		ConnectionID   uuid.UUID  `json:"connection_id"`
		Name           string     `json:"name"`
		AccountType    *string    `json:"account_type"`
		Balance        *float64   `json:"balance"`
		Currency       string     `json:"currency"`
		IsActive       bool       `json:"is_active"`
		ProviderID     string     `json:"provider_account_id"`
		Institution    *string    `json:"institution"`
		LastSync       *string    `json:"last_sync"`
		CreatedAt      string     `json:"created_at"`
		UpdatedAt      string     `json:"updated_at"`
	}

	_, err := s.client.Service.From("bank_accounts").
		Select("*", "", false).
		Eq("organization_id", orgID.String()).
		Order("name", nil).
		ExecuteTo(&dbResults)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	// Convert to API format
	accounts := make([]models.Account, len(dbResults))
	for i, result := range dbResults {
		// Parse timestamps
		createdAt, _ := parseTimestamp(result.CreatedAt)
		updatedAt, _ := parseTimestamp(result.UpdatedAt)
		
		var lastSyncDate time.Time
		if result.LastSync != nil {
			lastSyncDate, _ = parseTimestamp(*result.LastSync)
		}

		balance := 0.0
		if result.Balance != nil {
			balance = *result.Balance
		}

		accountType := ""
		if result.AccountType != nil {
			accountType = *result.AccountType
		}

		accounts[i] = models.Account{
			ID:             result.ID,
			OrganizationID: result.OrganizationID,
			ConnectionID:   result.ConnectionID,
			Name:           result.Name,
			Type:           accountType,
			Balance:        balance,
			Currency:       result.Currency,
			LastSyncDate:   lastSyncDate,
			IsActive:       result.IsActive,
			ProviderID:     result.ProviderID,
			ProviderName:   "SimpleFin", // Simplified for now
			Institution:    result.Institution,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}
	}

	return accounts, nil
}

// GetAccount returns a specific account
func (s *AccountService) GetAccount(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) (*models.Account, error) {
	if s.client == nil || s.client.Service == nil {
		return nil, fmt.Errorf("database client not available")
	}

	var dbResult struct {
		ID             uuid.UUID  `json:"id"`
		OrganizationID uuid.UUID  `json:"organization_id"`
		ConnectionID   uuid.UUID  `json:"connection_id"`
		Name           string     `json:"name"`
		AccountType    *string    `json:"account_type"`
		Balance        *float64   `json:"balance"`
		Currency       string     `json:"currency"`
		IsActive       bool       `json:"is_active"`
		ProviderID     string     `json:"provider_account_id"`
		Institution    *string    `json:"institution"`
		LastSync       *string    `json:"last_sync"`
		CreatedAt      string     `json:"created_at"`
		UpdatedAt      string     `json:"updated_at"`
	}

	_, err := s.client.Service.From("bank_accounts").
		Select("*", "", false).
		Eq("id", accountID.String()).
		Eq("organization_id", orgID.String()).
		Single().
		ExecuteTo(&dbResult)

	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	// Parse timestamps
	createdAt, _ := parseTimestamp(dbResult.CreatedAt)
	updatedAt, _ := parseTimestamp(dbResult.UpdatedAt)
	
	var lastSyncDate time.Time
	if dbResult.LastSync != nil {
		lastSyncDate, _ = parseTimestamp(*dbResult.LastSync)
	}

	balance := 0.0
	if dbResult.Balance != nil {
		balance = *dbResult.Balance
	}

	accountType := ""
	if dbResult.AccountType != nil {
		accountType = *dbResult.AccountType
	}

	account := &models.Account{
		ID:             dbResult.ID,
		OrganizationID: dbResult.OrganizationID,
		ConnectionID:   dbResult.ConnectionID,
		Name:           dbResult.Name,
		Type:           accountType,
		Balance:        balance,
		Currency:       dbResult.Currency,
		LastSyncDate:   lastSyncDate,
		IsActive:       dbResult.IsActive,
		ProviderID:     dbResult.ProviderID,
		ProviderName:   "SimpleFin", // Simplified for now
		Institution:    dbResult.Institution,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	return account, nil
}

// CreateAccounts creates bank accounts from provider accounts
func (s *AccountService) CreateAccounts(ctx context.Context, orgID uuid.UUID, connectionID uuid.UUID, accounts []providers.ProviderAccount) error {
	if s.client == nil || s.client.Service == nil {
		return fmt.Errorf("database client not available")
	}

	if len(accounts) == 0 {
		return nil
	}

	// Prepare data for batch insert
	accountData := make([]map[string]interface{}, len(accounts))
	for i, account := range accounts {
		accountData[i] = map[string]interface{}{
			"organization_id":      orgID.String(),
			"connection_id":        connectionID.String(),
			"provider_account_id":  account.ID,
			"name":                 account.Name,
			"institution":          account.Institution,
			"account_type":         account.Type,
			"balance":              account.Balance,
			"currency":             account.Currency,
			"is_active":            true,
			"last_sync":            time.Now(),
		}
	}

	// Use upsert to handle existing accounts
	_, _, err := s.client.Service.From("bank_accounts").
		Upsert(accountData, "connection_id,provider_account_id", "", "").
		Execute()

	if err != nil {
		return fmt.Errorf("failed to create accounts: %w", err)
	}

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

	_, _, err := s.client.Service.
		From("bank_accounts").
		Update(updateData, "", "").
		Eq("id", accountID.String()).
		Eq("organization_id", orgID.String()).
		Execute()

	return err
}

