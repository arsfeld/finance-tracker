---
id: task-25
title: Add tests for LiveViews and Financial context
status: To Do
assignee: []
created_date: '2025-11-04 04:22'
labels:
  - phoenix
  - liveview
  - testing
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create comprehensive test coverage for all LiveView modules and the Financial context module. Currently no tests exist for the new UI components. Should test rendering, user interactions, and data queries.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 DashboardLive tests: mount, rendering, action button clicks
- [ ] #2 SettingsLive tests: form validation, submission, error handling
- [ ] #3 AccountsLive tests: filtering, rendering, account display
- [ ] #4 AnalysisHistoryLive tests: listing, empty state
- [ ] #5 AnalysisShowLive tests: detail display, not found handling
- [ ] #6 Financial context tests: all query functions return correct data
- [ ] #7 Financial context tests: credit card filtering works correctly
- [ ] #8 Settings context tests: CRUD operations work correctly
- [ ] #9 All tests pass with mix test
<!-- AC:END -->
