package models

import (
	"time"

	"github.com/google/uuid"
)

// CategorizationMetadata represents the categorization metadata stored in transactions
type CategorizationMetadata struct {
	ConfidenceScore      *float64           `json:"confidence_score,omitempty"`
	CategorizationMethod *string            `json:"categorization_method,omitempty"`
	LLMBatchID          *uuid.UUID         `json:"llm_batch_id,omitempty"`
	UserCorrected       bool               `json:"user_corrected"`
	SimilarityMatches   []SimilarityMatch  `json:"similarity_matches,omitempty"`
	RuleMatches         []RuleMatch        `json:"rule_matches,omitempty"`
	ProcessingTimeMs    *int64             `json:"processing_time_ms,omitempty"`
	CostEstimate        *float64           `json:"cost_estimate,omitempty"`
	FeedbackID          *uuid.UUID         `json:"feedback_id,omitempty"`
	CorrectionDate      *time.Time         `json:"correction_date,omitempty"`
}

// SimilarityMatch represents a match from similarity search
type SimilarityMatch struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	CategoryID    int       `json:"category_id"`
	Similarity    float64   `json:"similarity"`
	Description   string    `json:"description"`
	MerchantName  string    `json:"merchant_name"`
	Amount        float64   `json:"amount"`
}

// RuleMatch represents a match from rule-based engine
type RuleMatch struct {
	RuleID     uuid.UUID `json:"rule_id"`
	RuleType   string    `json:"rule_type"`
	Pattern    string    `json:"pattern"`
	Confidence float64   `json:"confidence"`
	Priority   int       `json:"priority"`
}

// EnhancedCategoryRule represents an enhanced categorization rule
type EnhancedCategoryRule struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id" db:"organization_id"`
	CategoryID      int        `json:"category_id" db:"category_id"`
	RuleType        string     `json:"rule_type" db:"rule_type"`
	Pattern         string     `json:"pattern" db:"pattern"`
	Confidence      float64    `json:"confidence" db:"confidence"`
	Priority        int        `json:"priority" db:"priority"`
	IsCaseSensitive bool       `json:"is_case_sensitive" db:"is_case_sensitive"`
	IsRegex         bool       `json:"is_regex" db:"is_regex"`
	UsageCount      int        `json:"usage_count" db:"usage_count"`
	SuccessRate     float64    `json:"success_rate" db:"success_rate"`
	CreatedBy       *uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt      *time.Time `json:"last_used_at" db:"last_used_at"`

	// Joined fields
	Category *Category `json:"category,omitempty"`
}

// RuleType constants
const (
	RuleTypeMerchantPattern    = "merchant_pattern"
	RuleTypeDescriptionKeyword = "description_keyword"
	RuleTypeAmountRange        = "amount_range"
	RuleTypeRegexPattern       = "regex_pattern"
)

// MerchantPatternCache represents cached merchant patterns for fast lookup
type MerchantPatternCache struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id" db:"organization_id"`
	MerchantPattern string     `json:"merchant_pattern" db:"merchant_pattern"`
	CategoryID      int        `json:"category_id" db:"category_id"`
	Confidence      float64    `json:"confidence" db:"confidence"`
	UsageCount      int        `json:"usage_count" db:"usage_count"`
	LastUsedAt      *time.Time `json:"last_used_at" db:"last_used_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`

	// Joined fields
	Category *Category `json:"category,omitempty"`
}

// LLMCategorizationBatch represents a batch of transactions processed by LLM
type LLMCategorizationBatch struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	OrganizationID    uuid.UUID  `json:"organization_id" db:"organization_id"`
	TransactionCount  int        `json:"transaction_count" db:"transaction_count"`
	InputTokens       int        `json:"input_tokens" db:"input_tokens"`
	OutputTokens      int        `json:"output_tokens" db:"output_tokens"`
	TotalCost         float64    `json:"total_cost" db:"total_cost"`
	ModelUsed         string     `json:"model_used" db:"model_used"`
	BatchType         string     `json:"batch_type" db:"batch_type"`
	SuccessRate       *float64   `json:"success_rate" db:"success_rate"`
	AvgConfidence     *float64   `json:"avg_confidence" db:"avg_confidence"`
	ProcessingTimeMs  *int       `json:"processing_time_ms" db:"processing_time_ms"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// CategorizationFeedback represents user feedback on categorization
type CategorizationFeedback struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TransactionID    uuid.UUID  `json:"transaction_id" db:"transaction_id"`
	OrganizationID   uuid.UUID  `json:"organization_id" db:"organization_id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	OldCategoryID    *int       `json:"old_category_id" db:"old_category_id"`
	NewCategoryID    int        `json:"new_category_id" db:"new_category_id"`
	FeedbackType     string     `json:"feedback_type" db:"feedback_type"`
	ConfidenceBefore *float64   `json:"confidence_before" db:"confidence_before"`
	MethodUsed       *string    `json:"method_used" db:"method_used"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`

	// Joined fields
	Transaction   *Transaction `json:"transaction,omitempty"`
	OldCategory   *Category    `json:"old_category,omitempty"`
	NewCategory   *Category    `json:"new_category,omitempty"`
	User          *User        `json:"user,omitempty"`
}

