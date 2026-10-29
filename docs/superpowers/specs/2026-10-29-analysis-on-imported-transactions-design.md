# Analysis on imported transactions

## Problem

SimpleFin's TD connection has failed since late summer ("Auth required"). Its
newest TD transaction is from 2026-07-22. Since September it has delivered only
two balance readings for the card, both through seeding: −$7,129.23 on Sep 11
and −$8,024.04 on Sep 23. Statement imports now carry the data: 449 TD Visa rows
through Oct 27 are in production, all categorized, and the latest import
recorded a −$3,187.08 statement balance for Oct 27.

The analysis already reads the imported rows. Its output is still wrong:

- **The current cycle reports $0.00/day while $2,886 is itemized.** The Sep 23
  reading predates the $8,450 paid on Sep 22. Balance math books about $8.7k of
  spending in Sep 11–23 and leaves Sep 23–Oct 27 at $0. Across the whole span
  the money reconciles: the balance implies $8,744 of spending against $8,261
  itemized. The 34-day interval to the import snapshot is too long for payment
  netting and for proportional splitting across cycles.
- **"TD Visa stopped refreshing".** The SimpleFin sync still runs every two
  days. Its upsert writes the stale Sep 25 `balance_date` over the Oct 27 one
  from the import, so the staleness check fires.
- **Analysis runs only from the SimpleFin cron**, chained after a sync that
  fails.

SimpleFin may be reconnected later, so both sources have to coexist.

## Decisions

- Balance-trusted math applies only while SimpleFin data is fresh. Spending
  comes from transactions wherever SimpleFin is stale.
- Staleness is decided per stretch of time, not per card: a balance interval is
  trusted only when its two SimpleFin readings are close together. Outages, the
  time after the last reading, and any period without readings use
  transactions. After a reconnect, balance math resumes without intervention
  and the outage stays measured by transactions.
- Analysis runs on its own schedule, independent of SimpleFin.

## Design

### 1. Ledger: trusted balance stretches (`internal/ledger`)

- Only SimpleFin readings feed balance math: snapshot sources `sync`,
  `seed_anchor` and `seed_current`. `import` snapshots stay in
  `balance_snapshots` as a record but are filtered out before `Intervals`.
- `Intervals` runs over the full SimpleFin series, netting included, as today.
  Afterwards an interval is **trusted** when `To - From <= MaxTrustedGap`
  (7 days). Sync runs every two days, so one or two missed runs stay trusted.
  Untrusted intervals contribute no balance spending.
- Balance coverage ends at the last SimpleFin reading. The old unchanged-balance extension is dropped: once imports advance balance_date it would stretch SimpleFin coverage over an outage and hide imported charges. On a healthy connection only the days since the last sync are measured from charges.
- `CardPeriods` takes a list of trusted ranges (merged adjacent trusted
  intervals) instead of one `[covFrom, covTo]` range.
  For each period:
  - `BalanceSpend` is the trusted intervals' spending, split by time overlap as
    today.
  - Itemized charges inside a trusted range are compared with `BalanceSpend` to
    compute `NotItemized`, as today. Charges outside every trusted range add to
    `Total` directly.
  - `Source` is `balance` (every covered day trusted), `transactions` (none
    trusted) or `mixed`. `SourceItemizedOnly` is renamed `SourceTransactions`,
    and `SourceMixed` is added.
  - `CoveredDays` counts the period's days up to the card's **data-through
    date**: the later of the current account's `balance_date` (see section 2)
    and the card's newest transaction, so an import without an ending balance
    still counts.
    `DailyBurn = Total / CoveredDays`. An in-progress cycle is no longer
    diluted by days nobody has reported yet.
- `Combine` takes `mixed` when cards disagree, or when any card is `mixed`.

On today's production data no TD interval in the analysis window is trusted
(Aug 31→Sep 11 is 11 days, Sep 11→Sep 23 is 12). Every cycle is measured from
transactions: the current cycle shows $2,886 over 12.5 days instead of $0.

### 2. Store: SimpleFin never rolls back newer data (`internal/store`)

- `AccountStore.Upsert` updates `balance` and `balance_date` on conflict only
  when `excluded.balance_date >= accounts.balance_date`. Name, currency and org
  fields still update on every sync. This is the guard `UpdateBalance` already
  gives imports.
- `SnapshotStore.RecordCurrent` records `sync` readings only for accounts whose
  balance the sync just applied. An import balance in the `accounts` row is
  never re-recorded as a `sync` reading, which would otherwise let it into
  balance math.

### 3. Prompt (`internal/llm/analyze.go`)

- The Source column shows `balance`, `transactions` or `mixed`.
- The "itemized only … lower bound" guidance is replaced by: periods marked
  `transactions` are measured from individual charges (statement imports) and
  are complete through the card's data-through date. `mixed` periods combine
  both and are compared with `balance` periods by daily burn. Keep the instruction not to read a trend across a `balance` /
  `transactions` boundary.
- The data-lag warning keeps its 72-hour threshold, measured in the analysis
  from the newer of `balance_date` and the newest transaction (the sync alert
  keeps `balance_date` alone). The wording changes from "stopped refreshing" to "data
  through <date> (<n> days ago); spending after that date is missing", which
  fits both an import and a sync.

### 4. Scheduling (`internal/config`, `cmd/server`)

- New `ANALYSIS_SCHEDULE` (cron with seconds), default `0 0 8 * * 1` (Mondays
  08:00 in the server's TZ). It is always registered and calls
  `Analysis.RunAnalysis`.
- The SimpleFin cron (`SYNC_SCHEDULE`) is still registered only when
  `SIMPLEFIN_BRIDGE_URL` is set, and now runs only `Sync.RunSync`.
- Both jobs keep using the scheduler's mutex. A collision skips the run and
  logs it, as today.
- Unsetting `SIMPLEFIN_BRIDGE_URL` in production to stop the failing sync is an
  operational choice outside this change.

### 5. Documentation

- `CLAUDE.md`, "Balance-Trusted Card Spending": state that balance math applies
  only to SimpleFin readings at most 7 days apart and that transactions measure
  everything else. Document `ANALYSIS_SCHEDULE`.

## Testing

- `internal/ledger`:
  - SimpleFin readings 2 days apart are trusted and measured by balance.
  - Readings 12 days apart fall back to transactions.
  - Import snapshots never create balance spending.
  - A period with trusted and untrusted days is `mixed`, and its `Total` is
    balance spend plus outside charges.
  - Balance coverage ends at the last reading.
  - `DailyBurn` divides by days up to the data-through date.
  - A regression test built from the production TD readings and payments
    (Sep 11, Sep 23, import Oct 27) reports the current cycle from its
    itemized charges.
- `internal/store`:
  - A sync upsert with an older `balance_date` leaves a newer import balance in
    place.
  - `RecordCurrent` does not record a `sync` reading for a balance it did not
    apply.
- `internal/llm`: the prompt shows the new Source labels and data-lag wording.
- Acceptance: `promptdump` on a copy of the production DB shows a non-zero
  current-cycle daily burn and no "stopped refreshing" warning.

## Out of scope

- Running analysis automatically after an import.
- Reconciling SimpleFin and imported transactions beyond the importer's
  existing dedupe.
- Recording each import's statement date range. `balance_date` serves as the
  data-through date.
