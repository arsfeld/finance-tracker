package api

import (
	"net/http"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/config"
	"finance_tracker/internal/models"
)

type BillingPeriodsHandler struct {
	cfg *config.Config
}

func NewBillingPeriodsHandler(cfg *config.Config) *BillingPeriodsHandler {
	return &BillingPeriodsHandler{cfg: cfg}
}

type BillingPeriodResponse struct {
	Label string `json:"label"`
	Start int64  `json:"start"`
	End   int64  `json:"end"`
}

func (h *BillingPeriodsHandler) List(w http.ResponseWriter, r *http.Request) {
	// Current period.
	start, end, err := billing.CalculateDateRange(models.DateRangeTypeCurrentMonth, nil, nil, h.cfg.BillingDay)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DATE_ERROR", err.Error())
		return
	}

	// Generate last 6 billing periods.
	trendStart := start.AddDate(0, -5, 0)
	periods := billing.CalculateBillingPeriods(trendStart, end, h.cfg.BillingDay)

	var result []BillingPeriodResponse
	for _, p := range periods {
		result = append(result, BillingPeriodResponse{
			Label: p.Label,
			Start: p.Start.Unix(),
			End:   p.End.Unix(),
		})
	}

	WriteData(w, result)
}
