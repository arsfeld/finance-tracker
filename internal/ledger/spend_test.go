package ledger

import (
	"math"
	"testing"
	"time"

	"finance_tracker/internal/models"
)

func snap(balance float64, when time.Time) models.BalanceSnapshot {
	return models.BalanceSnapshot{Balance: balance, BalanceDate: when.Unix()}
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func midnight(month time.Month, d int) time.Time {
	return time.Date(2026, month, d, 0, 0, 0, 0, time.UTC)
}

// Regression fixture from galactica, 2026-09-23. The card delivered no
// transactions for any of this; the balances and the payments seen from the
// paying accounts are enough.
func TestIntervalsReproduceProductionTDSpending(t *testing.T) {
	snaps := []models.BalanceSnapshot{
		snap(-12124.34, day(time.August, 31)),
		snap(-7129.23, day(time.September, 11)),
		snap(-8024.04, day(time.September, 23)),
	}
	payments := []Payment{
		{Amount: 7340, At: day(time.September, 1).Unix()},
		{Amount: 1800, At: day(time.September, 22).Unix()},
	}

	got := Intervals(snaps, payments)

	if len(got) != 2 || !near(got[0].Spend, 2344.89) || !near(got[1].Spend, 2694.81) {
		t.Errorf("expected spends of 2344.89 and 2694.81, got %+v", got)
	}
}

// A drop with no known payment behind it is a payment the feeds missed. It
// must never produce negative spending.
func TestIntervalsTreatUnexplainedDropAsPayment(t *testing.T) {
	got := Intervals([]models.BalanceSnapshot{snap(-1000, day(time.September, 1)), snap(-400, day(time.September, 3))}, nil)

	if len(got) != 1 || got[0].Spend != 0 {
		t.Errorf("expected zero spend, got %+v", got)
	}
}

func totalSpend(ivs []Interval) float64 {
	var total float64
	for _, iv := range ivs {
		total += iv.Spend
	}
	return total
}

func assertNoNegative(t *testing.T, ivs []Interval) {
	t.Helper()
	for _, iv := range ivs {
		if iv.Spend < 0 {
			t.Errorf("negative interval: %+v", iv)
		}
	}
}

// The bank debits a payment a day or two before the card credits it. With a
// snapshot in between, one interval sees the payment but not the drop and the
// next sees the drop without the payment. Clamping each on its own counted the
// $5,000 payment as spending.
func TestIntervalsNetPaymentAgainstLaterCardCredit(t *testing.T) {
	snaps := []models.BalanceSnapshot{
		snap(-5000, day(time.September, 1)),
		snap(-5080, day(time.September, 3)),
		snap(-100, day(time.September, 5)),
	}
	payments := []Payment{{Amount: 5000, At: day(time.September, 2).Unix()}}

	got := Intervals(snaps, payments)

	assertNoNegative(t, got)
	if total := totalSpend(got); !near(total, 100) {
		t.Errorf("expected 100 of spending, got %.2f in %+v", total, got)
	}
}

// The mirror case: the card credit lands in the interval before the day the
// bank dates the debit.
func TestIntervalsNetCardCreditAgainstLaterPayment(t *testing.T) {
	snaps := []models.BalanceSnapshot{
		snap(-5000, day(time.September, 1)),
		snap(-50, day(time.September, 3)),
		snap(-130, day(time.September, 5)),
	}
	payments := []Payment{{Amount: 5000, At: day(time.September, 4).Unix()}}

	got := Intervals(snaps, payments)

	assertNoNegative(t, got)
	if total := totalSpend(got); !near(total, 130) {
		t.Errorf("expected the 50 + 80 of real charges, got %.2f in %+v", total, got)
	}
}

// Two lagged payments share the interval between them. Each takes only what
// the other left, and nothing is absorbed twice.
func TestIntervalsNetTwoDropsAgainstSharedNeighbour(t *testing.T) {
	snaps := []models.BalanceSnapshot{
		snap(-3000, day(time.September, 1)),
		snap(-1000, day(time.September, 2)), // 2000 credited before its Sep 3 debit
		snap(-1200, day(time.September, 4)), // 200 of charges
		snap(-300, day(time.September, 5)),  // 1000 credited after its Sep 3 debit; 100 of charges
	}
	payments := []Payment{
		{Amount: 2000, At: day(time.September, 3).Unix()},
		{Amount: 1000, At: day(time.September, 3).Unix()},
	}

	got := Intervals(snaps, payments)

	assertNoNegative(t, got)
	if total := totalSpend(got); !near(total, 300) {
		t.Errorf("expected the 200 + 100 of charges, got %.2f in %+v", total, got)
	}
}

// A drop larger than every neighbour within the window absorbs what it can;
// the rest is a payment the feeds missed and never becomes negative spending.
func TestIntervalsDropLargerThanNeighboursLeavesNoNegative(t *testing.T) {
	snaps := []models.BalanceSnapshot{
		snap(-1000, day(time.September, 1)),
		snap(-1200, day(time.September, 3)),
		snap(-100, day(time.September, 4)),
		snap(-150, day(time.September, 5)),
	}

	got := Intervals(snaps, nil)

	assertNoNegative(t, got)
	if total := totalSpend(got); !near(total, 0) {
		t.Errorf("the 1100 drop outweighs the 200 + 50 around it, expected 0, got %.2f in %+v", total, got)
	}
}

// Spending outside the window is not netted against a drop.
func TestIntervalsLeaveDistantSpendingAlone(t *testing.T) {
	snaps := []models.BalanceSnapshot{
		snap(0, day(time.September, 1)),
		snap(-500, day(time.September, 2)),
		snap(-500, day(time.September, 10)),
		snap(-100, day(time.September, 11)),
	}

	got := Intervals(snaps, nil)

	if total := totalSpend(got); !near(total, 500) {
		t.Errorf("the Sep 1-2 spending is 8 days before the drop, expected 500, got %.2f in %+v", total, got)
	}
}

func periodsAroundSep15(now time.Time) []models.BillingPeriod {
	return []models.BillingPeriod{
		{Label: "Aug 15 - Sep 14", Start: midnight(time.August, 15), End: midnight(time.September, 14), IsComplete: true},
		{Label: "Sep 15 - Sep 25", Start: midnight(time.September, 15), End: now, IsComplete: false},
	}
}

func TestCardPeriodsSplitsIntervalAtBillingDay(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 10)), snap(-1000, midnight(time.September, 20))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 20).Unix(), nil)

	if !near(got[0].BalanceSpend, 500) || !near(got[1].BalanceSpend, 500) {
		t.Errorf("expected 500 on each side of Sep 15, got %.2f and %.2f", got[0].BalanceSpend, got[1].BalanceSpend)
	}
}

