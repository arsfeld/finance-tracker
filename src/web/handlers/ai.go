package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/services"
)

// AIHandler handles AI-related API endpoints
type AIHandler struct {
	aiService *services.AIService
}

// NewAIHandler creates a new AI handler
func NewAIHandler(aiService *services.AIService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
	}
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Message             string                      `json:"message"`
	ConversationHistory []services.ChatMessage      `json:"conversation_history,omitempty"`
	UseRAG              bool                       `json:"use_rag"`
	Context             string                     `json:"context,omitempty"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response        string                     `json:"response"`
	Model           string                     `json:"model"`
	TokensUsed      int                        `json:"tokens_used"`
	ProcessingTime  int64                      `json:"processing_time_ms"`
	Suggestions     []string                   `json:"suggestions,omitempty"`
	Charts          []interface{}              `json:"charts,omitempty"`
	Insights        []interface{}              `json:"insights,omitempty"`
}

// BatchCategorizationRequest represents a batch categorization request  
type BatchCategorizationRequest struct {
	TransactionIDs    []string `json:"transaction_ids,omitempty"`
	ModelPreference   string   `json:"model_preference"`
	ForceRecategorize bool     `json:"force_recategorize"`
	UseRAG           bool     `json:"use_rag"`
}

// RegisterRoutes registers AI routes
func (h *AIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/ai", func(r chi.Router) {
		// Chat endpoints
		r.Post("/chat", h.HandleChat)
		r.Post("/chat/suggestions", h.GetChatSuggestions)
		
		// Enhanced categorization endpoints
		r.Post("/categorization/batch", h.HandleBatchCategorization)
		r.Post("/categorization/estimate", h.EstimateBatchCost)
		
		// Insights endpoints
		r.Get("/insights/spending", h.GetSpendingInsights)
		r.Get("/insights/trends", h.GetTrendAnalysis)
		r.Get("/insights/anomalies", h.DetectAnomalies)
		
		// Model management
		r.Get("/models", h.GetAvailableModels)
		r.Get("/models/optimal", h.GetOptimalModel)
	})
}

// HandleChat processes AI chat requests with RAG
func (h *AIHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Default to RAG-enabled chat for better responses
	if req.UseRAG == false && req.Context == "" {
		req.UseRAG = true
	}

	// Call AI service with RAG
	aiResponse, err := h.aiService.ChatWithRAG(ctx, orgID, req.Message, req.ConversationHistory)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get AI response: %v", err), http.StatusInternalServerError)
		return
	}

	if len(aiResponse.Choices) == 0 {
		http.Error(w, "No response from AI", http.StatusInternalServerError)
		return
	}

	// Build response with suggestions
	suggestions := h.generateContextualSuggestions(req.Message, req.Context)
	
	response := ChatResponse{
		Response:       aiResponse.Choices[0].Message.Content,
		Model:          aiResponse.Model,
		TokensUsed:     aiResponse.Usage.TotalTokens,
		ProcessingTime: 0, // Would be calculated in a real implementation
		Suggestions:    suggestions,
	}

	respondWithJSON(w, r, http.StatusOK, response)
}

// HandleBatchCategorization processes batch categorization requests
func (h *AIHandler) HandleBatchCategorization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	var req BatchCategorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert string IDs to UUIDs
	var transactionIDs []uuid.UUID
	for _, idStr := range req.TransactionIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			transactionIDs = append(transactionIDs, id)
		}
	}

	// Set defaults
	if req.ModelPreference == "" {
		req.ModelPreference = "cost"
	}

	// Create service request
	serviceReq := services.BatchCategorizationRequest{
		TransactionIDs:    transactionIDs,
		ModelPreference:   req.ModelPreference,
		ForceRecategorize: req.ForceRecategorize,
		UseRAG:           req.UseRAG,
	}

	// Call AI service
	response, err := h.aiService.BatchCategorizeTransactions(ctx, orgID, serviceReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to categorize transactions: %v", err), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, r, http.StatusOK, response)
}

// EstimateBatchCost estimates the cost for batch categorization
func (h *AIHandler) EstimateBatchCost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := auth.GetOrganization(ctx)
	if orgID == uuid.Nil {
		http.Error(w, "Organization not found", http.StatusUnauthorized)
		return
	}

	var req BatchCategorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults for estimation
	if req.ModelPreference == "" {
		req.ModelPreference = "cost"
	}

	// Create service request for estimation (empty transaction IDs = all uncategorized)
	serviceReq := services.BatchCategorizationRequest{
		TransactionIDs:    []uuid.UUID{}, // Empty for estimation
		ModelPreference:   req.ModelPreference,
		ForceRecategorize: req.ForceRecategorize,
		UseRAG:           req.UseRAG,
	}

	// Get estimate
	response, err := h.aiService.BatchCategorizeTransactions(ctx, orgID, serviceReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to estimate cost: %v", err), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, r, http.StatusOK, response)
}

// GetChatSuggestions provides contextual chat suggestions
func (h *AIHandler) GetChatSuggestions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Context string `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	suggestions := h.generateContextualSuggestions("", req.Context)
	
	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"suggestions": suggestions,
	})
}

// GetSpendingInsights provides AI-powered spending insights
func (h *AIHandler) GetSpendingInsights(w http.ResponseWriter, r *http.Request) {
	// Placeholder for spending insights
	insights := []map[string]interface{}{
		{
			"type":        "spending_pattern",
			"title":       "Frequent Coffee Purchases",
			"description": "You've spent $127 on coffee this month across 23 transactions",
			"suggestion":  "Consider brewing coffee at home to save approximately $80/month",
			"confidence":  0.85,
		},
		{
			"type":        "budget_alert",
			"title":       "Dining Budget Exceeded",
			"description": "Dining expenses are 23% over your typical monthly average",
			"suggestion":  "Try meal planning to reduce restaurant visits",
			"confidence":  0.92,
		},
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"insights": insights,
	})
}

