package jobs

import (
	"time"
	"github.com/google/uuid"
)

// Job argument types for different sync operations
type SyncTransactionsArgs struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	ConnectionID   uuid.UUID `json:"connection_id"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	ForceSync      bool       `json:"force_sync"`
}

func (SyncTransactionsArgs) Kind() string { return "sync_transactions" }

type SyncAccountsArgs struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	ConnectionID   uuid.UUID `json:"connection_id"`
	ForceSync      bool       `json:"force_sync"`
}

func (SyncAccountsArgs) Kind() string { return "sync_accounts" }

type FullSyncArgs struct {
	OrganizationID uuid.UUID  `json:"organization_id"`
	ConnectionID   uuid.UUID  `json:"connection_id"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	IncludeHistory bool       `json:"include_history"`
	ForceSync      bool       `json:"force_sync"`
}

func (FullSyncArgs) Kind() string { return "full_sync" }

type TestConnectionArgs struct {
	OrganizationID   uuid.UUID `json:"organization_id"`
	ConnectionID     uuid.UUID `json:"connection_id"`
	ValidateAccounts bool       `json:"validate_accounts"`
}

func (TestConnectionArgs) Kind() string { return "test_connection" }

// Job priorities
const (
	PriorityLow    = 1
	PriorityNormal = 2
	PriorityHigh   = 3
	PriorityUrgent = 4
)

// Job states (River provides these, but we define for reference)
const (
	JobStateAvailable = "available"
	JobStateRunning   = "running"
	JobStateRetryable = "retryable"
	JobStateScheduled = "scheduled"
	JobStateCompleted = "completed"
	JobStateCancelled = "cancelled"
	JobStateDiscarded = "discarded"
)

// Custom metadata keys
const (
	MetadataConnectionName = "connection_name"
	MetadataProviderType   = "provider_type"
	MetadataAccountCount   = "account_count"
	MetadataTransactionCount = "transaction_count"
	MetadataStartDate      = "start_date"
	MetadataEndDate        = "end_date"
)

// Extended job argument types for new River jobs

// AnalyzeSpendingArgs defines arguments for spending analysis jobs
type AnalyzeSpendingArgs struct {
	OrganizationID uuid.UUID   `json:"organization_id"`
	AccountIDs     []uuid.UUID `json:"account_ids,omitempty"`
	StartDate      *time.Time  `json:"start_date,omitempty"`
	EndDate        *time.Time  `json:"end_date,omitempty"`
	AnalysisType   string      `json:"analysis_type"` // "spending", "trends", "insights", "budgets"
	NotifyChannels []string    `json:"notify_channels,omitempty"` // "email", "ntfy", "webhook"
}

func (AnalyzeSpendingArgs) Kind() string { return "analyze_spending" }

// CleanupArgs defines arguments for cleanup jobs
type CleanupArgs struct {
	OrganizationID uuid.UUID  `json:"organization_id"`
	Type           string     `json:"type"` // "old_jobs", "cache", "temp_files", "duplicates"
	OlderThan      *time.Time `json:"older_than,omitempty"`
	DryRun         bool       `json:"dry_run"`
}

func (CleanupArgs) Kind() string { return "cleanup" }

// BackupArgs defines arguments for backup jobs
type BackupArgs struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Type           string    `json:"type"` // "full", "incremental", "config"
	Destination    string    `json:"destination"` // "s3", "local", "supabase"
	Encrypt        bool      `json:"encrypt"`
	Compress       bool      `json:"compress"`
}

func (BackupArgs) Kind() string { return "backup" }