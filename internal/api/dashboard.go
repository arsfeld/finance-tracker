package api

import (
	"math"
	"net/http"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/config"
	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

type DashboardHandler struct {
	cfg           *config.Config
	txnStore      *store.TransactionStore
	analysisStore *store.AnalysisStore
}

func NewDashboardHandler(cfg *config.Config, ts *store.TransactionStore, as *store.AnalysisStore) *DashboardHandler {
	return &DashboardHandler{cfg: cfg, txnStore: ts, analysisStore: as}
}

type DashboardResponse struct {
	Period             PeriodInfo             `json:"period"`
	Summary            SpendingSummary        `json:"summary"`
	CategoryTotals     map[string]float64     `json:"category_totals"`
	RecentTransactions []models.DBTransaction `json:"recent_transactions"`
	LatestAnalysis     *models.Analysis       `json:"latest_analysis"`
	TrendData          []TrendPoint           `json:"trend_data"`
}

type PeriodInfo struct {
	Label string `json:"label"`
	Start int64  `json:"start"`
	End   int64  `json:"end"`
}

type SpendingSummary struct {
	TotalSpending    float64 `json:"total_spending"`
	DailyAverage     float64 `json:"daily_average"`
	TransactionCount int     `json:"transaction_count"`
	DaysInPeriod     int     `json:"days_in_period"`
}

type TrendPoint struct {
	Label string  `json:"label"`
	Total float64 `json:"total"`
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	billingDay := h.cfg.BillingDay

	start, end, err := billing.CalculateDateRange(models.DateRangeTypeCurrentMonth, nil, nil, billingDay)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DATE_ERROR", err.Error())
		return
	}

	startUnix := start.Unix()
	endUnix := end.Unix()

	txns, err := h.txnStore.GetForPeriod(ctx, startUnix, endUnix)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	var totalSpending float64
	var expenseCount int
	for _, t := range txns {
		if t.Amount < 0 {
			totalSpending += math.Abs(t.Amount)
			expenseCount++
		}
	}

	daysInPeriod := int(end.Sub(start).Hours()/24) + 1
	dailyAvg := 0.0
	if daysInPeriod > 0 {
		dailyAvg = totalSpending / float64(daysInPeriod)
	}

	catTotals, _ := h.txnStore.CountByCategory(ctx, startUnix, endUnix)

	recentFilter := store.TransactionFilter{
		StartDate: startUnix,
		EndDate:   endUnix,
		Page:      1,
		Limit:     10,
		SortBy:    "posted",
		SortDir:   "desc",
	}
	recent, _, _ := h.txnStore.List(ctx, recentFilter)

	latest, _ := h.analysisStore.GetLatest(ctx)

	trendStart := start.AddDate(0, -4, 0)
	periods := billing.CalculateBillingPeriods(trendStart, end, billingDay)
	var trendData []TrendPoint
	for _, p := range periods {
		pTotals, _ := h.txnStore.CountByCategory(ctx, p.Start.Unix(), p.End.Unix())
		var total float64
		for _, v := range pTotals {
			total += v
		}
		trendData = append(trendData, TrendPoint{Label: p.Label, Total: total})
	}

	resp := DashboardResponse{
		Period: PeriodInfo{
			Label: start.Format("Jan 2") + " - " + end.Format("Jan 2, 2006"),
			Start: startUnix,
			End:   endUnix,
		},
		Summary: SpendingSummary{
			TotalSpending:    totalSpending,
			DailyAverage:     dailyAvg,
			TransactionCount: expenseCount,
			DaysInPeriod:     daysInPeriod,
		},
		CategoryTotals:     catTotals,
		RecentTransactions: recent,
		LatestAnalysis:     latest,
		TrendData:          trendData,
	}

	WriteData(w, resp)
}
