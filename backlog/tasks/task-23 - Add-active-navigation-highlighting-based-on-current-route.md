---
id: task-23
title: Add active navigation highlighting based on current route
status: To Do
assignee: []
created_date: '2025-11-04 04:22'
labels:
  - phoenix
  - liveview
  - ui
  - frontend
dependencies: []
priority: low
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Update navigation menu to highlight the current page the user is on. Currently all nav links use the same styling. Should show active state with proper border/color styling to improve UX.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Dashboard nav link is highlighted when on /dashboard
- [ ] #2 Accounts nav link is highlighted when on /accounts
- [ ] #3 History nav link is highlighted when on /analysis or /analysis/:id
- [ ] #4 Settings nav link is highlighted when on /settings
- [ ] #5 Active styling uses indigo-500 border and gray-900 text (matching existing design)
- [ ] #6 Inactive links use gray-500 text with hover states
<!-- AC:END -->
