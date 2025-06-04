package categorization

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"finance_tracker/src/internal/models"
)

// ruleEngine implements the RuleEngine interface
type ruleEngine struct {
	repo RuleRepository
}

// NewRuleEngine creates a new rule-based categorization engine
func NewRuleEngine(repo RuleRepository) RuleEngine {
	return &ruleEngine{
		repo: repo,
	}
}

// CategorizeByRules attempts to categorize transaction using predefined rules
func (e *ruleEngine) CategorizeByRules(ctx context.Context, transaction *models.Transaction) (*models.CategorizationResult, error) {
	startTime := time.Now()

	// Get all rules for the organization
	rules, err := e.repo.GetRulesByOrganization(ctx, transaction.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	if len(rules) == 0 {
		return &models.CategorizationResult{
			CategoryID:       nil,
			Confidence:       0.0,
			Method:           models.CategorizationMethodRuleBased,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			CostEstimate:     0.0,
			Explanation:      "No rules defined for organization",
		}, nil
	}

	// Sort rules by priority (highest first)
	sortedRules := make([]*models.EnhancedCategoryRule, len(rules))
	copy(sortedRules, rules)
	
	// Simple bubble sort by priority (descending)
	for i := 0; i < len(sortedRules)-1; i++ {
		for j := 0; j < len(sortedRules)-i-1; j++ {
			if sortedRules[j].Priority < sortedRules[j+1].Priority {
				sortedRules[j], sortedRules[j+1] = sortedRules[j+1], sortedRules[j]
			}
		}
	}

	var bestMatch *models.EnhancedCategoryRule
	var bestConfidence float64
	var ruleMatches []models.RuleMatch

	// Evaluate each rule
	for _, rule := range sortedRules {
		match, confidence := e.evaluateRule(rule, transaction)
		if match {
			ruleMatches = append(ruleMatches, models.RuleMatch{
				RuleID:     rule.ID,
				RuleType:   rule.RuleType,
				Pattern:    rule.Pattern,
				Confidence: confidence,
				Priority:   rule.Priority,
			})

			// Use the first (highest priority) matching rule
			if bestMatch == nil || confidence > bestConfidence {
				bestMatch = rule
				bestConfidence = confidence
			}
		}
	}

	processingTime := time.Since(startTime).Milliseconds()

	// If no rules matched
	if bestMatch == nil {
		return &models.CategorizationResult{
			CategoryID:       nil,
			Confidence:       0.0,
			Method:           models.CategorizationMethodRuleBased,
			ProcessingTimeMs: processingTime,
			CostEstimate:     0.0,
			Metadata: models.CategorizationMetadata{
				RuleMatches: ruleMatches,
			},
			Explanation: "No matching rules found",
		}, nil
	}

	// Update rule usage stats
	go func() {
		ctx := context.Background()
		if err := e.repo.UpdateRuleUsage(ctx, bestMatch.ID, true); err != nil {
			// Log error but don't fail the categorization
			fmt.Printf("Failed to update rule usage for rule %s: %v\n", bestMatch.ID, err)
		}
	}()

	return &models.CategorizationResult{
		CategoryID:       &bestMatch.CategoryID,
		Confidence:       bestConfidence,
		Method:           models.CategorizationMethodRuleBased,
		ProcessingTimeMs: processingTime,
		CostEstimate:     0.0,
		Metadata: models.CategorizationMetadata{
			RuleMatches: ruleMatches,
		},
		Explanation: fmt.Sprintf("Matched rule: %s (priority: %d)", bestMatch.Pattern, bestMatch.Priority),
	}, nil
}

// evaluateRule evaluates a single rule against a transaction
func (e *ruleEngine) evaluateRule(rule *models.EnhancedCategoryRule, transaction *models.Transaction) (bool, float64) {
	switch rule.RuleType {
	case models.RuleTypeMerchantPattern:
		return e.evaluateMerchantPattern(rule, transaction)
	case models.RuleTypeDescriptionKeyword:
		return e.evaluateDescriptionKeyword(rule, transaction)
	case models.RuleTypeAmountRange:
		return e.evaluateAmountRange(rule, transaction)
	case models.RuleTypeRegexPattern:
		return e.evaluateRegexPattern(rule, transaction)
	default:
		return false, 0.0
	}
}

// evaluateMerchantPattern evaluates merchant pattern rules
func (e *ruleEngine) evaluateMerchantPattern(rule *models.EnhancedCategoryRule, transaction *models.Transaction) (bool, float64) {
	if transaction.MerchantName == nil {
		return false, 0.0
	}

	merchantName := *transaction.MerchantName
	pattern := rule.Pattern

	if !rule.IsCaseSensitive {
		merchantName = strings.ToLower(merchantName)
		pattern = strings.ToLower(pattern)
	}

	if rule.IsRegex {
		matched, err := regexp.MatchString(pattern, merchantName)
		if err != nil {
			return false, 0.0
		}
		if matched {
			return true, rule.Confidence
		}
	} else {
		// Support wildcards (* and ?)
		if e.wildcardMatch(pattern, merchantName) {
			return true, rule.Confidence
		}
	}

	return false, 0.0
}

// evaluateDescriptionKeyword evaluates description keyword rules
func (e *ruleEngine) evaluateDescriptionKeyword(rule *models.EnhancedCategoryRule, transaction *models.Transaction) (bool, float64) {
	if transaction.Description == nil {
		return false, 0.0
	}

	description := *transaction.Description
	pattern := rule.Pattern

	if !rule.IsCaseSensitive {
		description = strings.ToLower(description)
		pattern = strings.ToLower(pattern)
	}

	if rule.IsRegex {
		matched, err := regexp.MatchString(pattern, description)
		if err != nil {
			return false, 0.0
		}
		if matched {
			return true, rule.Confidence
		}
	} else {
		// Check if pattern is contained in description
		if strings.Contains(description, pattern) {
			return true, rule.Confidence
		}
	}

	return false, 0.0
}

// evaluateAmountRange evaluates amount range rules
func (e *ruleEngine) evaluateAmountRange(rule *models.EnhancedCategoryRule, transaction *models.Transaction) (bool, float64) {
	// Pattern format: "min,max" or "min," (no max) or ",max" (no min)
	parts := strings.Split(rule.Pattern, ",")
	if len(parts) != 2 {
		return false, 0.0
	}

	amount := transaction.Amount

	// Parse min value
	if parts[0] != "" {
		min, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return false, 0.0
		}
		if amount < min {
			return false, 0.0
		}
	}

	// Parse max value
	if parts[1] != "" {
		max, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return false, 0.0
		}
		if amount > max {
			return false, 0.0
		}
	}

	return true, rule.Confidence
}

