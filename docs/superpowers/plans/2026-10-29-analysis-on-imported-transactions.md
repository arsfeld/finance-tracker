# Analysis on Imported Transactions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Measure card spending from transactions wherever SimpleFin balance data is stale, so analysis works on statement imports.

**Architecture:** `internal/ledger` keeps balance math only for intervals between SimpleFin readings at most 7 days apart and measures everything else from itemized charges. Periods are labelled `balance`, `transactions` or `mixed`. The SimpleFin upsert stops rolling back newer balances, the prompt describes the new sources, and analysis gets its own cron.

**Tech Stack:** Go, SQLite, robfig/cron, zerolog. Run everything through devenv: `devenv shell -- go test ./...` (skip the `devenv shell --` prefix only if `$DEVENV_PROFILE` is set).

**Spec:** `docs/superpowers/specs/2026-10-29-analysis-on-imported-transactions-design.md`

## Global Constraints

- `MaxTrustedGap` = 7 days (`7 * 24 * time.Hour`).
- Snapshot source `"import"` never feeds balance math. Sources `sync`, `seed_anchor`, `seed_current` and empty (test fixtures) do.
- Period sources are exactly `balance`, `transactions` and `mixed` (Go: `SourceBalance`, `SourceTransactions`, `SourceMixed`).
- `ANALYSIS_SCHEDULE` default is `0 0 8 * * 1` (cron with seconds).
- Match surrounding code: comments explain *why*, in the same terse prose. No new dependencies.
- Commit after each task with a conventional message (`feat(ledger): …`, `fix(store): …`). No attribution lines.

---

### Task 1: Ledger measures spending from trusted balance stretches only

**Files:**
- Modify: `internal/ledger/spend.go` (constants, `PeriodSpend`, `CardPeriods`, `Combine`)
- Modify: `internal/ledger/build.go` (`Build`: data-through date, itemized-only fallback)
- Modify: `internal/llm/analyze.go`, `internal/llm/analyze_test.go` (only the rename `SourceItemizedOnly` → `SourceTransactions` so the build compiles; the wording changes are Task 3)
- Test: `internal/ledger/spend_test.go`, `internal/ledger/build_test.go`

**Interfaces:**
- Produces:
  - `const MaxTrustedGap = 7 * 24 * time.Hour`
  - `const ImportSnapshotSource = "import"`
  - `SourceTransactions Source = "transactions"` (replaces `SourceItemizedOnly`), `SourceMixed Source = "mixed"`
  - `PeriodSpend.BalanceDays float64`: days of the period measured by the balance
  - `PeriodSpend.CoveredDays`: now means days of the period up to the card's data-through date (by balance *or* transactions)
  - `PeriodSpend.DailyBurn = Total / CoveredDays` (0 when CoveredDays is 0)
  - `CardPeriods(periods []models.BillingPeriod, snaps []models.BalanceSnapshot, payments []Payment, dataThrough int64, charges []models.DBTransaction) []PeriodSpend`: same signature; the 4th parameter is renamed `dataThrough`
  - `Intervals` is unchanged.

- [ ] **Step 1: Update and add the failing ledger tests**

In `internal/ledger/spend_test.go`, replace the tests from `TestCardPeriodsSplitsIntervalAtBillingDay` through `TestCombineAddsCardsAndKeepsBalanceSource` (keep `periodsAroundSep15` as is) with:

