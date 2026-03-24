package api

import (
	"net/http"
	"strconv"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/config"
	"finance_tracker/internal/store"
)

type AnalyticsHandler struct {
	cfg      *config.Config
	txnStore *store.TransactionStore
}

func NewAnalyticsHandler(cfg *config.Config, ts *store.TransactionStore) *AnalyticsHandler {
	return &AnalyticsHandler{cfg: cfg, txnStore: ts}
}

// CategoryTrendPeriod represents category totals for a single billing period.
type CategoryTrendPeriod struct {
	Label  string             `json:"label"`
	Start  int64              `json:"start"`
	End    int64              `json:"end"`
	Totals map[string]float64 `json:"totals"`
}

// CategoryTrendResponse is the response for the category trend endpoint.
type CategoryTrendResponse struct {
	Categories []string              `json:"categories"`
	Periods    []CategoryTrendPeriod `json:"periods"`
}

// CategoryTrend returns per-category spending totals across billing periods.
// GET /api/analytics/category-trend?months=N (default 6)
func (h *AnalyticsHandler) CategoryTrend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	months, _ := strconv.Atoi(r.URL.Query().Get("months"))
	if months < 1 || months > 24 {
		months = 6
	}

	billingDay := h.cfg.BillingDay
	start, end, err := billing.CalculateDateRange("current_month", nil, nil, billingDay)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DATE_ERROR", err.Error())
		return
	}

	trendStart := start.AddDate(0, -(months - 1), 0)
	periods := billing.CalculateBillingPeriods(trendStart, end, billingDay)

	categorySet := make(map[string]bool)
	var result []CategoryTrendPeriod

	for _, p := range periods {
		totals, err := h.txnStore.CountByCategory(ctx, p.Start.Unix(), p.End.Unix())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		for cat := range totals {
			categorySet[cat] = true
		}
		result = append(result, CategoryTrendPeriod{
			Label:  p.Label,
			Start:  p.Start.Unix(),
			End:    p.End.Unix(),
			Totals: totals,
		})
	}

	var categories []string
	for cat := range categorySet {
		categories = append(categories, cat)
	}

	WriteData(w, CategoryTrendResponse{
		Categories: categories,
		Periods:    result,
	})
}

// DailyTotals returns per-day expense totals for a date range.
// GET /api/analytics/daily-totals?start=X&end=Y
func (h *AnalyticsHandler) DailyTotals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	start, _ := strconv.ParseInt(q.Get("start"), 10, 64)
	end, _ := strconv.ParseInt(q.Get("end"), 10, 64)

	if start <= 0 || end <= 0 {
		// Default to last 90 days
		billingDay := h.cfg.BillingDay
		s, e, err := billing.CalculateDateRange("current_month", nil, nil, billingDay)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DATE_ERROR", err.Error())
			return
		}
		end = e.Unix()
		start = s.AddDate(0, -3, 0).Unix()
	}

	days, err := h.txnStore.DailyTotals(ctx, start, end)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteData(w, map[string]interface{}{
		"days":  days,
		"start": start,
		"end":   end,
	})
}

// TopMerchants returns top N merchants by total spending.
// GET /api/analytics/top-merchants?start=X&end=Y&limit=N
func (h *AnalyticsHandler) TopMerchants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	start, _ := strconv.ParseInt(q.Get("start"), 10, 64)
	end, _ := strconv.ParseInt(q.Get("end"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 10
	}

	if start <= 0 || end <= 0 {
		billingDay := h.cfg.BillingDay
		s, e, err := billing.CalculateDateRange("current_month", nil, nil, billingDay)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DATE_ERROR", err.Error())
			return
		}
		start = s.Unix()
		end = e.Unix()
	}

	merchants, err := h.txnStore.TopMerchants(ctx, start, end, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteData(w, map[string]interface{}{
		"merchants": merchants,
		"start":     start,
		"end":       end,
	})
}
