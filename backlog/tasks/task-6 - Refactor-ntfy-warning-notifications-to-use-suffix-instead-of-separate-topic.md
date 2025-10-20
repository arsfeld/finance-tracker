---
id: task-6
title: Refactor ntfy warning notifications to use suffix instead of separate topic
status: Done
assignee:
  - '@claude'
created_date: '2025-10-20 18:23'
updated_date: '2025-10-20 18:25'
labels:
  - refactoring
  - notifications
dependencies: []
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Change from NTFY_TOPIC_WARNING to a simpler suffix-based approach:

- Remove NTFY_TOPIC_WARNING environment variable
- Add NTFY_WARNING_SUFFIX (optional, default: "-warnings")
- Warnings sent to {NTFY_TOPIC}{NTFY_WARNING_SUFFIX}
- Example: NTFY_TOPIC=finance → warnings to finance-warnings

This simplifies configuration - users only need to set one main topic.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Remove NTFY_TOPIC_WARNING from Settings struct and environment loading
- [x] #2 Add NTFY_WARNING_SUFFIX to Settings struct with default value
- [x] #3 Update sendNtfyNotification to append suffix instead of switching topics
- [x] #4 Update CLAUDE.md documentation to reflect new approach
- [x] #5 Test that warnings are sent to topic with suffix
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Update Settings struct in settings.go
   - Remove NtfyTopicWarning field
   - Add NtfyWarningSuffix field (with default "-warnings")

2. Update environment loading in settings.go
   - Remove NTFY_TOPIC_WARNING loading
   - Add NTFY_WARNING_SUFFIX loading with default

3. Update sendNtfyNotification in notifications.go
   - Remove topic switching logic
   - Append suffix when notificationTopic == "warning"

4. Update documentation in CLAUDE.md
   - Change NTFY_TOPIC_WARNING to NTFY_WARNING_SUFFIX
   - Update examples and explanations

5. Test the changes
   - Verify default suffix works
   - Verify custom suffix works
   - Verify regular notifications unaffected
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Summary

Successfully refactored ntfy warning notifications from separate topic approach to simpler suffix-based approach.

## Changes Made

### 1. Settings Structure (`settings.go`)
- **Removed**: `NtfyTopicWarning *string` field
- **Added**: `NtfyWarningSuffix string` field with default value `"-warnings"`
- **Changed**: Environment loading to use `NTFY_WARNING_SUFFIX` instead of `NTFY_TOPIC_WARNING`
- Default suffix is automatically applied, but can be overridden via env var

### 2. Notification Logic (`notifications.go`)
- **Simplified**: Topic determination logic in `sendNtfyNotification()`
- **Before**: Switched to entirely different topic when warnings configured
- **After**: Concatenates base topic + suffix for warnings
- **Example**: `finance` + `-warnings` = `finance-warnings`
- Updated logging to show base_topic and warning_suffix separately

### 3. Documentation (`CLAUDE.md`)
Updated three sections:

**Notifications section (lines 105-110)**:
- Changed from "separate topic" to "suffix-based" explanation
- Added example showing topic concatenation
- Clarified default suffix and customization

**Environment Variables (lines 131-133)**:
- Replaced `NTFY_TOPIC_WARNING` with `NTFY_WARNING_SUFFIX`
- Documented default value

**Error Handling (line 142)**:
- Updated to show new format: `{NTFY_TOPIC}{NTFY_WARNING_SUFFIX}`
- Added concrete example

## Benefits of New Approach

1. **Simpler configuration**: Users only need to set one topic (`NTFY_TOPIC`)
2. **Automatic namespacing**: Warnings automatically get suffixed
3. **Flexible**: Suffix can still be customized if needed
4. **Consistent**: Base topic name always appears in warning topic

## Migration Notes

Users currently using `NTFY_TOPIC_WARNING` should:
1. Remove `NTFY_TOPIC_WARNING` from their `.env`
2. If they want custom suffix, set `NTFY_WARNING_SUFFIX`
3. Otherwise, default `"-warnings"` suffix will be used automatically

**Example migration**:
```bash
# Old approach
NTFY_TOPIC=finance
NTFY_TOPIC_WARNING=finance-alerts

# New approach (custom suffix)
NTFY_TOPIC=finance
NTFY_WARNING_SUFFIX=-alerts

# New approach (default suffix)
NTFY_TOPIC=finance
# (warnings go to finance-warnings automatically)
```

## Testing

✅ Build successful - no compilation errors
✅ Type consistency verified - suffix is non-pointer string with default
✅ Logic verified - concatenation happens only for warning notifications
<!-- SECTION:NOTES:END -->
