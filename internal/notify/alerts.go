package notify

import (
	"fmt"
	"strings"
	"time"

	"finance_tracker/internal/models"
)

// SyncHealthAlert renders the warning sent when an included account is not
// reporting properly. It returns "" when there is nothing to report, which is
// the caller's signal to stay quiet.
//
// Two distinct faults produce the same symptom of spending that reads too low,
// and both are covered here. An account goes stale when the connection stops
// refreshing at all. An account goes unreconciled when the connection keeps
// refreshing balances but stops delivering transactions — which the staleness
// check cannot see, because balance_date stays current throughout.
//
// The trigger is one of those two conditions rather than the presence of API
// errors: the bridge attaches a 45-day range advisory to every single response,
// so alerting on errors alone would fire on every sync forever. The errors are
// still quoted, because they carry the reason ("Auth required").
func SyncHealthAlert(
	stale []models.StaleConnection,
	drifted []models.UnreconciledAccount,
	apiErrors []string,
	now time.Time,
) string {
	if len(stale) == 0 && len(drifted) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(headline(stale, drifted) + "\n\n")

	for _, c := range stale {
		lastSync := time.Unix(c.BalanceDate, 0).UTC()
		fmt.Fprintf(&b, "- %s (%s): last refreshed %s, %d days ago",
			c.Name, c.OrgName, lastSync.Format("Jan 2, 2006"), daysSince(lastSync, now))
		if c.LastTransaction > 0 {
			lastTxn := time.Unix(c.LastTransaction, 0).UTC()
			fmt.Fprintf(&b, "; newest transaction %s, %d days ago\n",
				lastTxn.Format("Jan 2, 2006"), daysSince(lastTxn, now))
		} else {
			b.WriteString("; no transactions on record\n")
		}
	}

	for _, u := range drifted {
		fmt.Fprintf(&b, "- %s (%s): balance %.2f has moved %.2f more than its transactions account for\n",
			u.Name, u.OrgName, u.Balance, u.Unexplained)
	}

	if reasons := connectionErrors(apiErrors); len(reasons) > 0 {
		b.WriteString("\nSimpleFin reported:\n")
		for _, e := range reasons {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}

	b.WriteString("\nSpending totals will read low until these accounts report in full.")
	return b.String()
}

// headline names the dominant fault so the first line of the push notification
// says what is actually wrong.
func headline(stale []models.StaleConnection, drifted []models.UnreconciledAccount) string {
	switch {
	case len(stale) > 0 && len(drifted) > 0:
		return fmt.Sprintf("%d account(s) stopped syncing, %d reporting incomplete transactions",
			len(stale), len(drifted))
	case len(stale) > 0:
		return fmt.Sprintf("%d account(s) stopped syncing", len(stale))
	default:
		return fmt.Sprintf("%d account(s) reporting incomplete transactions", len(drifted))
	}
}

// connectionErrors keeps the API errors that describe a connection and drops
// advisories about our own request, which every response carries.
func connectionErrors(apiErrors []string) []string {
	var out []string
	for _, e := range apiErrors {
		if strings.Contains(e, "Connection to") || strings.Contains(e, "Auth required") {
			out = append(out, e)
		}
	}
	return out
}

func daysSince(from, to time.Time) int {
	d := int(to.Sub(from).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}
