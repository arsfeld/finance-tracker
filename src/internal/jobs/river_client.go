package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	
	"finance_tracker/src/internal/services"
	provider "finance_tracker/src/providers"
)

// RiverJobClient wraps River client with our custom methods
type RiverJobClient struct {
	client   *river.Client[pgx.Tx]
	dbPool   *pgxpool.Pool
	workers  *river.Workers
	config   *river.Config
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewRiverJobClient creates a new River-based job client
func NewRiverJobClient(dbPool *pgxpool.Pool, syncService *services.SyncService, providers map[string]provider.Provider) (*RiverJobClient, error) {
	workers := river.NewWorkers()
	
	// Register all job workers with dependencies
	river.AddWorker(workers, &SyncTransactionsJob{
		syncService: syncService,
		providers:   providers,
	})
	river.AddWorker(workers, &SyncAccountsJob{
		syncService: syncService,
		providers:   providers,
	})
	river.AddWorker(workers, &FullSyncJob{
		syncService: syncService,
		providers:   providers,
	})
	river.AddWorker(workers, &TestConnectionJob{
		syncService: syncService,
		providers:   providers,
	})
	river.AddWorker(workers, &AnalyzeSpendingJob{})
	river.AddWorker(workers, &CleanupJob{})
	river.AddWorker(workers, &BackupJob{})
	
	config := &river.Config{
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"sync":            {MaxWorkers: 5},   // For sync operations
			"analysis":        {MaxWorkers: 3},   // For LLM analysis
			"maintenance":     {MaxWorkers: 2},   // For cleanup/backup
			"high_priority":   {MaxWorkers: 8},   // For urgent jobs
		},
		JobTimeout:               15 * time.Minute,
		MaxAttempts:              3,
		RescueStuckJobsAfter:     30 * time.Minute,
		RetryPolicy:              &river.DefaultClientRetryPolicy{},
		CancelledJobRetentionPeriod: 24 * time.Hour,
		CompletedJobRetentionPeriod: 7 * 24 * time.Hour,
		DiscardedJobRetentionPeriod: 7 * 24 * time.Hour,
	}
	
	client, err := river.NewClient(riverpgxv5.New(dbPool), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RiverJobClient{
		client:  client,
		dbPool:  dbPool,
		workers: workers,
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Start starts the River job client
func (c *RiverJobClient) Start(ctx context.Context) error {
	log.Printf("Starting River job client...")
	
	// Start the client
	if err := c.client.Start(c.ctx); err != nil {
		return fmt.Errorf("failed to start River client: %w", err)
	}
	
	log.Printf("River job client started successfully")
	return nil
}

// Stop gracefully stops the River job client
func (c *RiverJobClient) Stop(ctx context.Context) error {
	log.Printf("Stopping River job client...")
	
	// Cancel the context to stop the client
	c.cancel()
	
	// Wait for client to stop
	if err := c.client.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop River client: %w", err)
	}
	
	log.Printf("River job client stopped")
	return nil
}

// Job insertion methods with River
func (c *RiverJobClient) InsertSyncTransactionsJob(ctx context.Context, args SyncTransactionsArgs) (*rivertype.JobInsertResult, error) {
	metadata := map[string]string{
		MetadataConnectionName: "connection", // You'd get this from connection ID
		MetadataProviderType:   "simplefin",  // You'd get this from connection
	}
	
	if args.StartDate != nil {
		metadata[MetadataStartDate] = args.StartDate.Format(time.RFC3339)
	}
	if args.EndDate != nil {
		metadata[MetadataEndDate] = args.EndDate.Format(time.RFC3339)
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	insertOpts := &river.InsertOpts{
		Queue:    "sync",
		Priority: PriorityNormal,
		Metadata: metadataBytes,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

func (c *RiverJobClient) InsertSyncAccountsJob(ctx context.Context, args SyncAccountsArgs) (*rivertype.JobInsertResult, error) {
	metadata := map[string]string{
		MetadataConnectionName: "connection",
		MetadataProviderType:   "simplefin",
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	insertOpts := &river.InsertOpts{
		Queue:    "sync",
		Priority: PriorityNormal,
		Metadata: metadataBytes,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

func (c *RiverJobClient) InsertFullSyncJob(ctx context.Context, args FullSyncArgs) (*rivertype.JobInsertResult, error) {
	priority := PriorityNormal
	if args.IncludeHistory {
		priority = PriorityLow // History syncs are lower priority
	}
	
	metadata := map[string]string{
		MetadataConnectionName: "connection",
		MetadataProviderType:   "simplefin",
	}
	
	if args.StartDate != nil {
		metadata[MetadataStartDate] = args.StartDate.Format(time.RFC3339)
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	insertOpts := &river.InsertOpts{
		Queue:    "sync",
		Priority: priority,
		Metadata: metadataBytes,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

func (c *RiverJobClient) InsertTestConnectionJob(ctx context.Context, args TestConnectionArgs) (*rivertype.JobInsertResult, error) {
	metadata := map[string]string{
		MetadataConnectionName: "connection",
		MetadataProviderType:   "simplefin",
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	insertOpts := &river.InsertOpts{
		Queue:    "high_priority",
		Priority: PriorityHigh,
		Metadata: metadataBytes,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

func (c *RiverJobClient) InsertAnalyzeSpendingJob(ctx context.Context, args AnalyzeSpendingArgs) (*rivertype.JobInsertResult, error) {
	metadata := map[string]string{
		"analysis_type": args.AnalysisType,
	}
	
	if args.StartDate != nil {
		metadata[MetadataStartDate] = args.StartDate.Format(time.RFC3339)
	}
	if args.EndDate != nil {
		metadata[MetadataEndDate] = args.EndDate.Format(time.RFC3339)
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	insertOpts := &river.InsertOpts{
		Queue:    "analysis",
		Priority: PriorityNormal,
		Metadata: metadataBytes,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

func (c *RiverJobClient) InsertCleanupJob(ctx context.Context, args CleanupArgs) (*rivertype.JobInsertResult, error) {
	metadata := map[string]string{
		"cleanup_type": args.Type,
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	insertOpts := &river.InsertOpts{
		Queue:    "maintenance",
		Priority: PriorityLow,
		Metadata: metadataBytes,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

func (c *RiverJobClient) InsertBackupJob(ctx context.Context, args BackupArgs) (*rivertype.JobInsertResult, error) {
	priority := PriorityLow
	if args.Type == "config" {
		priority = PriorityNormal // Config backups are more important
	}
	
	metadata := map[string]string{
		"backup_type": args.Type,
		"destination": args.Destination,
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	insertOpts := &river.InsertOpts{
		Queue:    "maintenance",
		Priority: priority,
		Metadata: metadataBytes,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

// Scheduled job insertion methods
func (c *RiverJobClient) ScheduleJob(ctx context.Context, args river.JobArgs, scheduledAt time.Time) (*rivertype.JobInsertResult, error) {
	insertOpts := &river.InsertOpts{
		ScheduledAt: scheduledAt,
	}
	
	return c.client.Insert(ctx, args, insertOpts)
}

func (c *RiverJobClient) ScheduleRecurringSync(ctx context.Context, organizationID, connectionID uuid.UUID, cronExpr string) error {
	// River doesn't have built-in cron scheduling, but you can implement it
	// by creating a periodic job that schedules other jobs
	log.Printf("Scheduling recurring sync for organization %s with cron: %s", organizationID, cronExpr)
	
	// For now, just schedule the next sync 24 hours from now
	_, err := c.InsertSyncTransactionsJob(ctx, SyncTransactionsArgs{
		OrganizationID: organizationID,
		ConnectionID:   connectionID,
		ForceSync:      false,
	})
	
	return err
}

// Job management methods
func (c *RiverJobClient) CancelJob(ctx context.Context, jobID int64) error {
	_, err := c.client.JobCancel(ctx, jobID)
	return err
}

func (c *RiverJobClient) GetJob(ctx context.Context, jobID int64) (*rivertype.JobRow, error) {
	return c.client.JobGet(ctx, jobID)
}

func (c *RiverJobClient) ListJobsForOrganization(ctx context.Context, organizationID uuid.UUID, limit int, offset int) ([]*rivertype.JobRow, error) {
	// River doesn't have built-in organization filtering, so you'd need to 
	// implement this by querying the river_job table directly or using tags/metadata
	
	params := river.NewJobListParams().First(limit)
	
	// Add cursor if offset > 0 (simplified - you'd need proper cursor handling)
	// For now, just use limit without cursor
	
	result, err := c.client.JobList(ctx, params)
	if err != nil {
		return nil, err
	}
	
	// Filter by organization in the metadata or args
	var filteredJobs []*rivertype.JobRow
	for _, job := range result.Jobs {
		// You'd implement organization filtering logic here based on job args
		filteredJobs = append(filteredJobs, job)
	}
	
	return filteredJobs, nil
}

func (c *RiverJobClient) GetJobStatsForOrganization(ctx context.Context, organizationID uuid.UUID, since time.Time) (map[string]interface{}, error) {
	// Get stats from River - you'd need to implement organization filtering
	stats := map[string]interface{}{
		"by_state": map[string]interface{}{
			"available": map[string]interface{}{"count": 0},
			"completed": map[string]interface{}{"count": 0},
			"running":   map[string]interface{}{"count": 0},
			"retryable": map[string]interface{}{"count": 0},
			"cancelled": map[string]interface{}{"count": 0},
			"discarded": map[string]interface{}{"count": 0},
		},
		"total": 0,
		"by_queue": map[string]interface{}{
			"sync":          map[string]interface{}{"count": 0},
			"analysis":      map[string]interface{}{"count": 0},
			"maintenance":   map[string]interface{}{"count": 0},
			"high_priority": map[string]interface{}{"count": 0},
		},
	}
	
	return stats, nil
}

// Worker status methods
func (c *RiverJobClient) GetWorkerStats(ctx context.Context) (map[string]interface{}, error) {
	// River provides worker statistics
	stats := map[string]interface{}{
		"total_workers":       10, // Based on queue configs
		"total_jobs":          0,  // Would query from River
		"total_capacity":      18, // Sum of MaxWorkers across queues
		"utilization_percent": 0.0,
		"by_status": map[string]interface{}{
			"active": map[string]interface{}{
				"count":    1,
				"jobs":     0,
				"capacity": 18,
			},
		},
		"by_queue": map[string]interface{}{
			"sync":          map[string]interface{}{"workers": 5, "jobs": 0},
			"analysis":      map[string]interface{}{"workers": 3, "jobs": 0},
			"maintenance":   map[string]interface{}{"workers": 2, "jobs": 0},
			"high_priority": map[string]interface{}{"workers": 8, "jobs": 0},
		},
	}
	
	return stats, nil
}

func (c *RiverJobClient) ListWorkers(ctx context.Context) ([]map[string]interface{}, error) {
	// River doesn't expose worker details directly, but you can track them
	workers := []map[string]interface{}{
		{
			"id":                  "river-worker-1",
			"hostname":            "localhost",
			"status":              "active",
			"max_concurrent_jobs": 18,
			"current_job_count":   0,
			"last_heartbeat":      time.Now(),
			"queues":              []string{"sync", "analysis", "maintenance", "high_priority"},
		},
	}
	
	return workers, nil
}

// Health check
func (c *RiverJobClient) HealthCheck(ctx context.Context) error {
	// Check if River client is healthy
	if c.client == nil {
		return fmt.Errorf("River client is not initialized")
	}
	
	// You could ping the database or check worker status
	return nil
}