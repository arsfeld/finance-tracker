package categorization

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"finance_tracker/src/internal/models"
)

// costManager implements the CostManager interface
type costManager struct {
	repo         CostRepository
	alertService AlertService
}

// CostRepository defines the interface for cost data access
type CostRepository interface {
	// GetCostTracker returns current cost tracking info
	GetCostTracker(ctx context.Context, organizationID uuid.UUID) (*models.CostTracker, error)
	
	// UpdateCostTracker updates cost tracking info
	UpdateCostTracker(ctx context.Context, tracker *models.CostTracker) error
	
	// GetMonthlySpend returns spending for the current month
	GetMonthlySpend(ctx context.Context, organizationID uuid.UUID) (float64, error)
	
	// GetDailySpend returns spending for today
	GetDailySpend(ctx context.Context, organizationID uuid.UUID) (float64, error)
	
	// RecordCost records a cost entry
	RecordCost(ctx context.Context, organizationID uuid.UUID, cost float64, transactionCount int, metadata map[string]interface{}) error
	
	// GetBudgetSettings returns budget settings for an organization
	GetBudgetSettings(ctx context.Context, organizationID uuid.UUID) (*BudgetSettings, error)
	
	// UpdateBudgetSettings updates budget settings
	UpdateBudgetSettings(ctx context.Context, organizationID uuid.UUID, settings *BudgetSettings) error
}

// AlertService defines the interface for sending cost alerts
type AlertService interface {
	// SendBudgetAlert sends a budget alert
	SendBudgetAlert(ctx context.Context, organizationID uuid.UUID, alert *BudgetAlert) error
}

