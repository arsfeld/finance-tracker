package ledger

import (
	"testing"
	"time"

	"finance_tracker/internal/models"
)

const tdKey = "TD Canada Trust|4520"

func productionAccounts() []models.DBAccount {
	return []models.DBAccount{
		{ID: "ACT-old", IsIncluded: true, IsCreditCard: true, CardKey: tdKey, FirstSeenAt: "2026-03-17 02:27:47",
			BalanceDate: day(time.September, 3).Unix()},
		{ID: "ACT-new", IsIncluded: true, IsCreditCard: true, CardKey: tdKey, FirstSeenAt: "2026-09-11 23:41:21",
			BalanceDate: day(time.September, 23).Unix()},
		{ID: "ACT-chq", IsIncluded: true, CardKey: "Tangerine Bank (CA)|2106"},
		{ID: "ACT-sav", IsIncluded: true, CardKey: "TD Canada Trust|2625"},
	}
}

func on(account string, t models.DBTransaction) models.DBTransaction {
	t.AccountID = account
	return t
}

func productionPeriods() []models.BillingPeriod {
	return []models.BillingPeriod{
		{Label: "Aug 15 - Sep 14", Start: midnight(time.August, 15), End: midnight(time.September, 14), IsComplete: true},
		{Label: "Sep 15 - Sep 23", Start: midnight(time.September, 15), End: day(time.September, 23), IsComplete: false},
	}
}

// End to end on the galactica fixture: no card transactions at all, payments
// found only on the chequing and savings side.
func TestBuildMeasuresSpendingWithoutCardTransactions(t *testing.T) {
	snaps := map[string][]models.BalanceSnapshot{tdKey: {
		snap(-12124.34, day(time.August, 31)),
		snap(-7129.23, day(time.September, 11)),
		snap(-8024.04, day(time.September, 23)),
	}}
	txns := []models.DBTransaction{
		on("ACT-chq", tx("Bill Payment - TD VISA - ************4533", -7340, "Payment", day(time.September, 1))),
		on("ACT-sav", tx("WW591 TFR-A  C/C", -1800, "Payment", day(time.September, 22))),
		on("ACT-chq", tx("EFT Withdrawal to HYDRO-QUEBEC", -74.17, "Utilities", day(time.September, 16))),
	}

	report := Build(productionPeriods(), productionAccounts(), txns, snaps, DefaultPaymentPatterns, []string{"Payment"})

	total := report.Periods[0].BalanceSpend + report.Periods[1].BalanceSpend
	if !near(total, 2344.89+2694.81) {
		t.Errorf("expected 5039.70 of balance spending across the periods, got %.2f", total)
	}
	if len(report.Charges) != 0 {
		t.Errorf("chequing charges are out of scope, got %+v", report.Charges)
	}
}

func TestBuildItemizesOnlyIncludedCardChargesOutsideExcludedCategories(t *testing.T) {
	txns := []models.DBTransaction{
		on("ACT-old", tx("METRO", -100, "Groceries", day(time.August, 20))),
		on("ACT-old", tx("ANNUAL FEE", -139, "Payment", day(time.August, 20))),
		on("ACT-old", tx("PAYMENT - THANK YOU", 5015, "Payment", day(time.August, 21))),
		on("ACT-chq", tx("EFT Withdrawal to SUBARU FINANCE", -702.61, "Automotive", day(time.August, 30))),
	}

	report := Build(productionPeriods(), productionAccounts(), txns, nil, DefaultPaymentPatterns, []string{"Payment"})

	if len(report.Charges) != 1 || report.Charges[0].Description != "METRO" {
		t.Errorf("expected only the grocery charge, got %+v", report.Charges)
	}
}

// After a re-auth SimpleFin can keep returning the dead ID with a fresh
// balance_date and a frozen balance. Its date must not stretch the card's
// coverage over days the current account has not reported, or the burn rate
// decays toward zero.
func TestBuildTakesCoverageFromTheCurrentAccountOnly(t *testing.T) {
	accounts := productionAccounts()
	accounts[0].BalanceDate = day(time.September, 23).Unix() // ACT-old, superseded
	accounts[1].BalanceDate = day(time.September, 11).Unix() // ACT-new, current
	snaps := map[string][]models.BalanceSnapshot{tdKey: {
		snap(-12124.34, day(time.August, 31)),
		snap(-7129.23, day(time.September, 11)),
	}}

	report := Build(productionPeriods(), accounts, nil, snaps, DefaultPaymentPatterns, nil)

	if cur := report.Periods[1]; cur.Source != SourceItemizedOnly || cur.CoveredDays != 0 {
		t.Errorf("the current account last reported Sep 11, so Sep 15 on is not covered, got %+v", cur)
	}
}

// Whether a card is analyzed is decided by its current account. Excluding the
// dead duplicate a re-auth left behind must not drop the card's history.
func TestBuildKeepsSupersededHistoryOfAnIncludedCard(t *testing.T) {
	accounts := productionAccounts()
	accounts[0].IsIncluded = false // ACT-old
	txns := []models.DBTransaction{on("ACT-old", tx("METRO", -100, "Groceries", day(time.August, 20)))}

	report := Build(productionPeriods(), accounts, txns, nil, DefaultPaymentPatterns, nil)

	if len(report.Cards) != 1 || report.Cards[0] != tdKey {
		t.Errorf("expected the TD card to be analyzed, got %v", report.Cards)
	}
	if len(report.Charges) != 1 || report.Charges[0].Description != "METRO" {
		t.Errorf("expected the old account's charge to be itemized, got %+v", report.Charges)
	}
}

func TestBuildSkipsACardWhoseCurrentAccountIsExcluded(t *testing.T) {
	accounts := productionAccounts()
	accounts[1].IsIncluded = false // ACT-new
	txns := []models.DBTransaction{on("ACT-old", tx("METRO", -100, "Groceries", day(time.August, 20)))}

	report := Build(productionPeriods(), accounts, txns, nil, DefaultPaymentPatterns, nil)

	if len(report.Cards) != 0 || len(report.Charges) != 0 {
		t.Errorf("an excluded card is not analyzed, got cards %v and charges %+v", report.Cards, report.Charges)
	}
}

// The interval that straddles the start of the range begins at the last
// snapshot before it, and its payments can post up to the match window earlier.
func TestFetchFromReachesBackToTheStraddlingSnapshot(t *testing.T) {
	snaps := map[string][]models.BalanceSnapshot{tdKey: {
		snap(-100, day(time.August, 1)), snap(-200, day(time.August, 10)), snap(-300, day(time.August, 20)),
	}}

	got := FetchFrom(snaps, midnight(time.August, 15))

	if want := day(time.August, 10).Add(-PaymentMatchWindow); !got.Equal(want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}
