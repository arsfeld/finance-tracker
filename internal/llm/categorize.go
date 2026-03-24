package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

const categorizationSystemPrompt = "You are a financial transaction categorization system. Respond with valid JSON only. Categorize merchants into clear spending categories like Dining, Groceries, Transportation, Entertainment, Subscriptions, Shopping, Healthcare, Utilities, etc."

// CategorizeTransactions identifies uncategorized merchants and calls the LLM to categorize them.
// Results are persisted in the category store.
func CategorizeTransactions(ctx context.Context, client *Client, catStore *store.CategoryStore, transactions []models.DBTransaction) error {
	// Collect unique descriptions and check existing categories.
	seen := make(map[string]bool)
	var uncategorized []string

	for _, t := range transactions {
		if seen[t.Description] {
			continue
		}
		seen[t.Description] = true

		if t.Category != "" {
			continue
		}

		existing, err := catStore.Get(ctx, t.Description)
		if err != nil {
			return fmt.Errorf("check category for %q: %w", t.Description, err)
		}
		if existing != "" {
			continue
		}
		uncategorized = append(uncategorized, t.Description)
	}

	log.Info().Int("known", len(seen)-len(uncategorized)).Int("uncategorized", len(uncategorized)).Msg("Category lookup results")

	if len(uncategorized) == 0 {
		log.Info().Msg("All merchants already categorized")
		return nil
	}

	// Get existing categories for context.
	existingEntries, err := catStore.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list categories: %w", err)
	}

	// Build category -> merchants map for the prompt, preserving source metadata.
	catMap := make(map[string][]categoryExample)
	for _, e := range existingEntries {
		catMap[e.Category] = append(catMap[e.Category], categoryExample{
			MerchantDescription: e.MerchantDescription,
			Source:              e.Source,
		})
	}

	prompt := generateCategorizationPrompt(uncategorized, catMap)

	log.Info().Int("count", len(uncategorized)).Msg("Categorizing new merchants with LLM")
	jsonResp, err := client.CategorizeJSON(categorizationSystemPrompt, prompt)
	if err != nil {
		return fmt.Errorf("LLM categorization: %w", err)
	}

	var catResp models.CategorizationResponse
	if err := json.Unmarshal([]byte(jsonResp), &catResp); err != nil {
		return fmt.Errorf("parse categorization response: %w (response: %s)", err, jsonResp)
	}

	// Persist new categories.
	var entries []models.CategoryEntry
	for _, tc := range catResp.Categories {
		entries = append(entries, models.CategoryEntry{
			MerchantDescription: tc.Description,
			Category:            tc.Category,
			Source:              "llm",
		})
	}

	if len(entries) > 0 {
		if err := catStore.BulkUpsert(ctx, entries); err != nil {
			return fmt.Errorf("save categories: %w", err)
		}
		log.Info().Int("count", len(entries)).Msg("Saved new categories")
	}

	return nil
}

type categoryExample struct {
	MerchantDescription string
	Source              string
}

const (
	maxUserExamples = 5
	maxLLMExamples  = 2
	maxTotalExamples = 200
)

