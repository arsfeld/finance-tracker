package categorization

import (
	"context"
	"time"

	"github.com/google/uuid"

	"finance_tracker/src/internal/models"
)

// CategorizationEngine defines the main interface for transaction categorization
type CategorizationEngine interface {
	// CategorizeTransaction categorizes a single transaction using the multi-layer approach
	CategorizeTransaction(ctx context.Context, transaction *models.Transaction) (*models.CategorizationResult, error)
	
	// CategorizeTransactionsBatch categorizes multiple transactions in batch
	CategorizeTransactionsBatch(ctx context.Context, transactions []*models.Transaction) ([]*models.CategorizationResult, error)
	
	// GetCategorizationStats returns categorization statistics for an organization
	GetCategorizationStats(ctx context.Context, organizationID uuid.UUID) (*models.CategorizationStats, error)
}

// RuleEngine defines the interface for rule-based categorization
type RuleEngine interface {
	// CategorizeByRules attempts to categorize transaction using predefined rules
	CategorizeByRules(ctx context.Context, transaction *models.Transaction) (*models.CategorizationResult, error)
	
	// AddRule adds a new categorization rule
	AddRule(ctx context.Context, rule *models.EnhancedCategoryRule) error
	
	// UpdateRule updates an existing categorization rule
	UpdateRule(ctx context.Context, rule *models.EnhancedCategoryRule) error
	
	// DeleteRule deletes a categorization rule
	DeleteRule(ctx context.Context, ruleID uuid.UUID) error
	
	// GetRules returns all rules for an organization
	GetRules(ctx context.Context, organizationID uuid.UUID) ([]*models.EnhancedCategoryRule, error)
	
	// TestRule tests a rule against historical transactions
	TestRule(ctx context.Context, rule *models.EnhancedCategoryRule) (*RuleTestResult, error)
}

// PatternEngine defines the interface for pattern matching categorization
type PatternEngine interface {
	// CategorizeByPatterns attempts to categorize transaction using pattern matching
	CategorizeByPatterns(ctx context.Context, transaction *models.Transaction) (*models.CategorizationResult, error)
	
	// UpdatePatternCache updates the pattern cache with new transaction data
	UpdatePatternCache(ctx context.Context, transaction *models.Transaction, categoryID int, confidence float64) error
	
	// GetSimilarPatterns returns similar merchant patterns for a transaction
	GetSimilarPatterns(ctx context.Context, organizationID uuid.UUID, merchantName string, threshold float64) ([]*models.SimilarMerchantPattern, error)
	
	// ClearPatternCache clears the pattern cache for an organization
	ClearPatternCache(ctx context.Context, organizationID uuid.UUID) error
}

// RAGEngine defines the interface for RAG-based similarity categorization
type RAGEngine interface {
	// CategorizeBySimilarity attempts to categorize transaction using similarity search
	CategorizeBySimilarity(ctx context.Context, transaction *models.Transaction) (*models.CategorizationResult, error)
	
	// GenerateEmbedding generates an embedding for a transaction
	GenerateEmbedding(ctx context.Context, transaction *models.Transaction) ([]float64, error)
	
	// FindSimilarTransactions finds similar transactions using vector search
	FindSimilarTransactions(ctx context.Context, transaction *models.Transaction, threshold float64, limit int) ([]*models.SimilarityMatch, error)
	
	// UpdateEmbeddings updates embeddings for transactions
	UpdateEmbeddings(ctx context.Context, transactions []*models.Transaction) error
	
	// LearnFromFeedback learns from user feedback to improve similarity matching
	LearnFromFeedback(ctx context.Context, feedback []*models.CategorizationFeedback) error
}

// LLMEngine defines the interface for LLM-based categorization
type LLMEngine interface {
	// CategorizeByLLM categorizes transactions using LLM in batch
	CategorizeByLLM(ctx context.Context, transactions []*models.Transaction) ([]*models.CategorizationResult, error)
	
	// GetBestModel returns the best model based on cost/accuracy tradeoff
	GetBestModel(ctx context.Context, organizationID uuid.UUID, strategy string) (*models.LLMModel, error)
	
	// EstimateCost estimates the cost for categorizing transactions
	EstimateCost(ctx context.Context, transactions []*models.Transaction, model *models.LLMModel) (float64, error)
	
	// RecordBatch records a completed batch for cost tracking
	RecordBatch(ctx context.Context, batch *models.LLMCategorizationBatch) error
}

