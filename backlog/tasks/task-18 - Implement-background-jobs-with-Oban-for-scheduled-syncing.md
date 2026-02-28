---
id: task-18
title: Implement background jobs with Oban for scheduled syncing
status: To Do
assignee: []
created_date: '2025-11-04 01:39'
labels:
  - phoenix
  - oban
  - background-jobs
  - scheduling
dependencies:
  - task-13
  - task-14
  - task-15
  - task-16
  - task-17
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add Oban dependency and implement background job workers for automated transaction fetching and analysis generation. Configure cron-based scheduling for daily syncs and provide manual trigger capability from the UI.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Oban dependency added to mix.exs
- [ ] #2 Oban configured in application supervisor with database-backed queues
- [ ] #3 FetchTransactions worker created for SimpleFin sync jobs
- [ ] #4 GenerateAnalysis worker created for LLM analysis jobs
- [ ] #5 Workers properly scope data to user_id (multi-user support)
- [ ] #6 Cron plugin configured for daily automatic transaction sync (configurable time)
- [ ] #7 Manual job triggers work from LiveView UI buttons (Sync Now, Analyze Now)
- [ ] #8 Job failures are logged and retried with exponential backoff
- [ ] #9 Oban dashboard accessible for monitoring jobs (admin only)
- [ ] #10 Workers send notifications after successful completion
<!-- AC:END -->
