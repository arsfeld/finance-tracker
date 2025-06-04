package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	
	"finance_tracker/src/internal/jobs"
)

// Helper function to get organization ID from request context
func GetOrganizationID(r *http.Request) uuid.UUID {
	// TODO: Extract this from the auth middleware context
	// For now, return a placeholder UUID
	return uuid.New()
}

type RiverJobHandler struct {
	jobClient *jobs.RiverJobClient
}

func NewRiverJobHandler(jobClient *jobs.RiverJobClient) *RiverJobHandler {
	return &RiverJobHandler{
		jobClient: jobClient,
	}
}

// CreateSyncJob handles POST /api/v1/connections/{id}/sync with River
func (h *RiverJobHandler) CreateSyncJob(w http.ResponseWriter, r *http.Request) {
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
		Type           string     `json:"type"`          // "full", "transactions", "accounts", "test"
		StartDate      *time.Time `json:"start_date"`
		EndDate        *time.Time `json:"end_date"`
		ForceSync      bool       `json:"force_sync"`
		IncludeHistory bool       `json:"include_history"`
		Priority       string     `json:"priority"`      // "low", "normal", "high", "urgent"
		ScheduledAt    *time.Time `json:"scheduled_at"`  // Optional scheduling
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
	var result *rivertype.JobInsertResult
	
	switch req.Type {
	case "full":
		result, err = h.jobClient.InsertFullSyncJob(r.Context(), jobs.FullSyncArgs{
			OrganizationID: organizationID,
			ConnectionID:   connectionID,
			StartDate:      req.StartDate,
			IncludeHistory: req.IncludeHistory,
			ForceSync:      req.ForceSync,
		})
		
	case "transactions":
		result, err = h.jobClient.InsertSyncTransactionsJob(r.Context(), jobs.SyncTransactionsArgs{
			OrganizationID: organizationID,
			ConnectionID:   connectionID,
			StartDate:      req.StartDate,
			EndDate:        req.EndDate,
			ForceSync:      req.ForceSync,
		})
		
	case "accounts":
		result, err = h.jobClient.InsertSyncAccountsJob(r.Context(), jobs.SyncAccountsArgs{
			OrganizationID: organizationID,
			ConnectionID:   connectionID,
			ForceSync:      req.ForceSync,
		})
		
	case "test":
		result, err = h.jobClient.InsertTestConnectionJob(r.Context(), jobs.TestConnectionArgs{
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
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"id":           result.Job.ID,
		"kind":         result.Job.Kind,
		"state":        string(result.Job.State),
		"queue":        result.Job.Queue,
		"priority":     result.Job.Priority,
		"created_at":   result.Job.CreatedAt,
		"scheduled_at": result.Job.ScheduledAt,
		"metadata":     result.Job.Metadata,
		"attempt":      result.Job.Attempt,
		"max_attempts": result.Job.MaxAttempts,
	})
}

// CreateAnalysisJob handles POST /api/v1/analysis/jobs
func (h *RiverJobHandler) CreateAnalysisJob(w http.ResponseWriter, r *http.Request) {
	organizationID := GetOrganizationID(r)
	if organizationID == uuid.Nil {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		AccountIDs     []uuid.UUID `json:"account_ids,omitempty"`
		StartDate      *time.Time  `json:"start_date"`
		EndDate        *time.Time  `json:"end_date"`
		AnalysisType   string      `json:"analysis_type"` // "spending", "trends", "insights", "budgets"
		NotifyChannels []string    `json:"notify_channels,omitempty"` // "email", "ntfy", "webhook"
		ScheduledAt    *time.Time  `json:"scheduled_at"`  // Optional scheduling
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default values
	if req.AnalysisType == "" {
		req.AnalysisType = "spending"
	}

	result, err := h.jobClient.InsertAnalyzeSpendingJob(r.Context(), jobs.AnalyzeSpendingArgs{
		OrganizationID: organizationID,
		AccountIDs:     req.AccountIDs,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		AnalysisType:   req.AnalysisType,
		NotifyChannels: req.NotifyChannels,
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create analysis job: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"id":           result.Job.ID,
		"kind":         result.Job.Kind,
		"state":        string(result.Job.State),
		"queue":        result.Job.Queue,
		"priority":     result.Job.Priority,
		"created_at":   result.Job.CreatedAt,
		"scheduled_at": result.Job.ScheduledAt,
		"metadata":     result.Job.Metadata,
	})
}

// CreateMaintenanceJob handles POST /api/v1/maintenance/jobs
func (h *RiverJobHandler) CreateMaintenanceJob(w http.ResponseWriter, r *http.Request) {
	organizationID := GetOrganizationID(r)
	if organizationID == uuid.Nil {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Type        string     `json:"type"`        // "cleanup", "backup"
		SubType     string     `json:"sub_type"`    // For cleanup: "old_jobs", "cache", etc. For backup: "full", "incremental"
		OlderThan   *time.Time `json:"older_than,omitempty"`
		DryRun      bool       `json:"dry_run"`
		Destination string     `json:"destination,omitempty"` // For backup jobs
		Encrypt     bool       `json:"encrypt"`
		Compress    bool       `json:"compress"`
		ScheduledAt *time.Time `json:"scheduled_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var result *rivertype.JobInsertResult
	var err error

	switch req.Type {
	case "cleanup":
		result, err = h.jobClient.InsertCleanupJob(r.Context(), jobs.CleanupArgs{
			OrganizationID: organizationID,
			Type:           req.SubType,
			OlderThan:      req.OlderThan,
			DryRun:         req.DryRun,
		})
		
	case "backup":
		if req.Destination == "" {
			req.Destination = "supabase"
		}
		result, err = h.jobClient.InsertBackupJob(r.Context(), jobs.BackupArgs{
			OrganizationID: organizationID,
			Type:           req.SubType,
			Destination:    req.Destination,
			Encrypt:        req.Encrypt,
			Compress:       req.Compress,
		})
		
	default:
		http.Error(w, "Invalid maintenance job type", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create maintenance job: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"id":           result.Job.ID,
		"kind":         result.Job.Kind,
		"state":        string(result.Job.State),
		"queue":        result.Job.Queue,
		"priority":     result.Job.Priority,
		"created_at":   result.Job.CreatedAt,
		"scheduled_at": result.Job.ScheduledAt,
		"metadata":     result.Job.Metadata,
	})
}

// ListJobs handles GET /api/v1/jobs with River backend
func (h *RiverJobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
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

	// Optional filters
	queue := r.URL.Query().Get("queue")
	state := r.URL.Query().Get("state")
	kind := r.URL.Query().Get("kind")

	jobs, err := h.jobClient.ListJobsForOrganization(r.Context(), organizationID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to list jobs", http.StatusInternalServerError)
		return
	}

	// Convert River jobs to API format
	apiJobs := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		// Apply filters if specified
		if queue != "" && job.Queue != queue {
			continue
		}
		if state != "" && string(job.State) != state {
			continue
		}
		if kind != "" && job.Kind != kind {
			continue
		}

		// Calculate progress (simplified - you'd get this from job metadata)
		progress := 0
		if job.State == rivertype.JobStateCompleted {
			progress = 100
		} else if job.State == rivertype.JobStateRunning {
			progress = 50 // You'd calculate this based on job metadata
		}

		apiJobs[i] = map[string]interface{}{
			"id":               job.ID,
			"type":             job.Kind,
			"status":           string(job.State),
			"queue":            job.Queue,
			"priority":         job.Priority,
			"title":            fmt.Sprintf("%s job", job.Kind),
			"progress_current": progress,
			"progress_total":   100,
			"created_at":       job.CreatedAt,
			"scheduled_at":     job.ScheduledAt,
			"attempted_at":     job.AttemptedAt,
			"finalized_at":     job.FinalizedAt,
			"attempt":          job.Attempt,
			"max_attempts":     job.MaxAttempts,
			"errors":           job.Errors,
			"metadata":         job.Metadata,
			"tags":             job.Tags,
		}
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"jobs": apiJobs,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  len(apiJobs), // You'd get actual total from database
		},
	})
}

// GetJob handles GET /api/v1/jobs/{id} with River backend
func (h *RiverJobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
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

	// Calculate progress
	progress := 0
	if job.State == rivertype.JobStateCompleted {
		progress = 100
	} else if job.State == rivertype.JobStateRunning {
		progress = 50 // You'd get this from job metadata
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"id":               job.ID,
		"type":             job.Kind,
		"status":           string(job.State),
		"queue":            job.Queue,
		"priority":         job.Priority,
		"title":            fmt.Sprintf("%s job", job.Kind),
		"progress_current": progress,
		"progress_total":   100,
		"created_at":       job.CreatedAt,
		"scheduled_at":     job.ScheduledAt,
		"attempted_at":     job.AttemptedAt,
		"finalized_at":     job.FinalizedAt,
		"attempt":          job.Attempt,
		"max_attempts":     job.MaxAttempts,
		"errors":           job.Errors,
		"metadata":         job.Metadata,
		"tags":             job.Tags,
		"args":             job.EncodedArgs, // Raw job arguments
	})
}

// CancelJob handles POST /api/v1/jobs/{id}/cancel with River backend
func (h *RiverJobHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
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

	respondWithJSON(w, r, http.StatusOK, map[string]string{"status": "cancelled"})
}

// GetJobStats handles GET /api/v1/jobs/stats with River backend
func (h *RiverJobHandler) GetJobStats(w http.ResponseWriter, r *http.Request) {
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

	respondWithJSON(w, r, http.StatusOK, stats)
}

// GetWorkerStats handles GET /api/v1/workers/stats with River backend
func (h *RiverJobHandler) GetWorkerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.jobClient.GetWorkerStats(r.Context())
	if err != nil {
		http.Error(w, "Failed to get worker stats", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, r, http.StatusOK, stats)
}

// ListWorkers handles GET /api/v1/workers with River backend
func (h *RiverJobHandler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := h.jobClient.ListWorkers(r.Context())
	if err != nil {
		http.Error(w, "Failed to list workers", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"workers": workers,
	})
}

// GetQueues handles GET /api/v1/queues
func (h *RiverJobHandler) GetQueues(w http.ResponseWriter, r *http.Request) {
	queues := []map[string]interface{}{
		{
			"name":        river.QueueDefault,
			"max_workers": 10,
			"state":       "active",
		},
		{
			"name":        "sync",
			"max_workers": 5,
			"state":       "active",
		},
		{
			"name":        "analysis",
			"max_workers": 3,
			"state":       "active",
		},
		{
			"name":        "maintenance",
			"max_workers": 2,
			"state":       "active",
		},
		{
			"name":        "high_priority",
			"max_workers": 8,
			"state":       "active",
		},
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"queues": queues,
	})
}

// HealthCheck for the job system
func (h *RiverJobHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.jobClient.HealthCheck(r.Context()); err != nil {
		http.Error(w, "Job system unhealthy", http.StatusServiceUnavailable)
		return
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "river-0.11.4",
	})
}