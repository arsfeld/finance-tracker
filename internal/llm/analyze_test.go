package llm

import (
	"strings"
	"testing"
	"time"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/models"
)

func txn(id, desc string, amount float64, category string, posted time.Time) models.DBTransaction {
	return models.DBTransaction{
		ID: id, AccountID: "ACT-test", Description: desc, Amount: amount, Posted: posted.Unix(), Category: category,
	}
}

func date(month time.Month, d int) time.Time {
	return time.Date(2026, month, d, 0, 0, 0, 0, time.UTC)
}

var (
	julyCycle = models.BillingPeriod{Label: "Jul 15 - Aug 14", Start: date(time.July, 15), End: date(time.August, 14), IsComplete: true, IsFocus: true}
	augCycle  = models.BillingPeriod{Label: "Aug 15 - Sep 14", Start: date(time.August, 15), End: date(time.September, 14), IsComplete: true, IsFocus: true}
	sepCycle  = models.BillingPeriod{Label: "Sep 15 - Sep 23", Start: date(time.September, 15), End: date(time.September, 23), IsFocus: true}
	now       = time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
)

// productionPeriods mirrors galactica on 2026-09-23: July has no balance
// history, August is partly covered, September is covered and unitemized.
func productionPeriods() []ledger.PeriodSpend {
	return []ledger.PeriodSpend{
		{Period: julyCycle, Source: ledger.SourceTransactions, Total: 2522.38, Itemized: 2522.38, CoveredDays: 31, DailyBurn: 81.37},
		{Period: augCycle, Source: ledger.SourceBalance, Total: 3131.30, BalanceSpend: 3131.30, CoveredDays: 14.5, DailyBurn: 215.95, NotItemized: 3131.30},
		{Period: sepCycle, Source: ledger.SourceBalance, Total: 1908.62, BalanceSpend: 1908.62, CoveredDays: 8.5, DailyBurn: 224.54, Itemized: 100, NotItemized: 1808.62},
	}
}

func prompt(periods []ledger.PeriodSpend, charges []models.DBTransaction) string {
	return GeneratePrompt(PromptInput{Periods: periods, Charges: charges, BillingDay: 15, Now: now})
}

// The headline must be the balance figure. Summing transactions is how a dead
// feed was reported as a $2,470 month.
func TestGeneratePromptLeadsWithBalanceDerivedTotals(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	for _, want := range []string{"| Sep 15 - Sep 23 |", "$1908.62", "$224.54/day", "$1808.62"} {
		if !strings.Contains(p, want) {
			t.Errorf("expected %q in the period table; prompt was:\n%s", want, p)
		}
	}
}

func TestGeneratePromptLabelsCategoriesAsPartial(t *testing.T) {
	charges := []models.DBTransaction{txn("t1", "METRO", -100, "Groceries", date(time.September, 16))}

	p := prompt(productionPeriods(), charges)

	// July is itemized only: counting it would make categories look like they
	// cover most of the spending the balance measured.
	if !strings.Contains(p, "Categories cover $100.00 of $5039.92 (2%) of card spending in balance-measured cycles.") {
		t.Errorf("the category breakdown must state its coverage of balance periods; prompt was:\n%s", p)
	}
}

func TestGeneratePromptOmitsCoverageWithoutBalancePeriods(t *testing.T) {
	periods := productionPeriods()[:1]
	charges := []models.DBTransaction{txn("t1", "METRO", -100, "Groceries", date(time.August, 1))}

	if p := prompt(periods, charges); strings.Contains(p, "Categories cover") {
		t.Errorf("no balance period, no coverage line; prompt was:\n%s", p)
	}
}

// August's balance history starts on Aug 31, so its total is only part of the
// cycle. Read as a whole cycle it looks like spending fell.
func TestGeneratePromptMarksPartlyCoveredBalancePeriods(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	for _, want := range []string{
		"| Aug 15 - Sep 14 | completed [FOCUS] | balance (14.5 of 31 days) | 14.5 |",
		"| Sep 15 - Sep 23 | in progress [FOCUS] | balance | 8.5 |",
		"compare cycles by Daily burn, not Total",
		"by daily burn (and total where fully covered)",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("expected %q; prompt was:\n%s", want, p)
		}
	}
}

