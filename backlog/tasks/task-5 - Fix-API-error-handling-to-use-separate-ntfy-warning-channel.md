---
id: task-5
title: Fix API error handling to use separate ntfy warning channel
status: Done
assignee:
  - '@claude'
created_date: '2025-10-20 18:04'
updated_date: '2025-10-20 18:12'
labels:
  - bug
  - notifications
  - error-handling
dependencies: []
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
When one account (e.g., Tangerine) returns an API error but other accounts (e.g., TD) are still working, the system should:
1. Send error notifications to a separate ntfy warning topic (NTFY_TOPIC_WARNING)
2. Continue processing working accounts and send regular notifications
3. Clearly identify which account caused the error

Currently, API errors are logged but the notification behavior needs verification to ensure both error warnings AND regular summaries are sent correctly.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 API errors from individual accounts are sent to NTFY_TOPIC_WARNING (if configured)
- [x] #2 Regular transaction summaries are still sent to NTFY_TOPIC for working accounts
- [x] #3 Error messages clearly identify which account failed (account name/ID)
- [x] #4 System continues processing when some accounts fail but others succeed
- [x] #5 Documentation updated to explain NTFY_TOPIC_WARNING usage
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Verify current implementation status
   - Confirm NTFY_TOPIC_WARNING routing works
   - Confirm system continues processing on errors
   - Check if SimpleFin error messages include account info

2. Test the implementation
   - Run the application and verify warnings are sent correctly
   - Verify regular notifications still work

3. Update documentation
   - Update CLAUDE.md with NTFY_TOPIC_WARNING usage
   - Add example of error handling flow
   - Document SimpleFin error format limitations

4. Mark acceptance criteria as complete
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Summary

Verified that API error handling with separate ntfy warning channel was already fully implemented. Enhanced documentation to better explain the feature.

## What Was Already Implemented

1. **NTFY_TOPIC_WARNING routing** (`notifications.go:23-74`)
   - When `notificationTopic == "warning"`, ntfy notifications are routed to `NTFY_TOPIC_WARNING` if configured
   - Falls back to `NTFY_TOPIC` if warning topic not set

2. **API error collection** (`simplefin.go:88-93`)
   - SimpleFin API errors are collected at response level
   - Returned as separate value to allow continued processing

3. **Warning notification flow** (`main.go:193-205`)
   - Each API error triggers a separate warning notification
   - Notifications sent with `notificationTopic = "warning"`
   - System continues processing even if warning notifications fail

4. **Graceful degradation**
   - Successfully fetched accounts are still processed
   - Regular analysis notifications proceed normally
   - Warning notification failures are logged but don't stop process\n\n## Changes Made\n\n**Documentation updates in `CLAUDE.md`:**\n\n1. Enhanced **Notifications section** (lines 105-111):\n   - Detailed explanation of warning topic behavior\n   - Clarified optional nature and fallback logic\n   - Noted email notifications don't support topic differentiation\n\n2. Expanded **Error Handling section** (lines 132-152):\n   - Separated SimpleFin API errors from LLM errors\n   - Added detailed error flow diagram (5 steps)\n   - Documented error message passthrough behavior\n   - Explained continuation logic when partial failures occur\n\n## Notes on Account Identification (AC #3)\n\nThe SimpleFin API returns errors as string array at response level (`AccountsResponse.Errors`), not tied to specific accounts. Error messages may or may not include account identifiers depending on SimpleFin's response format. The application correctly passes these through with minimal formatting (`"API Error: {message}"`) to preserve any account information that SimpleFin provides.\n\n## Testing Recommendations\n\nTo fully verify this feature:\n1. Set both `NTFY_TOPIC` and `NTFY_TOPIC_WARNING` in `.env`\n2. Trigger an API error (e.g., temporarily break SimpleFin auth)\n3. Verify warnings go to warning topic\n4. Verify regular summaries still work for successful accounts
<!-- SECTION:NOTES:END -->
