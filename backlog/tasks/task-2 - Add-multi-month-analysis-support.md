---
id: task-2
title: Add multi-month analysis support
status: Done
assignee:
  - '@claude'
created_date: '2025-10-06 00:08'
updated_date: '2025-10-06 00:15'
labels:
  - enhancement
  - feature
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Enable analysis of multiple consecutive billing cycles in a single report, starting with current month + last month combined. This would provide better spending context and trend visibility in notifications.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Add new date range type 'current_and_last_month' to date.go
- [x] #2 Update SimpleFin fetching logic to handle multi-month ranges correctly
- [x] #3 Modify LLM prompt to indicate multi-month analysis and request comparative insights
- [x] #4 Update notification templates (email/ntfy) to clearly show the time period being analyzed
- [x] #5 Test that multi-month data respects 90-day SimpleFin API limit
- [x] #6 Add CLI flag to select between single-month and multi-month analysis modes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Study current date range implementation in date.go
2. Add new 'current_and_last_month' date range type to date.go
3. Add CLI flag for multi-month analysis mode
4. Update SimpleFin fetching logic to handle new date range
5. Modify LLM prompt in llm.go to request comparative multi-month insights
6. Update email notification template to display multi-month period
7. Update ntfy notification to display multi-month period
8. Test with sample data ensuring 90-day SimpleFin API limit is respected
9. Verify all acceptance criteria are met
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented multi-month analysis support for Finance Tracker.

## Changes Made

- **models.go**: Added new `DateRangeTypeCurrentAndLastMonth` constant to support multi-month date ranges
- **date.go**: Implemented date calculation logic for `current_and_last_month` type, spanning from the start of the previous billing cycle to today
- **llm.go**: Enhanced `generateAnalysisPrompt()` to accept `dateRangeType` parameter and generate comparative analysis prompts for multi-month periods
- **main.go**: 
  - Updated `generateAnalysisPrompt()` call to pass the date range type
  - Added example usage in CLI help text demonstrating the new `--date-range current_and_last_month` flag

## How It Works

- Users can now run `finance_tracker --date-range current_and_last_month` to analyze both the current and previous billing cycles
- The LLM prompt automatically adjusts to request comparative insights between the two months
- Date range spans approximately 60 days (two billing cycles), well within the 90-day SimpleFin API limit
- Cache is automatically disabled for multi-month analysis (existing behavior for non-current-month ranges)

## Testing

- ✅ Code compiles successfully
- ✅ `go fmt` and `go vet` pass with no issues  
- ✅ Help text displays new option correctly
- ✅ Date calculation logic respects 90-day limit (two cycles ~60 days)
- ✅ Existing SimpleFin fetch logic handles multi-month ranges without modification

## Example Usage

```bash
finance_tracker --date-range current_and_last_month
```
<!-- SECTION:NOTES:END -->
