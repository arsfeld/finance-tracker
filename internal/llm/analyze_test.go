package llm

import (
	"strings"
	"testing"
	"time"

	"finance_tracker/internal/models"
)

func txn(id, desc string, amount float64, category string, posted time.Time) models.DBTransaction {
	return models.DBTransaction{
		ID:          id,
		AccountID:   "ACT-test",
		Description: desc,
		Amount:      amount,
		Posted:      posted.Unix(),
		Category:    category,
	}
}

// Paying the credit card is not spending: the purchases it settles are already
// counted on the card itself. Regression test for a $4,000 transfer that was
// reported as the single largest expense of the billing cycle.
func TestFilterExcludedCategoriesDropsTransferLegs(t *testing.T) {
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	txns := []models.DBTransaction{
		txn("t1", "METRO BLANCHARD ST ALP", -67.05, "Groceries", day),
		txn("t2", "JT231 TFR-TO C/C", -4000.00, "Payment", day),
		txn("t3", "PAYMENT - THANK YOU", 8830.00, "Payment", day),
	}

	got := FilterExcludedCategories(txns, []string{"Payment"})

	if len(got) != 1 {
		t.Fatalf("expected only the grocery transaction to survive, got %d: %+v", len(got), got)
	}
	if got[0].ID != "t1" {
		t.Errorf("expected transaction t1 to survive, got %q", got[0].ID)
	}
}

// Excluding a category must not depend on the sign of the amount. The negative
// leg of a transfer inflates spending; the positive leg would understate it.
func TestFilterExcludedCategoriesIgnoresSign(t *testing.T) {
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	txns := []models.DBTransaction{
		txn("t1", "JT231 TFR-TO C/C", -4000.00, "Payment", day),
		txn("t2", "PAYMENT - THANK YOU", 8830.00, "Payment", day),
	}

	got := FilterExcludedCategories(txns, []string{"Payment"})

	if len(got) != 0 {
		t.Errorf("both legs of the transfer should be dropped, got %+v", got)
	}
}

func TestFilterExcludedCategoriesKeepsEverythingWhenNoneExcluded(t *testing.T) {
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	txns := []models.DBTransaction{
		txn("t1", "METRO BLANCHARD ST ALP", -67.05, "Groceries", day),
		txn("t2", "CINEPLEX ODEON", -24.50, "Entertainment", day),
	}

	got := FilterExcludedCategories(txns, nil)

	if len(got) != 2 {
		t.Errorf("expected all transactions to survive, got %d", len(got))
	}
}

// Uncategorized transactions have an empty category and must not be swept up by
// an empty string sneaking into the excluded list.
func TestFilterExcludedCategoriesKeepsUncategorized(t *testing.T) {
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	txns := []models.DBTransaction{
		txn("t1", "SOME NEW MERCHANT", -12.00, "", day),
	}

	got := FilterExcludedCategories(txns, []string{""})

	if len(got) != 1 {
		t.Errorf("uncategorized transaction should survive, got %+v", got)
	}
}

// The headline figure the LLM is told must not include transfers.
func TestGeneratePromptTotalExcludesTransfers(t *testing.T) {
	start := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	txns := []models.DBTransaction{
		txn("t1", "METRO BLANCHARD ST ALP", -100.00, "Groceries", day),
		txn("t2", "SUSHI BEAUMONT", -50.00, "Dining", day),
		txn("t3", "JT231 TFR-TO C/C", -4000.00, "Payment", day),
	}

	kept := FilterExcludedCategories(txns, []string{"Payment"})
	prompt := GeneratePrompt(kept, nil, start, end, day, 15, false)

	if !strings.Contains(prompt, "Total Expenses: $150.00") {
		t.Errorf("expected total of $150.00 with the transfer excluded; prompt said:\n%s",
			firstLines(prompt, 12))
	}
	if strings.Contains(prompt, "JT231 TFR-TO C/C") {
		t.Error("the transfer should not appear in the transaction table sent to the LLM")
	}
}

// Charges in categories that are not excluded must keep counting as expenses.
func TestGeneratePromptCountsNonExcludedCharges(t *testing.T) {
	start := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	txns := []models.DBTransaction{
		txn("t1", "METRO BLANCHARD ST ALP", -100.00, "Groceries", day),
		txn("t2", "HYDRO QUEBEC", -139.00, "Utilities", day),
	}

	kept := FilterExcludedCategories(txns, []string{"Payment"})
	prompt := GeneratePrompt(kept, nil, start, end, day, 15, false)

	if !strings.Contains(prompt, "Total Expenses: $239.00") {
		t.Errorf("non-excluded charges should count as expenses; prompt said:\n%s", firstLines(prompt, 12))
	}
}

// Excluding Payment wholesale also drops card fees and interest, which are real
// costs. This is a deliberate tradeoff: the alternative is splitting transfers
// and fees into separate categories. Documented here so the loss is visible if
// the fee total ever grows enough to matter.
func TestGeneratePromptDropsFeesBundledWithPaymentCategory(t *testing.T) {
	start := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	txns := []models.DBTransaction{
		txn("t1", "METRO BLANCHARD ST ALP", -100.00, "Groceries", day),
		txn("t2", "ANNUAL FEE", -139.00, "Payment", day),
	}

	kept := FilterExcludedCategories(txns, []string{"Payment"})
	prompt := GeneratePrompt(kept, nil, start, end, day, 15, false)

	if !strings.Contains(prompt, "Total Expenses: $100.00") {
		t.Errorf("expected fees to be dropped with the Payment category; prompt said:\n%s",
			firstLines(prompt, 12))
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