// August's Total covers only 14 of its 31 days. Compared by Total, the first
// fully covered cycle after it reads as spending more than doubling.
func TestGeneratePromptComparesCyclesByBurnWhenCoverageDiffers(t *testing.T) {
	fullSep := models.BillingPeriod{Label: "Sep 15 - Oct 14", Start: date(time.September, 15), End: date(time.October, 14), IsComplete: true, IsFocus: true}
	periods := []ledger.PeriodSpend{
		{Period: augCycle, Source: ledger.SourceBalance, Total: 3047.15, CoveredDays: 14.1, DailyBurn: 216.80},
		{Period: fullSep, Source: ledger.SourceBalance, Total: 6774.30, CoveredDays: 30, DailyBurn: 225.81},
	}

	p := prompt(periods, nil)

	if !strings.Contains(p, "| +4.2% daily burn |") {
		t.Errorf("expected the change to compare daily burn; prompt was:\n%s", p)
	}
	if strings.Contains(p, "+122.3%") {
		t.Errorf("a partial total must not be compared with a full one; prompt was:\n%s", p)
	}
}

func TestGeneratePromptComparesFullyCoveredCyclesByTotal(t *testing.T) {
	fullSep := models.BillingPeriod{Label: "Sep 15 - Oct 14", Start: date(time.September, 15), End: date(time.October, 14), IsComplete: true}
	fullOct := models.BillingPeriod{Label: "Oct 15 - Nov 14", Start: date(time.October, 15), End: date(time.November, 14), IsComplete: true}
	periods := []ledger.PeriodSpend{
		{Period: fullSep, Source: ledger.SourceBalance, Total: 6000, CoveredDays: 30, DailyBurn: 200},
		{Period: fullOct, Source: ledger.SourceBalance, Total: 6600, CoveredDays: 31, DailyBurn: 212.90},
	}

	if p := prompt(periods, nil); !strings.Contains(p, "| +10.0% |") {
		t.Errorf("fully covered cycles compare totals; prompt was:\n%s", p)
	}
}

func TestGeneratePromptDatesTheItemizedCharges(t *testing.T) {
	charges := []models.DBTransaction{
		txn("t1", "METRO", -100, "Groceries", date(time.September, 16)),
		txn("t2", "IGA", -50, "Groceries", date(time.August, 20)),
	}

	p := prompt(productionPeriods(), charges)

	if !strings.Contains(p, "Itemized charges span Aug 20, 2026 – Sep 16, 2026 (by transaction date).") {
		t.Errorf("expected the charge span; prompt was:\n%s", p)
	}
}

// A feed that died in August still fills the categories. The model must know
// they describe August, not this cycle.
func TestGeneratePromptSaysWhenNoChargesArrivedThisCycle(t *testing.T) {
	charges := []models.DBTransaction{txn("t1", "IGA", -50, "Groceries", date(time.August, 20))}

	p := prompt(productionPeriods(), charges)

	want := "Itemized charges span Aug 20, 2026 – Aug 20, 2026 (by transaction date); none have arrived since, so categories describe that window, not current habits."
	if !strings.Contains(p, want) {
		t.Errorf("expected %q; prompt was:\n%s", want, p)
	}
}

func TestGeneratePromptLabelsTransactionPeriods(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if !strings.Contains(p, "| Jul 15 - Aug 14 | completed [FOCUS] | transactions |") {
		t.Errorf("July has no balance history and must say so; prompt was:\n%s", p)
	}
}

// Card payments settle spending already counted; they are never expenses.
func TestGeneratePromptNeverShowsPayments(t *testing.T) {
	charges := []models.DBTransaction{
		txn("t1", "METRO", -100, "Groceries", date(time.September, 16)),
		txn("t2", "PAYMENT - THANK YOU", 7340, "Payment", date(time.September, 16)),
	}

	if p := prompt(productionPeriods(), charges); strings.Contains(p, "PAYMENT - THANK YOU") {
		t.Error("a payment reached the prompt")
	}
}

func TestGeneratePromptComparesBurnWithLastCompletedCycle(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if !strings.Contains(p, "Current cycle burn: $224.54/day over 8.5 days vs $215.95/day in Aug 15 - Sep 14 (+4.0%)") {
		t.Errorf("expected the burn comparison; prompt was:\n%s", p)
	}
}

func TestGeneratePromptSaysWhenNoCompletedCycleShareTheCurrentSource(t *testing.T) {
	periods := productionPeriods()
	periods[1].Source = ledger.SourceTransactions

	if p := prompt(periods, nil); !strings.Contains(p, "No completed cycle is measured the same way, so there is no comparison.") {
		t.Errorf("expected the missing-comparison note; prompt was:\n%s", p)
	}
}

func TestGeneratePromptMarksBudgetsAsLowerBoundsWhenMostlyUnitemized(t *testing.T) {
	periods := productionPeriods()
	periods[2].Itemized, periods[2].NotItemized = 954.31, 954.31

	p := GeneratePrompt(PromptInput{
		Periods: periods, BillingDay: 15, Now: now,
		Budgets: []models.Budget{{Category: "Groceries", Amount: 990}},
	})

	if !strings.Contains(p, "these budget figures are lower bounds") {
		t.Errorf("50%% itemized must mark budgets as lower bounds; prompt was:\n%s", p)
	}
	if strings.Contains(p, "Budgets cannot be assessed") {
		t.Errorf("50%% itemized can still be assessed as a lower bound; prompt was:\n%s", p)
	}
}

