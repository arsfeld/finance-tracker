---
id: task-9
title: Calculate and include daily burn rate in AI analysis
status: Done
assignee:
  - '@claude'
created_date: '2025-10-20 18:46'
updated_date: '2025-10-20 18:50'
labels:
  - enhancement
  - llm
  - analysis
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add daily burn rate calculation (total spending / days in period) to provide better spending context for AI analysis. This metric helps identify if spending is on track, above, or below normal daily spending patterns for the billing cycle.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Calculate daily burn rate for the date range (total negative transactions / number of days)
- [x] #2 Include burn rate calculation in the data sent to the LLM (in the prompt/context)
- [x] #3 Format burn rate clearly (e.g., '$X.XX per day over Y days')
- [x] #4 Ensure calculation happens before AI analysis so LLM receives accurate values
- [x] #5 Consider including projection: 'At this rate, monthly spending would be $Z'
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Analyze existing code to understand where burn rate calculation should be added
2. Add burn rate calculation in generateAnalysisPrompt function (llm.go)
3. Include burn rate in the period description sent to LLM
4. Add monthly projection based on burn rate
5. Update prompt instructions to reference pre-calculated burn rate
6. Test the changes by running the application
7. Verify burn rate appears correctly in AI analysis
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented daily burn rate calculation and monthly spending projection for AI analysis.

Changes made in src/llm.go:
- Added daily burn rate calculation (total expenses / days) for both single-period and multi-month analysis
- For single-period: includes burn rate and 30-day projection in period description
- For multi-month: calculates individual burn rates for each of the 3 cycles, plus average burn rate from completed cycles
- Added monthly projection based on average burn rate from completed cycles (more accurate than including in-progress cycle)
- Updated prompt instructions to reference pre-calculated burn rates instead of asking LLM to calculate them

The burn rate provides better spending context, showing:
- Daily spending rate: $X.XX/day
- Monthly projection: $Y.YY (based on current/completed rates)

This helps the AI identify whether spending is on track, above, or below normal patterns for the billing cycle.
<!-- SECTION:NOTES:END -->
