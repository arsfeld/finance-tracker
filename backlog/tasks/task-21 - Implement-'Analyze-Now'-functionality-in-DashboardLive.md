---
id: task-21
title: Implement 'Analyze Now' functionality in DashboardLive
status: To Do
assignee: []
created_date: '2025-11-04 04:22'
labels:
  - phoenix
  - liveview
  - backend
  - llm
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add backend functionality to trigger financial analysis from the Dashboard. Currently shows placeholder message. Should run the analysis flow (fetch transactions, send to LLM, store results) and update the UI with new analysis.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Clicking 'Analyze Now' triggers analysis workflow
- [ ] #2 User receives immediate feedback that analysis was triggered
- [ ] #3 Dashboard updates with new analysis when complete
- [ ] #4 Real-time PubSub broadcast updates the UI when analysis completes
- [ ] #5 Analysis respects user's settings (account filtering, date range, model)
- [ ] #6 Error handling shows appropriate messages if analysis fails
<!-- AC:END -->
