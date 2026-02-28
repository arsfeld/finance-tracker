---
id: task-14
title: Implement SimpleFin API integration in Phoenix app
status: Done
assignee:
  - '@ai-agent'
created_date: '2025-11-04 01:30'
updated_date: '2025-11-04 02:47'
labels:
  - phoenix
  - integration
  - simplefin
dependencies:
  - task-13
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement the SimpleFin Bridge API client to fetch accounts and transactions. Port logic from Go implementation (src/simplefin.go) to Elixir, handling JSON parsing, balance type conversion, error handling, and zero-balance filtering.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 SimpleFin client makes HTTP GET requests to Bridge API with date range parameters
- [x] #2 JSON response parsing handles Balance type (both string and float formats)
- [x] #3 Accounts with zero balance are filtered out
- [x] #4 API errors array is captured and returned separately
- [x] #5 Client handles HTTP errors and timeouts gracefully
- [x] #6 Transaction data is correctly parsed with all fields (id, description, amount, posted, transacted_at, pending)
- [x] #7 Module includes tests for parsing and error handling
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review Go implementation and understand SimpleFin API structure
2. Implement HTTP client using Req library for SimpleFin API calls
3. Create custom JSON decoder for Balance type (handles string and float)
4. Implement fetch_accounts/3 function with date range parameters
5. Add zero-balance account filtering logic
6. Implement API error array capture and return
7. Add HTTP timeout and error handling
8. Write comprehensive tests for all functionality
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Summary

Implemented SimpleFin Bridge API client in Elixir, porting functionality from the Go implementation. The client fetches accounts and transactions from SimpleFin with proper error handling, zero-balance filtering, and support for both string and numeric balance formats.

## Key Changes

- **HTTP Client**: Implemented using Req library with 120-second timeout
- **Balance Parsing**: Custom `parse_balance/1` function handles both string and float formats from SimpleFin API
- **Zero-Balance Filtering**: Filters out accounts with zero balance before returning results
- **Error Handling**: Captures API errors array separately and handles HTTP/transport errors gracefully
- **Logging**: Comprehensive debug and warning logs for API interactions
- **Tests**: Full test suite with 15 tests covering all functionality using Bypass for HTTP mocking

## Files Modified

- `lib/finance_tracker/integrations/simplefin.ex`: Complete implementation of SimpleFin client
- `mix.exs`: Added Bypass dependency for testing
- `test/finance_tracker/integrations/simplefin_test.exs`: Comprehensive test suite

## Testing

All 15 tests pass. Run with:
```bash
MIX_ENV=test mix test test/finance_tracker/integrations/simplefin_test.exs
```

Tests cover:
- Balance parsing (string, float, integer, nil, invalid)
- HTTP requests with correct date range parameters  
- JSON parsing with accounts and transactions
- Zero-balance account filtering
- API error capture and separation
- HTTP error handling (500, timeout, connection refused)
- Invalid JSON handling
- Transaction field parsing
<!-- SECTION:NOTES:END -->
