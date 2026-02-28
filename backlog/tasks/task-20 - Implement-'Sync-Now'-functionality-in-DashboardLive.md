---
id: task-20
title: Implement 'Sync Now' functionality in DashboardLive
status: To Do
assignee: []
created_date: '2025-11-04 04:21'
labels:
  - phoenix
  - liveview
  - oban
  - backend
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add backend functionality to trigger SimpleFin data sync from the Dashboard. Currently shows placeholder message. Should trigger background job via Oban to fetch latest transactions and update account balances.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Clicking 'Sync Now' triggers background job to fetch SimpleFin data
- [ ] #2 User receives immediate feedback that sync was triggered
- [ ] #3 Account balances and last sync time update after sync completes
- [ ] #4 Real-time PubSub broadcast updates the UI when sync completes
- [ ] #5 Error handling shows appropriate messages if sync fails
<!-- AC:END -->
