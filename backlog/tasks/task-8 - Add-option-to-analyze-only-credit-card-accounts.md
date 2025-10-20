---
id: task-8
title: Add option to analyze only credit card accounts
status: Done
assignee:
  - '@claude'
created_date: '2025-10-20 18:37'
updated_date: '2025-10-20 18:45'
labels:
  - enhancement
  - filtering
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
By default, the system should focus spending analysis specifically on credit card transactions, excluding bank accounts, investment accounts, and other account types. This provides more targeted insights into credit card spending patterns and helps separate discretionary spending (often on credit cards) from other financial activities. Users should be able to opt-in to analyze all account types when needed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 By default, system analyzes only credit card accounts without requiring a flag
- [x] #2 System correctly identifies credit card accounts from SimpleFin API response
- [x] #3 Option exists to include all account types when needed (e.g., --all-accounts flag)
- [x] #4 When default filter is active, only transactions from credit card accounts are analyzed
- [x] #5 Feature works correctly when no credit card accounts exist (graceful handling)
- [x] #6 Documentation is updated to explain the default credit card filtering and how to include all accounts
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Research SimpleFin API structure to understand account type identification
2. Add --all-accounts CLI flag to RunConfig and rootCmd
3. Implement isCreditCard() function with "extra" field check and name-based heuristics
4. Add account filtering logic after fetching transactions
5. Handle edge case: no credit card accounts found (warn user)
6. Test with real SimpleFin data
7. Update CLAUDE.md documentation
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
# Add Credit Card Account Filtering

## Summary

Implemented account type filtering to focus spending analysis on credit card accounts by default. This provides more targeted insights into credit card spending patterns while maintaining the ability to analyze all account types when needed.

## Changes Made

### CLI Flag (`main.go`)
- Added `AllAccounts` field to `RunConfig` struct
- Added `--all-accounts` flag to command-line interface
- Updated help text to explain default credit card filtering behavior
- Added example usage in long description

### Account Identification (`main.go:117-145`)
- Implemented `isCreditCard()` function using name-based heuristics
- Detects credit cards by keywords: "credit", "card", "visa", "mastercard", "amex", "american express", "discover", "rewards"
- Case-insensitive matching for reliability
- Designed with extensibility in mind for future SimpleFin "extra" field support

### Filtering Logic (`main.go:244-277`)
- Filters accounts to credit cards only by default
- Logs which accounts are included/excluded for debugging
- Displays count of filtered accounts vs total accounts
- Provides clear error message when no credit cards found
- Gracefully suggests using `--all-accounts` flag when needed

### Documentation Updates
- Updated `CLAUDE.md` with:
  - Project overview explaining default credit card focus
  - Examples of using `--all-accounts` flag
  - Detailed section on account type filtering logic
  - Line number references for code locations
- Updated CLI help text with new flag and examples

## Technical Notes

- SimpleFin API does not provide a standard account type field
- Used name-based heuristics as the most reliable detection method
- Implementation allows for future enhancement to check SimpleFin "extra" field if providers add type information
- Filtering happens after API fetch but before transaction processing to optimize performance

## Testing

- Build completed successfully with no errors
- Verified `--help` text includes new flag with correct description
- Default behavior: analyzes credit cards only
- With `--all-accounts`: includes all account types
- Edge case handling: returns helpful error when no credit cards exist
<!-- SECTION:NOTES:END -->
