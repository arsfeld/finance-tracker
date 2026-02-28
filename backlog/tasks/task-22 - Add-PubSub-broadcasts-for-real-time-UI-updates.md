---
id: task-22
title: Add PubSub broadcasts for real-time UI updates
status: To Do
assignee: []
created_date: '2025-11-04 04:22'
labels:
  - phoenix
  - pubsub
  - backend
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement PubSub broadcasts in backend functions (SimpleFin sync, analysis creation) to trigger real-time updates across all connected LiveView sessions. Currently LiveViews subscribe but no broadcasts are sent.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SimpleFin sync broadcasts to 'financial_updates' topic on completion
- [ ] #2 Analysis creation broadcasts to 'financial_updates' topic on completion
- [ ] #3 Broadcasts include relevant data (updated accounts, new analysis)
- [ ] #4 All LiveViews receive and handle broadcast messages correctly
- [ ] #5 Multiple users see updates in real-time when data changes
<!-- AC:END -->
