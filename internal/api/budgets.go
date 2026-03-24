package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
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
}

func NewBudgetHandler(bs *store.BudgetStore, ts *store.TransactionStore, cfg *config.Config) *BudgetHandler {
	return &BudgetHandler{budgetStore: bs, txnStore: ts, cfg: cfg}
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
	if body.Category == "" || len(body.Category) > 100 {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category name")
		return
	}
	if body.Amount <= 0 || body.Amount > 1_000_000 || math.IsNaN(body.Amount) || math.IsInf(body.Amount, 0) {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Amount must be between 0 and 1,000,000")
		return
	}
	if strings.EqualFold(body.Category, "Uncategorized") {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Cannot set budget for Uncategorized")
		return
	}

	if err := h.budgetStore.Upsert(r.Context(), body.Category, body.Amount); err != nil {
		log.Error().Err(err).Str("category", body.Category).Msg("Failed to upsert budget")
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save budget")
		return
	}

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

	if err := h.budgetStore.Delete(r.Context(), category); err != nil {
		log.Error().Err(err).Str("category", category).Msg("Failed to delete budget")
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete budget")
		return
	}

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
		suggestions := h.computeSuggestedAmounts(ctx, start, h.cfg.BillingDay)
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
func (h *BudgetHandler) computeSuggestedAmounts(ctx context.Context, currentStart time.Time, billingDay int) map[string]float64 {
	totals := make(map[string]float64)
	counts := make(map[string]int)

	// Look at 3 prior billing periods.
	periodStart := currentStart
	for i := 0; i < 3; i++ {
		periodStart = periodStart.AddDate(0, -1, 0)
		periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

		spending, err := h.txnStore.CountByCategory(ctx, periodStart.Unix(), periodEnd.Unix())
		if err != nil {
			continue
		}
		for cat, amount := range spending {
			key := strings.ToLower(cat)
			totals[key] += amount
			counts[key]++
		}
	}

	result := make(map[string]float64)
	for key, total := range totals {
		if counts[key] > 0 {
			result[key] = total / float64(counts[key])
		}
	}
	return result
}

// sortBudgetedByPercent sorts budgeted categories by percent descending.
func sortBudgetedByPercent(items []BudgetedCategory) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Percent > items[j-1].Percent; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