// BudgetSettings represents budget configuration for an organization
type BudgetSettings struct {
	OrganizationID uuid.UUID    `json:"organization_id"`
	MonthlyBudget  float64      `json:"monthly_budget"`
	DailyBudget    float64      `json:"daily_budget"`
	AlertThresholds []float64   `json:"alert_thresholds"` // e.g., [0.5, 0.8, 0.95]
	AlertChannels  []string     `json:"alert_channels"`   // e.g., ["email", "slack"]
	IsEnabled      bool         `json:"is_enabled"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// BudgetAlert represents a budget alert
type BudgetAlert struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	AlertType      string    `json:"alert_type"`    // "daily", "monthly"
	Threshold      float64   `json:"threshold"`     // 0.8 = 80%
	CurrentSpend   float64   `json:"current_spend"`
	BudgetLimit    float64   `json:"budget_limit"`
	Percentage     float64   `json:"percentage"`
	Message        string    `json:"message"`
	Severity       string    `json:"severity"`      // "warning", "critical"
}

// NewCostManager creates a new cost manager
func NewCostManager(repo CostRepository, alertService AlertService) CostManager {
	return &costManager{
		repo:         repo,
		alertService: alertService,
	}
}

// CheckBudget checks if operation is within budget limits
func (m *costManager) CheckBudget(ctx context.Context, organizationID uuid.UUID, estimatedCost float64) error {
	// Get current cost tracker
	tracker, err := m.GetCostTracker(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get cost tracker: %w", err)
	}
	
	// Get budget settings
	settings, err := m.repo.GetBudgetSettings(ctx, organizationID)
	if err != nil {
		// If no settings found, use default budget limits
		settings = &BudgetSettings{
			OrganizationID: organizationID,
			MonthlyBudget:  50.0,  // $50 default monthly budget
			DailyBudget:    5.0,   // $5 default daily budget
			IsEnabled:      true,
		}
	}
	
	if !settings.IsEnabled {
		return nil // Budget checking disabled
	}
	
	// Check daily budget
	if settings.DailyBudget > 0 {
		projectedDailySpend := tracker.CurrentSpend + estimatedCost
		if projectedDailySpend > settings.DailyBudget {
			return fmt.Errorf("operation would exceed daily budget (current: $%.2f, estimated: $%.2f, limit: $%.2f)",
				tracker.CurrentSpend, estimatedCost, settings.DailyBudget)
		}
	}
	
	// Check monthly budget
	if settings.MonthlyBudget > 0 {
		projectedMonthlySpend := tracker.MonthlySpend + estimatedCost
		if projectedMonthlySpend > settings.MonthlyBudget {
			return fmt.Errorf("operation would exceed monthly budget (current: $%.2f, estimated: $%.2f, limit: $%.2f)",
				tracker.MonthlySpend, estimatedCost, settings.MonthlyBudget)
		}
	}
	
	return nil
}

// RecordCost records actual cost for an operation
func (m *costManager) RecordCost(ctx context.Context, organizationID uuid.UUID, cost float64, transactionCount int) error {
	// Record the cost entry
	metadata := map[string]interface{}{
		"timestamp":         time.Now(),
		"transaction_count": transactionCount,
		"source":           "categorization_engine",
	}
	
	if err := m.repo.RecordCost(ctx, organizationID, cost, transactionCount, metadata); err != nil {
		return fmt.Errorf("failed to record cost: %w", err)
	}
	
	// Update cost tracker
	tracker, err := m.GetCostTracker(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get cost tracker: %w", err)
	}
	
	// Update tracker values
	tracker.CurrentSpend += cost
	tracker.MonthlySpend += cost
	tracker.TransactionCount += int64(transactionCount)
	
	if tracker.TransactionCount > 0 {
		tracker.AvgCostPerTxn = tracker.MonthlySpend / float64(tracker.TransactionCount)
	}
	
	if err := m.repo.UpdateCostTracker(ctx, tracker); err != nil {
		return fmt.Errorf("failed to update cost tracker: %w", err)
	}
	
	// Check for budget alerts
	if err := m.checkBudgetAlerts(ctx, organizationID, tracker); err != nil {
		// Log error but don't fail the cost recording
		fmt.Printf("Failed to check budget alerts: %v\n", err)
	}
	
	return nil
}

// GetCostTracker returns current cost tracking info
func (m *costManager) GetCostTracker(ctx context.Context, organizationID uuid.UUID) (*models.CostTracker, error) {
	tracker, err := m.repo.GetCostTracker(ctx, organizationID)
	if err != nil {
		// If tracker doesn't exist, create a new one
		tracker = &models.CostTracker{
			OrganizationID:   organizationID,
			MonthlyBudget:    50.0,  // Default budget
			DailyBudget:      5.0,   // Default daily budget
			CurrentSpend:     0.0,
			MonthlySpend:     0.0,
			TransactionCount: 0,
			AvgCostPerTxn:    0.0,
		}
		
		// Get actual spending from database
		monthlySpend, err := m.repo.GetMonthlySpend(ctx, organizationID)
		if err == nil {
			tracker.MonthlySpend = monthlySpend
		}
		
		dailySpend, err := m.repo.GetDailySpend(ctx, organizationID)
		if err == nil {
			tracker.CurrentSpend = dailySpend
		}
	}
	
	return tracker, nil
}

// UpdateBudget updates budget limits for an organization
func (m *costManager) UpdateBudget(ctx context.Context, organizationID uuid.UUID, monthlyBudget, dailyBudget float64) error {
	settings, err := m.repo.GetBudgetSettings(ctx, organizationID)
	if err != nil {
		// Create new settings
		settings = &BudgetSettings{
			OrganizationID:  organizationID,
			AlertThresholds: []float64{0.5, 0.8, 0.95}, // Default alert thresholds
			AlertChannels:   []string{"email"},          // Default alert channels
			IsEnabled:       true,
			CreatedAt:       time.Now(),
		}
	}
	
	settings.MonthlyBudget = monthlyBudget
	settings.DailyBudget = dailyBudget
	settings.UpdatedAt = time.Now()
	
	if err := m.repo.UpdateBudgetSettings(ctx, organizationID, settings); err != nil {
		return fmt.Errorf("failed to update budget settings: %w", err)
	}
	
	// Update cost tracker with new budget limits
	tracker, err := m.GetCostTracker(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get cost tracker: %w", err)
	}
	
	tracker.MonthlyBudget = monthlyBudget
	tracker.DailyBudget = dailyBudget
	
	if err := m.repo.UpdateCostTracker(ctx, tracker); err != nil {
		return fmt.Errorf("failed to update cost tracker: %w", err)
	}
	
	return nil
}

// checkBudgetAlerts checks if any budget alerts should be sent
func (m *costManager) checkBudgetAlerts(ctx context.Context, organizationID uuid.UUID, tracker *models.CostTracker) error {
	settings, err := m.repo.GetBudgetSettings(ctx, organizationID)
	if err != nil || !settings.IsEnabled {
		return nil // No alerts if settings not found or disabled
	}
	
	// Check daily budget alerts
	if settings.DailyBudget > 0 && tracker.CurrentSpend > 0 {
		dailyPercentage := tracker.CurrentSpend / settings.DailyBudget
		
		for _, threshold := range settings.AlertThresholds {
			if dailyPercentage >= threshold {
				alert := &BudgetAlert{
					OrganizationID: organizationID,
					AlertType:      "daily",
					Threshold:      threshold,
					CurrentSpend:   tracker.CurrentSpend,
					BudgetLimit:    settings.DailyBudget,
					Percentage:     dailyPercentage,
					Message:        fmt.Sprintf("Daily categorization budget is %.1f%% used ($%.2f of $%.2f)", dailyPercentage*100, tracker.CurrentSpend, settings.DailyBudget),
					Severity:       m.getSeverity(dailyPercentage),
				}
				
				if err := m.alertService.SendBudgetAlert(ctx, organizationID, alert); err != nil {
					fmt.Printf("Failed to send daily budget alert: %v\n", err)
				}
			}
		}
	}
	
	// Check monthly budget alerts
	if settings.MonthlyBudget > 0 && tracker.MonthlySpend > 0 {
		monthlyPercentage := tracker.MonthlySpend / settings.MonthlyBudget
		
		for _, threshold := range settings.AlertThresholds {
			if monthlyPercentage >= threshold {
				alert := &BudgetAlert{
					OrganizationID: organizationID,
					AlertType:      "monthly",
					Threshold:      threshold,
					CurrentSpend:   tracker.MonthlySpend,
					BudgetLimit:    settings.MonthlyBudget,
					Percentage:     monthlyPercentage,
					Message:        fmt.Sprintf("Monthly categorization budget is %.1f%% used ($%.2f of $%.2f)", monthlyPercentage*100, tracker.MonthlySpend, settings.MonthlyBudget),
					Severity:       m.getSeverity(monthlyPercentage),
				}
				
				if err := m.alertService.SendBudgetAlert(ctx, organizationID, alert); err != nil {
					fmt.Printf("Failed to send monthly budget alert: %v\n", err)
				}
			}
		}
	}
	
	return nil
}

// getSeverity returns the severity level based on percentage
func (m *costManager) getSeverity(percentage float64) string {
	if percentage >= 0.95 {
		return "critical"
	} else if percentage >= 0.8 {
		return "warning"
	}
	return "info"
}

// CostOptimizer provides cost optimization strategies
type CostOptimizer struct {
	costManager CostManager
}

// NewCostOptimizer creates a new cost optimizer
func NewCostOptimizer(costManager CostManager) *CostOptimizer {
	return &CostOptimizer{
		costManager: costManager,
	}
}

// OptimizationStrategy represents a cost optimization strategy
type OptimizationStrategy struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	EstimatedSavings   float64 `json:"estimated_savings"`
	ImplementationCost float64 `json:"implementation_cost"`
	ROIMonths          int     `json:"roi_months"`
	Priority           string  `json:"priority"` // "high", "medium", "low"
}

// GetOptimizationStrategies returns cost optimization recommendations
func (o *CostOptimizer) GetOptimizationStrategies(ctx context.Context, organizationID uuid.UUID) ([]*OptimizationStrategy, error) {
	tracker, err := o.costManager.GetCostTracker(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost tracker: %w", err)
	}
	
	var strategies []*OptimizationStrategy
	
	// Strategy 1: Improve rule coverage
	if tracker.AvgCostPerTxn > 0.002 { // If average cost is high
		strategies = append(strategies, &OptimizationStrategy{
			Name:               "Improve Rule Coverage",
			Description:        "Add more categorization rules to reduce LLM usage",
			EstimatedSavings:   tracker.MonthlySpend * 0.3, // 30% savings
			ImplementationCost: 0.0,                        // Free
			ROIMonths:          1,
			Priority:           "high",
		})
	}
	
	// Strategy 2: Pattern learning optimization
	strategies = append(strategies, &OptimizationStrategy{
		Name:               "Enhanced Pattern Learning",
		Description:        "Improve pattern matching to reduce LLM dependency",
		EstimatedSavings:   tracker.MonthlySpend * 0.2, // 20% savings
		ImplementationCost: 0.0,                        // Free
		ROIMonths:          2,
		Priority:           "medium",
	})
	
	// Strategy 3: Batch size optimization
	if tracker.TransactionCount > 1000 {
		strategies = append(strategies, &OptimizationStrategy{
			Name:               "Optimize Batch Sizes",
			Description:        "Increase batch sizes to reduce per-transaction costs",
			EstimatedSavings:   tracker.MonthlySpend * 0.15, // 15% savings
			ImplementationCost: 0.0,                         // Free
			ROIMonths:          1,
			Priority:           "medium",
		})
	}
	
	// Strategy 4: Model selection optimization
	strategies = append(strategies, &OptimizationStrategy{
		Name:               "Model Selection Optimization",
		Description:        "Use cheaper models for simple categorizations",
		EstimatedSavings:   tracker.MonthlySpend * 0.25, // 25% savings
		ImplementationCost: 0.0,                         // Free
		ROIMonths:          1,
		Priority:           "high",
	})
	
	return strategies, nil
}

// RateLimiter provides rate limiting for LLM API calls
type RateLimiter struct {
	maxRequestsPerMinute int
	maxRequestsPerHour   int
	maxCostPerHour       float64
	currentRequests      int
	currentCost          float64
	resetTime            time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequestsPerMinute, maxRequestsPerHour int, maxCostPerHour float64) *RateLimiter {
	return &RateLimiter{
		maxRequestsPerMinute: maxRequestsPerMinute,
		maxRequestsPerHour:   maxRequestsPerHour,
		maxCostPerHour:       maxCostPerHour,
		resetTime:            time.Now().Add(time.Hour),
	}
}

// CheckLimit checks if a request can be made within rate limits
func (rl *RateLimiter) CheckLimit(estimatedCost float64) error {
	now := time.Now()
	
	// Reset counters if hour has passed
	if now.After(rl.resetTime) {
		rl.currentRequests = 0
		rl.currentCost = 0.0
		rl.resetTime = now.Add(time.Hour)
	}
	
	// Check request limits
	if rl.currentRequests >= rl.maxRequestsPerHour {
		return fmt.Errorf("hourly request limit exceeded (%d/%d)", rl.currentRequests, rl.maxRequestsPerHour)
	}
	
	// Check cost limits
	if rl.currentCost+estimatedCost > rl.maxCostPerHour {
		return fmt.Errorf("hourly cost limit would be exceeded ($%.2f + $%.2f > $%.2f)", 
			rl.currentCost, estimatedCost, rl.maxCostPerHour)
	}
	
	return nil
}

// RecordRequest records a completed request
func (rl *RateLimiter) RecordRequest(actualCost float64) {
	rl.currentRequests++
	rl.currentCost += actualCost
}