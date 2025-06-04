package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"finance_tracker/src/internal/models"
)

type WorkerService struct {
	db *sqlx.DB
}

func NewWorkerService(db *sqlx.DB) *WorkerService {
	return &WorkerService{db: db}
}

// RegisterWorker registers a new worker or updates existing one
func (s *WorkerService) RegisterWorker(ctx context.Context, workerID string, maxConcurrentJobs int) error {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	pid := os.Getpid()

	query := `SELECT update_worker_heartbeat($1, $2, $3, $4)`
	
	_, err = s.db.ExecContext(ctx, query, workerID, hostname, pid, maxConcurrentJobs)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	return nil
}

// UpdateHeartbeat updates worker heartbeat
func (s *WorkerService) UpdateHeartbeat(ctx context.Context, workerID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	pid := os.Getpid()

	query := `SELECT update_worker_heartbeat($1, $2, $3, $4)`
	
	_, err = s.db.ExecContext(ctx, query, workerID, hostname, pid, 1)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return nil
}

// GetWorker retrieves a worker by ID
func (s *WorkerService) GetWorker(ctx context.Context, workerID string) (*models.JobWorker, error) {
	var worker models.JobWorker
	query := `SELECT * FROM job_workers WHERE id = $1`
	
	err := s.db.GetContext(ctx, &worker, query, workerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worker not found")
		}
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	return &worker, nil
}

// ListWorkers lists all registered workers
func (s *WorkerService) ListWorkers(ctx context.Context) ([]*models.JobWorker, error) {
	var workers []*models.JobWorker
	query := `SELECT * FROM job_workers ORDER BY last_heartbeat DESC`
	
	err := s.db.SelectContext(ctx, &workers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}

	return workers, nil
}

// ListHealthyWorkers lists workers that have sent heartbeat recently
func (s *WorkerService) ListHealthyWorkers(ctx context.Context, threshold time.Duration) ([]*models.JobWorker, error) {
	var workers []*models.JobWorker
	query := `
		SELECT * FROM job_workers 
		WHERE last_heartbeat >= $1 AND status = 'active'
		ORDER BY last_heartbeat DESC
	`
	
	cutoff := time.Now().Add(-threshold)
	err := s.db.SelectContext(ctx, &workers, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to list healthy workers: %w", err)
	}

	return workers, nil
}

// UpdateWorkerStatus updates worker status (active, paused, stopping)
func (s *WorkerService) UpdateWorkerStatus(ctx context.Context, workerID, status string) error {
	query := `UPDATE job_workers SET status = $1 WHERE id = $2`
	
	result, err := s.db.ExecContext(ctx, query, status, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("worker not found")
	}

	return nil
}

// UpdateWorkerJobCount updates current job count for a worker
func (s *WorkerService) UpdateWorkerJobCount(ctx context.Context, workerID string, count int) error {
	query := `UPDATE job_workers SET current_job_count = $1 WHERE id = $2`
	
	result, err := s.db.ExecContext(ctx, query, count, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker job count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("worker not found")
	}

	return nil
}

// RemoveWorker removes a worker (when shutting down)
func (s *WorkerService) RemoveWorker(ctx context.Context, workerID string) error {
	query := `DELETE FROM job_workers WHERE id = $1`
	
	_, err := s.db.ExecContext(ctx, query, workerID)
	if err != nil {
		return fmt.Errorf("failed to remove worker: %w", err)
	}

	return nil
}

// CleanupStaleWorkers removes workers that haven't sent heartbeat for too long
func (s *WorkerService) CleanupStaleWorkers(ctx context.Context, threshold time.Duration) (int, error) {
	query := `DELETE FROM job_workers WHERE last_heartbeat < $1`
	
	cutoff := time.Now().Add(-threshold)
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup stale workers: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// GetWorkerStats gets worker statistics
func (s *WorkerService) GetWorkerStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			status,
			COUNT(*) as count,
			SUM(current_job_count) as total_jobs,
			SUM(max_concurrent_jobs) as total_capacity
		FROM job_workers 
		WHERE last_heartbeat >= $1
		GROUP BY status
	`

	// Consider workers healthy if heartbeat within last 5 minutes
	cutoff := time.Now().Add(-5 * time.Minute)
	
	rows, err := s.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker stats: %w", err)
	}
	defer rows.Close()

	stats := map[string]interface{}{
		"by_status": make(map[string]map[string]interface{}),
		"total_workers": 0,
		"total_jobs": 0,
		"total_capacity": 0,
	}

	totalWorkers := 0
	totalJobs := 0
	totalCapacity := 0

	for rows.Next() {
		var status string
		var count, jobs, capacity int

		err := rows.Scan(&status, &count, &jobs, &capacity)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker stats: %w", err)
		}

		stats["by_status"].(map[string]map[string]interface{})[status] = map[string]interface{}{
			"count": count,
			"jobs": jobs,
			"capacity": capacity,
		}

		totalWorkers += count
		totalJobs += jobs
		totalCapacity += capacity
	}

	stats["total_workers"] = totalWorkers
	stats["total_jobs"] = totalJobs
	stats["total_capacity"] = totalCapacity

	// Calculate utilization percentage
	if totalCapacity > 0 {
		stats["utilization_percent"] = float64(totalJobs) / float64(totalCapacity) * 100
	} else {
		stats["utilization_percent"] = 0.0
	}

	return stats, nil
}

// CanAcceptJob checks if a worker can accept more jobs
func (s *WorkerService) CanAcceptJob(ctx context.Context, workerID string) (bool, error) {
	var currentJobs, maxJobs int
	query := `SELECT current_job_count, max_concurrent_jobs FROM job_workers WHERE id = $1 AND status = 'active'`
	
	err := s.db.QueryRowContext(ctx, query, workerID).Scan(&currentJobs, &maxJobs)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("worker not found or not active")
		}
		return false, fmt.Errorf("failed to check worker capacity: %w", err)
	}

	return currentJobs < maxJobs, nil
}

// GetAvailableWorkers gets workers that can accept more jobs
func (s *WorkerService) GetAvailableWorkers(ctx context.Context) ([]*models.JobWorker, error) {
	var workers []*models.JobWorker
	query := `
		SELECT * FROM job_workers 
		WHERE status = 'active' 
			AND current_job_count < max_concurrent_jobs
			AND last_heartbeat >= $1
		ORDER BY current_job_count ASC, last_heartbeat DESC
	`
	
	// Consider workers healthy if heartbeat within last 2 minutes
	cutoff := time.Now().Add(-2 * time.Minute)
	
	err := s.db.SelectContext(ctx, &workers, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get available workers: %w", err)
	}

	return workers, nil
}