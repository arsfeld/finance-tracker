package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"finance_tracker/src/internal/models"
	"finance_tracker/src/internal/services/categorization"
)

// Real-time categorization job (triggered on transaction sync)
type RealtimeCategorizationJob struct {
	TransactionID  uuid.UUID `json:"transaction_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Priority       string    `json:"priority"` // "high", "normal", "low"
}

func (j RealtimeCategorizationJob) Kind() string {
	return "realtime_categorization"
}

func (j RealtimeCategorizationJob) InsertOpts() river.InsertOpts {
	opts := river.InsertOpts{
		Queue: "categorization",
	}
	
	switch j.Priority {
	case "high":
		opts.Priority = 3
	case "normal":
		opts.Priority = 2
	case "low":
		opts.Priority = 1
	default:
		opts.Priority = 2
	}
	
	return opts
}

// Batch categorization job
type BatchCategorizationJob struct {
	OrganizationID    uuid.UUID    `json:"organization_id"`
	TransactionIDs    []uuid.UUID  `json:"transaction_ids,omitempty"` // Specific transactions
	DateRange         *DateRange   `json:"date_range,omitempty"`      // Or date range
	ForceRecategorize bool         `json:"force_recategorize"`        // Recategorize existing
	MaxCost           float64      `json:"max_cost"`                  // Budget limit
	ConfidenceThresh  float64      `json:"confidence_threshold"`      // Minimum confidence threshold
}

func (j BatchCategorizationJob) Kind() string {
	return "batch_categorization"
}

func (j BatchCategorizationJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "batch_categorization",
		MaxAttempts: 3,
	}
}

// RAG learning job
type RAGLearningJob struct {
	OrganizationID uuid.UUID   `json:"organization_id"`
	FeedbackIDs    []uuid.UUID `json:"feedback_ids,omitempty"`
	RebuildIndex   bool        `json:"rebuild_index"`
}

func (j RAGLearningJob) Kind() string {
	return "rag_learning"
}

func (j RAGLearningJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "rag_learning",
		MaxAttempts: 2,
	}
}

// Pattern mining job
type PatternMiningJob struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	MinConfidence  float64   `json:"min_confidence"`
	MinUsageCount  int       `json:"min_usage_count"`
}

func (j PatternMiningJob) Kind() string {
	return "pattern_mining"
}

func (j PatternMiningJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "maintenance",
		MaxAttempts: 2,
	}
}

// DateRange represents a date range for filtering
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// Job workers

// RealtimeCategorizationWorker handles real-time categorization of individual transactions
type RealtimeCategorizationWorker struct {
	river.WorkerDefaults[RealtimeCategorizationJob]
	
	engine            categorization.CategorizationEngine
	transactionRepo   TransactionRepository
}

func NewRealtimeCategorizationWorker(engine categorization.CategorizationEngine, transactionRepo TransactionRepository) *RealtimeCategorizationWorker {
	return &RealtimeCategorizationWorker{
		engine:          engine,
		transactionRepo: transactionRepo,
	}
}

func (w *RealtimeCategorizationWorker) Work(ctx context.Context, job *river.Job[RealtimeCategorizationJob]) error {
	// Get the transaction
	transaction, err := w.transactionRepo.GetByID(ctx, job.Args.TransactionID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}
	
	if transaction == nil {
		return fmt.Errorf("transaction not found: %s", job.Args.TransactionID)
	}
	
	// Skip if already categorized (unless force recategorize)
	if transaction.CategoryID != nil {
		return nil // Already categorized
	}
	
	// Categorize the transaction
	result, err := w.engine.CategorizeTransaction(ctx, transaction)
	if err != nil {
		return fmt.Errorf("failed to categorize transaction: %w", err)
	}
	
	// Update the transaction with categorization result
	if result.CategoryID != nil {
		if err := w.transactionRepo.UpdateCategorization(ctx, transaction.ID, *result.CategoryID, result.Metadata); err != nil {
			return fmt.Errorf("failed to update transaction categorization: %w", err)
		}
	}
	
	return nil
}

// BatchCategorizationWorker handles batch categorization of multiple transactions
type BatchCategorizationWorker struct {
	river.WorkerDefaults[BatchCategorizationJob]
	
	engine          categorization.CategorizationEngine
	llmEngine       categorization.LLMEngine
	transactionRepo TransactionRepository
}

func NewBatchCategorizationWorker(engine categorization.CategorizationEngine, llmEngine categorization.LLMEngine, transactionRepo TransactionRepository) *BatchCategorizationWorker {
	return &BatchCategorizationWorker{
		engine:          engine,
		llmEngine:       llmEngine,
		transactionRepo: transactionRepo,
	}
}

func (w *BatchCategorizationWorker) Work(ctx context.Context, job *river.Job[BatchCategorizationJob]) error {
	var transactions []*models.Transaction
	var err error
	
	// Get transactions to categorize
	if len(job.Args.TransactionIDs) > 0 {
		// Specific transactions
		transactions, err = w.transactionRepo.GetByIDs(ctx, job.Args.TransactionIDs)
		if err != nil {
			return fmt.Errorf("failed to get transactions by IDs: %w", err)
		}
	} else if job.Args.DateRange != nil {
		// Date range
		transactions, err = w.transactionRepo.GetByDateRange(ctx, job.Args.OrganizationID, job.Args.DateRange.StartDate, job.Args.DateRange.EndDate)
		if err != nil {
			return fmt.Errorf("failed to get transactions by date range: %w", err)
		}
	} else {
		// Get all uncategorized transactions
		transactions, err = w.transactionRepo.GetUncategorized(ctx, job.Args.OrganizationID)
		if err != nil {
			return fmt.Errorf("failed to get uncategorized transactions: %w", err)
		}
	}
	
	if len(transactions) == 0 {
		return nil // No transactions to categorize
	}
	
	// Filter out already categorized transactions unless force recategorize
	if !job.Args.ForceRecategorize {
		var uncategorized []*models.Transaction
		for _, tx := range transactions {
			if tx.CategoryID == nil {
				uncategorized = append(uncategorized, tx)
			}
		}
		transactions = uncategorized
	}
	
	if len(transactions) == 0 {
		return nil // No transactions to categorize after filtering
	}
	
	// Separate transactions that need LLM processing
	var needsLLM []*models.Transaction
	var categorized int
	
	// First pass: try rule-based and pattern matching
	for _, tx := range transactions {
		result, err := w.engine.CategorizeTransaction(ctx, tx)
		if err != nil {
			// Log error but continue with other transactions
			fmt.Printf("Failed to categorize transaction %s: %v\n", tx.ID, err)
			continue
		}
		
		if result.CategoryID != nil && result.Confidence >= job.Args.ConfidenceThresh {
			// Successfully categorized with high confidence
			if err := w.transactionRepo.UpdateCategorization(ctx, tx.ID, *result.CategoryID, result.Metadata); err != nil {
				fmt.Printf("Failed to update categorization for transaction %s: %v\n", tx.ID, err)
				continue
			}
			categorized++
		} else {
			// Needs LLM processing
			needsLLM = append(needsLLM, tx)
		}
	}
	
	// Second pass: LLM processing for remaining transactions
	if len(needsLLM) > 0 {
		// Check if we have budget for LLM processing
		model, err := w.llmEngine.GetBestModel(ctx, job.Args.OrganizationID, "cost_optimized")
		if err != nil {
			return fmt.Errorf("failed to get LLM model: %w", err)
		}
		
		estimatedCost, err := w.llmEngine.EstimateCost(ctx, needsLLM, model)
		if err != nil {
			return fmt.Errorf("failed to estimate LLM cost: %w", err)
		}
		
		if estimatedCost <= job.Args.MaxCost {
			// Process with LLM
			llmResults, err := w.llmEngine.CategorizeByLLM(ctx, needsLLM)
			if err != nil {
				return fmt.Errorf("failed to categorize with LLM: %w", err)
			}
			
			// Update transactions with LLM results
			for i, result := range llmResults {
				if result.CategoryID != nil && result.Confidence >= job.Args.ConfidenceThresh {
					if err := w.transactionRepo.UpdateCategorization(ctx, needsLLM[i].ID, *result.CategoryID, result.Metadata); err != nil {
						fmt.Printf("Failed to update LLM categorization for transaction %s: %v\n", needsLLM[i].ID, err)
						continue
					}
					categorized++
				}
			}
		}
	}
	
	// Log results
	fmt.Printf("Batch categorization completed: %d/%d transactions categorized\n", categorized, len(transactions))
	
	return nil
}

// RAGLearningWorker handles RAG learning from user feedback
type RAGLearningWorker struct {
	river.WorkerDefaults[RAGLearningJob]
	
	ragEngine       categorization.RAGEngine
	feedbackRepo    categorization.FeedbackRepository
}

func NewRAGLearningWorker(ragEngine categorization.RAGEngine, feedbackRepo categorization.FeedbackRepository) *RAGLearningWorker {
	return &RAGLearningWorker{
		ragEngine:    ragEngine,
		feedbackRepo: feedbackRepo,
	}
}

func (w *RAGLearningWorker) Work(ctx context.Context, job *river.Job[RAGLearningJob]) error {
	var feedback []*models.CategorizationFeedback
	var err error
	
	if len(job.Args.FeedbackIDs) > 0 {
		// Process specific feedback
		for _, feedbackID := range job.Args.FeedbackIDs {
			// This would require a method to get feedback by ID
			// For now, just get all feedback for the organization
			_ = feedbackID
		}
	}
	
	if feedback == nil {
		// Get recent feedback for the organization
		feedback, err = w.feedbackRepo.GetFeedbackByOrganization(ctx, job.Args.OrganizationID, 100, 0)
		if err != nil {
			return fmt.Errorf("failed to get feedback: %w", err)
		}
	}
	
	if len(feedback) == 0 {
		return nil // No feedback to learn from
	}
	
	// Learn from feedback
	if err := w.ragEngine.LearnFromFeedback(ctx, feedback); err != nil {
		return fmt.Errorf("failed to learn from feedback: %w", err)
	}
	
	return nil
}

// PatternMiningWorker handles automatic pattern discovery
type PatternMiningWorker struct {
	river.WorkerDefaults[PatternMiningJob]
	
	patternEngine   categorization.PatternEngine
	transactionRepo TransactionRepository
}

func NewPatternMiningWorker(patternEngine categorization.PatternEngine, transactionRepo TransactionRepository) *PatternMiningWorker {
	return &PatternMiningWorker{
		patternEngine:   patternEngine,
		transactionRepo: transactionRepo,
	}
}

func (w *PatternMiningWorker) Work(ctx context.Context, job *river.Job[PatternMiningJob]) error {
	// Get recent categorized transactions
	transactions, err := w.transactionRepo.GetRecentCategorized(ctx, job.Args.OrganizationID, time.Hour*24*30) // Last 30 days
	if err != nil {
		return fmt.Errorf("failed to get recent transactions: %w", err)
	}
	
	// Mine patterns from categorized transactions
	for _, tx := range transactions {
		if tx.CategoryID != nil && tx.MerchantName != nil {
			// Update pattern cache with high confidence since it's already categorized
			err := w.patternEngine.UpdatePatternCache(ctx, tx, *tx.CategoryID, job.Args.MinConfidence)
			if err != nil {
				fmt.Printf("Failed to update pattern cache for transaction %s: %v\n", tx.ID, err)
			}
		}
	}
	
	return nil
}

// Repository interfaces needed by workers

type TransactionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Transaction, error)
	GetByDateRange(ctx context.Context, organizationID uuid.UUID, startDate, endDate time.Time) ([]*models.Transaction, error)
	GetUncategorized(ctx context.Context, organizationID uuid.UUID) ([]*models.Transaction, error)
	GetRecentCategorized(ctx context.Context, organizationID uuid.UUID, since time.Duration) ([]*models.Transaction, error)
	UpdateCategorization(ctx context.Context, transactionID uuid.UUID, categoryID int, metadata models.CategorizationMetadata) error
}

// Helper functions for creating categorization jobs

// CreateRealtimeCategorizationJob creates a real-time categorization job
func CreateRealtimeCategorizationJob(ctx context.Context, client *river.Client[any], transactionID, organizationID uuid.UUID, priority string) (*rivertype.JobInsertResult, error) {
	job := RealtimeCategorizationJob{
		TransactionID:  transactionID,
		OrganizationID: organizationID,
		Priority:       priority,
	}
	
	return client.Insert(ctx, job, nil)
}

// CreateBatchCategorizationJob creates a batch categorization job
func CreateBatchCategorizationJob(ctx context.Context, client *river.Client[any], request models.BatchCategorizationRequest) (*rivertype.JobInsertResult, error) {
	var dateRange *DateRange
	if request.DateRange != nil {
		dateRange = &DateRange{
			StartDate: request.DateRange.StartDate,
			EndDate:   request.DateRange.EndDate,
		}
	}
	
	job := BatchCategorizationJob{
		OrganizationID:    request.OrganizationID,
		TransactionIDs:    request.TransactionIDs,
		DateRange:         dateRange,
		ForceRecategorize: request.ForceRecategorize,
		MaxCost:           request.MaxCost,
		ConfidenceThresh:  request.ConfidenceThresh,
	}
	
	return client.Insert(ctx, job, nil)
}

// CreateRAGLearningJob creates a RAG learning job
func CreateRAGLearningJob(ctx context.Context, client *river.Client[any], organizationID uuid.UUID, feedbackIDs []uuid.UUID, rebuildIndex bool) (*rivertype.JobInsertResult, error) {
	job := RAGLearningJob{
		OrganizationID: organizationID,
		FeedbackIDs:    feedbackIDs,
		RebuildIndex:   rebuildIndex,
	}
	
	return client.Insert(ctx, job, nil)
}

// CreatePatternMiningJob creates a pattern mining job
func CreatePatternMiningJob(ctx context.Context, client *river.Client[any], organizationID uuid.UUID, minConfidence float64, minUsageCount int) (*rivertype.JobInsertResult, error) {
	job := PatternMiningJob{
		OrganizationID: organizationID,
		MinConfidence:  minConfidence,
		MinUsageCount:  minUsageCount,
	}
	
	return client.Insert(ctx, job, nil)
}