// CostManager defines the interface for cost management and budgeting
type CostManager interface {
	// CheckBudget checks if operation is within budget limits
	CheckBudget(ctx context.Context, organizationID uuid.UUID, estimatedCost float64) error
	
	// RecordCost records actual cost for an operation
	RecordCost(ctx context.Context, organizationID uuid.UUID, cost float64, transactionCount int) error
	
	// GetCostTracker returns current cost tracking info
	GetCostTracker(ctx context.Context, organizationID uuid.UUID) (*models.CostTracker, error)
	
	// UpdateBudget updates budget limits for an organization
	UpdateBudget(ctx context.Context, organizationID uuid.UUID, monthlyBudget, dailyBudget float64) error
}

// FeedbackManager defines the interface for managing user feedback
type FeedbackManager interface {
	// RecordFeedback records user feedback on categorization
	RecordFeedback(ctx context.Context, feedback *models.CategorizationFeedback) error
	
	// GetFeedback returns feedback for an organization
	GetFeedback(ctx context.Context, organizationID uuid.UUID, limit, offset int) ([]*models.CategorizationFeedback, error)
	
	// GetFeedbackByTransaction returns feedback for a specific transaction
	GetFeedbackByTransaction(ctx context.Context, transactionID uuid.UUID) ([]*models.CategorizationFeedback, error)
	
	// AnalyzeFeedback analyzes feedback patterns to improve categorization
	AnalyzeFeedback(ctx context.Context, organizationID uuid.UUID) (*FeedbackAnalysis, error)
}

// Repository interfaces for data access

// RuleRepository defines the interface for rule data access
type RuleRepository interface {
	// GetRulesByOrganization returns all rules for an organization
	GetRulesByOrganization(ctx context.Context, organizationID uuid.UUID) ([]*models.EnhancedCategoryRule, error)
	
	// GetRulesByType returns rules of a specific type
	GetRulesByType(ctx context.Context, organizationID uuid.UUID, ruleType string) ([]*models.EnhancedCategoryRule, error)
	
	// CreateRule creates a new rule
	CreateRule(ctx context.Context, rule *models.EnhancedCategoryRule) error
	
	// UpdateRule updates an existing rule
	UpdateRule(ctx context.Context, rule *models.EnhancedCategoryRule) error
	
	// DeleteRule deletes a rule
	DeleteRule(ctx context.Context, ruleID uuid.UUID) error
	
	// UpdateRuleUsage updates rule usage statistics
	UpdateRuleUsage(ctx context.Context, ruleID uuid.UUID, success bool) error
}

// PatternRepository defines the interface for pattern cache data access
type PatternRepository interface {
	// GetPatternsByOrganization returns all patterns for an organization
	GetPatternsByOrganization(ctx context.Context, organizationID uuid.UUID) ([]*models.MerchantPatternCache, error)
	
	// GetSimilarPatterns returns similar patterns for a merchant name
	GetSimilarPatterns(ctx context.Context, organizationID uuid.UUID, merchantName string, threshold float64) ([]*models.SimilarMerchantPattern, error)
	
	// UpdatePattern updates or creates a pattern cache entry
	UpdatePattern(ctx context.Context, organizationID uuid.UUID, merchantPattern string, categoryID int, confidence float64) error
	
	// ClearPatterns clears all patterns for an organization
	ClearPatterns(ctx context.Context, organizationID uuid.UUID) error
}

// FeedbackRepository defines the interface for feedback data access
type FeedbackRepository interface {
	// CreateFeedback creates new feedback
	CreateFeedback(ctx context.Context, feedback *models.CategorizationFeedback) error
	
	// GetFeedbackByOrganization returns feedback for an organization
	GetFeedbackByOrganization(ctx context.Context, organizationID uuid.UUID, limit, offset int) ([]*models.CategorizationFeedback, error)
	
	// GetFeedbackByTransaction returns feedback for a transaction
	GetFeedbackByTransaction(ctx context.Context, transactionID uuid.UUID) ([]*models.CategorizationFeedback, error)
	
	// GetFeedbackStats returns feedback statistics
	GetFeedbackStats(ctx context.Context, organizationID uuid.UUID) (*FeedbackStats, error)
}