// evaluateRegexPattern evaluates regex pattern rules
func (e *ruleEngine) evaluateRegexPattern(rule *models.EnhancedCategoryRule, transaction *models.Transaction) (bool, float64) {
	// Combine description and merchant name for regex matching
	var text string
	if transaction.Description != nil {
		text += *transaction.Description
	}
	if transaction.MerchantName != nil {
		if text != "" {
			text += " "
		}
		text += *transaction.MerchantName
	}

	if text == "" {
		return false, 0.0
	}

	if !rule.IsCaseSensitive {
		text = strings.ToLower(text)
	}

	matched, err := regexp.MatchString(rule.Pattern, text)
	if err != nil {
		return false, 0.0
	}

	if matched {
		return true, rule.Confidence
	}

	return false, 0.0
}

// wildcardMatch performs wildcard matching (* and ?)
func (e *ruleEngine) wildcardMatch(pattern, text string) bool {
	// Convert wildcard pattern to regex
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "\\?", ".")
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, text)
	if err != nil {
		return false
	}

	return matched
}

// AddRule adds a new categorization rule
func (e *ruleEngine) AddRule(ctx context.Context, rule *models.EnhancedCategoryRule) error {
	// Set ID if not provided
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}

	// Set timestamps
	rule.CreatedAt = time.Now()

	// Validate rule
	if err := e.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	return e.repo.CreateRule(ctx, rule)
}

// UpdateRule updates an existing categorization rule
func (e *ruleEngine) UpdateRule(ctx context.Context, rule *models.EnhancedCategoryRule) error {
	// Validate rule
	if err := e.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	return e.repo.UpdateRule(ctx, rule)
}

// DeleteRule deletes a categorization rule
func (e *ruleEngine) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	return e.repo.DeleteRule(ctx, ruleID)
}

// GetRules returns all rules for an organization
func (e *ruleEngine) GetRules(ctx context.Context, organizationID uuid.UUID) ([]*models.EnhancedCategoryRule, error) {
	return e.repo.GetRulesByOrganization(ctx, organizationID)
}

// TestRule tests a rule against historical transactions
func (e *ruleEngine) TestRule(ctx context.Context, rule *models.EnhancedCategoryRule) (*RuleTestResult, error) {
	// This would require access to transaction repository to test against historical data
	// For now, return a placeholder implementation
	return &RuleTestResult{
		Rule:                rule,
		MatchedTransactions: 0,
		AccuracyRate:        0.0,
		Examples:            []*models.Transaction{},
	}, nil
}

// validateRule validates a rule configuration
func (e *ruleEngine) validateRule(rule *models.EnhancedCategoryRule) error {
	if rule.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization_id is required")
	}

	if rule.CategoryID <= 0 {
		return fmt.Errorf("category_id must be positive")
	}

	if rule.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}

	if rule.Confidence < 0.0 || rule.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0")
	}

	if rule.Priority < 0 {
		return fmt.Errorf("priority must be non-negative")
	}

	// Validate rule type
	switch rule.RuleType {
	case models.RuleTypeMerchantPattern:
	case models.RuleTypeDescriptionKeyword:
	case models.RuleTypeAmountRange:
		// Validate amount range pattern
		parts := strings.Split(rule.Pattern, ",")
		if len(parts) != 2 {
			return fmt.Errorf("amount range pattern must be in format 'min,max'")
		}
		if parts[0] != "" {
			if _, err := strconv.ParseFloat(parts[0], 64); err != nil {
				return fmt.Errorf("invalid min amount: %w", err)
			}
		}
		if parts[1] != "" {
			if _, err := strconv.ParseFloat(parts[1], 64); err != nil {
				return fmt.Errorf("invalid max amount: %w", err)
			}
		}
	case models.RuleTypeRegexPattern:
		// Validate regex pattern
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	default:
		return fmt.Errorf("invalid rule type: %s", rule.RuleType)
	}

	return nil
}