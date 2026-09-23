# Balance-Trusted Credit Card Spending — Design

Date: 2026-09-23
Status: Approved design, pending implementation plan

## Problem

The AI analysis measures spending by summing transactions. The card's transaction
feed is unreliable, and when it breaks every report reads as a spending drop.

Production data on galactica (snapshot 2026-09-23):

- Re-authorizing the connections made SimpleFin issue new account IDs. The TD
  Aeroplan Visa (4520) now exists twice: the old account `ACT-1820c169…` has 637
  transactions, the newest posted Jul 22, and its balance is frozen at −$12,124.34.
  The new account `ACT-ca03b759…` has **zero** transactions, while its balance
  moved from −$7,129.23 (Sep 11) to −$8,024.04 (Sep 23). Both are included in
  analysis.
- The card feed was degrading before Jul 22. The last card-side payment credit is
  May 29, yet the card was paid $4,000 (line of credit, Jun 26), $1,213 (Jun 26),
  $1,000 (Jul 8) and $8,500 (Jul 24). Itemized charges fall from about $9–10k a
  month (Jan–Mar) to $3.3k (Jun) and $2.5k (Jul).
- The latest report put Aug 15–Sep 14 at **$2,470** of spending, computed over
  "6 transaction days". Real card spending is about $7–9k a month.
- The analysis also includes chequing, savings and line-of-credit transactions.
  The "Payment" category is a catch-all (payroll, transfers, TFSA contributions,
  interest, card annual fees).
- Only the current balance and a one-time anchor are stored. There is no balance
  history.

## Goals

1. The card **balance is the source of truth** for how much was spent. Debt going
   up means spending; debt going down means a payment. This holds even when
   transactions are missing.
2. **Card payments are never counted** as spending.
3. **Daily burn comes from the balance**, so rising debt shows spending is going
   up even when the reason can't be itemized.
4. Analysis covers **credit cards only**. Chequing expenses are stable and out of
   scope.
5. Transactions still supply categories and top expenses, labeled as covering
   only the itemized share.

## Non-Goals

- Web UI for card flags or payment patterns (API only).
- Analyzing chequing, savings or line-of-credit spending.
- Estimating spending for periods before the first balance reading (no
  payment-based proxy).
- Auto-merging accounts without a parseable last-4.
- UI charts built on the new ledger. Only the analysis prompt consumes it in this
  iteration.

## Design

### 1. Card identification and identity across re-auth

Migration `006_balance_snapshots.sql` adds to `accounts`:

- `is_credit_card INTEGER NOT NULL DEFAULT 0`
- `card_key TEXT`

**Classification.** When an account is first inserted, `is_credit_card` is set
from its name using the legacy CLI's keyword list (`src/main.go`: visa,
mastercard, amex, card, …). Names containing "line of credit" are never cards.
Later upserts do not overwrite the flag. There is no accounts UI in the web
app (even `is_included` is API-only), so `PATCH /api/accounts/{id}` gains
optional `is_credit_card` and `card_key` fields next to `is_included`.

**Card key.** `card_key = org_name + "|" + last4`, where `last4` is the trailing
`(NNNN)` in the account name. Example: `TD AEROPLAN VISA INFINITE (4520)` →
`TD Canada Trust|4520`. If no last-4 can be parsed, `card_key = account id` (no
merging) and a warning is logged naming the account. `card_key` can be edited by
hand to merge accounts.

**Backfill.** At startup, an idempotent Go step fills `is_credit_card` and
`card_key` for existing rows where `card_key IS NULL`. For current data this
marks both TD Aeroplan Visa (4520) accounts and the Tangerine Money-Back World
Mastercard (4268), and nothing else.

**Analysis scope.** The analysis uses transactions from accounts with
`is_included = 1 AND is_credit_card = 1`, grouped by `card_key`. Other accounts
keep syncing and stay visible in the UI. They reach the analysis only through
payment detection (section 3).

### 2. Balance ledger

New table:

```sql
CREATE TABLE balance_snapshots (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    card_key     TEXT    NOT NULL,
    account_id   TEXT    NOT NULL,
    balance      REAL    NOT NULL,   -- as reported by SimpleFin; negative = debt
    balance_date INTEGER NOT NULL,   -- unix seconds
    source       TEXT    NOT NULL,   -- 'sync' | 'seed_anchor' | 'seed_current'
    created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE (card_key, balance_date)
);
```

**Recording.** After the account upserts on each sync, for every account with
`is_credit_card = 1`:

1. Only the card's **current account** is read: the account with the newest
   `first_seen_at` for that `card_key` (ties broken by ID). A re-auth creates a
   new ID and the old one is dead from then on, even if SimpleFin keeps
   reporting it with a fresh `balance_date`. Every other account with the same
   key is **superseded**.
   A reading whose `balance_date` is not newer than the card's newest snapshot
   is skipped.
2. If `balance` equals the balance of the card's previous snapshot, skip it. An
   unchanged balance carries no information, and TD has reported a frozen
   balance with an advancing `balance_date` (the old account showed −$12,124.34
   on Sep 3, after the $7,340 payment on Sep 1). Writing that reading would put a
   fake $7,340 of spending into a 3-day window.
3. Otherwise `INSERT OR IGNORE` a `sync` row.

**Coverage end.** For each card, coverage runs to the maximum `balance_date`
across its accounts, not to the newest snapshot. A quiet card with an unchanged
balance still counts as covered.

**Seed.** The same startup step inserts, with `INSERT OR IGNORE`:

- each card account's `anchor_balance` / `anchor_balance_date` (`seed_anchor`);
- the current `balance` / `balance_date` of each card's current account
  (`seed_current`).

Readings are applied in date order through the same rules as sync.

For current data this gives three TD readings: −$12,124.34 (Aug 31),
−$7,129.23 (Sep 11) and −$8,024.04 (Sep 23). The old account's transactions are
**not** used to rebuild earlier balances, because the feed was already missing
payments from June.

### 3. Payment detection

A payment toward a card is recognized from either side:

- **Card side:** a positive transaction on the card whose effective category is
  `Payment`.
- **Paying side:** a negative transaction on a non-card account whose description
  contains one of the card's payment patterns.

Patterns live in the `settings` table under key `card_payment_patterns`, as JSON
keyed by `card_key`:

```json
{"TD Canada Trust|4520": ["Bill Payment - TD VISA", "TFR-TO C/C", "TFR-A C/C"]}
```

This default is written on first startup if the key is missing. The existing
settings API only reflects `.env`, so a new `GET`/`PUT
/api/card-payment-patterns` endpoint reads and replaces the map.

Matching is a case-insensitive substring match after collapsing runs of
whitespace on both sides. The real description is `WW591 TFR-A  C/C` (two
spaces), which a literal match on `TFR-A C/C` would miss.

**Dedup.** A card-side credit and a paying-side debit with the same absolute
amount (±$0.01) within ±3 days count as one payment, dated by the card-side
transaction.

### 4. Spending calculation

Package `internal/ledger` (new, pure functions, no DB access). Inputs: a card's
ordered snapshots, its deduplicated payments, its card transactions, the billing
day, and the analysis range.

Work with `debt = −balance`. For consecutive snapshots s₀, s₁:

```
spend = (debt₁ − debt₀) + Σ payments with t₀ < date ≤ t₁
if spend < 0: spend = 0        // an unexplained drop is a payment
```

- An itemized refund (a positive card transaction that is not `Payment`) is not
  in `payments`, so it correctly reduces spending. A refund that isn't itemized
  looks like a payment, so spending can come out slightly high. That's the safe
  direction.
- Fees and interest raise the debt, so they count in the balance total.

**Periods.** Billing periods come from the existing `calcBillingPeriods`. An
interval that crosses a period boundary is split pro-rata by seconds.

**Per period and card**, then summed across cards:

| Field | Definition |
|---|---|
| `BalanceSpend` | sum of the interval spend assigned to the period |
| `CoveredDays` | calendar days of the period between the card's first snapshot and its coverage end (a card needs at least two snapshots to have coverage) |
| `Itemized` | sum of the absolute values of the card's negative transactions in the period, excluding excluded categories |
| `Total` | the headline: `BalanceSpend` plus the itemized charges that fall outside the covered window |
| `DailyBurn` | `BalanceSpend / CoveredDays`; for `itemized_only`, `Itemized / elapsed days` |
| `NotItemized` | `max(0, BalanceSpend − itemized charges inside the covered window)`; reported only when above $50 |
| `Source` | `balance` if `CoveredDays > 0`, otherwise `itemized_only` |

Across cards, `Total`, `BalanceSpend`, `Itemized`, `NotItemized` and `DailyBurn`
add up; `CoveredDays` is the maximum; `Source` is `balance` if any card has
coverage.

A partially covered period (Aug 15–Sep 14 has balance data only from Aug 31)
uses balance data for the covered days and itemized charges for the rest, and
`CoveredDays` shows how much is covered.

**Regression expectations (TD, current data):** Aug 31→Sep 11 ≈ $2,344.89
((12,124.34 − 7,129.23) is a $4,995.11 drop, offset by the $7,340 payment on Sep 1).
Sep 11→Sep 23 ≈ $2,694.81 ($894.81 growth plus the $1,800 payment on Sep 22).

