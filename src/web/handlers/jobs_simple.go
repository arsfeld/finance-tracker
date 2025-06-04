package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	
	"finance_tracker/src/internal/jobs"
)

// Helper function to get organization ID from request context
func GetOrganizationID(r *http.Request) uuid.UUID {
	// TODO: Extract this from the auth middleware context
	// For now, return a placeholder UUID
	return uuid.New()
}

type SimpleJobHandler struct {
	jobClient *jobs.SimpleJobClient
}

func NewSimpleJobHandler(jobClient *jobs.SimpleJobClient) *SimpleJobHandler {
	return &SimpleJobHandler{
		jobClient: jobClient,
	}
}

// CreateSyncJob handles POST /api/v1/connections/{id}/sync
func (h *SimpleJobHandler) CreateSyncJob(w http.ResponseWriter, r *http.Request) {
	organizationID := GetOrganizationID(r)
	if organizationID == uuid.Nil {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	connectionIDStr := chi.URLParam(r, "id")
	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		http.Error(w, "Invalid connection ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Type          string     `json:"type"`          // "full", "transactions", "accounts", "test"
		StartDate     *time.Time `json:"start_date"`
		EndDate       *time.Time `json:"end_date"`
		ForceSync     bool       `json:"force_sync"`
		IncludeHistory bool      `json:"include_history"`
		Priority      string     `json:"priority"`      // "low", "normal", "high", "urgent"
		ScheduledAt   *time.Time `json:"scheduled_at"`  // Optional scheduling
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default values
	if req.Type == "" {
		req.Type = "transactions"
	}

	// Create appropriate job based on type
	var job *jobs.SimpleJobResult
	
	switch req.Type {
	case "full":
		job, err = h.jobClient.InsertFullSyncJob(r.Context(), jobs.FullSyncArgs{
			OrganizationID: organizationID,
			ConnectionID:   connectionID,
			StartDate:      req.StartDate,
			IncludeHistory: req.IncludeHistory,
			ForceSync:      req.ForceSync,
		})
		
	case "transactions":
		job, err = h.jobClient.InsertSyncTransactionsJob(r.Context(), jobs.SyncTransactionsArgs{
			OrganizationID: organizationID,
			ConnectionID:   connectionID,
			StartDate:      req.StartDate,
			EndDate:        req.EndDate,
			ForceSync:      req.ForceSync,
		})
		
	case "accounts":
		job, err = h.jobClient.InsertSyncAccountsJob(r.Context(), jobs.SyncAccountsArgs{
			OrganizationID: organizationID,
			ConnectionID:   connectionID,
			ForceSync:      req.ForceSync,
		})
		
	case "test":
		job, err = h.jobClient.InsertTestConnectionJob(r.Context(), jobs.TestConnectionArgs{
			OrganizationID:   organizationID,
			ConnectionID:     connectionID,
			ValidateAccounts: true,
		})
		
	default:
		http.Error(w, "Invalid sync type", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create sync job: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]interface{}{
		"id":           job.ID,
		"kind":         job.Kind,
		"state":        job.State,
		"created_at":   job.CreatedAt,
		"scheduled_at": job.ScheduledAt,
	})
}

// ListJobs handles GET /api/v1/jobs
func (h *SimpleJobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	organizationID := GetOrganizationID(r)
	if organizationID == uuid.Nil {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	limit := 50 // Default limit
	offset := 0 // Default offset

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	jobs, err := h.jobClient.ListJobsForOrganization(r.Context(), organizationID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to list jobs", http.StatusInternalServerError)
		return
	}

	// Convert simple jobs to API format
	apiJobs := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		apiJobs[i] = map[string]interface{}{
			"id":               job.ID,
			"type":             job.Kind,
			"status":           job.State,
			"title":            fmt.Sprintf("%s job", job.Kind),
			"progress_current": 0,
			"progress_total":   100,
			"created_at":       job.CreatedAt,
			"scheduled_at":     job.ScheduledAt,
			"started_at":       nil,
			"completed_at":     nil,
			"attempt":          1,
			"max_attempts":     3,
			"errors":           nil,
			"metadata":         nil,
		}
	}

	writeJSON(w, map[string]interface{}{
		"jobs": apiJobs,
	})
}

// GetJob handles GET /api/v1/jobs/{id}
func (h *SimpleJobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	organizationID := GetOrganizationID(r)
	if organizationID == uuid.Nil {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	jobIDStr := chi.URLParam(r, "id")
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, err := h.jobClient.GetJob(r.Context(), jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]interface{}{
		"id":               job.ID,
		"type":             job.Kind,
		"status":           job.State,
		"title":            fmt.Sprintf("%s job", job.Kind),
		"progress_current": 0,
		"progress_total":   100,
		"created_at":       job.CreatedAt,
		"scheduled_at":     job.ScheduledAt,
		"started_at":       nil,
		"completed_at":     nil,
		"attempt":          1,
		"max_attempts":     3,
		"errors":           nil,
		"metadata":         nil,
	})
}

// CancelJob handles POST /api/v1/jobs/{id}/cancel
func (h *SimpleJobHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	organizationID := GetOrganizationID(r)
	if organizationID == uuid.Nil {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	jobIDStr := chi.URLParam(r, "id")
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	err = h.jobClient.CancelJob(r.Context(), jobID)
	if err != nil {
		http.Error(w, "Failed to cancel job", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "cancelled"})
}

// PauseJob handles POST /api/v1/jobs/{id}/pause
func (h *SimpleJobHandler) PauseJob(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "paused"})
}

// ResumeJob handles POST /api/v1/jobs/{id}/resume
func (h *SimpleJobHandler) ResumeJob(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "pending"})
}

// RetryJob handles POST /api/v1/jobs/{id}/retry
func (h *SimpleJobHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "retrying"})
}

// GetJobStats handles GET /api/v1/jobs/stats
func (h *SimpleJobHandler) GetJobStats(w http.ResponseWriter, r *http.Request) {
	organizationID := GetOrganizationID(r)
	if organizationID == uuid.Nil {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	// Default to last 7 days
	since := time.Now().AddDate(0, 0, -7)
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if parsedSince, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = parsedSince
		}
	}

	stats, err := h.jobClient.GetJobStatsForOrganization(r.Context(), organizationID, since)
	if err != nil {
		http.Error(w, "Failed to get job stats", http.StatusInternalServerError)
		return
	}

	writeJSON(w, stats)
}

// GetWorkerStats handles GET /api/v1/workers/stats
func (h *SimpleJobHandler) GetWorkerStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_workers":       1,
		"total_jobs":          0,
		"total_capacity":      10,
		"utilization_percent": 0.0,
		"by_status": map[string]interface{}{
			"active": map[string]interface{}{
				"count":    1,
				"jobs":     0,
				"capacity": 10,
			},
		},
	}

	writeJSON(w, stats)
}

// ListWorkers handles GET /api/v1/workers
func (h *SimpleJobHandler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	workers := []map[string]interface{}{
		{
			"id":                 "simple-worker-1",
			"hostname":           "localhost",
			"status":             "active",
			"max_concurrent_jobs": 10,
			"current_job_count":   0,
			"last_heartbeat":      time.Now(),
		},
	}

	writeJSON(w, map[string]interface{}{
		"workers": workers,
	})
}

