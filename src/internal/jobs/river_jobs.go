package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/riverqueue/river"
	"finance_tracker/src/internal/services"
	"finance_tracker/src/providers"
)

// SyncTransactionsJob handles transaction synchronization
type SyncTransactionsJob struct {
	river.WorkerDefaults[SyncTransactionsArgs]
	syncService   *services.SyncService
	providers     map[string]providers.FinancialProvider
	cryptoService *services.CryptoService
}

func (SyncTransactionsJob) Kind() string { return "sync_transactions" }

func (job *SyncTransactionsJob) Work(ctx context.Context, j *river.Job[SyncTransactionsArgs]) error {
	args := j.Args
	log.Printf("Starting transaction sync for organization %s, connection %s", args.OrganizationID, args.ConnectionID)
	
	// Step 1: Get connection details and provider
	connection, err := job.syncService.GetConnection(ctx, args.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	
	provider, exists := job.providers[connection.ProviderType]
	if !exists {
		return fmt.Errorf("provider %s not found", connection.ProviderType)
	}
	
	// Step 2: Get accounts for this connection
	accounts, err := job.syncService.GetAccountsByConnection(ctx, args.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get accounts: %w", err)
	}
	
	// Step 3: Determine date range
	startDate := args.StartDate
	endDate := args.EndDate
	if startDate == nil || endDate == nil {
		// Default to last 30 days
		now := time.Now()
		defaultStart := now.AddDate(0, 0, -30)
		if startDate == nil {
			startDate = &defaultStart
		}
		if endDate == nil {
			endDate = &now
		}
	}
	
	log.Printf("Syncing transactions from %s to %s for %d accounts", 
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), len(accounts))
	
	// Step 4: Sync transactions for each account
	totalTransactions := 0
	for _, account := range accounts {
		log.Printf("Syncing transactions for account %s (%s)", account.Name, account.ID)
		
		// Get transactions from provider
		// Decrypt credentials
		credentials, err := job.cryptoService.DecryptCredentials(connection.CredentialsEncrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt credentials: %w", err)
		}
		transactions, err := provider.GetTransactions(ctx, credentials, account.ProviderAccountID, *startDate, *endDate)
		if err != nil {
			log.Printf("Warning: failed to get transactions for account %s: %v", account.ID, err)
			continue
		}
		
		// Use the account ID directly (it's already a UUID in our database)
		accountUUID := account.ID
		
		// Store transactions in database
		err = job.syncService.StoreTransactions(ctx, args.OrganizationID, accountUUID, transactions)
		if err != nil {
			log.Printf("Warning: failed to store transactions for account %s: %v", account.ID, err)
			continue
		}
		
		totalTransactions += len(transactions)
		log.Printf("Stored %d transactions for account %s", len(transactions), account.Name)
	}
	
	// Step 5: Update last sync time
	err = job.syncService.UpdateConnectionLastSync(ctx, args.ConnectionID, time.Now())
	if err != nil {
		log.Printf("Warning: failed to update last sync time: %v", err)
	}
	
	log.Printf("Transaction sync completed for organization %s. Synced %d total transactions", 
		args.OrganizationID, totalTransactions)
	return nil
}

// SyncAccountsJob handles account synchronization
type SyncAccountsJob struct {
	river.WorkerDefaults[SyncAccountsArgs]
	syncService   *services.SyncService
	providers     map[string]providers.FinancialProvider
	cryptoService *services.CryptoService
}

func (SyncAccountsJob) Kind() string { return "sync_accounts" }

