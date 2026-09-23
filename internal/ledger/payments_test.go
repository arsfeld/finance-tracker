package ledger

import (
	"testing"
	"time"

	"finance_tracker/internal/models"
)

var tdPatterns = DefaultPaymentPatterns["TD Canada Trust|4520"]

func tx(desc string, amount float64, category string, posted time.Time) models.DBTransaction {
	return models.DBTransaction{Description: desc, Amount: amount, Category: category, Posted: posted.Unix()}
}

func day(month time.Month, d int) time.Time {
	return time.Date(2026, month, d, 12, 0, 0, 0, time.UTC)
}

// The real description has two spaces; the configured pattern has one.
func TestMatchesPaymentPatternCollapsesWhitespace(t *testing.T) {
	if !MatchesPaymentPattern("WW591 TFR-A  C/C", tdPatterns) {
		t.Error("the Sep 22 TD savings transfer to the card must match")
	}
	if !MatchesPaymentPattern("Bill Payment - TD VISA - ************4533", tdPatterns) {
		t.Error("the Tangerine bill payment must match")
	}
}

// TD savings also sends transfers to the line of credit with the same TFR-A
// prefix. Those are not card payments.
func TestMatchesPaymentPatternIgnoresLineOfCreditTransfers(t *testing.T) {
	if MatchesPaymentPattern("RP012 TFR-A  XXX3871", tdPatterns) {
		t.Error("a transfer to the line of credit is not a card payment")
	}
}

func TestDetectPaymentsCountsOnePaymentSeenFromBothSides(t *testing.T) {
	card := []models.DBTransaction{tx("PAYMENT - THANK YOU", 7340, "Payment", day(time.September, 2))}
	paying := []models.DBTransaction{tx("Bill Payment - TD VISA - ****4533", -7340, "Payment", day(time.September, 1))}

	got := DetectPayments(card, paying, tdPatterns)

	if len(got) != 1 || got[0].Amount != 7340 || got[0].At != day(time.September, 2).Unix() {
		t.Errorf("expected one $7340 payment dated by the card side, got %+v", got)
	}
}

func TestDetectPaymentsKeepsSameAmountOutsideWindow(t *testing.T) {
	card := []models.DBTransaction{tx("PAYMENT - THANK YOU", 1000, "Payment", day(time.July, 8))}
	paying := []models.DBTransaction{tx("Bill Payment - TD VISA - ****4533", -1000, "Payment", day(time.July, 20))}

	if got := DetectPayments(card, paying, tdPatterns); len(got) != 2 {
		t.Errorf("payments 12 days apart are two payments, got %+v", got)
	}
}

// A refund lowers the balance too, but it is not a payment: leaving it out of
// the payments lets it reduce spending, which it did.
func TestDetectPaymentsIgnoresRefunds(t *testing.T) {
	card := []models.DBTransaction{tx("#446 SPORTS EXPERTS", 20.70, "Shopping", day(time.January, 23))}

	if got := DetectPayments(card, nil, tdPatterns); len(got) != 0 {
		t.Errorf("a refund is not a payment, got %+v", got)
	}
}

func TestDetectPaymentsNeedsPatternsForPayingSide(t *testing.T) {
	paying := []models.DBTransaction{tx("Bill Payment - BMO MASTERCARD - ****8", -150, "Payment", day(time.September, 4))}

	if got := DetectPayments(nil, paying, nil); len(got) != 0 {
		t.Errorf("a card without patterns has no paying-side payments, got %+v", got)
	}
}

func TestParsePaymentPatternsDefaultsWhenUnset(t *testing.T) {
	got, err := ParsePaymentPatterns("")
	if err != nil || len(got["TD Canada Trust|4520"]) != 3 {
		t.Errorf("expected the TD defaults, got %v (err %v)", got, err)
	}
}