```go
func TestCardPeriodsSplitsIntervalAtBillingDay(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 12)), snap(-600, midnight(time.September, 18))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 18).Unix(), nil)

	if !near(got[0].BalanceSpend, 300) || !near(got[1].BalanceSpend, 300) {
		t.Errorf("expected 300 on each side of Sep 15, got %.2f and %.2f", got[0].BalanceSpend, got[1].BalanceSpend)
	}
}

// Burn is spend over calendar days with data, not over days that happen to
// have transactions. Dividing by 6 "transaction days" is how a dead feed once
// reported a quiet month.
func TestCardPeriodsDailyBurnUsesCoveredCalendarDays(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 12)), snap(-600, midnight(time.September, 18))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 20).Unix(), nil)[1]

	if got.Source != SourceBalance || !near(got.CoveredDays, 5) || !near(got.DailyBurn, 60) {
		t.Errorf("expected a balance period of 5 covered days at $60/day, got %+v", got)
	}
}

func TestCardPeriodsReportsNotItemizedGap(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{
		snap(0, midnight(time.September, 15)), snap(-4000, midnight(time.September, 20)), snap(-8000, midnight(time.September, 25)),
	}
	charges := []models.DBTransaction{tx("COSTCO", -5000, "Groceries", day(time.September, 18))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 25).Unix(), charges)[1]

	if got.Source != SourceBalance || !near(got.Total, 8000) || !near(got.Itemized, 5000) || !near(got.NotItemized, 3000) {
		t.Errorf("expected total 8000, itemized 5000, not itemized 3000, got %+v", got)
	}
}

// Charges post a little before or after the balance refresh, so small gaps are
// noise and must not be reported as missing spending.
func TestCardPeriodsIgnoresSmallGap(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{
		snap(0, midnight(time.September, 15)), snap(-4000, midnight(time.September, 20)), snap(-8000, midnight(time.September, 25)),
	}
	charges := []models.DBTransaction{tx("COSTCO", -7970, "Groceries", day(time.September, 18))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 25).Unix(), charges)[1]

	if got.NotItemized != 0 {
		t.Errorf("a $30 gap is noise, got not itemized %.2f", got.NotItemized)
	}
}

func TestCardPeriodsSingleSnapshotIsMeasuredFromTransactions(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(-500, midnight(time.September, 20))}
	charges := []models.DBTransaction{tx("IGA", -300, "Groceries", day(time.September, 18))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 20).Unix(), charges)[1]

	if got.Source != SourceTransactions || !near(got.Total, 300) || !near(got.CoveredDays, 5) || !near(got.DailyBurn, 60) {
		t.Errorf("one snapshot has no interval; expected transactions with total 300 over 5 days, got %+v", got)
	}
}

// Readings 11 days apart span a failing connection; the one at the end may be
// a frozen balance. The period falls back to its charges.
func TestCardPeriodsLongGapFallsBackToTransactions(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(-12124.34, day(time.August, 31)), snap(-7129.23, day(time.September, 11))}
	payments := []Payment{{Amount: 7340, At: day(time.September, 1).Unix()}}
	charges := []models.DBTransaction{tx("METRO", -100, "Groceries", day(time.August, 20))}

	got := CardPeriods(periods, snaps, payments, day(time.September, 11).Unix(), charges)[0]

	if got.Source != SourceTransactions || got.BalanceSpend != 0 || !near(got.Total, 100) {
		t.Errorf("expected only the itemized 100 from transactions, got %+v", got)
	}
}

// Statement balances are recorded but are not SimpleFin readings: a reading
// per import spans weeks and is spread across cycles by elapsed time.
func TestCardPeriodsIgnoresImportSnapshots(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	imported := snap(-900, midnight(time.September, 18))
	imported.Source = ImportSnapshotSource
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 14)), imported}
	charges := []models.DBTransaction{tx("IGA", -250, "Groceries", day(time.September, 16))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 18).Unix(), charges)[1]

	if got.Source != SourceTransactions || got.BalanceSpend != 0 || !near(got.Total, 250) {
		t.Errorf("an import snapshot must not create balance spending, got %+v", got)
	}
}

// Aug 15 - Sep 14 has trusted balance data only from Sep 5 to Sep 10. Those
// days come from the balance and the rest from charges.
func TestCardPeriodsMixesBalanceAndTransactionDays(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 5)), snap(-500, midnight(time.September, 10))}
	charges := []models.DBTransaction{
		tx("METRO", -100, "Groceries", day(time.August, 20)),
		tx("IGA", -450, "Groceries", day(time.September, 7)),
	}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 10).Unix(), charges)[0]

	if got.Source != SourceMixed || !near(got.Total, 600) || !near(got.BalanceDays, 5) || !near(got.CoveredDays, 26) || got.NotItemized != 0 {
		t.Errorf("expected mixed: 500 balance + 100 outside over 26 days, 5 by balance, got %+v", got)
	}
}

// An unchanged balance is not stored as a snapshot, so coverage runs on to the
// account's balance date, but never more than MaxTrustedGap past the last
// reading: an import's balance date must not stretch SimpleFin over an outage.
func TestCardPeriodsCapsCoverageExtension(t *testing.T) {
	periods := []models.BillingPeriod{
		{Label: "Sep 1 - Sep 30", Start: midnight(time.September, 1), End: midnight(time.September, 30), IsComplete: true},
	}
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 1)), snap(-300, midnight(time.September, 3))}
	charges := []models.DBTransaction{
		tx("IGA", -300, "Groceries", day(time.September, 2)),
		tx("METRO", -80, "Groceries", day(time.September, 8)),  // inside the extension: the balance says it was paid off
		tx("COSTCO", -200, "Groceries", day(time.September, 20)), // past the cap: measured from transactions
	}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 25).Unix(), charges)[0]

	if !near(got.BalanceDays, 9) || !near(got.CoveredDays, 24) || !near(got.Total, 500) || got.Source != SourceMixed {
		t.Errorf("expected 9 balance days (2 measured + 7 extension) of 24, total 300 + 200, got %+v", got)
	}
}

func TestCombineAddsCardsAndMixesSources(t *testing.T) {
	p := models.BillingPeriod{Label: "Sep"}
	td := []PeriodSpend{{Period: p, Source: SourceBalance, Total: 2000, DailyBurn: 200, CoveredDays: 10, BalanceDays: 10}}
	mc := []PeriodSpend{{Period: p, Source: SourceTransactions, Total: 50, DailyBurn: 5, CoveredDays: 10}}

	got := Combine([][]PeriodSpend{td, mc})

	if len(got) != 1 || got[0].Source != SourceMixed || got[0].Total != 2050 || got[0].DailyBurn != 205 || got[0].CoveredDays != 10 {
		t.Errorf("unexpected combination: %+v", got)
	}
}

func TestCombineKeepsASharedSource(t *testing.T) {
	p := models.BillingPeriod{Label: "Sep"}
	a := []PeriodSpend{{Period: p, Source: SourceTransactions, Total: 10}}
	b := []PeriodSpend{{Period: p, Source: SourceTransactions, Total: 20}}

	if got := Combine([][]PeriodSpend{a, b}); got[0].Source != SourceTransactions {
		t.Errorf("two transaction cards stay transactions, got %+v", got[0])
	}
}
```

