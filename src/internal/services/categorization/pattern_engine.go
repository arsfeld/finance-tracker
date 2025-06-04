package categorization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"finance_tracker/src/internal/models"
)

// patternEngine implements the PatternEngine interface
type patternEngine struct {
	repo PatternRepository
}

// NewPatternEngine creates a new pattern matching categorization engine
func NewPatternEngine(repo PatternRepository) PatternEngine {
	return &patternEngine{
		repo: repo,
	}
}

// CategorizeByPatterns attempts to categorize transaction using pattern matching
func (e *patternEngine) CategorizeByPatterns(ctx context.Context, transaction *models.Transaction) (*models.CategorizationResult, error) {
	startTime := time.Now()

	if transaction.MerchantName == nil {
		return &models.CategorizationResult{
			CategoryID:       nil,
			Confidence:       0.0,
			Method:           models.CategorizationMethodPatternMatching,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			CostEstimate:     0.0,
			Explanation:      "No merchant name available for pattern matching",
		}, nil
	}

	merchantName := strings.TrimSpace(*transaction.MerchantName)
	if merchantName == "" {
		return &models.CategorizationResult{
			CategoryID:       nil,
			Confidence:       0.0,
			Method:           models.CategorizationMethodPatternMatching,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			CostEstimate:     0.0,
			Explanation:      "Empty merchant name for pattern matching",
		}, nil
	}

	// First try exact match with cached patterns
	exactMatch, err := e.findExactMatch(ctx, transaction.OrganizationID, merchantName)
	if err != nil {
		return nil, fmt.Errorf("failed to find exact match: %w", err)
	}

	if exactMatch != nil {
		// Update usage statistics
		go func() {
			ctx := context.Background()
			e.repo.UpdatePattern(ctx, transaction.OrganizationID, merchantName, exactMatch.CategoryID, exactMatch.Confidence)
		}()

		return &models.CategorizationResult{
			CategoryID:       &exactMatch.CategoryID,
			Confidence:       exactMatch.Confidence,
			Method:           models.CategorizationMethodPatternMatching,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			CostEstimate:     0.0,
			Explanation:      fmt.Sprintf("Exact pattern match: %s", merchantName),
		}, nil
	}

	// Try fuzzy matching with similar patterns
	similarPatterns, err := e.repo.GetSimilarPatterns(ctx, transaction.OrganizationID, merchantName, 0.3)
	if err != nil {
		return nil, fmt.Errorf("failed to get similar patterns: %w", err)
	}

	if len(similarPatterns) == 0 {
		return &models.CategorizationResult{
			CategoryID:       nil,
			Confidence:       0.0,
			Method:           models.CategorizationMethodPatternMatching,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			CostEstimate:     0.0,
			Explanation:      "No similar patterns found",
		}, nil
	}

	// Find the best match considering similarity and usage frequency
	bestMatch := e.selectBestPattern(similarPatterns)
	
	// Adjust confidence based on similarity score
	adjustedConfidence := bestMatch.Confidence * bestMatch.Similarity

	// Store the new pattern if confidence is high enough
	if adjustedConfidence > 0.6 {
		go func() {
			ctx := context.Background()
			e.repo.UpdatePattern(ctx, transaction.OrganizationID, merchantName, bestMatch.CategoryID, adjustedConfidence)
		}()
	}

	return &models.CategorizationResult{
		CategoryID:       &bestMatch.CategoryID,
		Confidence:       adjustedConfidence,
		Method:           models.CategorizationMethodPatternMatching,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		CostEstimate:     0.0,
		Explanation:      fmt.Sprintf("Fuzzy pattern match: %s (similarity: %.2f)", bestMatch.MerchantPattern, bestMatch.Similarity),
	}, nil
}

// findExactMatch looks for an exact match in the pattern cache
func (e *patternEngine) findExactMatch(ctx context.Context, organizationID uuid.UUID, merchantName string) (*models.MerchantPatternCache, error) {
	// Get all patterns for the organization
	patterns, err := e.repo.GetPatternsByOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	// Look for exact match (case-insensitive)
	merchantLower := strings.ToLower(merchantName)
	for _, pattern := range patterns {
		if strings.ToLower(pattern.MerchantPattern) == merchantLower {
			return pattern, nil
		}
	}

	return nil, nil
}

// selectBestPattern selects the best pattern from similar matches
func (e *patternEngine) selectBestPattern(patterns []*models.SimilarMerchantPattern) *models.SimilarMerchantPattern {
	if len(patterns) == 0 {
		return nil
	}

	bestPattern := patterns[0]
	bestScore := e.calculatePatternScore(bestPattern)

	for i := 1; i < len(patterns); i++ {
		score := e.calculatePatternScore(patterns[i])
		if score > bestScore {
			bestPattern = patterns[i]
			bestScore = score
		}
	}

	return bestPattern
}

