package ledger

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"finance_tracker/internal/models"
)

// PaymentCategory marks a card-side payment credit.
const PaymentCategory = "Payment"

// PaymentMatchWindow is how far apart the two sides of one payment may post.
// The card usually credits a day or two after the bank debits.
const PaymentMatchWindow = 3 * 24 * time.Hour

// PaymentPatternsSettingKey is where the per-card payment patterns live in the
// settings table.
const PaymentPatternsSettingKey = "card_payment_patterns"

// DefaultPaymentPatterns are the descriptions that pay the TD Visa from the
// other accounts: Tangerine bill payments, line of credit transfers and TD
// savings transfers. Seen on the paying side, they reveal payments the card's
// own feed dropped. The feed has not delivered a payment credit since May.
var DefaultPaymentPatterns = map[string][]string{
	"TD Canada Trust|4520": {"Bill Payment - TD VISA", "TFR-TO C/C", "TFR-A C/C"},
}

// ParsePaymentPatterns reads the stored patterns, falling back to the defaults
// when none are stored.
func ParsePaymentPatterns(raw string) (map[string][]string, error) {
	if raw == "" {
		return DefaultPaymentPatterns, nil
	}
	var patterns map[string][]string
	if err := json.Unmarshal([]byte(raw), &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

// Payment is money paid toward a card. Amount is positive.
type Payment struct {
	Amount float64
	At     int64
}

func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// MatchesPaymentPattern reports whether a description contains any pattern,
// ignoring case and runs of whitespace. Bank descriptions pad fields with
// spaces ("TFR-A  C/C"), which a literal match would miss.
func MatchesPaymentPattern(description string, patterns []string) bool {
	desc := normalize(description)
	for _, p := range patterns {
		if p = normalize(p); p != "" && strings.Contains(desc, p) {
			return true
		}
	}
	return false
}

// DetectPayments finds the payments made toward one card: payment credits on
// the card itself, and debits on the paying accounts that match the card's
// patterns. A debit that matches a card-side credit of the same amount within
// PaymentMatchWindow is the same payment and counts once.
func DetectPayments(cardTxns, payingTxns []models.DBTransaction, patterns []string) []Payment {
	var cardSide []Payment
	for _, t := range cardTxns {
		if t.Amount > 0 && t.Category == PaymentCategory {
			cardSide = append(cardSide, Payment{Amount: t.Amount, At: t.Posted})
		}
	}

	payments := append([]Payment(nil), cardSide...)
	matched := make([]bool, len(cardSide))
	window := int64(PaymentMatchWindow / time.Second)
	for _, t := range payingTxns {
		if t.Amount >= 0 || !MatchesPaymentPattern(t.Description, patterns) {
			continue
		}
		amount := -t.Amount
		duplicate := false
		for i, c := range cardSide {
			if !matched[i] && math.Abs(c.Amount-amount) <= 0.01 && abs(c.At-t.Posted) <= window {
				matched[i], duplicate = true, true
				break
			}
		}
		if !duplicate {
			payments = append(payments, Payment{Amount: amount, At: t.Posted})
		}
	}

	sort.Slice(payments, func(i, j int) bool { return payments[i].At < payments[j].At })
	return payments
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