// Burn is spend over calendar days covered by the balance, not over days that
// happen to have transactions. Dividing by 6 "transaction days" is how a dead
// feed once reported a quiet month.
func TestCardPeriodsDailyBurnUsesCoveredCalendarDays(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 10)), snap(-1000, midnight(time.September, 20))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 20).Unix(), nil)

	if !near(got[1].CoveredDays, 5) || !near(got[1].DailyBurn, 100) {
		t.Errorf("expected 5 covered days at $100/day, got %.2f days at %.2f", got[1].CoveredDays, got[1].DailyBurn)
	}
}

func TestCardPeriodsReportsNotItemizedGap(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 15)), snap(-8000, midnight(time.September, 25))}
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
	snaps := []models.BalanceSnapshot{snap(0, midnight(time.September, 15)), snap(-8000, midnight(time.September, 25))}
	charges := []models.DBTransaction{tx("COSTCO", -7970, "Groceries", day(time.September, 18))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 25).Unix(), charges)[1]

	if got.NotItemized != 0 {
		t.Errorf("a $30 gap is noise, got not itemized %.2f", got.NotItemized)
	}
}

func TestCardPeriodsSingleSnapshotIsItemizedOnly(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(-500, midnight(time.September, 20))}
	charges := []models.DBTransaction{tx("IGA", -300, "Groceries", day(time.September, 18))}

	got := CardPeriods(periods, snaps, nil, midnight(time.September, 20).Unix(), charges)[1]

	if got.Source != SourceItemizedOnly || !near(got.Total, 300) {
		t.Errorf("one snapshot has no interval; expected itemized only with total 300, got %+v", got)
	}
}

// Aug 15 - Sep 14 has balance data only from Aug 31. The covered part comes
// from the balance and the part before it from itemized charges.
func TestCardPeriodsPartialCoverageAddsItemizedOutsideWindow(t *testing.T) {
	periods := periodsAroundSep15(midnight(time.September, 25))
	snaps := []models.BalanceSnapshot{snap(-12124.34, day(time.August, 31)), snap(-7129.23, day(time.September, 11))}
	payments := []Payment{{Amount: 7340, At: day(time.September, 1).Unix()}}
	charges := []models.DBTransaction{tx("METRO", -100, "Groceries", day(time.August, 20))}

	got := CardPeriods(periods, snaps, payments, day(time.September, 11).Unix(), charges)[0]

	if !near(got.Total, 2444.89) || !near(got.CoveredDays, 11) {
		t.Errorf("expected total 2344.89 + 100 over 11 covered days, got %+v", got)
	}
}

func TestCombineAddsCardsAndKeepsBalanceSource(t *testing.T) {
	p := models.BillingPeriod{Label: "Sep"}
	td := []PeriodSpend{{Period: p, Source: SourceBalance, Total: 2000, DailyBurn: 200, CoveredDays: 10}}
	mc := []PeriodSpend{{Period: p, Source: SourceItemizedOnly, Total: 50, DailyBurn: 5}}

	got := Combine([][]PeriodSpend{td, mc})

	if len(got) != 1 || got[0].Source != SourceBalance || got[0].Total != 2050 || got[0].DailyBurn != 205 || got[0].CoveredDays != 10 {
		t.Errorf("unexpected combination: %+v", got)
	}
}
