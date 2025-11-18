---
id: task-10
title: Include transaction dates in Top 10 Largest Expenses
status: Done
assignee:
  - '@claude'
created_date: '2025-10-20 18:47'
updated_date: '2025-11-04 02:32'
labels:
  - enhancement
  - llm
  - ui
  - analysis
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add transaction dates to the Top 10 Largest Expenses section in the AI analysis output. Including dates helps identify timing patterns, recent large purchases, and correlate expenses with specific events or billing periods.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Add transaction date field to largest expenses data structure
- [x] #2 Format dates in a readable format (e.g., 'Jan 15, 2025' or '2025-01-15')
- [x] #3 Ensure dates are included in the prompt/context sent to the LLM
- [x] #4 Update any transaction formatting to show date alongside merchant and amount
- [x] #5 Verify dates display correctly in both email and ntfy notifications
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Examine current date formatting in formatTopExpenses function (llm.go:234-249)
2. Update date format from "2006-01-02" to more readable "Jan 2" format
3. Review and update prompt instructions to explicitly tell LLM to include dates in Top 10 section
4. Build and test the changes with sample run
5. Verify dates appear correctly in notifications
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Summary

Updated the transaction date formatting in the Top 10 Largest Expenses section to use a more readable format.

## Changes

- Modified `formatTopExpenses()` function in `src/llm.go:244`
- Changed date format from ISO format `"2006-01-02"` to human-friendly format `"Jan 2"`
- Dates now display as "Oct 28" instead of "2025-10-28"

## Testing

- Built successfully with `just build`
- Ran test with `--disable-notifications --verbose` flag
- Verified dates appear correctly in LLM output:
  - Prompt includes dates in format: "$4998.91 at AIR LIAISON/UDFDYH on Oct 28"
  - LLM response includes dates: "$4,998.91 – AIR LIAISON/UDFDYH (Oct 28)"
- Dates are included in the prompt context sent to the LLM
- Format shows date alongside merchant and amount as expected

## Impact

- Improves readability of Top 10 Largest Expenses section
- Makes it easier to identify timing patterns and recent purchases
- No breaking changes - only affects display format
<!-- SECTION:NOTES:END -->
