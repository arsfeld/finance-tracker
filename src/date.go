package main

import (
	"fmt"
	"time"
)

// calculateDateRange calculates the start and end dates based on the given date range type
// and optional custom start and end dates.
func calculateDateRange(
	dateRangeType DateRangeType,
	startDate *time.Time,
	endDate *time.Time,
) (time.Time, time.Time, error) {
	today := time.Now().UTC()

	// If current day is 5 or less and range type is current month, switch to last month
	if dateRangeType == DateRangeTypeCurrentMonth && today.Day() <= 5 {
		dateRangeType = DateRangeTypeLastMonth
	}

	switch dateRangeType {
	case DateRangeTypeCurrentMonth:
		start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, today, nil

	case DateRangeTypeLastMonth:
		var start time.Time
		if today.Month() == time.January {
			start = time.Date(today.Year()-1, time.December, 1, 0, 0, 0, 0, time.UTC)
		} else {
			start = time.Date(today.Year(), today.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		}
		end := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
		return start, end, nil

	case DateRangeTypeLast3Months:
		var start time.Time
		if today.Month() <= time.March {
			start = time.Date(today.Year()-1, today.Month()+9, 1, 0, 0, 0, 0, time.UTC)
		} else {
			start = time.Date(today.Year(), today.Month()-3, 1, 0, 0, 0, 0, time.UTC)
		}
		return start, today, nil

	case DateRangeTypeCurrentYear:
		start := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		return start, today, nil

	case DateRangeTypeLastYear:
		start := time.Date(today.Year()-1, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(today.Year()-1, time.December, 31, 23, 59, 59, 999999999, time.UTC)
		return start, end, nil

	case DateRangeTypeCustom:
		if startDate == nil || endDate == nil {
			return time.Time{}, time.Time{}, fmt.Errorf("custom date range requires both start_date and end_date")
		}
		return *startDate, *endDate, nil

	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date range type: %s", dateRangeType)
	}
}

// validateBillingPeriod ensures that the provided billing period is valid:
// - Start date must be before end date
// - Billing period can't exceed 90 days
func validateBillingPeriod(start, end time.Time) error {
	if start.After(end) {
		return fmt.Errorf("start date cannot be after end date")
	}

	if end.Sub(start).Hours() > 90*24 {
		return fmt.Errorf("billing period cannot exceed 90 days")
	}

	return nil
}
