// Package ledger turns credit card balances into spending figures.
//
// A card's transaction feed can stop at any time while its balance keeps
// refreshing, so the balance is the source of truth for how much was spent and
// transactions only explain where it went.
package ledger

import (
	"regexp"
	"strings"

	"finance_tracker/internal/models"
)

var creditCardKeywords = []string{
	"credit", "card", "visa", "mastercard", "amex", "american express", "discover", "rewards",
}

// IsCreditCardName reports whether an account name looks like a credit card.
// SimpleFin carries no account type, so the name is all there is. A line of
// credit matches "credit", but its balance is a loan rather than card spending.
func IsCreditCardName(name string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "line of credit") {
		return false
	}
	for _, k := range creditCardKeywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

var lastFour = regexp.MustCompile(`\((\d{4})\)\s*$`)

// CardKey identifies a card across the account IDs SimpleFin assigns it.
// Re-authorizing a connection issues new IDs for the same cards, so the key is
// built from the institution and the last four digits in the name. ok is false
// when no last-4 can be parsed: the account ID is returned so the account still
// gets a ledger of its own, it just cannot be merged automatically.
func CardKey(orgName, accountName, accountID string) (key string, ok bool) {
	m := lastFour.FindStringSubmatch(accountName)
	if m == nil || orgName == "" {
		return accountID, false
	}
	return orgName + "|" + m[1], true
}

// CurrentAccounts maps each card key to the account reporting for it now: the
// one first seen most recently. After a re-auth the old ID is dead even when
// SimpleFin keeps returning it with a fresh balance_date and a frozen balance,
// so balance recency cannot be trusted to pick it.
func CurrentAccounts(accounts []models.DBAccount) map[string]models.DBAccount {
	current := make(map[string]models.DBAccount)
	for _, a := range accounts {
		if !a.IsCreditCard || a.CardKey == "" {
			continue
		}
		cur, seen := current[a.CardKey]
		if !seen || a.FirstSeenAt > cur.FirstSeenAt || (a.FirstSeenAt == cur.FirstSeenAt && a.ID > cur.ID) {
			current[a.CardKey] = a
		}
	}
	return current
}

// SupersededAccounts returns the IDs of card accounts that another account with
// the same card key has replaced.
func SupersededAccounts(accounts []models.DBAccount) map[string]bool {
	current := CurrentAccounts(accounts)
	superseded := make(map[string]bool)
	for _, a := range accounts {
		if !a.IsCreditCard || a.CardKey == "" {
			continue
		}
		if current[a.CardKey].ID != a.ID {
			superseded[a.ID] = true
		}
	}
	return superseded
}
