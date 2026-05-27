package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
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

// BudgetPeriodSummary aggregates budget performance for a billing period.
type BudgetPeriodSummary struct {
	TotalBudget    float64 `json:"total_budget"`
	TotalSpent     float64 `json:"total_spent"`
	TotalRemaining float64 `json:"total_remaining"`
	TotalPercent   float64 `json:"total_percent"`
	OverCount      int     `json:"over_count"`
}

// BudgetHistoryPeriod is budget performance for one billing period.
type BudgetHistoryPeriod struct {
	Label      string              `json:"label"`
	Start      int64               `json:"start"`
	End        int64               `json:"end"`
	IsComplete bool                `json:"is_complete"`
	Budgeted   []BudgetedCategory  `json:"budgeted"`
	Summary    BudgetPeriodSummary `json:"summary"`
}

// BudgetCategoryPeriodValue is one category's result in a historical period.
type BudgetCategoryPeriodValue struct {
	Label   string  `json:"label"`
	Spent   float64 `json:"spent"`
	Percent float64 `json:"percent"`
}

// BudgetCategoryHistory summarizes a category across multiple periods.
type BudgetCategoryHistory struct {
	Category     string                      `json:"category"`
	Amount       float64                     `json:"amount"`
	AvgSpent     float64                     `json:"avg_spent"`
	PeriodsOver  int                         `json:"periods_over"`
	PeriodsUnder int                         `json:"periods_under"`
	ByPeriod     []BudgetCategoryPeriodValue `json:"by_period"`
}

// BudgetImprovementProposal suggests a budget adjustment based on history.
type BudgetImprovementProposal struct {
	Category        string  `json:"category"`
	CurrentAmount   float64 `json:"current_amount"`
	SuggestedAmount float64 `json:"suggested_amount"`
	Type            string  `json:"type"`
	Reason          string  `json:"reason"`
}

// BudgetHistoryResponse is the response for GET /api/budgets/history.
type BudgetHistoryResponse struct {
	Periods    []BudgetHistoryPeriod       `json:"periods"`
	Categories []BudgetCategoryHistory     `json:"categories"`
	Proposals  []BudgetImprovementProposal `json:"proposals"`
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

	budgeted, unbudgeted, _ := buildBudgetStatus(budgets, spending)

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

// History returns budget performance across recent billing periods.
// GET /api/budgets/history?months=N (default 6, max 12)
func (h *BudgetHandler) History(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	months, _ := strconv.Atoi(r.URL.Query().Get("months"))
	if months < 1 || months > 12 {
		months = 6
	}

	result, err := computeBudgetHistory(ctx, h.budgetStore, h.txnStore, h.cfg.BillingDay, months)
	if err != nil {
		log.Error().Err(err).Msg("Failed to compute budget history")
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to load budget history")
		return
	}

	WriteData(w, result)
}

func computeBudgetHistory(
	ctx context.Context,
	budgetStore *store.BudgetStore,
	txnStore *store.TransactionStore,
	billingDay, months int,
) (BudgetHistoryResponse, error) {
	budgets, err := budgetStore.GetAll(ctx)
	if err != nil {
		return BudgetHistoryResponse{}, err
	}
	if len(budgets) == 0 {
		return BudgetHistoryResponse{
			Periods:    []BudgetHistoryPeriod{},
			Categories: []BudgetCategoryHistory{},
			Proposals:  []BudgetImprovementProposal{},
		}, nil
	}

	currentStart, currentEnd := billing.CurrentBillingPeriod(billingDay)
	histStart := currentStart.AddDate(0, -(months - 1), 0)
	periods := billing.CalculateBillingPeriods(histStart, currentEnd, billingDay)

	var historyPeriods []BudgetHistoryPeriod
	categoryHistory := make(map[string]*BudgetCategoryHistory)

	for _, p := range periods {
		spending, err := txnStore.CountByCategory(ctx, p.Start.Unix(), p.End.Unix())
		if err != nil {
			return BudgetHistoryResponse{}, err
		}

		budgeted, _, summary := buildBudgetStatus(budgets, spending)
		sortBudgetedByPercent(budgeted)

		historyPeriods = append(historyPeriods, BudgetHistoryPeriod{
			Label:      p.Label,
			Start:      p.Start.Unix(),
			End:        p.End.Unix(),
			IsComplete: p.IsComplete,
			Budgeted:   budgeted,
			Summary:    summary,
		})

		for _, item := range budgeted {
			ch, ok := categoryHistory[item.Category]
			if !ok {
				ch = &BudgetCategoryHistory{
					Category: item.Category,
					Amount:   item.Amount,
				}
				categoryHistory[item.Category] = ch
			}
			ch.ByPeriod = append(ch.ByPeriod, BudgetCategoryPeriodValue{
				Label:   p.Label,
				Spent:   item.Spent,
				Percent: item.Percent,
			})
			if p.IsComplete {
				if item.Percent >= 100 {
					ch.PeriodsOver++
				} else {
					ch.PeriodsUnder++
				}
			}
		}
	}

	var categories []BudgetCategoryHistory
	for _, ch := range categoryHistory {
		if len(ch.ByPeriod) > 0 {
			total := 0.0
			for _, bp := range ch.ByPeriod {
				total += bp.Spent
			}
			ch.AvgSpent = math.Round((total/float64(len(ch.ByPeriod)))*100) / 100
		}
		categories = append(categories, *ch)
	}
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].PeriodsOver != categories[j].PeriodsOver {
			return categories[i].PeriodsOver > categories[j].PeriodsOver
		}
		return categories[i].Category < categories[j].Category
	})

	return BudgetHistoryResponse{
		Periods:    historyPeriods,
		Categories: categories,
		Proposals:  buildBudgetProposals(categories),
	}, nil
}

