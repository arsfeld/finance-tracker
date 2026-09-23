package ledger

import (
	"testing"

	"finance_tracker/internal/models"
)

func TestIsCreditCardNameRecognizesCards(t *testing.T) {
	for _, name := range []string{
		"TD AEROPLAN VISA INFINITE (4520)",
		"Money-Back World Mastercard®* (4268)",
	} {
		if !IsCreditCardName(name) {
			t.Errorf("%q is a credit card", name)
		}
	}
}

// A line of credit matches the "credit" keyword, but its balance is a loan.
// Treating it as a card would count every draw on it as spending.
func TestIsCreditCardNameRejectsLinesOfCreditAndBankAccounts(t *testing.T) {
	for _, name := range []string{
		"LINE OF CREDIT UNSECURED (3871)",
		"Tangerine Line of Credit (5058)",
		"Tangerine Chequing Account (2106)",
		"TD EVERY DAY SAVINGS ACCOUNT (2625)",
		"Tangerine Tax-Free Savings Account (9959)",
	} {
		if IsCreditCardName(name) {
			t.Errorf("%q is not a credit card", name)
		}
	}
}

// Re-authorizing TD issued a new account ID for the same Visa. The key must be
// identical for both so the card keeps one balance history.
func TestCardKeyIsStableAcrossAccountIDs(t *testing.T) {
	oldKey, ok1 := CardKey("TD Canada Trust", "TD AEROPLAN VISA INFINITE (4520)", "ACT-1820c169")
	newKey, ok2 := CardKey("TD Canada Trust", "TD AEROPLAN VISA INFINITE (4520)", "ACT-ca03b759")

	if !ok1 || !ok2 {
		t.Fatal("both names carry a last-4 and should produce a key")
	}
	if oldKey != "TD Canada Trust|4520" || newKey != oldKey {
		t.Errorf("expected both keys to be %q, got %q and %q", "TD Canada Trust|4520", oldKey, newKey)
	}
}

func TestCardKeyFallsBackToAccountIDWithoutLastFour(t *testing.T) {
	key, ok := CardKey("Some Bank", "Rewards Card", "ACT-xyz")

	if ok {
		t.Error("a name without (NNNN) cannot be matched across re-auths")
	}
	if key != "ACT-xyz" {
		t.Errorf("expected the account ID as the key, got %q", key)
	}
}

func card(id, key, firstSeen string) models.DBAccount {
	return models.DBAccount{ID: id, CardKey: key, IsCreditCard: true, FirstSeenAt: firstSeen}
}

// The old TD ID kept reporting a frozen balance with a fresh balance_date after
// re-auth, so recency of the balance cannot pick the live account. The account
// first seen most recently is the one the bank is reporting for.
func TestSupersededAccountsKeepsTheNewestFirstSeen(t *testing.T) {
	accounts := []models.DBAccount{
		card("ACT-old", "TD Canada Trust|4520", "2026-03-17 02:27:47"),
		card("ACT-new", "TD Canada Trust|4520", "2026-09-11 23:41:21"),
		{ID: "ACT-chq", CardKey: "Tangerine Bank (CA)|2106", FirstSeenAt: "2026-09-11 23:41:21"},
	}

	superseded := SupersededAccounts(accounts)

	if !superseded["ACT-old"] {
		t.Error("the old TD account is superseded")
	}
	if superseded["ACT-new"] || superseded["ACT-chq"] {
		t.Errorf("only the old TD account is superseded, got %v", superseded)
	}
	if got := CurrentAccounts(accounts)["TD Canada Trust|4520"].ID; got != "ACT-new" {
		t.Errorf("expected ACT-new to be current, got %q", got)
	}
}

// Inclusion follows an account's identity whatever its type, so every account
// needs a current row, not just cards. An account with no key is its own
// identity.
func TestCurrentByKeyCoversEveryAccount(t *testing.T) {
	accounts := []models.DBAccount{
		{ID: "ACT-loc-old", CardKey: "TD Canada Trust|3871", FirstSeenAt: "2026-07-25 10:00:00"},
		{ID: "ACT-loc-new", CardKey: "TD Canada Trust|3871", FirstSeenAt: "2026-09-11 23:41:21"},
		{ID: "ACT-nokey", FirstSeenAt: "2026-03-17 02:27:47"},
	}

	current := CurrentByKey(accounts)

	if got := current["TD Canada Trust|3871"].ID; got != "ACT-loc-new" {
		t.Errorf("expected the newer line of credit to be current, got %q", got)
	}
	if got := current["ACT-nokey"].ID; got != "ACT-nokey" {
		t.Errorf("an account without a key is keyed by its ID, got %q", got)
	}
	if len(current) != 2 {
		t.Errorf("expected 2 identities, got %d: %v", len(current), current)
	}
}