// LLMRepository defines the interface for LLM batch data access
type LLMRepository interface {
	// CreateBatch creates a new LLM batch record
	CreateBatch(ctx context.Context, batch *models.LLMCategorizationBatch) error
	
	// GetBatchesByOrganization returns batches for an organization
	GetBatchesByOrganization(ctx context.Context, organizationID uuid.UUID, limit, offset int) ([]*models.LLMCategorizationBatch, error)
	
	// GetCostSummary returns cost summary for an organization
	GetCostSummary(ctx context.Context, organizationID uuid.UUID, startDate, endDate time.Time) (*CostSummary, error)
}

// Supporting types for complex operations

// RuleTestResult represents the result of testing a rule
type RuleTestResult struct {
	Rule              *models.EnhancedCategoryRule `json:"rule"`
	MatchedTransactions int                        `json:"matched_transactions"`
	AccuracyRate      float64                      `json:"accuracy_rate"`
	Examples          []*models.Transaction        `json:"examples"`
}

// FeedbackAnalysis represents analysis of user feedback
type FeedbackAnalysis struct {
	OrganizationID      uuid.UUID                    `json:"organization_id"`
	TotalFeedback       int                          `json:"total_feedback"`
	CorrectionRate      float64                      `json:"correction_rate"`
	ConfirmationRate    float64                      `json:"confirmation_rate"`
	RejectionRate       float64                      `json:"rejection_rate"`
	MethodAccuracy      map[string]float64           `json:"method_accuracy"`
	CommonCorrections   []*CategoryCorrection        `json:"common_corrections"`
	ProblematicPatterns []*ProblematicPattern        `json:"problematic_patterns"`
}

// CategoryCorrection represents a common correction pattern
type CategoryCorrection struct {
	FromCategoryID   int     `json:"from_category_id"`
	ToCategoryID     int     `json:"to_category_id"`
	FromCategoryName string  `json:"from_category_name"`
	ToCategoryName   string  `json:"to_category_name"`
	Count            int     `json:"count"`
	AvgConfidence    float64 `json:"avg_confidence"`
}

// ProblematicPattern represents a pattern that often requires correction
type ProblematicPattern struct {
	Pattern       string  `json:"pattern"`
	ErrorRate     float64 `json:"error_rate"`
	Occurrences   int     `json:"occurrences"`
	SuggestedRule string  `json:"suggested_rule"`
}

// FeedbackStats represents feedback statistics
type FeedbackStats struct {
	TotalFeedback    int     `json:"total_feedback"`
	Corrections      int     `json:"corrections"`
	Confirmations    int     `json:"confirmations"`
	Rejections       int     `json:"rejections"`
	AvgConfidenceBefore float64 `json:"avg_confidence_before"`
}

// CostSummary represents cost summary information
type CostSummary struct {
	OrganizationID   uuid.UUID `json:"organization_id"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	TotalCost        float64   `json:"total_cost"`
	TransactionCount int       `json:"transaction_count"`
	AvgCostPerTxn    float64   `json:"avg_cost_per_txn"`
	BatchCount       int       `json:"batch_count"`
	ModelUsage       map[string]int `json:"model_usage"`
}

// Error types for categorization operations
type Error string

const (
	ErrInsufficientBudget Error = "insufficient budget for operation"
	ErrNoMatchingRules    Error = "no matching rules found"
	ErrLowConfidence      Error = "categorization confidence below threshold"
	ErrRateLimited        Error = "rate limited by LLM provider"
	ErrInvalidRule        Error = "invalid rule configuration"
	ErrPatternNotFound    Error = "pattern not found in cache"
	ErrEmbeddingFailed    Error = "failed to generate embedding"
	ErrModelNotAvailable  Error = "selected model not available"
)

func (e Error) Error() string {
	return string(e)
}