In `internal/ledger/build_test.go`:

1. Replace `TestBuildMeasuresSpendingWithoutCardTransactions` with a short-gap version (the production fixture's 11- and 12-day gaps are no longer trusted, which the new regression test below covers):

```go
// A healthy connection: readings a few days apart, no card transactions at
// all, payments found only on the paying side.
func TestBuildMeasuresSpendingWithoutCardTransactions(t *testing.T) {
	snaps := map[string][]models.BalanceSnapshot{tdKey: {
		snap(-5000, day(time.September, 4)),
		snap(-1000, day(time.September, 8)),
		snap(-1600, day(time.September, 12)),
	}}
	txns := []models.DBTransaction{
		on("ACT-chq", tx("Bill Payment - TD VISA - ************4533", -4500, "Payment", day(time.September, 6))),
		on("ACT-chq", tx("EFT Withdrawal to HYDRO-QUEBEC", -74.17, "Utilities", day(time.September, 9))),
	}

	report := Build(productionPeriods(), productionAccounts(), txns, snaps, DefaultPaymentPatterns, []string{"Payment"})

	if got := report.Periods[0]; !near(got.BalanceSpend, 1100) {
		t.Errorf("expected 500 + 600 of balance spending, got %+v", got)
	}
	if len(report.Charges) != 0 {
		t.Errorf("chequing charges are out of scope, got %+v", report.Charges)
	}
}
```

2. In `TestBuildTakesCoverageFromTheCurrentAccountOnly`, change `SourceItemizedOnly` to `SourceTransactions`.

3. Add the production regression test (galactica, 2026-10-29: the seeded Sep 23 reading predates the Sep 22 payments, and statement imports carry everything since):

```go
// Galactica on 2026-10-29: SimpleFin's last TD readings are seeds 11 and 12
// days apart, the Sep 23 one predating $8,450 of payments, and imports carry
// everything since. Balance math booked the payments as September spending
// and reported the current cycle at $0/day.
func TestBuildMeasuresAnImportedOutageFromTransactions(t *testing.T) {
	imported := snap(-3187.08, day(time.October, 27))
	imported.Source = ImportSnapshotSource
	snaps := map[string][]models.BalanceSnapshot{tdKey: {
		snap(-12124.34, day(time.August, 31)),
		snap(-7129.23, day(time.September, 11)),
		snap(-8024.04, day(time.September, 23)),
		imported,
	}}
	accounts := productionAccounts()
	accounts[1].BalanceDate = day(time.October, 27).Unix() // ACT-new, set by the import
	txns := []models.DBTransaction{
		on("ACT-new", tx("PAYMENT - THANK YOU", 1800, "Payment", day(time.September, 22))),
		on("ACT-new", tx("PAYMENT - THANK YOU", 6650, "Payment", day(time.September, 22))),
		on("ACT-new", tx("MAXI", -320, "Groceries", day(time.October, 2))),
		on("ACT-new", tx("QUALITY INN", -556.12, "Travel", day(time.October, 15))),
		on("ACT-new", tx("IGA", -129.85, "Groceries", day(time.October, 26))),
	}
	periods := []models.BillingPeriod{
		{Label: "Sep 15 - Oct 14", Start: midnight(time.September, 15), End: midnight(time.October, 14), IsComplete: true},
		{Label: "Oct 15 - Oct 29", Start: midnight(time.October, 15), End: day(time.October, 29), IsComplete: false},
	}

	report := Build(periods, accounts, txns, snaps, DefaultPaymentPatterns, []string{"Payment"})

	sep, oct := report.Periods[0], report.Periods[1]
	if sep.Source != SourceTransactions || !near(sep.Total, 320) {
		t.Errorf("September must come from its charges, got %+v", sep)
	}
	if oct.Source != SourceTransactions || !near(oct.Total, 685.97) || !near(oct.CoveredDays, 12.5) {
		t.Errorf("October must be its charges over the 12.5 days imported, got %+v", oct)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `devenv shell -- go test ./internal/ledger/`
Expected: compile failure on `SourceTransactions`, `SourceMixed`, `ImportSnapshotSource`, `BalanceDays`.

- [ ] **Step 3: Implement in `internal/ledger/spend.go`**

Replace the `Source` constants block with:

```go
const (
	SourceBalance      Source = "balance"
	SourceTransactions Source = "transactions"
	SourceMixed        Source = "mixed"
)

// MaxTrustedGap is the longest stretch between two SimpleFin readings whose
// balance change is trusted as spending. Sync runs every two days, so a missed
// run or two still fits. A longer gap means the connection was failing, and
// the reading at its end may be a frozen balance: TD's Sep 23 reading predated
// $8,450 of payments, and booking them as spending zeroed the next cycle.
const MaxTrustedGap = 7 * 24 * time.Hour

// ImportSnapshotSource marks a balance read from an imported statement. It is
// kept as a record but never measured: one reading per import spans weeks,
// and the statement's own transactions are complete.
const ImportSnapshotSource = "import"
```

Update `PeriodSpend`:

```go
// PeriodSpend is one billing period's card spending.
type PeriodSpend struct {
	Period       models.BillingPeriod
	Source       Source
	Total        float64 // headline: balance spending plus itemized charges outside the balance-measured days
	BalanceSpend float64
	CoveredDays  float64 // days of the period up to the card's data-through date
	BalanceDays  float64 // of those, days measured by the balance
	DailyBurn    float64 // Total over CoveredDays
	Itemized     float64
	NotItemized  float64 // balance spending the transactions don't account for; 0 when under NotItemizedMin
}
```

Replace `CardPeriods` (and its doc comment) with:

```go
type span struct{ from, to int64 }

func inSpans(spans []span, at int64) bool {
	for _, s := range spans {
		if at >= s.from && at < s.to {
			return true
		}
	}
	return false
}

// CardPeriods measures one card's spending in each billing period. The balance
// decides the total only between SimpleFin readings at most MaxTrustedGap
// apart; there an interval crossing a period boundary is split in proportion
// to time. Every other day, outages and imported stretches included, is
// measured by its itemized charges. dataThrough is the last moment the card
// has data for, from either source. An unchanged balance is not stored as a
// snapshot, so a trusted stretch runs on past the last reading toward it, but
// no further than MaxTrustedGap.
func CardPeriods(periods []models.BillingPeriod, snaps []models.BalanceSnapshot, payments []Payment,
	dataThrough int64, charges []models.DBTransaction) []PeriodSpend {
	var readings []models.BalanceSnapshot
	for _, s := range snaps {
		if s.Source != ImportSnapshotSource {
			readings = append(readings, s)
		}
	}

	// Netting runs over the whole series so a payment can still move back to
	// the interval whose drop it explains; trust is decided afterwards.
	gap := int64(MaxTrustedGap / time.Second)
	intervals := Intervals(readings, payments)
	var trusted []Interval
	var spans []span
	for _, iv := range intervals {
		if iv.To-iv.From > gap {
			continue
		}
		trusted = append(trusted, iv)
		if n := len(spans); n > 0 && spans[n-1].to == iv.From {
			spans[n-1].to = iv.To
		} else {
			spans = append(spans, span{iv.From, iv.To})
		}
	}
	if n := len(intervals); n > 0 && intervals[n-1].To-intervals[n-1].From <= gap {
		last := intervals[n-1].To
		spans[len(spans)-1].to = max(last, min(dataThrough, last+gap))
	}
	if n := len(spans); n > 0 {
		dataThrough = max(dataThrough, spans[n-1].to)
	}

	out := make([]PeriodSpend, len(periods))
	for i, p := range periods {
		from, to := PeriodBounds(p)
		ps := PeriodSpend{Period: p}

		for _, iv := range trusted {
			if ov := overlap(iv.From, iv.To, from, to); ov > 0 {
				ps.BalanceSpend += iv.Spend * float64(ov) / float64(iv.To-iv.From)
			}
		}

		dataEnd := to
		if dataThrough > 0 {
			dataEnd = min(to, dataThrough)
		}
		if dataEnd > from {
			ps.CoveredDays = float64(dataEnd-from) / 86400
		}
		var balanceSecs int64
		for _, s := range spans {
			balanceSecs += overlap(s.from, s.to, from, to)
		}
		ps.BalanceDays = float64(balanceSecs) / 86400

		var inside float64
		for _, t := range charges {
			d := EffectiveDate(t)
			if t.Amount >= 0 || d < from || d >= to {
				continue
			}
			ps.Itemized += -t.Amount
			if inSpans(spans, d) {
				inside += -t.Amount
			}
		}

		ps.Total = ps.BalanceSpend + (ps.Itemized - inside)
		if ps.CoveredDays > 0 {
			ps.DailyBurn = ps.Total / ps.CoveredDays
		}
		switch {
		case balanceSecs == 0:
			ps.Source = SourceTransactions
		case ps.BalanceDays >= ps.CoveredDays-0.5:
			ps.Source = SourceBalance
		default:
			ps.Source = SourceMixed
		}
		if balanceSecs > 0 {
			if gap := ps.BalanceSpend - inside; gap > NotItemizedMin {
				ps.NotItemized = gap
			}
		}
		out[i] = ps
	}
	return out
}
```

Replace `Combine` with:

```go
// Combine adds up several cards' periods. Totals and burn rates add; coverage
// is the widest card's. A period is balance or transactions when every card
// agrees, and mixed otherwise.
func Combine(perCard [][]PeriodSpend) []PeriodSpend {
	if len(perCard) == 0 {
		return nil
	}
	out := make([]PeriodSpend, len(perCard[0]))
	for i := range out {
		out[i] = PeriodSpend{Period: perCard[0][i].Period, Source: perCard[0][i].Source}
		for _, card := range perCard {
			c := card[i]
			out[i].Total += c.Total
			out[i].BalanceSpend += c.BalanceSpend
			out[i].Itemized += c.Itemized
			out[i].NotItemized += c.NotItemized
			out[i].DailyBurn += c.DailyBurn
			out[i].CoveredDays = max(out[i].CoveredDays, c.CoveredDays)
			out[i].BalanceDays = max(out[i].BalanceDays, c.BalanceDays)
			if c.Source != out[i].Source {
				out[i].Source = SourceMixed
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Implement in `internal/ledger/build.go`**

In `Build`, the data-through date is the later of the current account's `balance_date` and the card's newest transaction. An import without an ending balance still advances it. Replace:

```go
		// Coverage ends where the current account last reported. A superseded
		// ID can carry a fresh balance_date over a frozen balance, which would
		// stretch coverage over days nobody measured.
		coverageEnd := analyzed[key].BalanceDate
		perCard = append(perCard, CardPeriods(periods, snapshots[key], payments, coverageEnd, charges))
```

with:

```go
		// Data ends where the current account last reported, or at the card's
		// newest transaction when an import without an ending balance brought
		// it further. A superseded ID can carry a fresh balance_date over a
		// frozen balance, which would stretch coverage over days nobody
		// measured, so only the current account's date counts.
		dataThrough := analyzed[key].BalanceDate
		for _, t := range cardTxns {
			dataThrough = max(dataThrough, t.Posted)
		}
		perCard = append(perCard, CardPeriods(periods, snapshots[key], payments, dataThrough, charges))
```

Also replace `SourceItemizedOnly` with `SourceTransactions` in the `report.Periods == nil` fallback.

- [ ] **Step 5: Keep `internal/llm` compiling**

Mechanical rename only: replace `ledger.SourceItemizedOnly` with `ledger.SourceTransactions` in `internal/llm/analyze.go` and `internal/llm/analyze_test.go`. Don't change any wording; Task 3 owns that. Run `grep -rn SourceItemizedOnly internal cmd` and expect no matches.

- [ ] **Step 6: Run the tests**

Run: `devenv shell -- go test ./internal/ledger/ ./internal/llm/ ./internal/api/`
Expected: ledger and llm PASS (the llm label text is untouched until Task 3). If any `internal/api` test fails because its fixture relied on gaps over 7 days being balance-measured, update the fixture dates to readings ≤7 days apart and keep the assertion's intent. Do not weaken assertions.

- [ ] **Step 7: Vet and commit**

```bash
devenv shell -- go vet ./...
git add internal/ledger internal/llm internal/api
git commit -m "feat(ledger): measure spending from transactions where SimpleFin is stale"
```

---

### Task 2: SimpleFin sync never rolls back a newer balance

**Files:**
- Modify: `internal/store/accounts.go` (`Upsert`)
- Modify: `internal/store/snapshots.go` (`SnapshotSourceImport` aliases the ledger constant)
- Test: `internal/store/accounts_test.go`, `internal/store/snapshots_test.go`

**Interfaces:**
- Consumes: `ledger.ImportSnapshotSource` (Task 1)
- Produces: `AccountStore.Upsert` keeps the stored `balance`/`balance_date` when the incoming `balance_date` is older.

- [ ] **Step 1: Write the failing tests**

Append to `internal/store/accounts_test.go`:

```go
// SimpleFin keeps reporting TD's frozen Sep 25 balance after an Oct 27
// statement was imported. The sync must not roll the account back, or the
// card reads as stale and its data-through date regresses.
func TestUpsertKeepsANewerImportedBalance(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	card := models.DBAccount{
		ID: "ACT-card", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		Balance: -8024.04, BalanceDate: 1790380000, IsIncluded: true,
	}
	seedAccountFull(t, accts, card)
	if _, err := accts.UpdateBalance(ctx, "ACT-card", -3187.08, 1793102400); err != nil {
		t.Fatalf("import balance: %v", err)
	}

	card.Name = "TD AEROPLAN VISA INFINITE (4520) renamed"
	if err := accts.Upsert(ctx, card); err != nil {
		t.Fatalf("sync upsert: %v", err)
	}

	got, err := accts.GetByID(ctx, "ACT-card")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Balance != -3187.08 || got.BalanceDate != 1793102400 {
		t.Errorf("stale sync rolled the balance back to %v @ %d", got.Balance, got.BalanceDate)
	}
	if got.Name != card.Name {
		t.Errorf("a sync still refreshes the name, got %q", got.Name)
	}
}
```

Append to `internal/store/snapshots_test.go` (it uses the existing `seedProductionTD`, `seedAccountFull` and `at` helpers; check `at`'s signature in that file and adapt the two date arguments if it differs):

```go
// After an import, RecordCurrent must not copy the imported balance back as a
// SimpleFin reading: that would let a statement balance into balance math.
func TestRecordCurrentDoesNotRelabelAnImportedBalance(t *testing.T) {
	ctx := context.Background()
	_, accts, snaps := seedProductionTD(t)
	snaps.Seed(ctx)

	importedAt := at(time.October, 27)
	if _, err := accts.UpdateBalance(ctx, "ACT-new", -3187.08, importedAt); err != nil {
		t.Fatalf("import balance: %v", err)
	}
	if _, err := snaps.Record(ctx, models.BalanceSnapshot{
		CardKey: "TD Canada Trust|4520", AccountID: "ACT-new", Balance: -3187.08, BalanceDate: importedAt, Source: SnapshotSourceImport,
	}); err != nil {
		t.Fatalf("record import: %v", err)
	}
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-new", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		Balance: -8024.04, BalanceDate: at(time.September, 25),
	})

	n, err := snaps.RecordCurrent(ctx)
	if err != nil || n != 0 {
		t.Errorf("expected no sync reading after a stale sync, got %d (err %v)", n, err)
	}
	byCard, _ := snaps.ListByCard(ctx)
	hist := byCard["TD Canada Trust|4520"]
	if last := hist[len(hist)-1]; last.Source != SnapshotSourceImport {
		t.Errorf("latest snapshot should still be the import, got %+v", last)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `devenv shell -- go test ./internal/store/ -run 'TestUpsertKeepsANewerImportedBalance|TestRecordCurrentDoesNotRelabelAnImportedBalance' -v`
Expected: `TestUpsertKeepsANewerImportedBalance` FAILS (balance rolled back). The RecordCurrent test may already pass, because `Record` skips readings no newer than the latest snapshot. Keep it either way as a regression guard.

- [ ] **Step 3: Implement**

In `AccountStore.Upsert`, change the two balance lines of the `ON CONFLICT` clause to:

```sql
			balance = CASE WHEN excluded.balance_date >= COALESCE(accounts.balance_date, 0)
				THEN excluded.balance ELSE accounts.balance END,
			balance_date = MAX(excluded.balance_date, COALESCE(accounts.balance_date, 0)),
```

Extend the function's leading comment with one sentence: the balance only moves forward, because a failing connection can keep reporting a frozen balance after a newer statement was imported (the same guard as `UpdateBalance`).

In `internal/store/snapshots.go`, change `SnapshotSourceImport = "import"` to `SnapshotSourceImport = ledger.ImportSnapshotSource` (`ledger` is already imported).

- [ ] **Step 4: Run the store and api tests**

Run: `devenv shell -- go test ./internal/store/ ./internal/api/ ./internal/simplefin/ ./internal/importer/`
Expected: PASS. If an existing test upserted an older `balance_date` on purpose and expected it to stick, that expectation contradicts the spec. Update the test to use a newer date and say so in the commit body.

- [ ] **Step 5: Commit**

```bash
git add internal/store
git commit -m "fix(store): keep a newer balance when SimpleFin reports an older one"
```

---

### Task 3: Prompt describes balance, transactions and mixed periods

**Files:**
- Modify: `internal/llm/analyze.go` (`SystemPrompt`, `readingGuide`, `buildPeriodTable`, `buildBurnTrend`, `buildCoverageLine`, budget share, `buildDataLagWarning`)
- Test: `internal/llm/analyze_test.go`

**Interfaces:**
- Consumes: `ledger.SourceBalance`, `ledger.SourceTransactions`, `ledger.SourceMixed`, `PeriodSpend.BalanceDays`, and `PeriodSpend.CoveredDays` as "days with data" (Task 1)

- [ ] **Step 1: Update and add the failing tests**

In `internal/llm/analyze_test.go`:

1. In `productionPeriods()`, give the July entry `CoveredDays: 31` (it is a full, completed transactions period).
2. Rename `TestGeneratePromptLabelsItemizedOnlyPeriods` to `TestGeneratePromptLabelsTransactionPeriods` and change its expectation to `"| Jul 15 - Aug 14 | completed [FOCUS] | transactions |"`.
3. Rename `TestGeneratePromptSaysWhenNoCompletedCycleHasBalanceData` to `TestGeneratePromptSaysWhenNoCompletedCycleShareTheCurrentSource`. Keep its body (it sets `periods[1].Source = ledger.SourceTransactions`, so the current balance cycle has no completed balance cycle). Change the expected text to `"No completed cycle is measured the same way yet"`.
4. Replace `TestGeneratePromptNamesCardsThatStoppedRefreshing`'s second assertion string with `"1 card(s) have no data after the date shown, so their figures for later days are missing."`. Keep the `DATA LAG` and card-name assertion.
5. Add:

```go
// Imported cycles are complete through the import, so cycles measured the
// same way are compared like balance cycles.
func TestGeneratePromptComparesTransactionCycles(t *testing.T) {
	periods := []ledger.PeriodSpend{
		{Period: augCycle, Source: ledger.SourceTransactions, Total: 3100, Itemized: 3100, CoveredDays: 31, DailyBurn: 100},
		{Period: sepCycle, Source: ledger.SourceTransactions, Total: 1200, Itemized: 1200, CoveredDays: 8.5, DailyBurn: 141.18},
	}
	periods[0].Period.IsComplete = true

	p := prompt(periods, nil)

	if !strings.Contains(p, "Current cycle burn: $141.18/day over 8.5 days vs $100.00/day in Aug 15 - Sep 14 (+41.2%)") {
		t.Errorf("expected a transactions-to-transactions burn comparison; prompt was:\n%s", p)
	}
	if strings.Contains(p, "lower bound. Do not compare them with balance periods") {
		t.Errorf("transaction periods are complete, not lower bounds; prompt was:\n%s", p)
	}
}

func TestGeneratePromptLabelsMixedPeriods(t *testing.T) {
	periods := productionPeriods()
	periods[1].Source, periods[1].CoveredDays, periods[1].BalanceDays = ledger.SourceMixed, 31, 14.5

	if p := prompt(periods, nil); !strings.Contains(p, "| mixed (14.5 of 31 days by balance) |") {
		t.Errorf("a mixed period must say how much the balance measured; prompt was:\n%s", p)
	}
}

func TestGeneratePromptMarksPartlyCoveredTransactionPeriods(t *testing.T) {
	periods := productionPeriods()
	periods[0].CoveredDays = 20

	if p := prompt(periods, nil); !strings.Contains(p, "| transactions (20.0 of 31 days) |") {
		t.Errorf("a transactions period with missing days must say so; prompt was:\n%s", p)
	}
}

func TestGeneratePromptExplainsTransactionPeriods(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if !strings.Contains(p, `Periods marked "transactions" are measured from individual charges`) {
		t.Errorf("the reading guide must explain transaction periods; prompt was:\n%s", p)
	}
	if strings.Contains(p, "itemized only") {
		t.Errorf("the old label must be gone; prompt was:\n%s", p)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `devenv shell -- go test ./internal/llm/ -v`
Expected: the new and renamed tests FAIL on wording.

- [ ] **Step 3: Implement**

`SystemPrompt`: replace its 4th and 5th lines with:

```
Spending totals and daily burn rates come from card balances where the balance feed is healthy, and from individual charges (bank statement imports) elsewhere. Both are authoritative.
Where totals come from balances, categories and individual charges describe only the itemized part of that spending.
```

`readingGuide`: replace the "Total"/"Daily burn" line and the two period-marker lines so the guide reads:

```go
const readingGuide = `How to read these numbers:
- "Source" says how a period was measured. "balance": from the card balance (debt going up is spending, debt going down is a payment), authoritative even when transactions are missing. "transactions": from individual charges imported from bank statements, complete for the days covered. "mixed": some days each way.
- "Itemized" is the part of that spending with individual transactions. Categories and top expenses describe only this part.
- "Not itemized" is spending the balance shows but the transaction feed did not deliver. It is real spending with an unknown category — never treat it as savings.
- Card payments and transfers are not expenses and are not shown.
- Periods marked "transactions" are measured from individual charges and are complete through the covered days. Do not read a trend across a change between "balance" and "transactions" periods.
- A period marked "(X of Y days)" has data for only part of its days, so its Total is a lower bound. When coverage differs, compare cycles by Daily burn, not Total.
`
```

`buildPeriodTable`: replace the `source, covered := …` block with:

```go
		source, covered := string(p.Source), fmt.Sprintf("%.1f", p.CoveredDays)
		days := math.Round(periodDays(p.Period)*10) / 10
		switch {
		case p.Source == ledger.SourceMixed:
			source = fmt.Sprintf("mixed (%.1f of %g days by balance)", p.BalanceDays, days)
		case !fullyCovered(p):
			// A cycle with data for only part of its days has a partial Total
			// that otherwise reads as the whole cycle.
			source = fmt.Sprintf("%s (%.1f of %g days)", p.Source, p.CoveredDays, days)
		}
```

Replace the change condition `p.Source == ledger.SourceBalance && prev.Source == ledger.SourceBalance` with `comparable(p, prev)`, and add:

```go
// comparable reports whether two cycles were measured the same way. A trend
// read across a switch between balance and transactions measures the switch.
func comparable(a, b ledger.PeriodSpend) bool {
	return a.Source == b.Source && a.Source != ledger.SourceMixed
}
```

Update the comment above the change computation to say "only completed cycles measured the same way are compared".

`buildBurnTrend`: drop the `cur.Source != ledger.SourceBalance` early return (keep `cur.Period.IsComplete`). Pick the comparison cycle with `prev.Period.IsComplete && comparable(cur, prev) && prev.DailyBurn > 0`. The fallback message becomes:

```go
	return fmt.Sprintf("\nCurrent cycle burn: %s/day over %.1f days. No completed cycle is measured the same way yet, so there is no comparison.\n",
		money(cur.DailyBurn), cur.CoveredDays)
```

Update its doc comment to "the last completed cycle measured the same way".

`fullyCovered`: update its comment to "whether the period has data for every day". The code is unchanged.

`buildCoverageLine`: sum periods with `p.Source != ledger.SourceTransactions`, so balance and mixed periods count. Update its comment ("Transaction periods are left out…").

Budget share: change `current.Source == ledger.SourceBalance` to `current.Source != ledger.SourceTransactions`.

`buildDataLagWarning`: change the header sentence and the per-card line so that they fit an import as well as a sync:

```go
		fmt.Fprintf(&b, "%d card(s) have no data after the date shown, so their figures for later days are missing. ", len(stale))
		b.WriteString("Do not congratulate the user on low spending, and do not read a downward trend into it.\n")
		for _, c := range stale {
			through := time.Unix(c.BalanceDate, 0).UTC()
			line := fmt.Sprintf("> - **%s** (%s): data through %s (%d days ago)",
				c.Name, c.OrgName, through.Format("Jan 2, 2006"), daysBetween(through, now))
```

Keep the newest-transaction suffix as is. Update the function's doc comment: "tells the model which cards have no recent data, from a sync or an import".

- [ ] **Step 4: Run the tests**

Run: `devenv shell -- go test ./internal/llm/ ./internal/api/`
Expected: PASS. Fix any other existing llm test whose expected string contained "balance (" partial-coverage text, using the new `"<source> (X of Y days)"` format. Do not weaken it.

- [ ] **Step 5: Commit**

```bash
git add internal/llm
git commit -m "feat(llm): describe balance, transactions and mixed periods in the prompt"
```

---

### Task 4: Analysis runs on its own schedule; docs

**Files:**
- Modify: `internal/config/config.go` (`AnalysisSchedule`)
- Modify: `cmd/server/main.go` (cron registration)
- Modify: `CLAUDE.md` (Balance-Trusted section, env vars)
- Modify: `README.md` and `.env.example` / `docker-compose.yml`, but only if they already list `SYNC_SCHEDULE` (`grep -rn SYNC_SCHEDULE --include='*' . | grep -v node_modules`)

**Interfaces:**
- Produces: `config.Config.AnalysisSchedule string`

- [ ] **Step 1: Config**

In `Config`, under `// Scheduler`:

```go
	SyncSchedule     string
	AnalysisSchedule string
```

In `Load`:

```go
		SyncSchedule:          getEnvOr("SYNC_SCHEDULE", "0 0 */6 * * *"),
		AnalysisSchedule:      getEnvOr("ANALYSIS_SCHEDULE", "0 0 8 * * 1"),
```

- [ ] **Step 2: Scheduling in `cmd/server/main.go`**

Replace the "Schedule periodic sync + analysis" block with:

```go
	// SimpleFin sync runs only while a bridge is configured. Analysis has its
	// own schedule: statement imports keep the data current even when the
	// SimpleFin connection is down.
	if cfg.SimplefinBridgeURL != "" {
		if err := sched.AddFunc(cfg.SyncSchedule, srv.Sync.RunSync); err != nil {
			log.Error().Err(err).Str("schedule", cfg.SyncSchedule).Msg("Failed to add sync schedule")
		} else {
			log.Info().Str("schedule", cfg.SyncSchedule).Msg("Scheduled periodic sync")
		}
	}
	if err := sched.AddFunc(cfg.AnalysisSchedule, srv.Analysis.RunAnalysis); err != nil {
		log.Error().Err(err).Str("schedule", cfg.AnalysisSchedule).Msg("Failed to add analysis schedule")
	} else {
		log.Info().Str("schedule", cfg.AnalysisSchedule).Msg("Scheduled periodic analysis")
	}
```

Confirm that `srv.Sync.RunSync` and `srv.Analysis.RunAnalysis` have signature `func()` (`grep -n "func (h \*SyncHandler) RunSync\|func (h \*AnalysisRunHandler) RunAnalysis" internal/api`). If not, wrap them in closures.

- [ ] **Step 3: Build**

Run: `devenv shell -- go build ./... && devenv shell -- go vet ./...`
Expected: success.

- [ ] **Step 4: Docs**

In `CLAUDE.md`, section "Balance-Trusted Card Spending (web server)", replace the bullet starting "Card spending comes from **balance snapshots**" with:

```markdown
- Card spending comes from **balance snapshots** (`balance_snapshots`, one per card per sync when the balance changes) only where SimpleFin is healthy: between two SimpleFin readings at most 7 days apart (`ledger.MaxTrustedGap`), spend = debt growth + payments in between, and an unexplained drop is treated as a payment. Everywhere else (outages, statement imports) spending is the sum of itemized charges. Import snapshots (`source = 'import'`) are kept as a record but never measured. Periods are labelled `balance`, `transactions` or `mixed`.
- A SimpleFin sync never moves an account's balance back to an older `balance_date`, so a failing connection cannot undo an import.
```

Under "Environment Variables", add an "Optional (Scheduling)" group:

```markdown
Optional (Scheduling):
- `SYNC_SCHEDULE`: SimpleFin sync cron with seconds (default `0 0 */6 * * *`), only when `SIMPLEFIN_BRIDGE_URL` is set
- `ANALYSIS_SCHEDULE`: analysis cron with seconds (default `0 0 8 * * 1`, Mondays 08:00), always active
```

If README / `.env.example` / `docker-compose.yml` document `SYNC_SCHEDULE`, add `ANALYSIS_SCHEDULE` next to it with the same wording.

- [ ] **Step 5: Full test run and commit**

Run: `devenv shell -- go test ./...`
Expected: PASS.

```bash
git add internal/config cmd/server CLAUDE.md README.md
git commit -m "feat(server): run analysis on its own schedule"
```

(Only add README/other files if you changed them.)

---

### Task 5: Acceptance on a production copy (controller, not a subagent)

- [ ] Copy the prod DB copy to a fresh scratch dir. Never touch galactica.
- [ ] Run `DB_PATH=<copy> ENV_FILE=/dev/null devenv shell -- go run ./cmd/promptdump`.
- [ ] Expect the Oct 15 cycle to have source `transactions`, a non-zero daily burn of about $2,886 / 12.5 days, and no "stopped refreshing" text. The TD data-lag line appears only if `balance_date` is more than 72h old relative to now.
- [ ] `devenv shell -- go test ./...` green, then push to main.