// At 5% itemized every category is trivially under budget. Listing them
// invites the model to praise it.
func TestGeneratePromptWithholdsBudgetsWhenBarelyItemized(t *testing.T) {
	p := GeneratePrompt(PromptInput{
		Periods: productionPeriods(), BillingDay: 15, Now: now,
		Budgets: []models.Budget{{Category: "Groceries", Amount: 990}},
	})

	if !strings.Contains(p, "Budgets cannot be assessed this cycle: only 5% of card spending is itemized.") {
		t.Errorf("expected budgets to be withheld; prompt was:\n%s", p)
	}
	if strings.Contains(p, "- Groceries: $") {
		t.Errorf("no per-category budget lines at 5%% itemized; prompt was:\n%s", p)
	}
}

// A quiet card and a de-authorized connection look identical from the
// transactions alone. Regression test: TD broke on Aug 21 and every report
// showed falling spend.
func TestGeneratePromptNamesCardsThatStoppedRefreshing(t *testing.T) {
	stale := []models.StaleConnection{{
		Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		BalanceDate: date(time.August, 21).Unix(), LastTransaction: date(time.July, 22).Unix(),
	}}

	p := GeneratePrompt(PromptInput{Periods: productionPeriods(), Stale: stale, Now: now})

	if !strings.Contains(p, "DATA LAG") || !strings.Contains(p, "TD AEROPLAN VISA INFINITE (4520)") {
		t.Errorf("a stale card must raise a named data lag warning; prompt was:\n%s", p)
	}
	// Only the stale card's figures are out of date; the rest of the analysis
	// still stands.
	if !strings.Contains(p, "1 card(s) have no data after the date shown, so their figures for later days are missing.") {
		t.Errorf("the warning must be scoped to the stale card; prompt was:\n%s", p)
	}
}

func TestGeneratePromptOmitsWarningsWhenConnectionsAreHealthy(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if strings.Contains(p, "DATA LAG") || strings.Contains(p, "ITEMIZATION GAP") {
		t.Errorf("nothing is wrong, so nothing should be flagged; prompt was:\n%s", p)
	}
}

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

	if p := prompt(periods, nil); !strings.Contains(p, "| mixed: 14.5 d balance, 16.5 d charges |") {
		t.Errorf("a mixed period must say how much the balance measured; prompt was:\n%s", p)
	}
}

// A mixed cycle with missing days is a lower bound like any other, and the
// marker must survive the day split.
func TestGeneratePromptMarksPartlyCoveredMixedPeriods(t *testing.T) {
	periods := productionPeriods()
	periods[1].Source, periods[1].CoveredDays, periods[1].BalanceDays = ledger.SourceMixed, 20, 5

	if p := prompt(periods, nil); !strings.Contains(p, "| mixed: 5.0 d balance, 15.0 d charges (20.0 of 31 days) |") {
		t.Errorf("a partly covered mixed period must keep its coverage marker; prompt was:\n%s", p)
	}
}

// Cards that disagree on the source have no common day split.
func TestGeneratePromptLabelsMixedPeriodsWithoutASplit(t *testing.T) {
	periods := productionPeriods()
	periods[1].Source, periods[1].CoveredDays, periods[1].BalanceDays = ledger.SourceMixed, 20, 0

	p := prompt(periods, nil)

	if !strings.Contains(p, "| mixed (20.0 of 31 days) |") {
		t.Errorf("expected a plain mixed label with its coverage marker; prompt was:\n%s", p)
	}
	if !strings.Contains(p, "A mixed period measures some days by the balance and the rest from charges; its Total covers every covered day.") {
		t.Errorf("the reading guide must explain mixed periods; prompt was:\n%s", p)
	}
}

// Snapshots are stored only when the balance changes, so a healthy card's
// current cycle usually ends in a few days measured from charges. Rejecting
// mixed cycles dropped the burn comparison on every healthy connection.
func TestGeneratePromptComparesAMixedCurrentCycleWithABalanceCycle(t *testing.T) {
	periods := productionPeriods()
	periods[2].Source, periods[2].BalanceDays = ledger.SourceMixed, 6.5

	if p := prompt(periods, nil); !strings.Contains(p, "Current cycle burn: $224.54/day over 8.5 days vs $215.95/day in Aug 15 - Sep 14 (+4.0%)") {
		t.Errorf("a mixed cycle compares with a balance cycle; prompt was:\n%s", p)
	}
}

