# Account inclusion: choose which accounts count

## Problem

Tangerine's connection on galactica is stale, yet its accounts still show up in
the spending on the Transactions page. The connection must stay in SimpleFin, so
the choice has to live in Finance Tracker.

The mechanism exists but is unreachable and leaky:

- `accounts.is_included` is honoured by the analysis, the stale/drift alerts,
  Dashboard, Analytics and Budgets.
- No UI changes it. Only `PATCH /api/accounts/{id}` does.
- The Transactions page ignores it unless the URL carries `included_only=true`.
- A SimpleFin re-auth issues new account IDs, and new rows arrive included. In
  production the March Tangerine rows are excluded, but the rows created by the
  2026-09-19 re-auth are all included.

## Decisions

- Granularity: per account, with a per-institution switch.
- Transactions page: excluded accounts are hidden by default, with a "Show
  excluded accounts" escape hatch.
- Persistence: inclusion belongs to the account's identity key
  (`accounts.card_key`, `org|last4`, falling back to the account ID), not to
  the row, so it survives re-auths and renames.

## Design

### 1. Backend: inclusion belongs to an identity

- `AccountStore.SetIncluded(ctx, id, included)` resolves the account's
  `card_key` and sets `is_included` on every row with that key.
  `PATCH /api/accounts/{id}` routes `is_included` through it. The other patch
  fields (`is_credit_card`, `card_key`) stay per row. A PATCH for an unknown ID
  returns 404.
- `AccountStore.Upsert`, on insert only: a new row takes `is_included` from an
  existing row with the same key, the most recently first-seen one. A key never
  seen before is inserted included, as today. The conflict path never touches
  `is_included`, so hand edits survive every sync.
- Normalization: `AccountStore.NormalizeInclusion(ctx)` sets every row of a key
  to the value of that key's most recently first-seen row. It runs in
  `InitCardLedger` after `BackfillCardIdentity` on every startup. It is
  idempotent, and a no-op once `SetIncluded` keeps keys consistent.
- `GET /api/accounts` adds `is_current`: true for the most recently first-seen
  row of each key, with ties broken by the higher ID. This is the rule
  `ledger.CurrentAccounts` uses, applied to all accounts rather than cards only.
  The UI filters on it instead of re-implementing the rule.
- Unchanged: the ledger, the analysis, alerts and every `a.is_included = 1`
  query. Excluded non-card accounts are still scanned for card payments, because
  card spend depends on it.

### 2. Settings: new "Accounts" tab

- Lists accounts with `is_current`, grouped by `org_name`, sorted by name.
- Group header: the institution name and a switch. The switch is on when any
  account in the group is included; turning it off excludes all of them, and
  turning it on includes all of them.
- Row: the account name, a "card" badge when `is_credit_card`, the balance,
  "updated <balance_date>", and its own switch.
- The group switch sends one PATCH per account in parallel. When all have
  settled it invalidates `accounts`, `transactions`, `dashboard`, `analytics`
  and `budgets` queries. If any request failed, the reloaded list shows the
  true state and an error message is displayed.
- A note explains that excluded accounts are left out of spending, analysis and
  alerts, but are still checked for card payments.

### 3. Transactions page

- Sends `included_only=true` unless the URL has `show_excluded=true`.
- A "Show excluded accounts" checkbox beside the existing filters toggles
  `show_excluded` in the URL. The list, summary cards, category breakdown and
  CSV export all follow it, because they share `apiParams`.
- Incoming links with `included_only=true` keep working. The parameter is
  redundant but harmless.

## Testing

- Store tests:
  - `SetIncluded` updates every row sharing a key and no other key.
  - A new row inherits exclusion from its key's siblings.
  - A never-seen key is inserted included.
  - `NormalizeInclusion` resolves mixed rows toward the most recently
    first-seen row.
  - The `is_current` rule picks the newest row per key.
- API test: `GET /api/accounts` returns `is_current`; PATCH with `is_included`
  propagates across the key.
- `devenv shell -- just build-web` type-checks the frontend.
- Against a copy of the production DB: after normalization, excluding every
  Tangerine account removes Tangerine transactions from
  `/api/transactions/summary` (the default Transactions view), while
  `show_excluded=true` brings them back.

## Out of scope

- Editing `is_credit_card` or `card_key` from the UI.
- Removing or pausing connections in SimpleFin.
