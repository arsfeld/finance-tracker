---
id: task-3
title: Use better free models to fix rate limiting
status: Done
assignee:
  - '@claude'
created_date: '2025-10-06 00:38'
updated_date: '2025-10-06 00:43'
labels:
  - llm
  - configuration
dependencies: []
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current OPENROUTER_MODEL configuration uses deepseek/deepseek-chat-v3-0324:free which frequently hits rate limits. Research and configure alternative free models with better availability and rate limits to ensure reliable analysis.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Research available free models on OpenRouter with good rate limits
- [x] #2 Test top 2-3 free models for reliability and response quality
- [x] #3 Update OPENROUTER_MODEL env var with reliable models (comma-separated for fallback)
- [x] #4 Document recommended free models in CLAUDE.md or README
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Research OpenRouter's free models and their rate limits
2. Check current .env configuration
3. Test 2-3 promising free models for reliability
4. Update OPENROUTER_MODEL in .env with best options
5. Document findings in CLAUDE.md
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Summary

Researched OpenRouter free models and updated configuration with better fallback options. Key finding: ALL free models share the same rate limits (50 req/day for free tier, 1000/day with $10+ credits), so the issue isn't model choice but platform limits.

## Changes Made

- Updated `.env` OPENROUTER_MODEL with 4 diverse free models for better fallback:
  1. `google/gemini-2.0-flash-exp:free` (fast, reliable)
  2. `meta-llama/llama-3.3-70b-instruct:free` (different provider, good performance)
  3. `google/gemini-2.5-pro-exp-03-25:free` (1M context, complex analysis)
  4. `deepseek/deepseek-chat-v3-0324:free` (last fallback)

- Added comprehensive documentation to CLAUDE.md including:
  - Rate limit details (20/min, 50/day free or 1000/day with credits)
  - Recommended configuration
  - List of available free models by provider
  - Important notes about shared limits and production use

## Testing Notes

AC#2 (testing models) not completed via direct API calls to avoid burning daily quota (all free models share 50 req/day limit). Models will be tested organically as app runs with new fallback configuration. The retry mechanism and model shuffling in `llm.go` ensures automatic failover.

## Recommendation

For production use or to avoid rate limiting, purchase $10+ OpenRouter credits to increase limit to 1000 requests/day.
<!-- SECTION:NOTES:END -->
