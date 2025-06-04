package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusPaused    JobStatus = "paused"
)

// JobPriority represents the priority of a job
type JobPriority string

const (
	JobPriorityLow    JobPriority = "low"
	JobPriorityNormal JobPriority = "normal"
	JobPriorityHigh   JobPriority = "high"
	JobPriorityUrgent JobPriority = "urgent"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeSyncTransactions JobType = "sync_transactions"
	JobTypeSyncAccounts     JobType = "sync_accounts"
	JobTypeFullSync         JobType = "full_sync"
	JobTypeTestConnection   JobType = "test_connection"
)

// Job represents a background job
type Job struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	Type           JobType    `json:"type" db:"type"`
	Status         JobStatus  `json:"status" db:"status"`
	Priority       JobPriority `json:"priority" db:"priority"`
	
	// Job metadata
	Title       string  `json:"title" db:"title"`
	Description *string `json:"description" db:"description"`
	
	// Target data
	ProviderConnectionID *uuid.UUID `json:"provider_connection_id" db:"provider_connection_id"`
	BankAccountID        *uuid.UUID `json:"bank_account_id" db:"bank_account_id"`
	
	// Job parameters
	Parameters json.RawMessage `json:"parameters" db:"parameters"`
	
	// Progress tracking
	ProgressCurrent int     `json:"progress_current" db:"progress_current"`
	ProgressTotal   int     `json:"progress_total" db:"progress_total"`
	ProgressMessage *string `json:"progress_message" db:"progress_message"`
	
	// Timestamps
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ScheduledAt time.Time  `json:"scheduled_at" db:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
	
	// Worker info
	WorkerID *string `json:"worker_id" db:"worker_id"`
	
	// Results and errors
	Result       json.RawMessage `json:"result" db:"result"`
	ErrorMessage *string         `json:"error_message" db:"error_message"`
	ErrorDetails json.RawMessage `json:"error_details" db:"error_details"`
	
	// Retry logic
	RetryCount         int `json:"retry_count" db:"retry_count"`
	MaxRetries         int `json:"max_retries" db:"max_retries"`
	RetryDelaySeconds  int `json:"retry_delay_seconds" db:"retry_delay_seconds"`
}

// GetProgressPercentage returns progress as a percentage
func (j *Job) GetProgressPercentage() float64 {
	if j.ProgressTotal == 0 {
		return 0
	}
	return float64(j.ProgressCurrent) / float64(j.ProgressTotal) * 100
}

// IsComplete returns true if the job is in a terminal state
func (j *Job) IsComplete() bool {
	return j.Status == JobStatusCompleted || 
		   j.Status == JobStatusFailed || 
		   j.Status == JobStatusCancelled
}

// CanRetry returns true if the job can be retried
func (j *Job) CanRetry() bool {
	return j.Status == JobStatusFailed && j.RetryCount < j.MaxRetries
}

// JobWorker represents an active worker
type JobWorker struct {
	ID                 string    `json:"id" db:"id"`
	Hostname           string    `json:"hostname" db:"hostname"`
	PID                int       `json:"pid" db:"pid"`
	StartedAt          time.Time `json:"started_at" db:"started_at"`
	LastHeartbeat      time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	Status             string    `json:"status" db:"status"`
	MaxConcurrentJobs  int       `json:"max_concurrent_jobs" db:"max_concurrent_jobs"`
	CurrentJobCount    int       `json:"current_job_count" db:"current_job_count"`
	Version            *string   `json:"version" db:"version"`
	WorkerType         *string   `json:"worker_type" db:"worker_type"`
}

// IsHealthy returns true if worker has sent heartbeat recently
func (w *JobWorker) IsHealthy(threshold time.Duration) bool {
	return time.Since(w.LastHeartbeat) < threshold
}

// JobQueueSettings represents queue configuration per organization
type JobQueueSettings struct {
	OrganizationID             uuid.UUID  `json:"organization_id" db:"organization_id"`
	MaxConcurrentJobs          int        `json:"max_concurrent_jobs" db:"max_concurrent_jobs"`
	MaxRetries                 int        `json:"max_retries" db:"max_retries"`
	RetryDelaySeconds          int        `json:"retry_delay_seconds" db:"retry_delay_seconds"`
	AutoSyncEnabled            bool       `json:"auto_sync_enabled" db:"auto_sync_enabled"`
	AutoSyncIntervalMinutes    int        `json:"auto_sync_interval_minutes" db:"auto_sync_interval_minutes"`
	AutoSyncQuietHoursStart    *string    `json:"auto_sync_quiet_hours_start" db:"auto_sync_quiet_hours_start"`
	AutoSyncQuietHoursEnd      *string    `json:"auto_sync_quiet_hours_end" db:"auto_sync_quiet_hours_end"`
	MaxJobsPerHour             int        `json:"max_jobs_per_hour" db:"max_jobs_per_hour"`
	UpdatedAt                  time.Time  `json:"updated_at" db:"updated_at"`
}

// JobExecution represents job execution history
type JobExecution struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	JobID           uuid.UUID  `json:"job_id" db:"job_id"`
	WorkerID        *string    `json:"worker_id" db:"worker_id"`
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at" db:"completed_at"`
	Status          JobStatus  `json:"status" db:"status"`
	DurationSeconds *int       `json:"duration_seconds" db:"duration_seconds"`
	ErrorMessage    *string    `json:"error_message" db:"error_message"`
	ItemsProcessed  int        `json:"items_processed" db:"items_processed"`
	ItemsTotal      int        `json:"items_total" db:"items_total"`
}

