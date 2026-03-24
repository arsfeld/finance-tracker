package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/config"
	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

type BudgetHandler struct {
	budgetStore *store.BudgetStore
	txnStore    *store.TransactionStore
	cfg         *config.Config
	events      *EventHub
}

func NewBudgetHandler(bs *store.BudgetStore, ts *store.TransactionStore, cfg *config.Config, events *EventHub) *BudgetHandler {
	return &BudgetHandler{budgetStore: bs, txnStore: ts, cfg: cfg, events: events}
}

// BudgetedCategory represents a category with a budget and its current spending.
type BudgetedCategory struct {
	Category  string  `json:"category"`
	Amount    float64 `json:"amount"`
	Spent     float64 `json:"spent"`
	Remaining float64 `json:"remaining"`
	Percent   float64 `json:"percent"`
}

// UnbudgetedCategory represents a category with spending but no budget set.
type UnbudgetedCategory struct {
	Category        string   `json:"category"`
	Spent           float64  `json:"spent"`
	SuggestedAmount *float64 `json:"suggested_amount,omitempty"`
}

// BudgetPeriodInfo contains the current billing period boundaries.
type BudgetPeriodInfo struct {
	Label string `json:"label"`
	Start int64  `json:"start"`
	End   int64  `json:"end"`
}

// BudgetStatusResponse is the response for GET /api/budgets/status.
type BudgetStatusResponse struct {
	Period     BudgetPeriodInfo     `json:"period"`
	Budgeted   []BudgetedCategory   `json:"budgeted"`
	Unbudgeted []UnbudgetedCategory `json:"unbudgeted"`
}

// validateBudgetInput checks category and amount validity. Returns an error message or empty string.
func validateBudgetInput(category string, amount float64) string {
	if category == "" || len(category) > 100 {
		return "Invalid category name"
	}
	if amount <= 0 || amount > 1_000_000 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "Amount must be between 0 and 1,000,000"
	}
	if strings.EqualFold(category, "Uncategorized") {
		return "Cannot set budget for Uncategorized"
	}
	return ""
}

// Upsert creates or updates a budget for a category.
// POST /api/budgets
func (h *BudgetHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body struct {
		Category string  `json:"category"`
		Amount   float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	body.Category = strings.TrimSpace(body.Category)
	if msg := validateBudgetInput(body.Category, body.Amount); msg != "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", msg)
		return
	}

	if err := h.budgetStore.Upsert(r.Context(), body.Category, body.Amount); err != nil {
		log.Error().Err(err).Str("category", body.Category).Msg("Failed to upsert budget")
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save budget")
		return
	}

	h.events.Broadcast("budgets_updated", `{"status":"ok"}`)
	WriteData(w, models.Budget{Category: body.Category, Amount: body.Amount})
}

// Delete removes a budget for a category.
// DELETE /api/budgets/{category}
func (h *BudgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("category")
	if category == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Category is required")
		return
	}

	deleted, err := h.budgetStore.Delete(r.Context(), category)
	if err != nil {
		log.Error().Err(err).Str("category", category).Msg("Failed to delete budget")
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete budget")
		return
	}
	if !deleted {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "No budget found for that category")
		return
	}

	h.events.Broadcast("budgets_updated", `{"status":"ok"}`)
	WriteData(w, map[string]string{"deleted": category})
}

// Status returns budget progress for the current billing period.
// GET /api/budgets/status
func (h *BudgetHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	start, end := billing.CurrentBillingPeriod(h.cfg.BillingDay)

	budgets, err := h.budgetStore.GetAll(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load budgets")
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to load budgets")
		return
	}

	spending, err := h.txnStore.CountByCategory(ctx, start.Unix(), end.Unix())
	if err != nil {
		log.Error().Err(err).Msg("Failed to load spending data")
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to load spending data")
		return
	}

	// Build a set of budgeted category names (lowercased for case-insensitive matching).
	budgetMap := make(map[string]models.Budget)
	for _, b := range budgets {
		budgetMap[strings.ToLower(b.Category)] = b
	}

	var budgeted []BudgetedCategory
	var unbudgeted []UnbudgetedCategory
	matchedBudgets := make(map[string]bool)

	// Process all spending categories.
	for cat, spent := range spending {
		key := strings.ToLower(cat)
		if b, ok := budgetMap[key]; ok {
			matchedBudgets[key] = true
			remaining := b.Amount - spent
			percent := 0.0
			if b.Amount > 0 {
				percent = (spent / b.Amount) * 100
			}
			budgeted = append(budgeted, BudgetedCategory{
				Category:  cat,
				Amount:    b.Amount,
				Spent:     math.Round(spent*100) / 100,
				Remaining: math.Round(remaining*100) / 100,
				Percent:   math.Round(percent*100) / 100,
			})
		} else {
			unbudgeted = append(unbudgeted, UnbudgetedCategory{
				Category: cat,
				Spent:    math.Round(spent*100) / 100,
			})
		}
	}

	// Include budgeted categories with zero spending.
	for _, b := range budgets {
		key := strings.ToLower(b.Category)
		if !matchedBudgets[key] {
			budgeted = append(budgeted, BudgetedCategory{
				Category:  b.Category,
				Amount:    b.Amount,
				Spent:     0,
				Remaining: b.Amount,
				Percent:   0,
			})
		}
	}

	// Compute 3-month average suggestions for unbudgeted categories.
	if len(unbudgeted) > 0 {
		suggestions := h.computeSuggestedAmounts(ctx, start)
		for i := range unbudgeted {
			key := strings.ToLower(unbudgeted[i].Category)
			if avg, ok := suggestions[key]; ok {
				rounded := math.Ceil(avg/10) * 10
				unbudgeted[i].SuggestedAmount = &rounded
			}
		}
	}

	// Sort budgeted by percent descending (highest risk first).
	sortBudgetedByPercent(budgeted)

	period := BudgetPeriodInfo{
		Label: fmt.Sprintf("%s %d", start.Month().String()[:3], start.Year()),
		Start: start.Unix(),
		End:   end.Unix(),
	}

	WriteData(w, BudgetStatusResponse{
		Period:     period,
		Budgeted:   budgeted,
		Unbudgeted: unbudgeted,
	})
}

// computeSuggestedAmounts returns average spending per category over the prior 3 billing periods.
// Uses a single grouped query instead of 3 separate queries.
func (h *BudgetHandler) computeSuggestedAmounts(ctx context.Context, currentStart time.Time) map[string]float64 {
	// Range: 3 months before current period start.
	histStart := currentStart.AddDate(0, -3, 0)
	histEnd := currentStart.Add(-time.Second)

	grouped, err := h.txnStore.CountByCategoryGrouped(ctx, histStart.Unix(), histEnd.Unix())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to compute budget suggestions")
		return nil
	}

	result := make(map[string]float64)
	for cat, months := range grouped {
		key := strings.ToLower(cat)
		total := 0.0
		for _, v := range months {
			total += v
		}
		if len(months) > 0 {
			result[key] = total / float64(len(months))
		}
	}
	return result
}

// sortBudgetedByPercent sorts budgeted categories by percent descending.
func sortBudgetedByPercent(items []BudgetedCategory) {
	sort.Slice(items, func(i, j int) bool { return items[i].Percent > items[j].Percent })
}