// GetTrendAnalysis provides trend analysis
func (h *AIHandler) GetTrendAnalysis(w http.ResponseWriter, r *http.Request) {
	// Placeholder for trend analysis
	trends := map[string]interface{}{
		"monthly_spending": []map[string]interface{}{
			{"month": "2024-10", "amount": 2450.67, "change": 5.2},
			{"month": "2024-11", "amount": 2678.34, "change": 9.3},
			{"month": "2024-12", "amount": 2543.89, "change": -5.0},
		},
		"category_trends": map[string]interface{}{
			"groceries": map[string]interface{}{"trend": "increasing", "change_percent": 12.5},
			"entertainment": map[string]interface{}{"trend": "decreasing", "change_percent": -8.3},
			"transportation": map[string]interface{}{"trend": "stable", "change_percent": 2.1},
		},
	}

	respondWithJSON(w, r, http.StatusOK, trends)
}

// DetectAnomalies detects spending anomalies
func (h *AIHandler) DetectAnomalies(w http.ResponseWriter, r *http.Request) {
	// Placeholder for anomaly detection
	anomalies := []map[string]interface{}{
		{
			"type":        "unusual_amount",
			"transaction": "UBER TRIP - $87.45",
			"date":        "2024-12-15",
			"reason":      "Amount is 340% higher than typical Uber rides",
			"severity":    "medium",
		},
		{
			"type":        "new_merchant",
			"transaction": "LUXURY STORE - $234.67",
			"date":        "2024-12-14",
			"reason":      "First time shopping at this merchant",
			"severity":    "low",
		},
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"anomalies": anomalies,
	})
}

// GetAvailableModels returns available AI models
func (h *AIHandler) GetAvailableModels(w http.ResponseWriter, r *http.Request) {
	models := []map[string]interface{}{
		{
			"id":           "anthropic/claude-3.5-sonnet",
			"name":         "Claude 3.5 Sonnet",
			"provider":     "Anthropic",
			"capabilities": []string{"chat", "categorization", "analysis"},
			"cost_tier":    "premium",
			"speed_tier":   "medium",
			"accuracy":     "very_high",
		},
		{
			"id":           "openai/gpt-4o-mini",
			"name":         "GPT-4o Mini",
			"provider":     "OpenAI",
			"capabilities": []string{"chat", "categorization"},
			"cost_tier":    "budget",
			"speed_tier":   "fast",
			"accuracy":     "high",
		},
		{
			"id":           "google/gemini-pro-1.5",
			"name":         "Gemini Pro 1.5",
			"provider":     "Google",
			"capabilities": []string{"chat", "categorization", "analysis"},
			"cost_tier":    "medium",
			"speed_tier":   "fast",
			"accuracy":     "high",
		},
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"models": models,
	})
}

// GetOptimalModel suggests the optimal model for a task
func (h *AIHandler) GetOptimalModel(w http.ResponseWriter, r *http.Request) {
	taskType := r.URL.Query().Get("task")
	priority := r.URL.Query().Get("priority") // "cost", "speed", "accuracy"

	var recommendation map[string]interface{}

	switch taskType {
	case "categorization":
		switch priority {
		case "cost":
			recommendation = map[string]interface{}{
				"model":  "openai/gpt-4o-mini",
				"reason": "Best balance of cost and accuracy for categorization tasks",
			}
		case "speed":
			recommendation = map[string]interface{}{
				"model":  "google/gemini-pro-1.5",
				"reason": "Fastest processing with good accuracy",
			}
		default:
			recommendation = map[string]interface{}{
				"model":  "anthropic/claude-3.5-sonnet",
				"reason": "Highest accuracy for financial categorization",
			}
		}
	case "chat":
		recommendation = map[string]interface{}{
			"model":  "anthropic/claude-3.5-sonnet",
			"reason": "Best conversational abilities and financial understanding",
		}
	default:
		recommendation = map[string]interface{}{
			"model":  "anthropic/claude-3.5-sonnet",
			"reason": "Most versatile model for general financial tasks",
		}
	}

	respondWithJSON(w, r, http.StatusOK, recommendation)
}

// generateContextualSuggestions creates contextual chat suggestions
func (h *AIHandler) generateContextualSuggestions(lastMessage, context string) []string {
	switch context {
	case "Analytics Dashboard":
		return []string{
			"What are my biggest spending categories this month?",
			"Show me spending trends over the last 6 months",
			"Are there any unusual transactions I should review?",
			"How does my spending compare to last month?",
			"What subscriptions am I paying for?",
		}
	case "Transactions":
		return []string{
			"Categorize my uncategorized transactions",
			"Find duplicate transactions",
			"What's my average daily spending?",
			"Show me all transactions over $100",
			"Which merchants do I spend the most at?",
		}
	case "Categories & Rules":
		return []string{
			"Help me create categorization rules",
			"Analyze my spending patterns",
			"Suggest new categories based on my transactions",
			"Find transactions that might be miscategorized",
			"Optimize my categorization rules",
		}
	default:
		return []string{
			"Give me a spending summary for this month",
			"What are some ways I can save money?",
			"Help me create a budget",
			"Find recurring subscriptions I might want to cancel",
			"Analyze my financial health",
		}
	}
}