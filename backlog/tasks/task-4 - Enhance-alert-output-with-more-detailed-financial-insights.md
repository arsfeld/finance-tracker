---
id: task-4
title: Enhance alert output with more detailed financial insights
status: Done
assignee:
  - '@claude'
created_date: '2025-10-06 01:04'
updated_date: '2025-10-06 01:09'
labels:
  - enhancement
  - notifications
dependencies: []
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current alerts provide basic spending summary, but could include more actionable information like spending trends, comparisons to previous periods, budget warnings, and unusual patterns to make notifications more valuable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Add spending trend analysis (increase/decrease vs previous period)
- [x] #2 Include budget alerts if spending exceeds thresholds
- [x] #3 Highlight unusual or anomalous transactions
- [x] #4 Add payment due date reminders or warnings
- [x] #5 Include spending velocity (daily/weekly burn rate)
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review current LLM prompt structure in llm.go (generateAnalysisPrompt function)
2. Enhance LLM prompt to request additional analyses:
   - Spending trend analysis (compare periods if multi-month data)
   - Anomaly/unusual transaction detection
   - Spending velocity (daily/weekly burn rate)
   - Budget alerts (thresholds based on historical patterns)
   - Payment due date reminders
3. Test enhanced prompt with existing data
4. Verify output quality in both email and ntfy notifications
5. Update any documentation if needed
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Enhanced financial alerts with actionable insights by improving the LLM prompt in generateAnalysisPrompt (llm.go:208-273).

## Changes Made

### Enhanced LLM Prompt Structure
- Added new "🔍 Insights & Alerts" section requesting:
  - **Anomaly detection**: Identifies unusual transactions, unfamiliar merchants, duplicate charges
  - **Spending velocity**: Calculates daily/weekly burn rate (total expenses ÷ days)
  - **Budget warnings**: Flags high spending patterns and concerning categories
  - **Payment reminders**: Detects recurring subscriptions and upcoming bills

- Added conditional "📈 Spending Trends" section for multi-month analysis:
  - Month-over-month comparison with percentage changes
  - Category-level spending increases/decreases
  - Overall spending trajectory assessment

- Increased response word limit from 150 to 200 words to accommodate richer insights

### Files Modified
- `src/llm.go`: Enhanced generateAnalysisPrompt function with new prompt sections

## Testing Results

**Single-month analysis** produces:
- Daily burn rate calculation ($106.46/day for 21-day period)
- Anomaly detection (verified all merchants legitimate)
- Budget warnings (flagged high retail spending at $648.56)
- Payment reminders (identified Netflix, ChatGPT, LinkedIn subscriptions)

**Multi-month analysis** additionally provides:
- Month-over-month comparison (September +9.7% vs August)
- Category trend analysis (Groceries +12%, Shopping +15%)
- Spending trajectory assessment (discretionary spending uptick)
- Pattern recognition (impulse buying via Amazon/AliExpress)

Notifications display beautifully in both email (HTML with formatted sections) and ntfy (plain text with emojis).
<!-- SECTION:NOTES:END -->
