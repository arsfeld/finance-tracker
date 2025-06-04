package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SimpleJobClient provides a minimal job client interface for testing
type SimpleJobClient struct {
	db *sqlx.DB
}

// NewSimpleJobClient creates a simple job client
func NewSimpleJobClient(db *sqlx.DB) *SimpleJobClient {
	return &SimpleJobClient{db: db}
}

// Start starts the job client (no-op for simple implementation)
func (c *SimpleJobClient) Start(ctx context.Context) error {
	return nil
}

// Stop stops the job client (no-op for simple implementation)
func (c *SimpleJobClient) Stop(ctx context.Context) error {
	return nil
}

// Simple job result type
type SimpleJobResult struct {
	ID          int64     `json:"id"`
	Kind        string    `json:"kind"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// Job insertion methods (simplified)
func (c *SimpleJobClient) InsertSyncTransactionsJob(ctx context.Context, args SyncTransactionsArgs) (*SimpleJobResult, error) {
	// Simulate job creation
	result := &SimpleJobResult{
		ID:          time.Now().Unix(),
		Kind:        "sync_transactions",
		State:       "available",
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}
	return result, nil
}

func (c *SimpleJobClient) InsertSyncAccountsJob(ctx context.Context, args SyncAccountsArgs) (*SimpleJobResult, error) {
	result := &SimpleJobResult{
		ID:          time.Now().Unix(),
		Kind:        "sync_accounts",
		State:       "available",
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}
	return result, nil
}

func (c *SimpleJobClient) InsertFullSyncJob(ctx context.Context, args FullSyncArgs) (*SimpleJobResult, error) {
	result := &SimpleJobResult{
		ID:          time.Now().Unix(),
		Kind:        "full_sync",
		State:       "available",
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}
	return result, nil
}

func (c *SimpleJobClient) InsertTestConnectionJob(ctx context.Context, args TestConnectionArgs) (*SimpleJobResult, error) {
	result := &SimpleJobResult{
		ID:          time.Now().Unix(),
		Kind:        "test_connection",
		State:       "available",
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}
	return result, nil
}

// Job management methods
func (c *SimpleJobClient) CancelJob(ctx context.Context, jobID int64) error {
	return nil
}

func (c *SimpleJobClient) GetJob(ctx context.Context, jobID int64) (*SimpleJobResult, error) {
	return &SimpleJobResult{
		ID:    jobID,
		Kind:  "unknown",
		State: "completed",
	}, nil
}

func (c *SimpleJobClient) ListJobsForOrganization(ctx context.Context, organizationID uuid.UUID, limit int, offset int) ([]*SimpleJobResult, error) {
	// Return empty list for now
	return []*SimpleJobResult{}, nil
}

func (c *SimpleJobClient) GetJobStatsForOrganization(ctx context.Context, organizationID uuid.UUID, since time.Time) (map[string]interface{}, error) {
	return map[string]interface{}{
		"by_state": map[string]interface{}{
			"completed": map[string]interface{}{
				"count": 0,
			},
		},
		"total": 0,
	}, nil
}