// calculatePatternScore calculates a composite score for pattern ranking
func (e *patternEngine) calculatePatternScore(pattern *models.SimilarMerchantPattern) float64 {
	// Weighted scoring: similarity (70%) + usage frequency (20%) + confidence (10%)
	similarityScore := pattern.Similarity * 0.7
	
	// Normalize usage count (assuming max usage of 100)
	usageScore := float64(pattern.UsageCount) / 100.0
	if usageScore > 1.0 {
		usageScore = 1.0
	}
	usageScore *= 0.2
	
	confidenceScore := pattern.Confidence * 0.1
	
	return similarityScore + usageScore + confidenceScore
}

// UpdatePatternCache updates the pattern cache with new transaction data
func (e *patternEngine) UpdatePatternCache(ctx context.Context, transaction *models.Transaction, categoryID int, confidence float64) error {
	if transaction.MerchantName == nil {
		return nil // Nothing to cache
	}

	merchantName := strings.TrimSpace(*transaction.MerchantName)
	if merchantName == "" {
		return nil // Nothing to cache
	}

	return e.repo.UpdatePattern(ctx, transaction.OrganizationID, merchantName, categoryID, confidence)
}

// GetSimilarPatterns returns similar merchant patterns for a transaction
func (e *patternEngine) GetSimilarPatterns(ctx context.Context, organizationID uuid.UUID, merchantName string, threshold float64) ([]*models.SimilarMerchantPattern, error) {
	return e.repo.GetSimilarPatterns(ctx, organizationID, merchantName, threshold)
}

// ClearPatternCache clears the pattern cache for an organization
func (e *patternEngine) ClearPatternCache(ctx context.Context, organizationID uuid.UUID) error {
	return e.repo.ClearPatterns(ctx, organizationID)
}

// ExtractMerchantName extracts a clean merchant name from raw transaction data
func (e *patternEngine) ExtractMerchantName(transaction *models.Transaction) string {
	var merchantName string

	// Try merchant_name field first
	if transaction.MerchantName != nil && strings.TrimSpace(*transaction.MerchantName) != "" {
		merchantName = strings.TrimSpace(*transaction.MerchantName)
	} else if transaction.Description != nil && strings.TrimSpace(*transaction.Description) != "" {
		// Fall back to description if no merchant name
		merchantName = strings.TrimSpace(*transaction.Description)
	} else {
		return ""
	}

	// Clean up common patterns in merchant names
	merchantName = e.cleanMerchantName(merchantName)

	return merchantName
}

// cleanMerchantName cleans up merchant names for better pattern matching
func (e *patternEngine) cleanMerchantName(merchantName string) string {
	// Convert to uppercase for consistency
	cleaned := strings.ToUpper(merchantName)

	// Remove common suffixes and prefixes
	patterns := []string{
		// Card processor patterns
		"*",
		"- DEBIT",
		"- CREDIT",
		"- CHECKCARD",
		"DEBIT CARD",
		"CREDIT CARD",
		"CHECKCARD",
		
		// Location patterns
		" #[0-9]+", // Store numbers
		" [0-9]{5}", // ZIP codes
		" [A-Z]{2}$", // State codes at end
		
		// Transaction IDs and references
		" [0-9]{4,}", // Long numbers
		"REF #[A-Z0-9]+",
		"TXN [A-Z0-9]+",
		
		// Common business suffixes
		" LLC$",
		" INC$",
		" CORP$",
		" LTD$",
		" CO$",
	}

	// Apply cleaning patterns
	for _, pattern := range patterns {
		if strings.Contains(pattern, "[") || strings.Contains(pattern, "*") {
			// Regex pattern - would need proper regex implementation
			continue
		}
		cleaned = strings.ReplaceAll(cleaned, pattern, "")
	}

	// Remove extra whitespace
	cleaned = strings.TrimSpace(cleaned)
	
	// Replace multiple spaces with single space
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}

	return cleaned
}

// GeneratePatternVariations generates variations of a merchant name for better matching
func (e *patternEngine) GeneratePatternVariations(merchantName string) []string {
	variations := []string{merchantName}
	
	// Add cleaned version
	cleaned := e.cleanMerchantName(merchantName)
	if cleaned != merchantName {
		variations = append(variations, cleaned)
	}
	
	// Add variations without common words
	commonWords := []string{"THE", "AND", "&", "OF", "AT", "IN", "ON"}
	for _, word := range commonWords {
		variant := strings.ReplaceAll(cleaned, " "+word+" ", " ")
		variant = strings.TrimSpace(variant)
		if variant != cleaned && variant != "" {
			variations = append(variations, variant)
		}
	}
	
	// Add first significant word if multiple words
	words := strings.Fields(cleaned)
	if len(words) > 1 {
		// Skip common prefixes
		firstWord := words[0]
		if len(firstWord) > 3 && !e.isCommonPrefix(firstWord) {
			variations = append(variations, firstWord)
		}
	}
	
	return variations
}

// isCommonPrefix checks if a word is a common business prefix
func (e *patternEngine) isCommonPrefix(word string) bool {
	commonPrefixes := []string{"THE", "A", "AN", "AND", "&"}
	for _, prefix := range commonPrefixes {
		if word == prefix {
			return true
		}
	}
	return false
}