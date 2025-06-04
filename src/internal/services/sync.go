package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"finance_tracker/src/internal/models"
	"finance_tracker/src/providers"
)

// RiverJobClient interface to avoid circular dependency
type RiverJobClient interface {
	HealthCheck(ctx context.Context) error
}

type SyncService struct {
	db          *sqlx.DB
	jobService  *JobService
	riverClient RiverJobClient
	providers   map[string]providers.FinancialProvider
}

func NewSyncService(db *sqlx.DB, jobService *JobService) *SyncService {
	return &SyncService{
		db:         db,
		jobService: jobService,
		providers:  make(map[string]providers.FinancialProvider),
	}
}

// SetRiverClient sets the River client after initialization
func (s *SyncService) SetRiverClient(riverClient RiverJobClient) {
	s.riverClient = riverClient
}

// RegisterProvider registers a financial data provider
func (s *SyncService) RegisterProvider(name string, p providers.FinancialProvider) {
	s.providers[name] = p
}

// CreateSyncJob creates a new sync job for a provider connection
func (s *SyncService) CreateSyncJob(ctx context.Context, organizationID, connectionID uuid.UUID, jobType models.JobType, params interface{}) (*models.Job, error) {
	// For now, return a placeholder job
	job := &models.Job{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Type:           jobType,
		Status:         models.JobStatusPending,
		Priority:       models.JobPriorityNormal,
		Title:          "Sync Job",
		CreatedAt:      time.Now(),
		ScheduledAt:    time.Now(),
	}
	
	return job, nil
}

// CreateFullSyncJob creates a comprehensive sync job that syncs accounts and transactions
func (s *SyncService) CreateFullSyncJob(ctx context.Context, organizationID, connectionID uuid.UUID, includeHistory bool, startDate *time.Time) (*models.Job, error) {
	return s.CreateSyncJob(ctx, organizationID, connectionID, models.JobTypeFullSync, nil)
}

// CreateTransactionSyncJob creates a job to sync transactions for a specific date range
func (s *SyncService) CreateTransactionSyncJob(ctx context.Context, organizationID, connectionID uuid.UUID, startDate, endDate *time.Time, forceSync bool) (*models.Job, error) {
	return s.CreateSyncJob(ctx, organizationID, connectionID, models.JobTypeSyncTransactions, nil)
}

// ProcessJob processes a sync job - simplified implementation
func (s *SyncService) ProcessJob(ctx context.Context, job *models.Job, workerID string) error {
	log.Printf("Processing job %s (type: %s) with worker %s", job.ID, job.Type, workerID)
	
	// Simulate some work
	time.Sleep(1 * time.Second)
	
	log.Printf("Job %s completed successfully", job.ID)
	return nil
}

func (s *SyncService) getJobTitle(jobType models.JobType, connectionName string) string {
	switch jobType {
	case models.JobTypeFullSync:
		return fmt.Sprintf("Full sync for %s", connectionName)
	case models.JobTypeSyncTransactions:
		return fmt.Sprintf("Sync transactions for %s", connectionName)
	case models.JobTypeSyncAccounts:
		return fmt.Sprintf("Sync accounts for %s", connectionName)
	case models.JobTypeTestConnection:
		return fmt.Sprintf("Test connection for %s", connectionName)
	default:
		return fmt.Sprintf("Sync job for %s", connectionName)
	}
}

func (s *SyncService) getJobDescription(jobType models.JobType, connectionName string) *string {
	var desc string
	switch jobType {
	case models.JobTypeFullSync:
		desc = "Synchronize both accounts and transactions from the financial provider"
	case models.JobTypeSyncTransactions:
		desc = "Synchronize transaction data from the financial provider"
	case models.JobTypeSyncAccounts:
		desc = "Synchronize account information from the financial provider"
	case models.JobTypeTestConnection:
		desc = "Test connectivity and authentication with the financial provider"
	default:
		desc = "Synchronize data from the financial provider"
	}
	return &desc
}

// River-based job creation methods (commented out to avoid circular dependency)
// These methods can be moved to a separate service or implemented differently

// TODO: Implement River job creation methods without circular dependency
// Consider using a job queue service that doesn't import back to sync service

// Connection-related methods