// Two fully covered cycles are still compared by daily burn when one is
// mixed: its charge-measured days count only what was itemized.
func TestGeneratePromptComparesMixedCyclesByDailyBurn(t *testing.T) {
	fullSep := models.BillingPeriod{Label: "Sep 15 - Oct 14", Start: date(time.September, 15), End: date(time.October, 14), IsComplete: true}
	fullOct := models.BillingPeriod{Label: "Oct 15 - Nov 14", Start: date(time.October, 15), End: date(time.November, 14), IsComplete: true}
	periods := []ledger.PeriodSpend{
		{Period: fullSep, Source: ledger.SourceBalance, Total: 6000, CoveredDays: 30, DailyBurn: 200},
		{Period: fullOct, Source: ledger.SourceMixed, Total: 6600, CoveredDays: 31, BalanceDays: 28, DailyBurn: 212.90},
	}

	if p := prompt(periods, nil); !strings.Contains(p, "| +6.5% daily burn |") {
		t.Errorf("a mixed cycle compares by daily burn; prompt was:\n%s", p)
	}
}

func TestGeneratePromptDoesNotCompareTransactionsWithMixedCycles(t *testing.T) {
	periods := productionPeriods()
	periods[1].Source = ledger.SourceTransactions
	periods[2].Source, periods[2].BalanceDays = ledger.SourceMixed, 6.5

	if p := prompt(periods, nil); !strings.Contains(p, "No completed cycle is measured the same way, so there is no comparison.") {
		t.Errorf("transactions and mixed cycles must not be compared; prompt was:\n%s", p)
	}
}

// When the newest data predates the cycle, its burn is not zero spending; it
// is no data. Comparing it reported a -100% burn.
func TestGeneratePromptSaysWhenTheCurrentCycleHasNoData(t *testing.T) {
	periods := productionPeriods()
	periods[2].Total, periods[2].BalanceSpend, periods[2].CoveredDays, periods[2].DailyBurn = 0, 0, 0, 0
	periods[2].Itemized, periods[2].NotItemized = 0, 0

	p := prompt(periods, nil)

	if !strings.Contains(p, "\nCurrent cycle: no data yet; the latest data predates this cycle.\n") {
		t.Errorf("expected the no-data note; prompt was:\n%s", p)
	}
	if strings.Contains(p, "-100.0%") {
		t.Errorf("an empty cycle must not read as a -100%% burn; prompt was:\n%s", p)
	}
}

func TestGeneratePromptShowsNoChangeForAnEmptyCycle(t *testing.T) {
	fullSep := models.BillingPeriod{Label: "Sep 15 - Oct 14", Start: date(time.September, 15), End: date(time.October, 14), IsComplete: true}
	fullOct := models.BillingPeriod{Label: "Oct 15 - Nov 14", Start: date(time.October, 15), End: date(time.November, 14), IsComplete: true}
	periods := []ledger.PeriodSpend{
		{Period: fullSep, Source: ledger.SourceBalance, Total: 6000, CoveredDays: 30, DailyBurn: 200},
		{Period: fullOct, Source: ledger.SourceBalance},
	}

	if p := prompt(periods, nil); !strings.Contains(p, "| $0.00 | $0.00/day | — |") {
		t.Errorf("a cycle with no covered days has no change; prompt was:\n%s", p)
	}
}

// Half a day of data is not a cycle to compare against.
func TestGeneratePromptSkipsBarelyCoveredCyclesAsComparisonBase(t *testing.T) {
	periods := productionPeriods()
	periods[0].Source = ledger.SourceBalance
	periods[1].CoveredDays, periods[1].DailyBurn = 0.6, 500

	if p := prompt(periods, nil); !strings.Contains(p, "vs $81.37/day in Jul 15 - Aug 14") {
		t.Errorf("expected the comparison to skip the barely covered cycle; prompt was:\n%s", p)
	}
}

// A card's balance can lag while its charges keep arriving; the warning must
// date the newest data, not the balance.
func TestGeneratePromptDatesDataLagByTheNewestData(t *testing.T) {
	stale := []models.StaleConnection{{
		Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		BalanceDate: date(time.August, 21).Unix(), LastTransaction: date(time.September, 10).Unix(),
	}}

	p := GeneratePrompt(PromptInput{Periods: productionPeriods(), Stale: stale, Now: now})

	if !strings.Contains(p, "data through Sep 10, 2026 (13 days ago)") {
		t.Errorf("expected the newest transaction to date the data; prompt was:\n%s", p)
	}
	if strings.Contains(p, "Aug 21") {
		t.Errorf("the older balance date must not be reported; prompt was:\n%s", p)
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
