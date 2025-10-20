---
id: task-10
title: Include transaction dates in Top 10 Largest Expenses
status: In Progress
assignee:
  - '@claude'
created_date: '2025-10-20 18:47'
updated_date: '2025-10-20 18:48'
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
- [ ] #1 Add transaction date field to largest expenses data structure
- [ ] #2 Format dates in a readable format (e.g., 'Jan 15, 2025' or '2025-01-15')
- [ ] #3 Ensure dates are included in the prompt/context sent to the LLM
- [ ] #4 Update any transaction formatting to show date alongside merchant and amount
- [ ] #5 Verify dates display correctly in both email and ntfy notifications
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Examine current date formatting in formatTopExpenses function (llm.go:234-249)
2. Update date format from "2006-01-02" to more readable "Jan 2" format
3. Review and update prompt instructions to explicitly tell LLM to include dates in Top 10 section
4. Build and test the changes with sample run
5. Verify dates appear correctly in notifications
<!-- SECTION:PLAN:END -->
