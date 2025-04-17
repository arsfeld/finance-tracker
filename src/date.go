package main

import (
	"fmt"
	"time"
)

// calculateDateRange calculates the start and end dates based on the given date range type,
// optional custom start/end dates, and the billing cycle day.
func calculateDateRange(
	dateRangeType DateRangeType,
	startDate *time.Time,
	endDate *time.Time,
	billingDay int,
) (time.Time, time.Time, error) {
	today := time.Now().UTC()
	currentYear, currentMonth, _ := today.Date()

	// Adjust billingDay to be within valid range (1-28)
	if billingDay < 1 {
		billingDay = 1
	} else if billingDay > 28 {
		billingDay = 28
	}

	// Determine the start of the current billing cycle
	var currentCycleStart time.Time
	if today.Day() >= billingDay {
		// Current cycle started this month
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC)
	} else {
		// Current cycle started last month
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	}

	// If today is within 5 days after the *previous* billing day, switch to last month's cycle
	// Example: Billing day 15th. Today is April 18th. Previous billing day was March 15th.
	// If today was April 19th (<= 15 + 5 -1), we'd still show the March 15 - April 14 cycle.
	// (We use previousBillingDay because currentCycleStart might already be last month)
	previousBillingDayDate := currentCycleStart
	if today.Day() < billingDay {
		// If current cycle started last month, the *previous* was the month before that
		previousBillingDayDate = previousBillingDayDate.AddDate(0, -1, 0)
	}
	if dateRangeType == DateRangeTypeCurrentMonth && today.Sub(previousBillingDayDate).Hours() <= 5*24 {
		dateRangeType = DateRangeTypeLastMonth
	}

	switch dateRangeType {
	case DateRangeTypeCurrentMonth:
		// Start is the beginning of the current cycle, end is today
		return currentCycleStart, today, nil

	case DateRangeTypeLastMonth:
		// End is the day before the current cycle started
		end := currentCycleStart.Add(-24 * time.Hour)
		// Start is one month before the end date, at the billing day
		start := time.Date(end.Year(), end.Month(), billingDay, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		// Handle edge case where end month doesn't have billingDay (e.g. billingDay=31)
		/*
		if start.Day() != billingDay {
			// Go to the first of the *next* month, then subtract one day to get the last day of the correct month
			start = time.Date(end.Year(), end.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
			// Now set the start day to the billing day for the correct month/year
			start = time.Date(start.Year(), start.Month(), billingDay, 0, 0, 0, 0, time.UTC)
		}
		*/

		return start, end, nil

	case DateRangeTypeLast3Months:
		// End is today
		end := today
		// Start is 3 months prior to the current cycle start, on the billing day
		start := currentCycleStart.AddDate(0, -3, 0)
		return start, end, nil

	case DateRangeTypeCurrentYear:
		// Start is Jan 1st of the current year
		start := time.Date(currentYear, time.January, 1, 0, 0, 0, 0, time.UTC)
		// End is today
		return start, today, nil

	case DateRangeTypeLastYear:
		// Start is Jan 1st of last year
		start := time.Date(currentYear-1, time.January, 1, 0, 0, 0, 0, time.UTC)
		// End is Dec 31st of last year
		end := time.Date(currentYear-1, time.December, 31, 23, 59, 59, 999999999, time.UTC)
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
// - Billing period can't exceed 90 days (SimpleFIN limit?)
func validateBillingPeriod(start, end time.Time) error {
	if start.After(end) {
		return fmt.Errorf("start date cannot be after end date")
	}

	if end.Sub(start).Hours() > 90*24 {
		// Note: SimpleFIN seems to have a 90-day limit for fetching transactions
		return fmt.Errorf("billing period cannot exceed 90 days")
	}

	return nil
}
