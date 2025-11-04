---
id: task-11
title: Remove cache mechanism and always send notifications
status: Done
assignee:
  - '@claude'
created_date: '2025-11-03 23:59'
updated_date: '2025-11-04 00:06'
labels:
  - refactoring
  - simplification
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The current BadgerDB cache implementation causes more issues than it solves. Simplify the flow by always fetching transactions and sending notifications without checking for balance updates or enforcing cooldown periods.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Remove BadgerDB dependency and all cache-related code from db.go
- [x] #2 Remove cache directory creation and management logic
- [x] #3 Remove balance update detection logic
- [x] #4 Remove 2-day notification cooldown enforcement
- [x] #5 Remove --disable-cache flag (no longer needed)
- [x] #6 Update documentation to reflect cache removal
- [x] #7 Test that notifications are sent on every run
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Analyze codebase to understand all cache usage points
2. Remove DisableCache flag and related code from main.go
3. Remove early return logic for hasUpdatedAccounts
4. Remove 2-day cooldown check (GetLastMessageTime logic)
5. Delete db.go file entirely
6. Remove BadgerDB dependencies from go.mod using go mod tidy
7. Update CLAUDE.md documentation to remove cache references
8. Build and test the application
9. Verify notifications are sent on every run
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Successfully removed BadgerDB cache mechanism from finance_tracker:

- Removed all cache-related code from main.go including DisableCache flag, hasUpdatedAccounts tracking, and 2-day cooldown checks
- Deleted db.go file entirely (122 lines removed)
- Removed BadgerDB and xdg dependencies from go.mod using go mod tidy
- Updated CLAUDE.md documentation to remove all cache references
- Application now sends notifications on every run (when transactions exist)
- Build successful with no errors
- All flags verified: --disable-cache and --force removed from help output

The notification flow is now simplified:
1. Fetch transactions from SimpleFin
2. Filter and analyze with AI
3. Send notifications immediately

No more cache state, balance tracking, or cooldown periods.
<!-- SECTION:NOTES:END -->