// GetConnection retrieves a connection by ID
func (s *SyncService) GetConnection(ctx context.Context, connectionID uuid.UUID) (*models.ProviderConnection, error) {
	var connection models.ProviderConnection
	query := `SELECT * FROM provider_connections WHERE id = $1`
	err := s.db.GetContext(ctx, &connection, query, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	return &connection, nil
}

// GetAccountsByConnection retrieves all accounts for a connection
func (s *SyncService) GetAccountsByConnection(ctx context.Context, connectionID uuid.UUID) ([]*models.BankAccount, error) {
	var accounts []*models.BankAccount
	query := `SELECT * FROM bank_accounts WHERE connection_id = $1 ORDER BY name`
	err := s.db.SelectContext(ctx, &accounts, query, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts by connection: %w", err)
	}
	return accounts, nil
}

// StoreAccounts stores or updates accounts in the database
func (s *SyncService) StoreAccounts(ctx context.Context, organizationID, connectionID uuid.UUID, accounts []providers.ProviderAccount) error {
	for _, account := range accounts {
		// Check if account already exists
		existingID, err := s.getAccountByProviderID(ctx, connectionID, account.ID)
		if err == nil {
			// Update existing account
			query := `
				UPDATE bank_accounts 
				SET name = $1, account_type = $2, balance = $3, currency = $4, updated_at = NOW()
				WHERE id = $5
			`
			_, err = s.db.ExecContext(ctx, query, account.Name, account.Type, account.Balance, account.Currency, existingID)
			if err != nil {
				return fmt.Errorf("failed to update account %s: %w", account.ID, err)
			}
		} else {
			// Insert new account
			accountID := uuid.New()
			query := `
				INSERT INTO bank_accounts (id, organization_id, connection_id, provider_account_id, name, account_type, balance, currency, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			`
			_, err = s.db.ExecContext(ctx, query, accountID, organizationID, connectionID, 
				account.ID, account.Name, account.Type, account.Balance, account.Currency)
			if err != nil {
				return fmt.Errorf("failed to insert account %s: %w", account.ID, err)
			}
		}
	}
	return nil
}

// StoreTransactions stores transactions in the database
func (s *SyncService) StoreTransactions(ctx context.Context, organizationID, accountID uuid.UUID, transactions []providers.ProviderTransaction) error {
	for _, transaction := range transactions {
		// Check if transaction already exists
		exists, err := s.transactionExists(ctx, accountID, transaction.ID)
		if err != nil {
			return fmt.Errorf("failed to check transaction existence: %w", err)
		}
		
		if !exists {
			// Insert new transaction
			transactionID := uuid.New()
			query := `
				INSERT INTO transactions (id, organization_id, bank_account_id, provider_transaction_id, amount, description, date, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			`
			_, err = s.db.ExecContext(ctx, query, transactionID, organizationID, accountID,
				transaction.ID, transaction.Amount, transaction.Description, 
				transaction.Date)
			if err != nil {
				return fmt.Errorf("failed to insert transaction %s: %w", transaction.ID, err)
			}
		}
	}
	return nil
}

// UpdateConnectionLastSync updates the last sync time for a connection
func (s *SyncService) UpdateConnectionLastSync(ctx context.Context, connectionID uuid.UUID, lastSync time.Time) error {
	query := `UPDATE provider_connections SET last_sync = $1, updated_at = NOW() WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, lastSync, connectionID)
	if err != nil {
		return fmt.Errorf("failed to update connection last sync: %w", err)
	}
	return nil
}

// UpdateConnectionStatus updates the status of a connection
func (s *SyncService) UpdateConnectionStatus(ctx context.Context, connectionID uuid.UUID, status, message string) error {
	query := `UPDATE provider_connections SET sync_status = $1, error_message = $2, updated_at = NOW() WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, status, message, connectionID)
	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}
	return nil
}

// Helper methods

func (s *SyncService) getAccountByProviderID(ctx context.Context, connectionID uuid.UUID, providerID string) (uuid.UUID, error) {
	var accountID uuid.UUID
	query := `SELECT id FROM bank_accounts WHERE connection_id = $1 AND provider_account_id = $2`
	err := s.db.GetContext(ctx, &accountID, query, connectionID, providerID)
	return accountID, err
}

func (s *SyncService) transactionExists(ctx context.Context, accountID uuid.UUID, providerID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM transactions WHERE bank_account_id = $1 AND provider_transaction_id = $2`
	err := s.db.GetContext(ctx, &count, query, accountID, providerID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}