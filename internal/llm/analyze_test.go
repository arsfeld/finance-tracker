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

	if !strings.Contains(p, "Categories cover $2622.38 of $7562.30 (35%)") {
		t.Errorf("the category breakdown must state its coverage; prompt was:\n%s", p)
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
	p := GeneratePrompt(PromptInput{
		Periods: productionPeriods(), BillingDay: 15, Now: now,
		Budgets: []models.Budget{{Category: "Groceries", Amount: 990}},
	})

	if !strings.Contains(p, "these budget figures are lower bounds") {
		t.Errorf("5%% itemized must mark budgets as lower bounds; prompt was:\n%s", p)
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
}

func TestGeneratePromptOmitsWarningsWhenConnectionsAreHealthy(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if strings.Contains(p, "DATA LAG") || strings.Contains(p, "ITEMIZATION GAP") {
		t.Errorf("nothing is wrong, so nothing should be flagged; prompt was:\n%s", p)
	}
}

// With a live balance the totals are right; only the categories are short.
// Calling that "incomplete data" would contradict the totals.
func TestGeneratePromptDescribesDriftAsItemizationGap(t *testing.T) {
	drifted := []models.UnreconciledAccount{{
		Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust", Unexplained: -894.81,
	}}

	p := GeneratePrompt(PromptInput{Periods: productionPeriods(), Drifted: drifted, Now: now})

	if !strings.Contains(p, "ITEMIZATION GAP") || !strings.Contains(p, "itemization missing for $894.81") {
		t.Errorf("expected an itemization gap note; prompt was:\n%s", p)
	}
	if strings.Contains(p, "DATA LAG") {
		t.Error("a live balance is not a data lag")
	}
}
