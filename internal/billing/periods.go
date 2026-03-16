package billing

import (
	"fmt"
	"time"

	"finance_tracker/internal/models"
)

// CalculateDateRange computes the start and end dates for a billing period.
func CalculateDateRange(
	dateRangeType models.DateRangeType,
	startDate *time.Time,
	endDate *time.Time,
	billingDay int,
) (time.Time, time.Time, error) {
	today := time.Now().UTC()
	currentYear, currentMonth, _ := today.Date()

	if billingDay < 1 {
		billingDay = 1
	} else if billingDay > 28 {
		billingDay = 28
	}

	var currentCycleStart time.Time
	if today.Day() >= billingDay {
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC)
	} else {
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	}

	// Auto-rollback if within 5 days after previous billing day.
	previousBillingDayDate := currentCycleStart
	if today.Day() < billingDay {
		previousBillingDayDate = previousBillingDayDate.AddDate(0, -1, 0)
	}
	if dateRangeType == models.DateRangeTypeCurrentMonth && today.Sub(previousBillingDayDate).Hours() <= 5*24 {
		dateRangeType = models.DateRangeTypeLastMonth
	}

	switch dateRangeType {
	case models.DateRangeTypeCurrentMonth:
		return currentCycleStart, today, nil

	case models.DateRangeTypeLastMonth:
		end := currentCycleStart.Add(-24 * time.Hour)
		start := time.Date(end.Year(), end.Month(), billingDay, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		return start, end, nil

	case models.DateRangeTypeCurrentAndLastMonth:
		end := today
		start := currentCycleStart.AddDate(0, -4, 0)
		return start, end, nil

	case models.DateRangeTypeLast3Months:
		end := today
		start := currentCycleStart.AddDate(0, -3, 0)
		return start, end, nil

	case models.DateRangeTypeCurrentYear:
		start := time.Date(currentYear, time.January, 1, 0, 0, 0, 0, time.UTC)
		return start, today, nil

	case models.DateRangeTypeLastYear:
		start := time.Date(currentYear-1, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(currentYear-1, time.December, 31, 23, 59, 59, 999999999, time.UTC)
		return start, end, nil

	case models.DateRangeTypeCustom:
		if startDate == nil || endDate == nil {
			return time.Time{}, time.Time{}, fmt.Errorf("custom date range requires both start_date and end_date")
		}
		return *startDate, *endDate, nil

	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date range type: %s", dateRangeType)
	}
}

// CalculateBillingPeriods generates billing period boundaries between start and end.
func CalculateBillingPeriods(start, end time.Time, billingDay int) []models.BillingPeriod {
	if billingDay < 1 {
		billingDay = 1
	} else if billingDay > 28 {
		billingDay = 28
	}

	var periods []models.BillingPeriod
	now := time.Now().UTC()

	periodStart := time.Date(start.Year(), start.Month(), billingDay, 0, 0, 0, 0, time.UTC)
	if periodStart.After(start) {
		periodStart = periodStart.AddDate(0, -1, 0)
	}

	for periodStart.Before(end) {
		periodEnd := periodStart.AddDate(0, 1, 0).Add(-24 * time.Hour)
		if periodEnd.After(end) {
			periodEnd = end
		}

		isComplete := periodEnd.Before(now)
		periods = append(periods, models.BillingPeriod{
			Label:      periodStart.Format("Jan 2006"),
			Start:      periodStart,
			End:        periodEnd,
			IsComplete: isComplete,
			IsFocus:    false,
		})

		periodStart = periodStart.AddDate(0, 1, 0)
	}

	// Mark the last 3 periods as focus.
	for i := len(periods) - 1; i >= 0 && i >= len(periods)-3; i-- {
		periods[i].IsFocus = true
	}

	return periods
}

// ValidateBillingPeriod ensures start is before end.
func ValidateBillingPeriod(start, end time.Time) error {
	if start.After(end) {
		return fmt.Errorf("start date cannot be after end date")
	}
	return nil
}