func (job *SyncAccountsJob) Work(ctx context.Context, j *river.Job[SyncAccountsArgs]) error {
	args := j.Args
	log.Printf("Starting account sync for organization %s, connection %s", args.OrganizationID, args.ConnectionID)
	
	// Step 1: Get connection details and provider
	connection, err := job.syncService.GetConnection(ctx, args.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	
	provider, exists := job.providers[connection.ProviderType]
	if !exists {
		return fmt.Errorf("provider %s not found", connection.ProviderType)
	}
	
	// Step 2: Fetch accounts from provider
	// Decrypt credentials
	credentials, err := job.cryptoService.DecryptCredentials(connection.CredentialsEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}
	accounts, err := provider.ListAccounts(ctx, credentials)
	if err != nil {
		return fmt.Errorf("failed to fetch accounts from provider: %w", err)
	}
	
	log.Printf("Fetched %d accounts from provider", len(accounts))
	
	// Step 3: Store or update accounts in database
	err = job.syncService.StoreAccounts(ctx, args.OrganizationID, args.ConnectionID, accounts)
	if err != nil {
		return fmt.Errorf("failed to store accounts: %w", err)
	}
	
	// Step 4: Update connection last sync time
	err = job.syncService.UpdateConnectionLastSync(ctx, args.ConnectionID, time.Now())
	if err != nil {
		log.Printf("Warning: failed to update last sync time: %v", err)
	}
	
	log.Printf("Account sync completed for organization %s. Synced %d accounts", args.OrganizationID, len(accounts))
	return nil
}

// FullSyncJob handles comprehensive synchronization of both accounts and transactions
type FullSyncJob struct {
	river.WorkerDefaults[FullSyncArgs]
	syncService   *services.SyncService
	providers     map[string]providers.FinancialProvider
	cryptoService *services.CryptoService
}

func (FullSyncJob) Kind() string { return "full_sync" }

func (job *FullSyncJob) Work(ctx context.Context, j *river.Job[FullSyncArgs]) error {
	args := j.Args
	log.Printf("Starting full sync for organization %s, connection %s", args.OrganizationID, args.ConnectionID)
	
	// Step 1: Validate connection
	log.Printf("Full sync step 1/4: Validating connection")
	connection, err := job.syncService.GetConnection(ctx, args.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	
	provider, exists := job.providers[connection.ProviderType]
	if !exists {
		return fmt.Errorf("provider %s not found", connection.ProviderType)
	}
	
	// Step 2: Sync accounts
	log.Printf("Full sync step 2/4: Syncing accounts")
	// Decrypt credentials
	credentials, err := job.cryptoService.DecryptCredentials(connection.CredentialsEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}
	accounts, err := provider.ListAccounts(ctx, credentials)
	if err != nil {
		return fmt.Errorf("failed to fetch accounts from provider: %w", err)
	}
	
	err = job.syncService.StoreAccounts(ctx, args.OrganizationID, args.ConnectionID, accounts)
	if err != nil {
		return fmt.Errorf("failed to store accounts: %w", err)
	}
	log.Printf("Synced %d accounts", len(accounts))
	
	// Step 3: Sync transactions
	log.Printf("Full sync step 3/4: Syncing transactions")
	
	// Determine date range for transaction sync
	startDate := args.StartDate
	if startDate == nil {
		if args.IncludeHistory {
			// Include last 2 years for full history
			historyStart := time.Now().AddDate(-2, 0, 0)
			startDate = &historyStart
		} else {
			// Default to last 90 days for regular sync
			recentStart := time.Now().AddDate(0, 0, -90)
			startDate = &recentStart
		}
	}
	endDate := time.Now()
	
	totalTransactions := 0
	for _, account := range accounts {
		transactions, err := provider.GetTransactions(ctx, credentials, account.ID, *startDate, endDate)
		if err != nil {
			log.Printf("Warning: failed to get transactions for account %s: %v", account.ID, err)
			continue
		}
		
		// Get the UUID for this account from our database (accounts here are from provider)
		accountUUID, err := job.syncService.GetAccountByProviderID(ctx, args.ConnectionID, account.ID)
		if err != nil {
			log.Printf("Warning: failed to get account UUID for provider account %s: %v", account.ID, err)
			continue
		}
		
		// Store transactions in database
		err = job.syncService.StoreTransactions(ctx, args.OrganizationID, accountUUID, transactions)
		if err != nil {
			log.Printf("Warning: failed to store transactions for account %s: %v", account.ID, err)
			continue
		}
		
		totalTransactions += len(transactions)
	}
	log.Printf("Synced %d total transactions", totalTransactions)
	
	// Step 4: Update connection last sync time
	log.Printf("Full sync step 4/4: Updating sync metadata")
	err = job.syncService.UpdateConnectionLastSync(ctx, args.ConnectionID, time.Now())
	if err != nil {
		log.Printf("Warning: failed to update last sync time: %v", err)
	}
	
	log.Printf("Full sync completed for organization %s. Synced %d accounts and %d transactions", 
		args.OrganizationID, len(accounts), totalTransactions)
	return nil
}

// TestConnectionJob validates provider connectivity
type TestConnectionJob struct {
	river.WorkerDefaults[TestConnectionArgs]
	syncService   *services.SyncService
	providers     map[string]providers.FinancialProvider
	cryptoService *services.CryptoService
}

func (TestConnectionJob) Kind() string { return "test_connection" }

func (job *TestConnectionJob) Work(ctx context.Context, j *river.Job[TestConnectionArgs]) error {
	args := j.Args
	log.Printf("Testing connection for organization %s, connection %s", args.OrganizationID, args.ConnectionID)
	
	// Step 1: Get connection details
	connection, err := job.syncService.GetConnection(ctx, args.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	
	// Step 2: Get provider
	provider, exists := job.providers[connection.ProviderType]
	if !exists {
		return fmt.Errorf("provider %s not found", connection.ProviderType)
	}
	
	// Step 3: Test connection by listing accounts
	log.Printf("Testing connection by fetching accounts...")
	// Decrypt credentials
	credentials, err := job.cryptoService.DecryptCredentials(connection.CredentialsEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}
	accounts, err := provider.ListAccounts(ctx, credentials)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	
	log.Printf("Connection test successful. Found %d accounts", len(accounts))
	
	// Step 4: Optional account validation
	if args.ValidateAccounts && len(accounts) > 0 {
		log.Printf("Validating accounts for connection %s", args.ConnectionID)
		
		// Test transaction fetching for the first account
		if len(accounts) > 0 {
			test_account := accounts[0]
			now := time.Now()
			lastWeek := now.AddDate(0, 0, -7)
			
			transactions, err := provider.GetTransactions(ctx, credentials, test_account.ID, lastWeek, now)
			if err != nil {
				return fmt.Errorf("account validation failed: %w", err)
			}
			log.Printf("Account validation successful. Test account has %d recent transactions", len(transactions))
		}
	}
	
	// Step 5: Update connection status
	err = job.syncService.UpdateConnectionStatus(ctx, args.ConnectionID, "active", "Connection test passed")
	if err != nil {
		log.Printf("Warning: failed to update connection status: %v", err)
	}
	
	log.Printf("Connection test completed successfully for organization %s", args.OrganizationID)
	return nil
}

// AnalyzeSpendingJob generates spending insights using LLM
type AnalyzeSpendingJob struct {
	river.WorkerDefaults[AnalyzeSpendingArgs]
}

func (AnalyzeSpendingJob) Kind() string { return "analyze_spending" }

func (job AnalyzeSpendingJob) Work(ctx context.Context, j *river.Job[AnalyzeSpendingArgs]) error {
	args := j.Args
	log.Printf("Starting spending analysis for organization %s", args.OrganizationID)
	
	// Simulate LLM analysis work
	steps := []string{
		"Gathering transaction data",
		"Categorizing expenses",
		"Analyzing patterns",
		"Generating insights",
		"Preparing notifications",
	}
	
	for i, step := range steps {
		log.Printf("Analysis step %d/%d: %s", i+1, len(steps), step)
		time.Sleep(time.Second)
	}
	
	// Send notifications if requested
	for _, channel := range args.NotifyChannels {
		log.Printf("Sending analysis notification via %s", channel)
		// Implement notification sending logic
	}
	
	log.Printf("Spending analysis completed for organization %s", args.OrganizationID)
	return nil
}

// CleanupJob handles data cleanup and maintenance
type CleanupJob struct {
	river.WorkerDefaults[CleanupArgs]
}

func (CleanupJob) Kind() string { return "cleanup" }

func (job CleanupJob) Work(ctx context.Context, j *river.Job[CleanupArgs]) error {
	args := j.Args
	log.Printf("Starting cleanup job for organization %s, type: %s", args.OrganizationID, args.Type)
	
	if args.DryRun {
		log.Printf("Running in dry-run mode - no actual cleanup will be performed")
	}
	
	switch args.Type {
	case "old_jobs":
		log.Printf("Cleaning up old job records")
	case "cache":
		log.Printf("Cleaning up cache files")
	case "temp_files":
		log.Printf("Cleaning up temporary files")
	case "duplicates":
		log.Printf("Removing duplicate transactions")
	default:
		return fmt.Errorf("unknown cleanup type: %s", args.Type)
	}
	
	// Simulate cleanup work
	time.Sleep(2 * time.Second)
	
	log.Printf("Cleanup job completed for organization %s", args.OrganizationID)
	return nil
}

// BackupJob handles data backup operations
type BackupJob struct {
	river.WorkerDefaults[BackupArgs]
}

func (BackupJob) Kind() string { return "backup" }

func (job BackupJob) Work(ctx context.Context, j *river.Job[BackupArgs]) error {
	args := j.Args
	log.Printf("Starting backup job for organization %s, type: %s", args.OrganizationID, args.Type)
	
	steps := []string{"Preparing data", "Creating backup", "Compressing", "Uploading"}
	if args.Encrypt {
		steps = append(steps, "Encrypting")
	}
	
	for i, step := range steps {
		log.Printf("Backup step %d/%d: %s", i+1, len(steps), step)
		time.Sleep(time.Second)
	}
	
	log.Printf("Backup job completed for organization %s", args.OrganizationID)
	return nil
}