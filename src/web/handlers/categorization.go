package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/jobs"
	"finance_tracker/src/internal/models"
	"finance_tracker/src/internal/services/categorization"
)

// CategorizationHandler handles categorization-related API endpoints
type CategorizationHandler struct {
	engine          categorization.CategorizationEngine
	ruleEngine      categorization.RuleEngine
	patternEngine   categorization.PatternEngine
	llmEngine       categorization.LLMEngine
	costManager     categorization.CostManager
	feedbackManager categorization.FeedbackManager
	riverClient     *river.Client[any]
}

// NewCategorizationHandler creates a new categorization handler
func NewCategorizationHandler(
	engine categorization.CategorizationEngine,
	ruleEngine categorization.RuleEngine,
	patternEngine categorization.PatternEngine,
	llmEngine categorization.LLMEngine,
	costManager categorization.CostManager,
	feedbackManager categorization.FeedbackManager,
	riverClient *river.Client[any],
) *CategorizationHandler {
	return &CategorizationHandler{
		engine:          engine,
		ruleEngine:      ruleEngine,
		patternEngine:   patternEngine,
		llmEngine:       llmEngine,
		costManager:     costManager,
		feedbackManager: feedbackManager,
		riverClient:     riverClient,
	}
}

// RegisterRoutes registers categorization routes
func (h *CategorizationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/categorization", func(r chi.Router) {
		// Statistics and overview
		r.Get("/stats", h.GetStats)
		r.Get("/cost", h.GetCostSummary)
		
		// Rule management
		r.Route("/rules", func(r chi.Router) {
			r.Get("/", h.GetRules)
			r.Post("/", h.CreateRule)
			r.Put("/{ruleID}", h.UpdateRule)
			r.Delete("/{ruleID}", h.DeleteRule)
			r.Post("/{ruleID}/test", h.TestRule)
		})
		
		// Pattern management
		r.Route("/patterns", func(r chi.Router) {
			r.Get("/", h.GetPatterns)
			r.Get("/similar", h.GetSimilarPatterns)
			r.Delete("/cache", h.ClearPatternCache)
		})
		
		// Batch operations
		r.Route("/batch", func(r chi.Router) {
			r.Post("/categorize", h.CreateBatchCategorizationJob)
			r.Post("/estimate", h.EstimateBatchCost)
		})
		
		// Individual transaction categorization
		r.Route("/transactions", func(r chi.Router) {
			r.Post("/{transactionID}/categorize", h.CategorizeTransaction)
			r.Post("/{transactionID}/feedback", h.RecordFeedback)
		})
		
		// Feedback management
		r.Route("/feedback", func(r chi.Router) {
			r.Get("/", h.GetFeedback)
			r.Get("/analysis", h.GetFeedbackAnalysis)
		})
		
		// Model and cost management
		r.Route("/models", func(r chi.Router) {
			r.Get("/", h.GetAvailableModels)
			r.Get("/best", h.GetBestModel)
		})
		
		// Budget management
		r.Route("/budget", func(r chi.Router) {
			r.Get("/", h.GetBudget)
			r.Put("/", h.UpdateBudget)
		})
	})
}

// GetStats returns categorization statistics for the organization
func (h *CategorizationHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	stats, err := h.engine.GetCategorizationStats(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, stats)
}

// GetCostSummary returns cost summary for the organization
func (h *CategorizationHandler) GetCostSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	tracker, err := h.costManager.GetCostTracker(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get cost tracker: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, tracker)
}

// GetRules returns all categorization rules for the organization
func (h *CategorizationHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	rules, err := h.ruleEngine.GetRules(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get rules: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, rules)
}

// CreateRule creates a new categorization rule
func (h *CategorizationHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	userID := getUserID(ctx)
	
	var rule models.EnhancedCategoryRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	// Set organization and user
	rule.OrganizationID = orgID
	rule.CreatedBy = &userID
	
	if err := h.ruleEngine.AddRule(ctx, &rule); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create rule: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, rule)
}

// UpdateRule updates an existing categorization rule
func (h *CategorizationHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}
	
	var rule models.EnhancedCategoryRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	// Ensure rule belongs to organization
	rule.ID = ruleID
	rule.OrganizationID = orgID
	
	if err := h.ruleEngine.UpdateRule(ctx, &rule); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update rule: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, rule)
}

// DeleteRule deletes a categorization rule
func (h *CategorizationHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}
	
	if err := h.ruleEngine.DeleteRule(ctx, ruleID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete rule: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// TestRule tests a rule against historical transactions
func (h *CategorizationHandler) TestRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}
	
	// Get the rule first
	orgID := getOrganizationID(ctx)
	rules, err := h.ruleEngine.GetRules(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get rules: %v", err), http.StatusInternalServerError)
		return
	}
	
	var rule *models.EnhancedCategoryRule
	for _, r := range rules {
		if r.ID == ruleID {
			rule = r
			break
		}
	}
	
	if rule == nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}
	
	result, err := h.ruleEngine.TestRule(ctx, rule)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to test rule: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, result)
}

// GetPatterns returns merchant patterns for the organization
func (h *CategorizationHandler) GetPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	_ = orgID // TODO: use this when pattern method is implemented
	
	// This would need a method to get all patterns
	// For now, return empty array
	respondWithJSON(w, r, http.StatusOK, []interface{}{})
}

