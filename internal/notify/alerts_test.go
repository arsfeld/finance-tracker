package notify

import (
	"strings"
	"testing"
	"time"

	"finance_tracker/internal/models"
)

func tdStale(now time.Time) []models.StaleConnection {
	return []models.StaleConnection{{
		ID:              "ACT-td",
		Name:            "TD AEROPLAN VISA INFINITE (4520)",
		OrgName:         "TD Canada Trust",
		BalanceDate:     now.AddDate(0, 0, -10).Unix(),
		LastTransaction: now.AddDate(0, 0, -40).Unix(),
	}}
}

// The alert has to be actionable on a phone lock screen: which account, how
// long, and what the bridge said is wrong. Regression test: the connection died
// on Aug 21 and the only record was a "partial" row in a table nobody reads.
func TestStaleConnectionAlertNamesAccountAndReason(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	msg := StaleConnectionAlert(tdStale(now), []string{
		"Requested date range exceeds recommended range of 45 days. In the future, this may be capped.",
		"Connection to TD Canada Trust may need attention. Auth required",
	}, now)

	for _, want := range []string{
		"TD AEROPLAN VISA INFINITE (4520)",
		"10 days",
		"Auth required",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("alert must mention %q; got:\n%s", want, msg)
		}
	}
}

// Every sync carries the bridge's 45-day range advisory. It is about our own
// request, not a broken connection, so repeating it as the headline would train
// the reader to ignore the alert.
func TestStaleConnectionAlertLeadsWithTheAccountNotTheAdvisory(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	msg := StaleConnectionAlert(tdStale(now), []string{
		"Requested date range exceeds recommended range of 45 days. In the future, this may be capped.",
	}, now)

	headline := strings.SplitN(msg, "\n", 2)[0]
	if strings.Contains(headline, "45 days") {
		t.Errorf("the range advisory must not be the headline; got %q", headline)
	}
	if !strings.Contains(headline, "stopped syncing") {
		t.Errorf("headline should say what is wrong; got %q", headline)
	}
}

// Nothing stale means nothing to send. An empty string is the signal not to
// notify, so a healthy fortnight stays silent.
func TestStaleConnectionAlertEmptyWhenNothingIsStale(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	if msg := StaleConnectionAlert(nil, []string{"some advisory"}, now); msg != "" {
		t.Errorf("no stale connection means no alert; got:\n%s", msg)
	}
}

// An account that has never returned a transaction should say so rather than
// print a 1970 date.
func TestStaleConnectionAlertHandlesAccountWithNoTransactions(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	msg := StaleConnectionAlert([]models.StaleConnection{{
		Name:        "Tangerine Chequing Account (2106)",
		OrgName:     "Tangerine Bank (CA)",
		BalanceDate: now.AddDate(0, -9, 0).Unix(),
	}}, nil, now)

	if strings.Contains(msg, "1970") {
		t.Errorf("a missing transaction date must not render as an epoch date; got:\n%s", msg)
	}
	if !strings.Contains(msg, "no transactions") {
		t.Errorf("alert should say the account has no transactions; got:\n%s", msg)
	}
}
