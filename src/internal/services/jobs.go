package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"finance_tracker/src/internal/models"
)

type JobService struct {
	db *sqlx.DB
}

func NewJobService(db *sqlx.DB) *JobService {
	return &JobService{db: db}
}

// CreateJob creates a new job
func (s *JobService) CreateJob(ctx context.Context, params models.CreateJobParams) (*models.Job, error) {
	job := &models.Job{
		ID:             uuid.New(),
		OrganizationID: params.OrganizationID,
		Type:           params.Type,
		Status:         models.JobStatusPending,
		Priority:       params.Priority,
		Title:          params.Title,
		Description:    params.Description,
		ProviderConnectionID: params.ProviderConnectionID,
		BankAccountID:  params.BankAccountID,
		Parameters:     params.Parameters,
		ProgressCurrent: 0,
		ProgressTotal:   100,
		CreatedAt:      time.Now(),
		ScheduledAt:    time.Now(),
		RetryCount:     0,
		MaxRetries:     3,
		RetryDelaySeconds: 60,
	}

	// Apply custom scheduling and retry settings
	if params.ScheduledAt != nil {
		job.ScheduledAt = *params.ScheduledAt
	}
	if params.MaxRetries != nil {
		job.MaxRetries = *params.MaxRetries
	}
	if params.RetryDelaySeconds != nil {
		job.RetryDelaySeconds = *params.RetryDelaySeconds
	}

	query := `
		INSERT INTO jobs (
			id, organization_id, type, status, priority, title, description,
			provider_connection_id, bank_account_id, parameters, progress_current,
			progress_total, created_at, scheduled_at, retry_count, max_retries,
			retry_delay_seconds
		) VALUES (
			:id, :organization_id, :type, :status, :priority, :title, :description,
			:provider_connection_id, :bank_account_id, :parameters, :progress_current,
			:progress_total, :created_at, :scheduled_at, :retry_count, :max_retries,
			:retry_delay_seconds
		)`

	_, err := s.db.NamedExecContext(ctx, query, job)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// GetJob retrieves a job by ID
func (s *JobService) GetJob(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	query := `SELECT * FROM jobs WHERE id = $1`
	
	err := s.db.GetContext(ctx, &job, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// ListJobs lists jobs with filters
func (s *JobService) ListJobs(ctx context.Context, filters models.JobFilters) ([]*models.Job, error) {
	query := `SELECT * FROM jobs WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	// Build dynamic query based on filters
	if filters.OrganizationID != nil {
		query += fmt.Sprintf(" AND organization_id = $%d", argIndex)
		args = append(args, *filters.OrganizationID)
		argIndex++
	}

	if len(filters.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}

	if len(filters.Type) > 0 {
		query += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
		args = append(args, filters.Type)
		argIndex++
	}

	if len(filters.Priority) > 0 {
		query += fmt.Sprintf(" AND priority = ANY($%d)", argIndex)
		args = append(args, filters.Priority)
		argIndex++
	}

	if filters.ProviderConnectionID != nil {
		query += fmt.Sprintf(" AND provider_connection_id = $%d", argIndex)
		args = append(args, *filters.ProviderConnectionID)
		argIndex++
	}

	if filters.WorkerID != nil {
		query += fmt.Sprintf(" AND worker_id = $%d", argIndex)
		args = append(args, *filters.WorkerID)
		argIndex++
	}

	if filters.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filters.CreatedAfter)
		argIndex++
	}

	if filters.CreatedBefore != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filters.CreatedBefore)
		argIndex++
	}

	// Order and pagination
	query += " ORDER BY priority DESC, scheduled_at ASC"
	
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	var jobs []*models.Job
	err := s.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}

// UpdateJob updates job fields
func (s *JobService) UpdateJob(ctx context.Context, id uuid.UUID, params models.UpdateJobParams) error {
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if params.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *params.Status)
		argIndex++
		
		// Set completed_at when job completes
		if *params.Status == models.JobStatusCompleted || 
		   *params.Status == models.JobStatusFailed || 
		   *params.Status == models.JobStatusCancelled {
			updates = append(updates, fmt.Sprintf("completed_at = $%d", argIndex))
			args = append(args, time.Now())
			argIndex++
		}
	}

	if params.Priority != nil {
		updates = append(updates, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, *params.Priority)
		argIndex++
	}

	if params.ProgressCurrent != nil {
		updates = append(updates, fmt.Sprintf("progress_current = $%d", argIndex))
		args = append(args, *params.ProgressCurrent)
		argIndex++
	}

	if params.ProgressTotal != nil {
		updates = append(updates, fmt.Sprintf("progress_total = $%d", argIndex))
		args = append(args, *params.ProgressTotal)
		argIndex++
	}

	if params.ProgressMessage != nil {
		updates = append(updates, fmt.Sprintf("progress_message = $%d", argIndex))
		args = append(args, *params.ProgressMessage)
		argIndex++
	}

	if params.Result != nil {
		updates = append(updates, fmt.Sprintf("result = $%d", argIndex))
		args = append(args, *params.Result)
		argIndex++
	}

	if params.ErrorMessage != nil {
		updates = append(updates, fmt.Sprintf("error_message = $%d", argIndex))
		args = append(args, *params.ErrorMessage)
		argIndex++
	}

	if params.ErrorDetails != nil {
		updates = append(updates, fmt.Sprintf("error_details = $%d", argIndex))
		args = append(args, *params.ErrorDetails)
		argIndex++
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf("UPDATE jobs SET %s WHERE id = $%d", 
		fmt.Sprintf("%s", updates), argIndex)
	args = append(args, id)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found")
	}

	return nil
}

// DeleteJob deletes a job
func (s *JobService) DeleteJob(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM jobs WHERE id = $1`
	
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found")
	}

	return nil
}

// GetNextJob gets the next available job for a worker using database function
func (s *JobService) GetNextJob(ctx context.Context, workerID string) (*models.Job, error) {
	var job models.Job
	query := `
		SELECT job_id as id, job_type as type, parameters, provider_connection_id, bank_account_id
		FROM get_next_job($1)
	`
	
	rows, err := s.db.QueryContext(ctx, query, workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next job: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil // No jobs available
	}

	err = rows.Scan(&job.ID, &job.Type, &job.Parameters, &job.ProviderConnectionID, &job.BankAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to scan job result: %w", err)
	}

	// Get full job details
	return s.GetJob(ctx, job.ID)
}

// UpdateJobProgress updates job progress using database function
func (s *JobService) UpdateJobProgress(ctx context.Context, jobID uuid.UUID, current, total int, message string) error {
	query := `SELECT update_job_progress($1, $2, $3, $4)`
	
	var success bool
	err := s.db.GetContext(ctx, &success, query, jobID, current, total, message)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	if !success {
		return fmt.Errorf("job not found or not running")
	}

	return nil
}

// CompleteJob completes a job using database function
func (s *JobService) CompleteJob(ctx context.Context, jobID uuid.UUID, status models.JobStatus, result json.RawMessage, errorMessage string) error {
	query := `SELECT complete_job($1, $2, $3, $4)`
	
	var success bool
	var resultParam interface{}
	var errorParam interface{}
	
	if result != nil {
		resultParam = result
	}
	if errorMessage != "" {
		errorParam = errorMessage
	}
	
	err := s.db.GetContext(ctx, &success, query, jobID, status, resultParam, errorParam)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	if !success {
		return fmt.Errorf("job not found or not running")
	}

	return nil
}

// CancelJob cancels a pending or running job
func (s *JobService) CancelJob(ctx context.Context, id uuid.UUID) error {
	return s.UpdateJob(ctx, id, models.UpdateJobParams{
		Status: &[]models.JobStatus{models.JobStatusCancelled}[0],
	})
}

// PauseJob pauses a pending job
func (s *JobService) PauseJob(ctx context.Context, id uuid.UUID) error {
	return s.UpdateJob(ctx, id, models.UpdateJobParams{
		Status: &[]models.JobStatus{models.JobStatusPaused}[0],
	})
}

// ResumeJob resumes a paused job
func (s *JobService) ResumeJob(ctx context.Context, id uuid.UUID) error {
	return s.UpdateJob(ctx, id, models.UpdateJobParams{
		Status: &[]models.JobStatus{models.JobStatusPending}[0],
	})
}

// RetryJob creates a new job from a failed job
func (s *JobService) RetryJob(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	// Get the original job
	originalJob, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get original job: %w", err)
	}

	if !originalJob.CanRetry() {
		return nil, fmt.Errorf("job cannot be retried")
	}

	// Create new job with updated retry count
	params := models.CreateJobParams{
		OrganizationID:       originalJob.OrganizationID,
		Type:                 originalJob.Type,
		Priority:             originalJob.Priority,
		Title:                originalJob.Title,
		Description:          originalJob.Description,
		ProviderConnectionID: originalJob.ProviderConnectionID,
		BankAccountID:        originalJob.BankAccountID,
		Parameters:           originalJob.Parameters,
		MaxRetries:           &originalJob.MaxRetries,
		RetryDelaySeconds:    &originalJob.RetryDelaySeconds,
	}

	// Schedule retry with delay
	retryAt := time.Now().Add(time.Duration(originalJob.RetryDelaySeconds) * time.Second)
	params.ScheduledAt = &retryAt

	newJob, err := s.CreateJob(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create retry job: %w", err)
	}

	// Update original job retry count
	_, err = s.db.ExecContext(ctx, 
		`UPDATE jobs SET retry_count = retry_count + 1 WHERE id = $1`, 
		originalJob.ID)
	if err != nil {
		log.Printf("Warning: failed to update retry count for job %s: %v", originalJob.ID, err)
	}

	return newJob, nil
}

// GetJobStats gets job statistics for an organization
func (s *JobService) GetJobStats(ctx context.Context, organizationID uuid.UUID, since time.Time) (map[string]interface{}, error) {
	query := `
		SELECT 
			status,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (COALESCE(completed_at, NOW()) - started_at)))::int as avg_duration_seconds
		FROM jobs 
		WHERE organization_id = $1 AND created_at >= $2
		GROUP BY status
	`

	rows, err := s.db.QueryContext(ctx, query, organizationID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}
	defer rows.Close()

	stats := map[string]interface{}{
		"by_status": make(map[string]map[string]interface{}),
		"total": 0,
	}

	total := 0
	for rows.Next() {
		var status string
		var count int
		var avgDuration sql.NullInt64

		err := rows.Scan(&status, &count, &avgDuration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job stats: %w", err)
		}

		statusStats := map[string]interface{}{
			"count": count,
		}
		
		if avgDuration.Valid {
			statusStats["avg_duration_seconds"] = avgDuration.Int64
		}

		stats["by_status"].(map[string]map[string]interface{})[status] = statusStats
		total += count
	}

	stats["total"] = total
	return stats, nil
}