// GetSimilarPatterns returns similar patterns for a merchant name
func (h *CategorizationHandler) GetSimilarPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	merchantName := r.URL.Query().Get("merchant")
	if merchantName == "" {
		http.Error(w, "merchant parameter is required", http.StatusBadRequest)
		return
	}
	
	thresholdStr := r.URL.Query().Get("threshold")
	threshold := 0.3 // default
	if thresholdStr != "" {
		if t, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			threshold = t
		}
	}
	
	patterns, err := h.patternEngine.GetSimilarPatterns(ctx, orgID, merchantName, threshold)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get similar patterns: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, patterns)
}

// ClearPatternCache clears the pattern cache for the organization
func (h *CategorizationHandler) ClearPatternCache(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	if err := h.patternEngine.ClearPatternCache(ctx, orgID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to clear pattern cache: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// CreateBatchCategorizationJob creates a batch categorization job
func (h *CategorizationHandler) CreateBatchCategorizationJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	var request models.BatchCategorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	// Set organization ID
	request.OrganizationID = orgID
	
	// Create the job
	result, err := jobs.CreateBatchCategorizationJob(ctx, h.riverClient, request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create job: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"job_id": result.Job.ID,
		"state":  result.Job.State,
	})
}

// EstimateBatchCost estimates the cost for batch categorization
func (h *CategorizationHandler) EstimateBatchCost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	var request models.BatchCategorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	request.OrganizationID = orgID
	
	// Get best model for cost estimation
	model, err := h.llmEngine.GetBestModel(ctx, orgID, "cost_optimized")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get model: %v", err), http.StatusInternalServerError)
		return
	}
	
	// This would need to get the actual transactions to estimate properly
	// For now, return a placeholder estimation
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"estimated_cost":    0.0,
		"estimated_tokens":  0,
		"model":            model.Name,
		"transaction_count": 0,
	})
}

// CategorizeTransaction categorizes a single transaction
func (h *CategorizationHandler) CategorizeTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	transactionIDStr := chi.URLParam(r, "transactionID")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}
	
	orgID := getOrganizationID(ctx)
	
	// Create real-time categorization job
	result, err := jobs.CreateRealtimeCategorizationJob(ctx, h.riverClient, transactionID, orgID, "high")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create categorization job: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"job_id": result.Job.ID,
		"state":  result.Job.State,
	})
}

// RecordFeedback records user feedback on categorization
func (h *CategorizationHandler) RecordFeedback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	transactionIDStr := chi.URLParam(r, "transactionID")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}
	
	orgID := getOrganizationID(ctx)
	userID := getUserID(ctx)
	
	var feedback models.CategorizationFeedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	// Set required fields
	feedback.TransactionID = transactionID
	feedback.OrganizationID = orgID
	feedback.UserID = userID
	
	if err := h.feedbackManager.RecordFeedback(ctx, &feedback); err != nil {
		http.Error(w, fmt.Sprintf("Failed to record feedback: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, feedback)
}

// GetFeedback returns feedback for the organization
func (h *CategorizationHandler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	offsetStr := r.URL.Query().Get("offset")
	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	
	feedback, err := h.feedbackManager.GetFeedback(ctx, orgID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get feedback: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, feedback)
}

// GetFeedbackAnalysis returns analysis of user feedback
func (h *CategorizationHandler) GetFeedbackAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	analysis, err := h.feedbackManager.AnalyzeFeedback(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze feedback: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, analysis)
}

// GetAvailableModels returns available LLM models
func (h *CategorizationHandler) GetAvailableModels(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, r, http.StatusOK, models.DefaultLLMModels)
}

// GetBestModel returns the best model for a given strategy
func (h *CategorizationHandler) GetBestModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	strategy := r.URL.Query().Get("strategy")
	if strategy == "" {
		strategy = "balanced"
	}
	
	model, err := h.llmEngine.GetBestModel(ctx, orgID, strategy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get best model: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, model)
}

// GetBudget returns budget information for the organization
func (h *CategorizationHandler) GetBudget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	tracker, err := h.costManager.GetCostTracker(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get budget: %v", err), http.StatusInternalServerError)
		return
	}
	
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"monthly_budget":    tracker.MonthlyBudget,
		"daily_budget":      tracker.DailyBudget,
		"current_spend":     tracker.CurrentSpend,
		"monthly_spend":     tracker.MonthlySpend,
		"transaction_count": tracker.TransactionCount,
		"avg_cost_per_txn":  tracker.AvgCostPerTxn,
	})
}

// UpdateBudget updates budget limits for the organization
func (h *CategorizationHandler) UpdateBudget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := getOrganizationID(ctx)
	
	var request struct {
		MonthlyBudget float64 `json:"monthly_budget"`
		DailyBudget   float64 `json:"daily_budget"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	if err := h.costManager.UpdateBudget(ctx, orgID, request.MonthlyBudget, request.DailyBudget); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update budget: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions (these would need to be implemented based on your auth middleware)

func getOrganizationID(ctx context.Context) uuid.UUID {
	return auth.GetOrganization(ctx)
}

func getUserID(ctx context.Context) uuid.UUID {
	user := auth.GetUser(ctx)
	if user == nil {
		return uuid.Nil
	}
	return user.ID
}

