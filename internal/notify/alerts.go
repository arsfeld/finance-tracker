package notify

import (
	"fmt"
	"strings"
	"time"

	"finance_tracker/internal/models"
)

// StaleConnectionAlert renders the warning sent when an included account has
// stopped syncing. It returns "" when there is nothing to report, which is the
// caller's signal to stay quiet.
//
// The trigger is a stale account rather than the presence of API errors: the
// bridge attaches a 45-day range advisory to every single response, so alerting
// on errors alone would fire on every sync forever. The errors are still
// included, because they carry the reason ("Auth required") the account went
// stale.
func StaleConnectionAlert(stale []models.StaleConnection, apiErrors []string, now time.Time) string {
	if len(stale) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d account(s) stopped syncing\n\n", len(stale))

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

	if reasons := connectionErrors(apiErrors); len(reasons) > 0 {
		b.WriteString("\nSimpleFin reported:\n")
		for _, e := range reasons {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}

	b.WriteString("\nSpending totals will read low until these connections are re-authorized.")
	return b.String()
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