### 5. Prompt

`GeneratePrompt` takes the ledger result instead of raw accounts and all
transactions.

**User prompt sections:**

1. **Period table:** label, status (completed / in progress), covered days,
   balance spend, daily burn, change vs. the previous period, itemized, not
   itemized, source label.
2. **Current-cycle trend:** daily burn so far vs. the last completed cycle's
   daily burn, with the percentage change.
3. **Category breakdown and top 10 expenses**, headed with the itemized share,
   e.g. "Categories cover $5,120 of $8,010 (64%) of card spending".
4. **Budgets** compared against itemized category spending. When the current
   period is under 80% itemized, a note says the figures are lower bounds.
5. **Card transactions table:** card transactions only, excluded categories
   removed.

**Removed:** the accounts table, non-card transactions, the "transaction days"
burn rate, and the "consider only outgoing expenses / ignore income" note.

**Instructions added** (system prompt or user prompt):

- Headline totals and burn rates come from card balances and are authoritative.
  Categories describe only the itemized portion.
- Never treat not-itemized spending as savings, and never read a drop in itemized
  spending as a drop in spending.
- Card payments and transfers are not expenses, and they are not shown.
- Do not compare `itemized_only` periods against `balance` periods, and do not
  read a trend across that boundary.

**Data-lag warning (`buildDataLagWarning`):**

- A stale connection (no balance refresh) keeps the current "incomplete data"
  warning.
- A card whose balance is current but whose transactions drift (`UnreconciledAccounts`)
  is described as "balance tracked; itemization missing for $X". The totals are
  correct in that case, and the old wording would contradict them.

### 6. Edge cases

- **Single snapshot:** the card has no intervals, so its periods are
  `itemized_only` until the next reading that changes the balance.
- **Stale balance:** no snapshots are written and the interval grows. The existing
  stale-connection alert fires. Spending in a long interval is spread pro-rata.
- **Re-auth:** matched by `card_key`, so there's no gap. If the key can't be
  parsed, a new ledger starts and a warning is logged.
- **Payment toward an unknown card:** a paying-side match needs a `card_key` from
  the settings map. Patterns are never matched against unmapped cards.
- **Drift detector:** `UnreconciledAccounts` and its sync alert stay unchanged,
  apart from the prompt wording above.
- **Superseded accounts** are dropped from the stale and drift lists, both in the
  prompt and in the sync alert. Otherwise the dead old TD ID would alert on every
  sync forever.

## Components Touched

| Area | Change |
|---|---|
| `internal/database/migrations/006_balance_snapshots.sql` | new columns and table |
| `internal/store/accounts.go` | classification on insert, card-key backfill, inclusion + card listing |
| `internal/store/snapshots.go` (new) | record, seed, and list snapshots per card |
| `internal/store/transactions.go` | fetch card transactions and paying-side candidates |
| `internal/ledger/` (new) | card key parsing, payment dedup, interval spend, period aggregation |
| `internal/api/sync.go` | record snapshots after upserts |
| `internal/api/analysis_run.go` | build the ledger, pass it to the prompt |
| `internal/llm/analyze.go` | prompt restructure, instructions, lag wording |
| `internal/server` / startup | seed step, default payment patterns |
| `internal/api/accounts.go` | PATCH gains `is_credit_card`, `card_key` |
| `internal/api/payment_patterns.go` (new) | GET/PUT payment patterns |
| `cmd/promptdump` (new) | print the analysis prompt for a DB without calling the LLM, for tuning against a copy of production |

## Testing

Table-driven Go tests, following the existing `*_test.go` style.

- **`ledger`:** debt growth; a card-side payment in the window; a paying-side
  payment only; the same payment on both sides (counted once); an unexplained
  drop (spend = 0); an itemized refund; an interval crossing the billing day
  (pro-rata split); a single snapshot (`itemized_only`).
- **Card key:** `TD AEROPLAN VISA INFINITE (4520)` → `TD Canada Trust|4520`;
  `LINE OF CREDIT UNSECURED (3871)` is not a card; a name without last-4 falls
  back to the account ID.
- **Snapshot recording:** a superseded account is skipped; an unchanged balance
  is skipped; the seed is idempotent.
- **Prompt:** balance totals and not-itemized lines appear; categories are
  labeled as partial; itemized-only periods are labeled; payment and chequing
  transactions never appear.
- **Regression fixture:** today's three TD readings plus the $7,340 (Sep 1) and
  $1,800 (Sep 22) payments give $2,344.89 and $2,694.81.