// Job creation parameters
type CreateJobParams struct {
	OrganizationID       uuid.UUID       `json:"organization_id"`
	Type                 JobType         `json:"type"`
	Priority             JobPriority     `json:"priority"`
	Title                string          `json:"title"`
	Description          *string         `json:"description"`
	ProviderConnectionID *uuid.UUID      `json:"provider_connection_id"`
	BankAccountID        *uuid.UUID      `json:"bank_account_id"`
	Parameters           json.RawMessage `json:"parameters"`
	ScheduledAt          *time.Time      `json:"scheduled_at"`
	MaxRetries           *int            `json:"max_retries"`
	RetryDelaySeconds    *int            `json:"retry_delay_seconds"`
}

// Job update parameters
type UpdateJobParams struct {
	Status          *JobStatus       `json:"status"`
	Priority        *JobPriority     `json:"priority"`
	ProgressCurrent *int             `json:"progress_current"`
	ProgressTotal   *int             `json:"progress_total"`
	ProgressMessage *string          `json:"progress_message"`
	Result          *json.RawMessage `json:"result"`
	ErrorMessage    *string          `json:"error_message"`
	ErrorDetails    *json.RawMessage `json:"error_details"`
}

// Job filters for querying
type JobFilters struct {
	OrganizationID       *uuid.UUID       `json:"organization_id"`
	Status               []JobStatus      `json:"status"`
	Type                 []JobType        `json:"type"`
	Priority             []JobPriority    `json:"priority"`
	ProviderConnectionID *uuid.UUID       `json:"provider_connection_id"`
	WorkerID             *string          `json:"worker_id"`
	Limit                int              `json:"limit"`
	Offset               int              `json:"offset"`
	CreatedAfter         *time.Time       `json:"created_after"`
	CreatedBefore        *time.Time       `json:"created_before"`
}

// Sync job parameters for different job types
type SyncTransactionsParams struct {
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	ForceSync bool       `json:"force_sync"`
}

type SyncAccountsParams struct {
	ForceSync bool `json:"force_sync"`
}

type FullSyncParams struct {
	IncludeHistory bool       `json:"include_history"`
	StartDate      *time.Time `json:"start_date"`
	ForceSync      bool       `json:"force_sync"`
}

type TestConnectionParams struct {
	ValidateAccounts bool `json:"validate_accounts"`
}

// Implement driver.Valuer for JSON fields
func (j JobStatus) Value() (driver.Value, error) {
	return string(j), nil
}

func (j *JobStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if s, ok := value.(string); ok {
		*j = JobStatus(s)
		return nil
	}
	return fmt.Errorf("cannot scan %T into JobStatus", value)
}

func (j JobPriority) Value() (driver.Value, error) {
	return string(j), nil
}

func (j *JobPriority) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if s, ok := value.(string); ok {
		*j = JobPriority(s)
		return nil
	}
	return fmt.Errorf("cannot scan %T into JobPriority", value)
}

func (j JobType) Value() (driver.Value, error) {
	return string(j), nil
}

func (j *JobType) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if s, ok := value.(string); ok {
		*j = JobType(s)
		return nil
	}
	return fmt.Errorf("cannot scan %T into JobType", value)
}