func generateCategorizationPrompt(uncategorized []string, existingCategories map[string][]categoryExample) string {
	var sb strings.Builder

	sb.WriteString("Categorize the following merchant/transaction descriptions into spending categories.\n\n")

	if len(existingCategories) > 0 {
		sb.WriteString("Here are the existing categories and their merchants (use these as a guide for consistency):\n")
		catNames := make([]string, 0, len(existingCategories))
		for cat := range existingCategories {
			catNames = append(catNames, cat)
		}
		sort.Strings(catNames)

		totalExamples := 0
		for _, cat := range catNames {
			if totalExamples >= maxTotalExamples {
				break
			}

			examples := existingCategories[cat]

			// Partition by source: user-verified first, then LLM-assigned.
			var userExamples, llmExamples []string
			for _, ex := range examples {
				if ex.Source == "user" {
					userExamples = append(userExamples, ex.MerchantDescription)
				} else {
					llmExamples = append(llmExamples, ex.MerchantDescription)
				}
			}

			// Cap each source type.
			if len(userExamples) > maxUserExamples {
				userExamples = userExamples[:maxUserExamples]
			}
			if len(llmExamples) > maxLLMExamples {
				llmExamples = llmExamples[:maxLLMExamples]
			}

			// Build example strings with labels.
			var parts []string
			for _, m := range userExamples {
				parts = append(parts, m+" (verified by user)")
			}
			parts = append(parts, llmExamples...)

			if len(parts) == 0 {
				continue
			}

			totalExamples += len(parts)
			sb.WriteString(fmt.Sprintf("- %s: [%s]\n", cat, strings.Join(parts, ", ")))
		}
		sb.WriteString("\nAssign each new merchant to an existing category above if it fits. Create a new category only if none fit.\n")
		sb.WriteString("Categories with merchants marked '(verified by user)' are confirmed correct — prefer matching their patterns.\n\n")
	}

	sb.WriteString("Merchant descriptions to categorize:\n")
	for _, desc := range uncategorized {
		sb.WriteString(fmt.Sprintf("- %s\n", desc))
	}

	sb.WriteString("\nRespond with valid JSON only in this exact format:\n")
	sb.WriteString(`{"categories": [{"description": "MERCHANT NAME", "category": "Category Name"}, ...]}`)
	sb.WriteString("\n")

	return sb.String()
}

const similaritySystemPrompt = "You are a merchant matching system. Given a merchant that a user has categorized, identify which candidate merchants are the same brand or company. Respond with valid JSON only."

// SimilarityResponse is the LLM response for similarity queries.
type SimilarityResponse struct {
	Similar []string `json:"similar"`
}

// FindSimilarMerchants asks the LLM to identify which candidate merchants are likely the same
// brand/company as the corrected merchant. Returns only merchant descriptions that exist in candidates.
func FindSimilarMerchants(client *Client, correctedMerchant, category string, candidates []models.CategoryEntry) ([]string, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	// Build candidate set for the LLM prompt.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("The user has categorized %q as %q.\n", correctedMerchant, category))
	sb.WriteString("Which of the following merchants are likely the SAME merchant, brand, or company?\n")
	sb.WriteString("Only include merchants that are clearly the same company/brand. Be conservative.\n\n")
	sb.WriteString("Candidates:\n")

	// Track valid candidates for validation.
	validCandidates := make(map[string]bool)
	for _, c := range candidates {
		sb.WriteString(fmt.Sprintf("- %s (current: %s)\n", c.MerchantDescription, c.Category))
		validCandidates[c.MerchantDescription] = true
	}

	sb.WriteString("\nRespond with valid JSON only in this exact format:\n")
	sb.WriteString(`{"similar": ["MERCHANT NAME 1", "MERCHANT NAME 2"]}`)
	sb.WriteString("\nIf no merchants are similar, respond with: {\"similar\": []}\n")

	prompt := sb.String()

	log.Info().
		Str("merchant", correctedMerchant).
		Str("category", category).
		Int("candidates", len(candidates)).
		Msg("Finding similar merchants via LLM")

	jsonResp, err := client.CategorizeJSON(similaritySystemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM similarity: %w", err)
	}

	var resp SimilarityResponse
	if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
		return nil, fmt.Errorf("parse similarity response: %w (response: %s)", err, jsonResp)
	}

	// Validate results against actual candidate list to filter hallucinations.
	var validated []string
	for _, m := range resp.Similar {
		if validCandidates[m] {
			validated = append(validated, m)
		} else {
			log.Warn().Str("merchant", m).Msg("LLM returned merchant not in candidate set, filtering out")
		}
	}

	log.Info().Int("found", len(validated)).Msg("Similar merchants identified")
	return validated, nil
}
