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
		{Period: julyCycle, Source: ledger.SourceItemizedOnly, Total: 2522.38, Itemized: 2522.38, DailyBurn: 81.37},
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
	if !strings.Contains(p, "Categories cover $100.00 of $5039.92 (2%) of balance-tracked card spending.") {
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

func TestGeneratePromptLabelsItemizedOnlyPeriods(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if !strings.Contains(p, "| Jul 15 - Aug 14 | completed [FOCUS] | itemized only |") {
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

func TestGeneratePromptSaysWhenNoCompletedCycleHasBalanceData(t *testing.T) {
	periods := productionPeriods()
	periods[1].Source = ledger.SourceItemizedOnly

	if p := prompt(periods, nil); !strings.Contains(p, "No completed cycle has balance data yet") {
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
	if !strings.Contains(p, "1 card(s) stopped refreshing, so their figures are out of date.") {
		t.Errorf("the warning must be scoped to the stale card; prompt was:\n%s", p)
	}
}

func TestGeneratePromptOmitsWarningsWhenConnectionsAreHealthy(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if strings.Contains(p, "DATA LAG") || strings.Contains(p, "ITEMIZATION GAP") {
		t.Errorf("nothing is wrong, so nothing should be flagged; prompt was:\n%s", p)
	}
}