// buildBudgetStatus joins budget limits with spending for a period.
func buildBudgetStatus(budgets []models.Budget, spending map[string]float64) ([]BudgetedCategory, []UnbudgetedCategory, BudgetPeriodSummary) {
	budgetMap := make(map[string]models.Budget)
	for _, b := range budgets {
		budgetMap[strings.ToLower(b.Category)] = b
	}

	var budgeted []BudgetedCategory
	var unbudgeted []UnbudgetedCategory
	matchedBudgets := make(map[string]bool)

	for cat, spent := range spending {
		key := strings.ToLower(cat)
		if b, ok := budgetMap[key]; ok {
			matchedBudgets[key] = true
			budgeted = append(budgeted, makeBudgetedCategory(b.Category, b.Amount, spent))
		} else {
			unbudgeted = append(unbudgeted, UnbudgetedCategory{
				Category: cat,
				Spent:    math.Round(spent*100) / 100,
			})
		}
	}

	for _, b := range budgets {
		key := strings.ToLower(b.Category)
		if !matchedBudgets[key] {
			budgeted = append(budgeted, makeBudgetedCategory(b.Category, b.Amount, 0))
		}
	}

	summary := summarizeBudgeted(budgeted)
	return budgeted, unbudgeted, summary
}

func makeBudgetedCategory(category string, amount, spent float64) BudgetedCategory {
	remaining := amount - spent
	percent := 0.0
	if amount > 0 {
		percent = (spent / amount) * 100
	}
	return BudgetedCategory{
		Category:  category,
		Amount:    amount,
		Spent:     math.Round(spent*100) / 100,
		Remaining: math.Round(remaining*100) / 100,
		Percent:   math.Round(percent*100) / 100,
	}
}

func summarizeBudgeted(budgeted []BudgetedCategory) BudgetPeriodSummary {
	var summary BudgetPeriodSummary
	for _, item := range budgeted {
		summary.TotalBudget += item.Amount
		summary.TotalSpent += item.Spent
		if item.Percent >= 100 {
			summary.OverCount++
		}
	}
	summary.TotalRemaining = math.Round((summary.TotalBudget-summary.TotalSpent)*100) / 100
	summary.TotalSpent = math.Round(summary.TotalSpent*100) / 100
	summary.TotalBudget = math.Round(summary.TotalBudget*100) / 100
	if summary.TotalBudget > 0 {
		summary.TotalPercent = math.Round((summary.TotalSpent/summary.TotalBudget)*10000) / 100
	}
	return summary
}

func buildBudgetProposals(categories []BudgetCategoryHistory) []BudgetImprovementProposal {
	var proposals []BudgetImprovementProposal
	completePeriods := func(ch BudgetCategoryHistory) int {
		return ch.PeriodsOver + ch.PeriodsUnder
	}

	for _, ch := range categories {
		periodCount := completePeriods(ch)
		if periodCount == 0 {
			continue
		}

		// Consistently over budget: suggest increase to rounded average.
		if ch.PeriodsOver >= 2 && ch.AvgSpent > ch.Amount {
			suggested := math.Ceil(ch.AvgSpent/10) * 10
			if suggested <= ch.Amount {
				suggested = math.Ceil(ch.AvgSpent/10)*10 + 10
			}
			proposals = append(proposals, BudgetImprovementProposal{
				Category:        ch.Category,
				CurrentAmount:   ch.Amount,
				SuggestedAmount: suggested,
				Type:            "increase",
				Reason: fmt.Sprintf(
					"Over budget in %d of %d complete periods (avg $%.0f vs $%.0f limit)",
					ch.PeriodsOver, periodCount, ch.AvgSpent, ch.Amount,
				),
			})
			continue
		}

		// Consistently well under budget: suggest decrease.
		if ch.PeriodsUnder >= 2 && ch.AvgSpent < ch.Amount*0.5 {
			suggested := math.Ceil(ch.AvgSpent/10) * 10
			if suggested < 10 {
				suggested = 10
			}
			if suggested >= ch.Amount {
				continue
			}
			proposals = append(proposals, BudgetImprovementProposal{
				Category:        ch.Category,
				CurrentAmount:   ch.Amount,
				SuggestedAmount: suggested,
				Type:            "decrease",
				Reason: fmt.Sprintf(
					"Under 50%% of budget in all %d complete periods (avg $%.0f vs $%.0f limit)",
					periodCount, ch.AvgSpent, ch.Amount,
				),
			})
			continue
		}

		// Watch list: over in latest complete period but not consistent enough to auto-increase.
		if ch.PeriodsOver == 1 && periodCount >= 2 {
			proposals = append(proposals, BudgetImprovementProposal{
				Category:        ch.Category,
				CurrentAmount:   ch.Amount,
				SuggestedAmount: ch.Amount,
				Type:            "watch",
				Reason: fmt.Sprintf(
					"Over budget once in %d complete periods — monitor before changing",
					periodCount,
				),
			})
		}
	}

	sort.Slice(proposals, func(i, j int) bool {
		order := map[string]int{"increase": 0, "watch": 1, "decrease": 2}
		if order[proposals[i].Type] != order[proposals[j].Type] {
			return order[proposals[i].Type] < order[proposals[j].Type]
		}
		return proposals[i].Category < proposals[j].Category
	})

	return proposals
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