// FeedbackType constants
const (
	FeedbackTypeCorrection   = "correction"
	FeedbackTypeConfirmation = "confirmation"
	FeedbackTypeRejection    = "rejection"
)

// CategorizationMethod constants
const (
	CategorizationMethodRuleBased      = "rule_based"
	CategorizationMethodPatternMatching = "pattern_matching"
	CategorizationMethodRAGSimilarity  = "rag_similarity"
	CategorizationMethodLLMBatch       = "llm_batch"
	CategorizationMethodManual         = "manual"
)

// CategorizationResult represents the result of categorization
type CategorizationResult struct {
	CategoryID       *int                   `json:"category_id"`
	Confidence       float64                `json:"confidence"`
	Method           string                 `json:"method"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
	CostEstimate     float64                `json:"cost_estimate"`
	Metadata         CategorizationMetadata `json:"metadata"`
	Explanation      string                 `json:"explanation,omitempty"`
}

// BatchCategorizationRequest represents a request to categorize multiple transactions
type BatchCategorizationRequest struct {
	OrganizationID    uuid.UUID    `json:"organization_id"`
	TransactionIDs    []uuid.UUID  `json:"transaction_ids,omitempty"`
	DateRange         *DateRange   `json:"date_range,omitempty"`
	ForceRecategorize bool         `json:"force_recategorize"`
	MaxCost           float64      `json:"max_cost"`
	ConfidenceThresh  float64      `json:"confidence_threshold"`
}

// DateRange represents a date range for filtering
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// CategorizationStats represents categorization statistics for an organization
type CategorizationStats struct {
	OrganizationID          uuid.UUID `json:"organization_id" db:"organization_id"`
	TotalTransactions       int       `json:"total_transactions" db:"total_transactions"`
	CategorizedTransactions int       `json:"categorized_transactions" db:"categorized_transactions"`
	UserCorrected          int       `json:"user_corrected" db:"user_corrected"`
	AvgConfidence          *float64  `json:"avg_confidence" db:"avg_confidence"`
	RuleBasedCount         int       `json:"rule_based_count" db:"rule_based_count"`
	PatternMatchingCount   int       `json:"pattern_matching_count" db:"pattern_matching_count"`
	LLMBatchCount          int       `json:"llm_batch_count" db:"llm_batch_count"`
}

// SimilarMerchantPattern represents a similar merchant pattern result
type SimilarMerchantPattern struct {
	MerchantPattern string  `json:"merchant_pattern" db:"merchant_pattern"`
	CategoryID      int     `json:"category_id" db:"category_id"`
	Confidence      float64 `json:"confidence" db:"confidence"`
	Similarity      float64 `json:"similarity" db:"similarity"`
	UsageCount      int     `json:"usage_count" db:"usage_count"`

	// Joined fields
	Category *Category `json:"category,omitempty"`
}

// LLMModel represents an LLM model configuration
type LLMModel struct {
	Name        string  `json:"name"`
	CostPer1K   float64 `json:"cost_per_1k"`
	MaxTokens   int     `json:"max_tokens"`
	Accuracy    float64 `json:"accuracy"`
	IsDefault   bool    `json:"is_default"`
	Provider    string  `json:"provider"`
}

// CostTracker tracks LLM costs for budgeting
type CostTracker struct {
	OrganizationID  uuid.UUID `json:"organization_id"`
	MonthlyBudget   float64   `json:"monthly_budget"`
	DailyBudget     float64   `json:"daily_budget"`
	CurrentSpend    float64   `json:"current_spend"`
	MonthlySpend    float64   `json:"monthly_spend"`
	TransactionCount int64    `json:"transaction_count"`
	AvgCostPerTxn   float64   `json:"avg_cost_per_txn"`
}

// BatchConfig represents configuration for batch processing
type BatchConfig struct {
	MaxBatchSize     int           `json:"max_batch_size"`
	MinBatchSize     int           `json:"min_batch_size"`
	TimeWindow       time.Duration `json:"time_window"`
	ConfidenceThresh float64       `json:"confidence_thresh"`
	MaxConcurrent    int           `json:"max_concurrent"`
	CostLimit        float64       `json:"cost_limit"`
}

// Default configuration values
var (
	DefaultBatchConfig = BatchConfig{
		MaxBatchSize:     100,
		MinBatchSize:     20,
		TimeWindow:       time.Hour,
		ConfidenceThresh: 0.7,
		MaxConcurrent:    3,
		CostLimit:        50.0,
	}

	DefaultLLMModels = []LLMModel{
		{
			Name:      "gpt-4o-mini",
			CostPer1K: 0.00015,
			MaxTokens: 128000,
			Accuracy:  0.92,
			IsDefault: true,
			Provider:  "openrouter",
		},
		{
			Name:      "claude-3-haiku-20240307",
			CostPer1K: 0.00025,
			MaxTokens: 200000,
			Accuracy:  0.90,
			IsDefault: false,
			Provider:  "openrouter",
		},
	}
)