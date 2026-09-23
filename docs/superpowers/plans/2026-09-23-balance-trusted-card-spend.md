# Balance-Trusted Card Spending Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Measure credit card spending from card balances (debt going up means spending; debt going down means a payment), so the AI analysis stays correct when the transaction feed breaks. Card payments are never counted, and the analysis covers credit cards only.

**Architecture:** A new pure package `internal/ledger` identifies cards across SimpleFin account-ID churn, detects payments from both sides, and turns an ordered list of balance snapshots into per-billing-period spending. A new `balance_snapshots` table (`internal/store/snapshots.go`) records one reading per card per sync. It is seeded at startup from balances already stored. `internal/llm/analyze.go` is rebuilt around the ledger output. `cmd/promptdump` prints the prompt for a DB copy, so prompts can be tuned on real data without calling the LLM.

**Tech Stack:** Go 1.26, SQLite (`modernc.org/sqlite`), goose migrations (embedded SQL), zerolog, the standard `testing` package (table-less, one behavior per test, matching the existing `*_test.go` style).

**Spec:** `docs/superpowers/specs/2026-09-23-balance-trusted-card-spend-design.md`

## Global Constraints

- Run every Go command through devenv: `devenv shell -- go test ./...`. If `$DEVENV_PROFILE` is already set in your shell, plain `go test ./...` is fine.
- Balances are stored as SimpleFin reports them: **negative = debt**. The ledger works with `debt = −balance`.
- Card key format: `org_name + "|" + last4`, e.g. `TD Canada Trust|4520`.
- Card-side payment category: exactly `Payment`.
- Payment match window: **±3 days**; amount tolerance **±$0.01**.
- `NotItemized` is reported only when **above $50**.
- The budget lower-bound note is shown when the current period is **under 80% itemized**.
- Settings key for payment patterns: `card_payment_patterns`. Default value: `{"TD Canada Trust|4520": ["Bill Payment - TD VISA", "TFR-TO C/C", "TFR-A C/C"]}`.
- Pattern matching: case-insensitive substring after collapsing runs of whitespace on both sides.
- Snapshot sources: `sync`, `seed_anchor`, `seed_current`.
- A card's **current account** is the one with the newest `first_seen_at` for its `card_key` (ties broken by the larger ID). All others are **superseded**.
- Code comments explain *why*, in the style of the existing code (see `internal/store/accounts.go`). No emoji in logs beyond what the file already uses.
- Commit after every task. Never add attribution lines to commit messages.

## File Map

| File | Responsibility |
|---|---|
| `internal/models/models.go` | + `DBAccount.IsCreditCard`, `DBAccount.CardKey`, new `BalanceSnapshot` |
| `internal/ledger/card.go` (new) | card name classification, card key, current/superseded accounts |
| `internal/ledger/payments.go` (new) | payment patterns, payment detection and dedup |
| `internal/ledger/spend.go` (new) | intervals, per-period spending, combining cards |
| `internal/ledger/build.go` (new) | group DB rows into cards and build the report |
| `internal/database/migrations/006_balance_snapshots.sql` (new) | columns + table |
| `internal/store/accounts.go` | classify on insert, backfill, `Update` with `AccountPatch` |
| `internal/store/snapshots.go` (new) | record, seed, list balance snapshots |
| `internal/store/bootstrap.go` (new) | `InitCardLedger` startup step |
| `internal/store/transactions.go` | `GetForPeriodAllAccounts` |
| `internal/llm/analyze.go` | new `PromptInput`, prompt structure, lag wording, `CyclePeriods` |
| `internal/api/analysis_run.go` | `BuildPrompt` via the ledger |
| `internal/api/sync.go` | record snapshots; ignore superseded accounts in alerts |
| `internal/api/accounts.go` | PATCH accepts `is_credit_card`, `card_key` |
| `internal/api/payment_patterns.go` (new) | GET/PUT payment patterns |
| `internal/api/helpers.go` | `filterHealth` helper |
| `internal/server/server.go` | construct new stores, routes |
| `cmd/server/main.go` | call `InitCardLedger` |
| `cmd/promptdump/main.go` (new) | print the prompt for a DB |

---

### Task 1: Card identity in `ledger`

**Files:**
- Modify: `internal/models/models.go` (the `DBAccount` struct, plus a new type at the end of the file)
- Create: `internal/ledger/card.go`
- Test: `internal/ledger/card_test.go`

**Interfaces:**
- Produces:
  - `models.DBAccount` gains `IsCreditCard bool \`json:"is_credit_card"\`` and `CardKey string \`json:"card_key"\``
  - `models.BalanceSnapshot{CardKey, AccountID string; Balance float64; BalanceDate int64; Source string}`
  - `ledger.IsCreditCardName(name string) bool`
  - `ledger.CardKey(orgName, accountName, accountID string) (key string, ok bool)`
  - `ledger.CurrentAccounts(accounts []models.DBAccount) map[string]models.DBAccount` (keyed by card key)
  - `ledger.SupersededAccounts(accounts []models.DBAccount) map[string]bool` (keyed by account ID)

- [ ] **Step 1: Add the model fields**

In `internal/models/models.go`, replace the `DBAccount` struct with:

```go
// DBAccount represents an account row in the database.
type DBAccount struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Balance      float64 `json:"balance"`
	BalanceDate  int64   `json:"balance_date"`
	Currency     string  `json:"currency"`
	OrgName      string  `json:"org_name"`
	OrgDomain    string  `json:"org_domain"`
	IsIncluded   bool    `json:"is_included"`
	IsCreditCard bool    `json:"is_credit_card"`
	CardKey      string  `json:"card_key"`
	FirstSeenAt  string  `json:"first_seen_at"`
	UpdatedAt    string  `json:"updated_at"`
}
```

Append to the end of the file:

```go
// BalanceSnapshot is one reading of a card's balance. Card spending is measured
// between consecutive snapshots, so the balance stays the source of truth even
// when the transaction feed stops delivering.
type BalanceSnapshot struct {
	CardKey     string  `json:"card_key"`
	AccountID   string  `json:"account_id"`
	Balance     float64 `json:"balance"` // as reported by SimpleFin; negative = debt
	BalanceDate int64   `json:"balance_date"`
	Source      string  `json:"source"`
}
```

- [ ] **Step 2: Write the failing tests**

Create `internal/ledger/card_test.go`:

```go
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
```

- [ ] **Step 3: Run the tests to make sure they fail**

Run: `devenv shell -- go test ./internal/ledger/ -v`
Expected: build failure, `undefined: IsCreditCardName` (and similar).

- [ ] **Step 4: Implement**

Create `internal/ledger/card.go`:

```go
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
```

- [ ] **Step 5: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/ledger/ -v && devenv shell -- go build ./...`
Expected: all PASS, build succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/models/models.go internal/ledger/card.go internal/ledger/card_test.go
git commit -m "feat(ledger): identify credit cards across account-ID churn"
```

---

### Task 2: Migration and account classification

**Files:**
- Create: `internal/database/migrations/006_balance_snapshots.sql`
- Modify: `internal/store/accounts.go` (`Upsert`, `List`, `GetByID`; replace `UpdateInclusion`; add `BackfillCardIdentity`)
- Modify: `internal/api/accounts.go` (`UpdateInclusion` handler → `Update`)
- Modify: `internal/server/server.go:73` (route handler name)
- Test: `internal/store/accounts_test.go` (append)

**Interfaces:**
- Consumes: `ledger.IsCreditCardName`, `ledger.CardKey` (Task 1)
- Produces:
  - `store.AccountPatch{IsIncluded *bool \`json:"is_included"\`; IsCreditCard *bool \`json:"is_credit_card"\`; CardKey *string \`json:"card_key"\`}`
  - `(*AccountStore).Update(ctx, id string, p AccountPatch) error`
  - `(*AccountStore).BackfillCardIdentity(ctx) (int, error)`
  - `List`/`GetByID` return `IsCreditCard` and `CardKey`
  - `(*AccountHandler).Update` HTTP handler

- [ ] **Step 1: Write the migration**

Create `internal/database/migrations/006_balance_snapshots.sql`:

```sql
-- +goose Up
-- Card spending is measured from balances, not transactions: the transaction
-- feed can stop while the balance keeps refreshing. That needs to know which
-- accounts are cards, a key that survives SimpleFin issuing new account IDs on
-- re-auth, and a history of balances to take differences of.
ALTER TABLE accounts ADD COLUMN is_credit_card INTEGER NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN card_key TEXT;

CREATE TABLE balance_snapshots (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    card_key     TEXT    NOT NULL,
    account_id   TEXT    NOT NULL,
    balance      REAL    NOT NULL,
    balance_date INTEGER NOT NULL,
    source       TEXT    NOT NULL,
    created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE (card_key, balance_date)
);

-- +goose Down
DROP TABLE balance_snapshots;
ALTER TABLE accounts DROP COLUMN card_key;
ALTER TABLE accounts DROP COLUMN is_credit_card;
```

- [ ] **Step 2: Write the failing tests**

Append to `internal/store/accounts_test.go`:

```go
func TestUpsertClassifiesNewAccounts(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust", IsIncluded: true,
	})
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-loc", Name: "LINE OF CREDIT UNSECURED (3871)", OrgName: "TD Canada Trust", IsIncluded: true,
	})

	td, _ := accts.GetByID(ctx, "ACT-td")
	loc, _ := accts.GetByID(ctx, "ACT-loc")

	if !td.IsCreditCard || td.CardKey != "TD Canada Trust|4520" {
		t.Errorf("TD Visa should be a card keyed TD Canada Trust|4520, got %+v", td)
	}
	if loc.IsCreditCard {
		t.Errorf("a line of credit is not a card, got %+v", loc)
	}
}

// Classification is a guess from the name, so a hand correction has to survive
// the next sync rather than being re-guessed away.
func TestUpsertKeepsHandEditedCardFields(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	acct := models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust", IsIncluded: true,
	}
	seedAccountFull(t, accts, acct)

	notCard, key := false, "manual-key"
	if err := accts.Update(ctx, "ACT-td", AccountPatch{IsCreditCard: &notCard, CardKey: &key}); err != nil {
		t.Fatalf("update: %v", err)
	}
	seedAccountFull(t, accts, acct)

	got, _ := accts.GetByID(ctx, "ACT-td")
	if got.IsCreditCard || got.CardKey != "manual-key" {
		t.Errorf("hand-edited fields were overwritten by the sync: %+v", got)
	}
}

func TestBackfillCardIdentityClassifiesLegacyRows(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust", IsIncluded: true,
	})
	if _, err := db.Write.Exec(`UPDATE accounts SET card_key = NULL, is_credit_card = 0`); err != nil {
		t.Fatalf("simulate legacy row: %v", err)
	}

	n, err := accts.BackfillCardIdentity(ctx)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	again, _ := accts.BackfillCardIdentity(ctx)

	got, _ := accts.GetByID(ctx, "ACT-td")
	if n != 1 || again != 0 {
		t.Errorf("expected 1 row backfilled then 0, got %d then %d", n, again)
	}
	if !got.IsCreditCard || got.CardKey != "TD Canada Trust|4520" {
		t.Errorf("legacy row not classified: %+v", got)
	}
}
```

- [ ] **Step 3: Run the tests to make sure they fail**

Run: `devenv shell -- go test ./internal/store/ -run 'Classif|HandEdited|Backfill' -v`
Expected: build failure, `accts.Update undefined` / `undefined: AccountPatch`.

- [ ] **Step 4: Implement the store changes**

In `internal/store/accounts.go`, update the imports:

```go
import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/models"
)
```

Replace `Upsert`:

```go
func (s *AccountStore) Upsert(ctx context.Context, acct models.DBAccount) error {
	// Card fields are set on insert only: they are a guess from the name, and a
	// hand correction must survive every later sync.
	isCard := ledger.IsCreditCardName(acct.Name)
	cardKey, ok := ledger.CardKey(acct.OrgName, acct.Name, acct.ID)
	if isCard && !ok {
		log.Warn().Str("account", acct.Name).Str("id", acct.ID).
			Msg("Card name has no last-4; its balance history will not follow it across re-authorizations")
	}

	_, err := s.write.ExecContext(ctx, `
		INSERT INTO accounts (id, name, balance, balance_date, currency, org_name, org_domain, is_included,
			anchor_balance, anchor_balance_date, is_credit_card, card_key, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			balance = excluded.balance,
			balance_date = excluded.balance_date,
			currency = excluded.currency,
			org_name = excluded.org_name,
			org_domain = excluded.org_domain,
			updated_at = datetime('now')`,
		acct.ID, acct.Name, acct.Balance, acct.BalanceDate, acct.Currency, acct.OrgName, acct.OrgDomain, acct.IsIncluded,
		acct.Balance, acct.BalanceDate, isCard, cardKey,
	)
	return err
}
```

In both `List` and `GetByID`, change the column list to:

```sql
SELECT id, name, balance, balance_date, currency, org_name, org_domain, is_included,
	is_credit_card, COALESCE(card_key, ''), first_seen_at, updated_at
```

and the `Scan` calls to:

```go
Scan(&a.ID, &a.Name, &a.Balance, &a.BalanceDate, &a.Currency, &a.OrgName, &a.OrgDomain, &a.IsIncluded,
	&a.IsCreditCard, &a.CardKey, &a.FirstSeenAt, &a.UpdatedAt)
```

Replace `UpdateInclusion` with:

```go
// AccountPatch changes the hand-editable fields of an account. Nil fields are
// left as they are.
type AccountPatch struct {
	IsIncluded   *bool   `json:"is_included"`
	IsCreditCard *bool   `json:"is_credit_card"`
	CardKey      *string `json:"card_key"`
}

func (s *AccountStore) Update(ctx context.Context, id string, p AccountPatch) error {
	var sets []string
	var args []any
	if p.IsIncluded != nil {
		sets, args = append(sets, "is_included = ?"), append(args, *p.IsIncluded)
	}
	if p.IsCreditCard != nil {
		sets, args = append(sets, "is_credit_card = ?"), append(args, *p.IsCreditCard)
	}
	if p.CardKey != nil {
		sets, args = append(sets, "card_key = ?"), append(args, *p.CardKey)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)
	_, err := s.write.ExecContext(ctx, `UPDATE accounts SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	return err
}

// BackfillCardIdentity classifies accounts stored before cards were tracked.
// Rows that already have a card key are left alone, so hand edits survive
// restarts.
func (s *AccountStore) BackfillCardIdentity(ctx context.Context) (int, error) {
	type row struct{ id, name, org string }
	rows, err := s.write.QueryContext(ctx, `SELECT id, name, org_name FROM accounts WHERE card_key IS NULL`)
	if err != nil {
		return 0, err
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.org); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, r)
	}
	// The write pool has a single connection, so the cursor must be closed
	// before the updates can run.
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, r := range pending {
		key, _ := ledger.CardKey(r.org, r.name, r.id)
		if _, err := s.write.ExecContext(ctx,
			`UPDATE accounts SET is_credit_card = ?, card_key = ? WHERE id = ? AND card_key IS NULL`,
			ledger.IsCreditCardName(r.name), key, r.id); err != nil {
			return 0, err
		}
	}
	return len(pending), nil
}
```

- [ ] **Step 5: Update the HTTP handler**

In `internal/api/accounts.go`, replace `UpdateInclusion` with:

```go
func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var patch store.AccountPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	if patch.CardKey != nil && *patch.CardKey == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "card_key cannot be empty")
		return
	}

	if err := h.store.Update(r.Context(), id, patch); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, map[string]string{"status": "ok"})
}
```

In `internal/server/server.go`, change `acctHandler.UpdateInclusion` to `acctHandler.Update`.

- [ ] **Step 6: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/store/ ./internal/api/ -v && devenv shell -- go build ./...`
Expected: all PASS, including the existing stale/drift tests.

- [ ] **Step 7: Commit**

```bash
git add internal/database/migrations/006_balance_snapshots.sql internal/store/accounts.go internal/store/accounts_test.go internal/api/accounts.go internal/server/server.go
git commit -m "feat(accounts): classify credit cards and key them across re-auth"
```

---

### Task 3: Balance snapshot store

**Files:**
- Create: `internal/store/snapshots.go`
- Test: `internal/store/snapshots_test.go`

**Interfaces:**
- Consumes: `ledger.CurrentAccounts` (Task 1); `models.BalanceSnapshot` (Task 1); account columns from Task 2
- Produces:
  - `store.SnapshotSourceSync`, `store.SnapshotSourceSeedAnchor`, `store.SnapshotSourceSeedCurrent` (string consts)
  - `store.NewSnapshotStore(read, write *sql.DB) *SnapshotStore`
  - `(*SnapshotStore).Record(ctx, models.BalanceSnapshot) (bool, error)`
  - `(*SnapshotStore).RecordCurrent(ctx) (int, error)`: after each sync
  - `(*SnapshotStore).Seed(ctx) (int, error)`: at startup
  - `(*SnapshotStore).ListByCard(ctx) (map[string][]models.BalanceSnapshot, error)`: ascending by date

- [ ] **Step 1: Write the failing tests**

Create `internal/store/snapshots_test.go`:

```go
package store

import (
	"context"
	"testing"
	"time"

	"finance_tracker/internal/database"
	"finance_tracker/internal/models"
)

const tdKey = "TD Canada Trust|4520"

func at(month time.Month, day int) int64 {
	return time.Date(2026, month, day, 12, 0, 0, 0, time.UTC).Unix()
}

// seedCard creates a card account first seen with anchor, then reporting current.
func seedCard(t *testing.T, db *database.DB, accts *AccountStore, id, firstSeen string,
	anchor float64, anchorDate int64, current float64, currentDate int64) {
	t.Helper()
	acct := models.DBAccount{
		ID: id, Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust", IsIncluded: true,
		Balance: anchor, BalanceDate: anchorDate,
	}
	seedAccountFull(t, accts, acct)
	acct.Balance, acct.BalanceDate = current, currentDate
	seedAccountFull(t, accts, acct)
	if _, err := db.Write.Exec(`UPDATE accounts SET first_seen_at = ? WHERE id = ?`, firstSeen, id); err != nil {
		t.Fatalf("set first_seen_at: %v", err)
	}
}

// seedProductionTD reproduces the TD Visa as it stood on galactica on
// 2026-09-23: the old ID frozen after re-auth, the new ID live.
func seedProductionTD(t *testing.T) (*database.DB, *AccountStore, *SnapshotStore) {
	t.Helper()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	seedCard(t, db, accts, "ACT-old", "2026-03-17 02:27:47",
		-12124.34, at(time.August, 31), -12124.34, at(time.September, 3))
	seedCard(t, db, accts, "ACT-new", "2026-09-11 23:41:21",
		-7129.23, at(time.September, 11), -8024.04, at(time.September, 23))
	return db, accts, NewSnapshotStore(db.Read, db.Write)
}

func balances(snaps []models.BalanceSnapshot) []float64 {
	var out []float64
	for _, s := range snaps {
		out = append(out, s.Balance)
	}
	return out
}

// The old account's Sep 3 reading repeats the Aug 31 balance after a $7,340
// payment on Sep 1: TD froze the balance while advancing its date. Seeding it
// would book the payment as spending, so only the three real readings survive.
func TestSeedReproducesProductionTDHistory(t *testing.T) {
	ctx := context.Background()
	_, _, snaps := seedProductionTD(t)

	n, err := snaps.Seed(ctx)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	byCard, err := snaps.ListByCard(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	got := balances(byCard[tdKey])
	want := []float64{-12124.34, -7129.23, -8024.04}
	if n != 3 || len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("expected %v (3 inserted), got %v (%d inserted)", want, got, n)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	ctx := context.Background()
	_, _, snaps := seedProductionTD(t)

	snaps.Seed(ctx)
	n, err := snaps.Seed(ctx)

	if err != nil || n != 0 {
		t.Errorf("a second seed must insert nothing, got %d (err %v)", n, err)
	}
}

// A quiet card reports the same balance with a newer date on every sync. The
// reading carries no information, and skipping it is what keeps a frozen feed
// from inventing spending.
func TestRecordCurrentSkipsUnchangedBalance(t *testing.T) {
	ctx := context.Background()
	_, accts, snaps := seedProductionTD(t)
	snaps.Seed(ctx)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-new", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		Balance: -8024.04, BalanceDate: at(time.September, 25),
	})
	unchanged, _ := snaps.RecordCurrent(ctx)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-new", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		Balance: -8300.00, BalanceDate: at(time.September, 27),
	})
	changed, _ := snaps.RecordCurrent(ctx)

	if unchanged != 0 || changed != 1 {
		t.Errorf("expected 0 rows for an unchanged balance and 1 for a changed one, got %d and %d", unchanged, changed)
	}
}

// If the dead ID ever reports again, its frozen -12,124.34 must not be
// interleaved into the live card's history as a $4k swing.
func TestRecordCurrentIgnoresSupersededAccount(t *testing.T) {
	ctx := context.Background()
	_, accts, snaps := seedProductionTD(t)
	snaps.Seed(ctx)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-old", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		Balance: -12124.34, BalanceDate: at(time.September, 29),
	})
	n, err := snaps.RecordCurrent(ctx)

	if err != nil || n != 0 {
		t.Errorf("the superseded account must not be recorded, got %d rows (err %v)", n, err)
	}
}
```

- [ ] **Step 2: Run the tests to make sure they fail**

Run: `devenv shell -- go test ./internal/store/ -run 'Seed|RecordCurrent' -v`
Expected: build failure, `undefined: NewSnapshotStore`.

- [ ] **Step 3: Implement**

Create `internal/store/snapshots.go`:

```go
package store

import (
	"context"
	"database/sql"
	"math"
	"sort"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/models"
)

const (
	SnapshotSourceSync        = "sync"
	SnapshotSourceSeedAnchor  = "seed_anchor"
	SnapshotSourceSeedCurrent = "seed_current"
)

// SnapshotStore keeps the history of card balances that spending is measured
// from.
type SnapshotStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewSnapshotStore(read, write *sql.DB) *SnapshotStore {
	return &SnapshotStore{read: read, write: write}
}

// Record stores a balance reading unless it adds nothing: a reading no newer
// than the card's latest snapshot, or one whose balance has not moved. TD has
// reported a frozen balance with an advancing date after a payment had posted;
// storing it would book that payment as spending.
func (s *SnapshotStore) Record(ctx context.Context, snap models.BalanceSnapshot) (bool, error) {
	var lastBalance float64
	var lastDate int64
	err := s.write.QueryRowContext(ctx, `
		SELECT balance, balance_date FROM balance_snapshots
		WHERE card_key = ? ORDER BY balance_date DESC LIMIT 1`, snap.CardKey).Scan(&lastBalance, &lastDate)
	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return false, err
	case snap.BalanceDate <= lastDate, math.Abs(snap.Balance-lastBalance) < 0.005:
		return false, nil
	}

	res, err := s.write.ExecContext(ctx, `
		INSERT OR IGNORE INTO balance_snapshots (card_key, account_id, balance, balance_date, source)
		VALUES (?, ?, ?, ?, ?)`,
		snap.CardKey, snap.AccountID, snap.Balance, snap.BalanceDate, snap.Source)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// RecordCurrent records each card's current balance. It runs after every sync.
func (s *SnapshotStore) RecordCurrent(ctx context.Context) (int, error) {
	return s.recordReadings(ctx, false)
}

// Seed builds the starting history from what the accounts table already holds:
// every card account's anchor (the first balance ever seen for it) and each
// card's current balance. Anything older was never recorded. The card's
// transactions cannot rebuild it, because the feed was already missing payments
// months before it stopped.
func (s *SnapshotStore) Seed(ctx context.Context) (int, error) {
	return s.recordReadings(ctx, true)
}

type cardAccount struct {
	models.DBAccount
	anchorBalance float64
	anchorDate    int64
}

func (s *SnapshotStore) recordReadings(ctx context.Context, withAnchors bool) (int, error) {
	cards, err := s.cardAccounts(ctx)
	if err != nil {
		return 0, err
	}

	plain := make([]models.DBAccount, len(cards))
	for i, c := range cards {
		plain[i] = c.DBAccount
	}
	current := ledger.CurrentAccounts(plain)

	var readings []models.BalanceSnapshot
	if withAnchors {
		for _, c := range cards {
			if c.anchorDate > 0 {
				readings = append(readings, models.BalanceSnapshot{
					CardKey: c.CardKey, AccountID: c.ID,
					Balance: c.anchorBalance, BalanceDate: c.anchorDate, Source: SnapshotSourceSeedAnchor,
				})
			}
		}
	}
	source := SnapshotSourceSync
	if withAnchors {
		source = SnapshotSourceSeedCurrent
	}
	for _, a := range current {
		if a.BalanceDate > 0 {
			readings = append(readings, models.BalanceSnapshot{
				CardKey: a.CardKey, AccountID: a.ID, Balance: a.Balance, BalanceDate: a.BalanceDate, Source: source,
			})
		}
	}

	// Record's rules compare each reading with the latest one stored, so
	// readings have to arrive in date order.
	sort.SliceStable(readings, func(i, j int) bool { return readings[i].BalanceDate < readings[j].BalanceDate })

	recorded := 0
	for _, r := range readings {
		ok, err := s.Record(ctx, r)
		if err != nil {
			return recorded, err
		}
		if ok {
			recorded++
		}
	}
	return recorded, nil
}

func (s *SnapshotStore) cardAccounts(ctx context.Context) ([]cardAccount, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, name, card_key, balance, COALESCE(balance_date, 0),
			COALESCE(anchor_balance, 0), COALESCE(anchor_balance_date, 0), first_seen_at
		FROM accounts
		WHERE is_credit_card = 1 AND card_key IS NOT NULL AND card_key != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []cardAccount
	for rows.Next() {
		var c cardAccount
		if err := rows.Scan(&c.ID, &c.Name, &c.CardKey, &c.Balance, &c.BalanceDate,
			&c.anchorBalance, &c.anchorDate, &c.FirstSeenAt); err != nil {
			return nil, err
		}
		c.IsCreditCard = true
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

// ListByCard returns every snapshot grouped by card key, oldest first.
func (s *SnapshotStore) ListByCard(ctx context.Context) (map[string][]models.BalanceSnapshot, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT card_key, account_id, balance, balance_date, source
		FROM balance_snapshots ORDER BY card_key, balance_date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byCard := make(map[string][]models.BalanceSnapshot)
	for rows.Next() {
		var s models.BalanceSnapshot
		if err := rows.Scan(&s.CardKey, &s.AccountID, &s.Balance, &s.BalanceDate, &s.Source); err != nil {
			return nil, err
		}
		byCard[s.CardKey] = append(byCard[s.CardKey], s)
	}
	return byCard, rows.Err()
}
```

- [ ] **Step 4: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/store/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/snapshots.go internal/store/snapshots_test.go
git commit -m "feat(store): record card balance snapshots and seed their history"
```

---

### Task 4: Payment detection

**Files:**
- Create: `internal/ledger/payments.go`
- Test: `internal/ledger/payments_test.go`

**Interfaces:**
- Produces:
  - `ledger.PaymentCategory = "Payment"`
  - `ledger.PaymentMatchWindow = 3 * 24 * time.Hour`
  - `ledger.PaymentPatternsSettingKey = "card_payment_patterns"`
  - `ledger.DefaultPaymentPatterns map[string][]string`
  - `ledger.ParsePaymentPatterns(raw string) (map[string][]string, error)` (`""` → defaults)
  - `ledger.Payment{Amount float64; At int64}` (Amount is positive)
  - `ledger.MatchesPaymentPattern(description string, patterns []string) bool`
  - `ledger.DetectPayments(cardTxns, payingTxns []models.DBTransaction, patterns []string) []Payment` (sorted by `At`)

- [ ] **Step 1: Write the failing tests**

Create `internal/ledger/payments_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests to make sure they fail**

Run: `devenv shell -- go test ./internal/ledger/ -run 'Payment' -v`
Expected: build failure, `undefined: DefaultPaymentPatterns`.

- [ ] **Step 3: Implement**

Create `internal/ledger/payments.go`:

```go
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
```

- [ ] **Step 4: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/ledger/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ledger/payments.go internal/ledger/payments_test.go
git commit -m "feat(ledger): detect card payments from the card and the paying accounts"
```

---

### Task 5: Balance-derived spending per period

**Files:**
- Create: `internal/ledger/spend.go`
- Test: `internal/ledger/spend_test.go`

**Interfaces:**
- Consumes: `ledger.Payment` (Task 4), `models.BalanceSnapshot`, `models.BillingPeriod`
- Produces:
  - `ledger.NotItemizedMin = 50.0`
  - `ledger.Source` with `SourceBalance = "balance"` and `SourceItemizedOnly = "itemized_only"`
  - `ledger.Interval{From, To int64; Spend float64}`
  - `ledger.Intervals(snaps []models.BalanceSnapshot, payments []Payment) []Interval`
  - `ledger.PeriodSpend{Period models.BillingPeriod; Source Source; Total, BalanceSpend, CoveredDays, DailyBurn, Itemized, NotItemized float64}`
  - `ledger.PeriodBounds(p models.BillingPeriod) (from, to int64)`: half-open `[from, to)`
  - `ledger.EffectiveDate(t models.DBTransaction) int64`
  - `ledger.CardPeriods(periods []models.BillingPeriod, snaps []models.BalanceSnapshot, payments []Payment, coverageEnd int64, charges []models.DBTransaction) []PeriodSpend`
  - `ledger.Combine(perCard [][]PeriodSpend) []PeriodSpend`

- [ ] **Step 1: Write the failing tests**

Create `internal/ledger/spend_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests to make sure they fail**

Run: `devenv shell -- go test ./internal/ledger/ -run 'Intervals|CardPeriods|Combine' -v`
Expected: build failure, `undefined: Intervals`.

- [ ] **Step 3: Implement**

Create `internal/ledger/spend.go`:

```go
package ledger

import (
	"time"

	"finance_tracker/internal/models"
)

// NotItemizedMin is the smallest gap between balance spending and itemized
// charges worth reporting. Below it the gap is the normal lag between a charge
// hitting the balance and posting.
const NotItemizedMin = 50.0

// Source says where a period's headline figure came from.
type Source string

const (
	SourceBalance      Source = "balance"
	SourceItemizedOnly Source = "itemized_only"
)

// Interval is the spending between two consecutive balance snapshots.
type Interval struct {
	From, To int64
	Spend    float64
}

// Intervals turns a card's snapshots into spending. Between two readings,
// spending is the growth in debt plus whatever was paid in between. A drop
// that known payments don't explain is itself a payment, so spending never
// goes negative. The card is paid in full monthly, and the feeds do miss
// payments.
func Intervals(snaps []models.BalanceSnapshot, payments []Payment) []Interval {
	var out []Interval
	for i := 1; i < len(snaps); i++ {
		s0, s1 := snaps[i-1], snaps[i]
		spend := -s1.Balance - -s0.Balance
		for _, p := range payments {
			if p.At > s0.BalanceDate && p.At <= s1.BalanceDate {
				spend += p.Amount
			}
		}
		if spend < 0 {
			spend = 0
		}
		out = append(out, Interval{From: s0.BalanceDate, To: s1.BalanceDate, Spend: spend})
	}
	return out
}

// PeriodSpend is one billing period's card spending.
type PeriodSpend struct {
	Period       models.BillingPeriod
	Source       Source
	Total        float64 // headline: balance spending plus itemized charges outside the covered window
	BalanceSpend float64
	CoveredDays  float64
	DailyBurn    float64
	Itemized     float64
	NotItemized  float64 // balance spending the transactions don't account for; 0 when under NotItemizedMin
}

// PeriodBounds returns a period as a half-open [from, to) range. A completed
// period's End is midnight of its last day, so the day itself is added back.
func PeriodBounds(p models.BillingPeriod) (from, to int64) {
	end := p.End
	if p.IsComplete {
		end = end.Add(24 * time.Hour)
	}
	return p.Start.Unix(), end.Unix()
}

// EffectiveDate is when a charge happened, falling back to when it posted.
func EffectiveDate(t models.DBTransaction) int64 {
	if t.TransactedAt != nil {
		return *t.TransactedAt
	}
	return t.Posted
}

func overlap(a0, a1, b0, b1 int64) int64 {
	lo, hi := max(a0, b0), min(a1, b1)
	if hi < lo {
		return 0
	}
	return hi - lo
}

// CardPeriods measures one card's spending in each billing period. Where the
// balance history covers the period, the balance decides the total, and an
// interval crossing a period boundary is split in proportion to time. Where it
// doesn't, the period falls back to its itemized charges and is labeled as
// such. coverageEnd is the card's latest balance_date: an unchanged balance is
// not stored as a snapshot but still counts as coverage.
func CardPeriods(periods []models.BillingPeriod, snaps []models.BalanceSnapshot, payments []Payment,
	coverageEnd int64, charges []models.DBTransaction) []PeriodSpend {
	intervals := Intervals(snaps, payments)
	var covFrom, covTo int64
	if len(snaps) >= 2 {
		covFrom = snaps[0].BalanceDate
		covTo = max(coverageEnd, snaps[len(snaps)-1].BalanceDate)
	}

	out := make([]PeriodSpend, len(periods))
	for i, p := range periods {
		from, to := PeriodBounds(p)
		ps := PeriodSpend{Period: p}

		for _, iv := range intervals {
			if ov := overlap(iv.From, iv.To, from, to); ov > 0 {
				ps.BalanceSpend += iv.Spend * float64(ov) / float64(iv.To-iv.From)
			}
		}

		coveredFrom, coveredTo := max(from, covFrom), min(to, covTo)
		covered := len(snaps) >= 2 && coveredTo > coveredFrom
		if covered {
			ps.CoveredDays = float64(coveredTo-coveredFrom) / 86400
		}

		var inside float64
		for _, t := range charges {
			d := EffectiveDate(t)
			if t.Amount >= 0 || d < from || d >= to {
				continue
			}
			ps.Itemized += -t.Amount
			if covered && d >= coveredFrom && d < coveredTo {
				inside += -t.Amount
			}
		}

		if covered {
			ps.Source = SourceBalance
			ps.Total = ps.BalanceSpend + (ps.Itemized - inside)
			ps.DailyBurn = ps.BalanceSpend / ps.CoveredDays
			if gap := ps.BalanceSpend - inside; gap > NotItemizedMin {
				ps.NotItemized = gap
			}
		} else {
			ps.Source = SourceItemizedOnly
			ps.Total = ps.Itemized
			if days := float64(to-from) / 86400; days > 0 {
				ps.DailyBurn = ps.Itemized / days
			}
		}
		out[i] = ps
	}
	return out
}

// Combine adds up several cards' periods. Totals and burn rates add; coverage
// is the widest card's; a period has balance data if any card does.
func Combine(perCard [][]PeriodSpend) []PeriodSpend {
	if len(perCard) == 0 {
		return nil
	}
	out := make([]PeriodSpend, len(perCard[0]))
	for i := range out {
		out[i] = PeriodSpend{Period: perCard[0][i].Period, Source: SourceItemizedOnly}
		for _, card := range perCard {
			c := card[i]
			out[i].Total += c.Total
			out[i].BalanceSpend += c.BalanceSpend
			out[i].Itemized += c.Itemized
			out[i].NotItemized += c.NotItemized
			out[i].DailyBurn += c.DailyBurn
			out[i].CoveredDays = max(out[i].CoveredDays, c.CoveredDays)
			if c.Source == SourceBalance {
				out[i].Source = SourceBalance
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/ledger/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ledger/spend.go internal/ledger/spend_test.go
git commit -m "feat(ledger): measure card spending per billing period from balances"
```

---

### Task 6: Building the report from database rows

**Files:**
- Create: `internal/ledger/build.go`
- Test: `internal/ledger/build_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1, 4 and 5
- Produces:
  - `ledger.Report{Periods []PeriodSpend; Charges []models.DBTransaction}`
  - `ledger.Build(periods []models.BillingPeriod, accounts []models.DBAccount, txns []models.DBTransaction, snapshots map[string][]models.BalanceSnapshot, patterns map[string][]string, excluded []string) Report`
  - `ledger.FetchFrom(snapshots map[string][]models.BalanceSnapshot, start time.Time) time.Time`

- [ ] **Step 1: Write the failing tests**

Create `internal/ledger/build_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests to make sure they fail**

Run: `devenv shell -- go test ./internal/ledger/ -run 'Build|FetchFrom' -v`
Expected: build failure, `undefined: Build`.

- [ ] **Step 3: Implement**

Create `internal/ledger/build.go`:

```go
package ledger

import (
	"sort"
	"time"

	"finance_tracker/internal/models"
)

// Report is the card spending picture the analysis prompt is built from.
type Report struct {
	Periods []PeriodSpend
	Charges []models.DBTransaction // itemized card charges in range, excluded categories removed, newest first
}

// Build groups accounts into cards and measures each card's spending.
//
// Only included credit card accounts are analyzed. Every non-card account,
// included or not, is scanned for payments toward the cards, because that is
// where a payment shows up when the card's own feed drops it. Charges in
// excluded categories stay out of the itemized figures, but the balance still
// counts them. Card fees and interest are real costs, and the balance is the
// authority.
func Build(periods []models.BillingPeriod, accounts []models.DBAccount, txns []models.DBTransaction,
	snapshots map[string][]models.BalanceSnapshot, patterns map[string][]string, excluded []string) Report {
	isExcluded := make(map[string]bool, len(excluded))
	for _, c := range excluded {
		if c != "" {
			isExcluded[c] = true
		}
	}

	cardOf := make(map[string]string)    // included card account ID -> card key
	isCard := make(map[string]bool)      // any card account, included or not
	coverageEnd := make(map[string]int64) // card key -> latest balance_date
	for _, a := range accounts {
		if !a.IsCreditCard || a.CardKey == "" {
			continue
		}
		isCard[a.ID] = true
		coverageEnd[a.CardKey] = max(coverageEnd[a.CardKey], a.BalanceDate)
		if a.IsIncluded {
			cardOf[a.ID] = a.CardKey
		}
	}

	byCard := make(map[string][]models.DBTransaction)
	var paying []models.DBTransaction
	for _, t := range txns {
		if key, ok := cardOf[t.AccountID]; ok {
			byCard[key] = append(byCard[key], t)
		} else if !isCard[t.AccountID] {
			paying = append(paying, t)
		}
	}

	keys := make([]string, 0, len(coverageEnd))
	for _, key := range cardOf {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	keys = dedupSorted(keys)

	var report Report
	if len(periods) == 0 {
		return report
	}
	rangeFrom, _ := PeriodBounds(periods[0])
	_, rangeTo := PeriodBounds(periods[len(periods)-1])

	var perCard [][]PeriodSpend
	for _, key := range keys {
		cardTxns := byCard[key]
		var charges []models.DBTransaction
		for _, t := range cardTxns {
			if t.Amount < 0 && !isExcluded[t.Category] {
				charges = append(charges, t)
				if d := EffectiveDate(t); d >= rangeFrom && d < rangeTo {
					report.Charges = append(report.Charges, t)
				}
			}
		}
		payments := DetectPayments(cardTxns, paying, patterns[key])
		perCard = append(perCard, CardPeriods(periods, snapshots[key], payments, coverageEnd[key], charges))
	}

	report.Periods = Combine(perCard)
	if report.Periods == nil {
		for _, p := range periods {
			report.Periods = append(report.Periods, PeriodSpend{Period: p, Source: SourceItemizedOnly})
		}
	}
	sort.SliceStable(report.Charges, func(i, j int) bool {
		return EffectiveDate(report.Charges[i]) > EffectiveDate(report.Charges[j])
	})
	return report
}

func dedupSorted(s []string) []string {
	out := s[:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

// FetchFrom returns how far back transactions must be loaded to measure the
// range starting at start. The interval straddling start begins at the last
// snapshot before it, and payments settling it may post up to
// PaymentMatchWindow earlier.
func FetchFrom(snapshots map[string][]models.BalanceSnapshot, start time.Time) time.Time {
	from := start.Unix()
	for _, snaps := range snapshots {
		for i := len(snaps) - 1; i >= 0; i-- {
			if snaps[i].BalanceDate < start.Unix() {
				from = min(from, snaps[i].BalanceDate)
				break
			}
		}
	}
	return time.Unix(from, 0).UTC().Add(-PaymentMatchWindow)
}
```

- [ ] **Step 4: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/ledger/ -v && devenv shell -- go vet ./internal/ledger/`
Expected: all PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add internal/ledger/build.go internal/ledger/build_test.go
git commit -m "feat(ledger): build the card spending report from accounts and transactions"
```

---

### Task 7: Loading transactions from every account

**Files:**
- Modify: `internal/store/transactions.go` (add after `GetForPeriod`, ~line 276)
- Test: `internal/store/transactions_test.go` (append)

**Interfaces:**
- Produces: `(*TransactionStore).GetForPeriodAllAccounts(ctx, start, end int64) ([]models.DBTransaction, error)`, same row shape as `GetForPeriod` (category resolved from overrides and then categories), with no `is_included` filter.

- [ ] **Step 1: Write the failing test**

Append to `internal/store/transactions_test.go`:

```go
// Card payments often come from accounts the user excluded from analysis. They
// still have to be visible for payment detection.
func TestGetForPeriodAllAccountsIncludesExcludedAccounts(t *testing.T) {
	ctx := context.Background()
	accts, txns := upsertStores(t)
	if err := accts.Upsert(ctx, models.DBAccount{ID: "ACT-sav", Name: "ACT-sav", IsIncluded: false}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, _, err := txns.UpsertBatch(ctx, []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-sav", Description: "WW591 TFR-A  C/C", Amount: -1800, Posted: 1790000000},
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := txns.GetForPeriodAllAccounts(ctx, 1789999999, 1790000001)

	if err != nil || len(got) != 1 {
		t.Errorf("expected the excluded account's transaction, got %+v (err %v)", got, err)
	}
}
```

- [ ] **Step 2: Run the test to make sure it fails**

Run: `devenv shell -- go test ./internal/store/ -run AllAccounts -v`
Expected: build failure, `txns.GetForPeriodAllAccounts undefined`.

- [ ] **Step 3: Implement**

Add to `internal/store/transactions.go` after `GetForPeriod`:

```go
// GetForPeriodAllAccounts returns transactions in a date range from every
// account, included or not. Card payments are detected on the paying side, and
// the paying account is often one the user has excluded from analysis.
func (s *TransactionStore) GetForPeriodAllAccounts(ctx context.Context, start, end int64) ([]models.DBTransaction, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT t.id, t.account_id, t.description, t.amount, t.posted, t.transacted_at, t.pending,
			COALESCE(co.category, c.category, '') as category,
			t.cached_at, t.updated_at
		FROM transactions t
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		WHERE t.posted >= ? AND t.posted <= ?
		ORDER BY t.posted DESC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []models.DBTransaction
	for rows.Next() {
		var t models.DBTransaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Description, &t.Amount, &t.Posted, &t.TransactedAt, &t.Pending, &t.Category, &t.CachedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}
```

- [ ] **Step 4: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/store/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/transactions.go internal/store/transactions_test.go
git commit -m "feat(store): load transactions across all accounts for payment detection"
```

---

### Task 8: Rebuild the analysis prompt around the ledger

**Files:**
- Modify: `internal/llm/analyze.go` (most of the file changes; the replacement code is below)
- Rewrite: `internal/llm/analyze_test.go`

**Interfaces:**
- Consumes: `ledger.PeriodSpend`, `ledger.SourceBalance`, `ledger.SourceItemizedOnly` (Task 5)
- Produces:
  - `llm.SystemPrompt` (updated text)
  - `llm.PromptInput{Periods []ledger.PeriodSpend; Charges []models.DBTransaction; Stale []models.StaleConnection; Drifted []models.UnreconciledAccount; Budgets []models.Budget; BillingDay int; Now time.Time}`
  - `llm.GeneratePrompt(in PromptInput) string`
  - `llm.CyclePeriods(start, end time.Time, billingDay int) []models.BillingPeriod` (renamed from `calcBillingPeriods`, body unchanged)
- Removes: `FilterExcludedCategories` (the ledger does exclusion now), `generateSinglePeriodPrompt`, `generateMultiPeriodPrompt`, `formatDBAccounts`, `calcPeriodTotals`, `countTxnDays`

- [ ] **Step 1: Write the new tests**

Replace `internal/llm/analyze_test.go` entirely with:

```go
package llm

import (
	"strings"
	"testing"
	"time"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/models"
)

func txn(id, desc string, amount float64, category string, posted time.Time) models.DBTransaction {
	return models.DBTransaction{
		ID: id, AccountID: "ACT-test", Description: desc, Amount: amount, Posted: posted.Unix(), Category: category,
	}
}

func date(month time.Month, d int) time.Time {
	return time.Date(2026, month, d, 0, 0, 0, 0, time.UTC)
}

var (
	julyCycle = models.BillingPeriod{Label: "Jul 15 - Aug 14", Start: date(time.July, 15), End: date(time.August, 14), IsComplete: true, IsFocus: true}
	augCycle  = models.BillingPeriod{Label: "Aug 15 - Sep 14", Start: date(time.August, 15), End: date(time.September, 14), IsComplete: true, IsFocus: true}
	sepCycle  = models.BillingPeriod{Label: "Sep 15 - Sep 23", Start: date(time.September, 15), End: date(time.September, 23), IsFocus: true}
	now       = time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
)

// productionPeriods mirrors galactica on 2026-09-23: July has no balance
// history, August is partly covered, September is covered and unitemized.
func productionPeriods() []ledger.PeriodSpend {
	return []ledger.PeriodSpend{
		{Period: julyCycle, Source: ledger.SourceItemizedOnly, Total: 2522.38, Itemized: 2522.38, DailyBurn: 81.37},
		{Period: augCycle, Source: ledger.SourceBalance, Total: 3131.30, BalanceSpend: 3131.30, CoveredDays: 14.5, DailyBurn: 215.95, NotItemized: 3131.30},
		{Period: sepCycle, Source: ledger.SourceBalance, Total: 1908.62, BalanceSpend: 1908.62, CoveredDays: 8.5, DailyBurn: 224.54, Itemized: 100, NotItemized: 1808.62},
	}
}

func prompt(periods []ledger.PeriodSpend, charges []models.DBTransaction) string {
	return GeneratePrompt(PromptInput{Periods: periods, Charges: charges, BillingDay: 15, Now: now})
}

// The headline must be the balance figure. Summing transactions is how a dead
// feed was reported as a $2,470 month.
func TestGeneratePromptLeadsWithBalanceDerivedTotals(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	for _, want := range []string{"| Sep 15 - Sep 23 |", "$1908.62", "$224.54/day", "$1808.62"} {
		if !strings.Contains(p, want) {
			t.Errorf("expected %q in the period table; prompt was:\n%s", want, p)
		}
	}
}

func TestGeneratePromptLabelsCategoriesAsPartial(t *testing.T) {
	charges := []models.DBTransaction{txn("t1", "METRO", -100, "Groceries", date(time.September, 16))}

	p := prompt(productionPeriods(), charges)

	if !strings.Contains(p, "Categories cover $2622.38 of $7562.30 (35%)") {
		t.Errorf("the category breakdown must state its coverage; prompt was:\n%s", p)
	}
}

func TestGeneratePromptLabelsItemizedOnlyPeriods(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if !strings.Contains(p, "| Jul 15 - Aug 14 | completed [FOCUS] | itemized only |") {
		t.Errorf("July has no balance history and must say so; prompt was:\n%s", p)
	}
}

// Card payments settle spending already counted; they are never expenses.
func TestGeneratePromptNeverShowsPayments(t *testing.T) {
	charges := []models.DBTransaction{
		txn("t1", "METRO", -100, "Groceries", date(time.September, 16)),
		txn("t2", "PAYMENT - THANK YOU", 7340, "Payment", date(time.September, 16)),
	}

	if p := prompt(productionPeriods(), charges); strings.Contains(p, "PAYMENT - THANK YOU") {
		t.Error("a payment reached the prompt")
	}
}

func TestGeneratePromptComparesBurnWithLastCompletedCycle(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if !strings.Contains(p, "Current cycle burn: $224.54/day over 8.5 days vs $215.95/day in Aug 15 - Sep 14 (+4.0%)") {
		t.Errorf("expected the burn comparison; prompt was:\n%s", p)
	}
}

func TestGeneratePromptSaysWhenNoCompletedCycleHasBalanceData(t *testing.T) {
	periods := productionPeriods()
	periods[1].Source = ledger.SourceItemizedOnly

	if p := prompt(periods, nil); !strings.Contains(p, "No completed cycle has balance data yet") {
		t.Errorf("expected the missing-comparison note; prompt was:\n%s", p)
	}
}

func TestGeneratePromptMarksBudgetsAsLowerBoundsWhenMostlyUnitemized(t *testing.T) {
	p := GeneratePrompt(PromptInput{
		Periods: productionPeriods(), BillingDay: 15, Now: now,
		Budgets: []models.Budget{{Category: "Groceries", Amount: 990}},
	})

	if !strings.Contains(p, "these budget figures are lower bounds") {
		t.Errorf("5%% itemized must mark budgets as lower bounds; prompt was:\n%s", p)
	}
}

// A quiet card and a de-authorized connection look identical from the
// transactions alone. Regression test: TD broke on Aug 21 and every report
// showed falling spend.
func TestGeneratePromptNamesCardsThatStoppedRefreshing(t *testing.T) {
	stale := []models.StaleConnection{{
		Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		BalanceDate: date(time.August, 21).Unix(), LastTransaction: date(time.July, 22).Unix(),
	}}

	p := GeneratePrompt(PromptInput{Periods: productionPeriods(), Stale: stale, Now: now})

	if !strings.Contains(p, "DATA LAG") || !strings.Contains(p, "TD AEROPLAN VISA INFINITE (4520)") {
		t.Errorf("a stale card must raise a named data lag warning; prompt was:\n%s", p)
	}
}

func TestGeneratePromptOmitsWarningsWhenConnectionsAreHealthy(t *testing.T) {
	p := prompt(productionPeriods(), nil)

	if strings.Contains(p, "DATA LAG") || strings.Contains(p, "ITEMIZATION GAP") {
		t.Errorf("nothing is wrong, so nothing should be flagged; prompt was:\n%s", p)
	}
}

// With a live balance the totals are right; only the categories are short.
// Calling that "incomplete data" would contradict the totals.
func TestGeneratePromptDescribesDriftAsItemizationGap(t *testing.T) {
	drifted := []models.UnreconciledAccount{{
		Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust", Unexplained: -894.81,
	}}

	p := GeneratePrompt(PromptInput{Periods: productionPeriods(), Drifted: drifted, Now: now})

	if !strings.Contains(p, "ITEMIZATION GAP") || !strings.Contains(p, "itemization missing for $894.81") {
		t.Errorf("expected an itemization gap note; prompt was:\n%s", p)
	}
	if strings.Contains(p, "DATA LAG") {
		t.Error("a live balance is not a data lag")
	}
}
```

- [ ] **Step 2: Run the tests to make sure they fail**

Run: `devenv shell -- go test ./internal/llm/ -v`
Expected: build failure, `undefined: PromptInput`.

- [ ] **Step 3: Implement the prompt**

In `internal/llm/analyze.go`:

1. Add `"finance_tracker/internal/ledger"` to the imports.
2. Delete `GeneratePrompt`, `generateSinglePeriodPrompt`, `generateMultiPeriodPrompt`, `formatDBAccounts`, `countTxnDays`, `calcPeriodTotals`, `FilterExcludedCategories` and `buildDataLagWarning`.
3. Rename `calcBillingPeriods` to `CyclePeriods` and give it this doc comment:
   `// CyclePeriods splits [start, end] into billing cycles, oldest first; the last three are marked as focus.`
4. Replace `SystemPrompt` and add the new code below. Keep `formatDBTransactions`, `calcTotalExpenses`, `formatTopExpenses`, `buildCategoryBreakdown` and `daysBetween` as they are.

```go
// SystemPrompt is the default LLM system prompt for spending analysis.
const SystemPrompt = `You are an expert financial analyst specializing in personal finance and spending pattern analysis for families.
You are analyzing credit card spending for a Canadian family of 4: 2 adults, 2 daughters (born 2021 and 2024, currently ~4 and ~1 years old).
The family pays the card in full every month, so card spending is what the household spends day to day.
Spending totals and daily burn rates come from card balances and are authoritative, even when individual transactions are missing.
Categories and individual charges describe only the itemized part of that spending.
Be concise, specific, and use the pre-calculated data provided — do not recalculate totals or percentages.`

// PromptInput is everything the analysis prompt is built from.
type PromptInput struct {
	Periods    []ledger.PeriodSpend   // one per billing cycle, oldest first
	Charges    []models.DBTransaction // itemized card charges, excluded categories removed
	Stale      []models.StaleConnection
	Drifted    []models.UnreconciledAccount
	Budgets    []models.Budget
	BillingDay int
	Now        time.Time
}

const readingGuide = `How to read these numbers:
- "Total" and "Daily burn" come from the card balance: debt going up is spending, debt going down is a payment. They are authoritative even when transactions are missing.
- "Itemized" is the part of that spending with individual transactions. Categories and top expenses describe only this part.
- "Not itemized" is spending the balance shows but the transaction feed did not deliver. It is real spending with an unknown category — never treat it as savings.
- Card payments and transfers are not expenses and are not shown.
- Periods marked "itemized only" have no balance history; their totals are a lower bound. Do not compare them with balance periods or read a trend across that boundary.
`

const reportInstructions = `### Instructions

Analyze this family's credit card spending. Write a concise report (~250 words) with:

1. **Summary**: The last 3 billing cycles by total and daily burn. What's the trajectory?
2. **Burn Rate**: Is the current cycle's daily burn higher or lower than the last completed cycle? Say so even when the reason can't be itemized.
3. **What's Improving / What Needs Attention**: From the itemized categories, framed as a share of what is itemized.
4. **Top Expenses**: The 10 largest itemized charges:
%s5. **Family-Specific Insights**: Anything relevant to a family with young children.
6. **Actionable Suggestions**: 2-3 concrete things to try next billing cycle.

Notes:
- All calculations are pre-computed. Use them directly.
- Focus the narrative on FOCUS periods.
- Use CAD ($) for all amounts.
- When a period has "Not itemized" spending, say how much could not be broken down instead of guessing what it was.
`

// GeneratePrompt builds the analysis prompt from balance-derived card spending.
func GeneratePrompt(in PromptInput) string {
	// Only charges are itemized spending; a credit reaching this point would be
	// a payment or refund and has no place in the prompt.
	var charges []models.DBTransaction
	for _, t := range in.Charges {
		if t.Amount < 0 {
			charges = append(charges, t)
		}
	}

	var b strings.Builder
	b.WriteString("## Credit Card Spending Analysis — Family of 4\n\n")
	b.WriteString("Household: 2 adults, 2 children (born 2021 and 2024, currently ~4 and ~1 years old).\n\n")
	b.WriteString(readingGuide)
	b.WriteString("\n")
	b.WriteString(buildPeriodTable(in.Periods))
	b.WriteString(buildBurnTrend(in.Periods))
	b.WriteString(buildDataLagWarning(in.Stale, in.Drifted, in.Now))
	b.WriteString("\n")
	b.WriteString(buildCoverageLine(in.Periods))
	b.WriteString(buildCategoryBreakdown(charges))
	b.WriteString("\n")
	fmt.Fprintf(&b, reportInstructions, formatTopExpenses(charges, 10))
	if len(in.Budgets) > 0 {
		b.WriteString("\n" + buildBudgetSection(in.Budgets, charges, in.BillingDay, currentPeriod(in.Periods)) + "\n")
	}
	b.WriteString("\nCard Charges:\n")
	b.WriteString(formatDBTransactions(charges))
	return b.String()
}

func money(v float64) string { return fmt.Sprintf("$%.2f", v) }

func buildPeriodTable(periods []ledger.PeriodSpend) string {
	var b strings.Builder
	b.WriteString("Billing Periods:\n\n")
	b.WriteString("| Period | Status | Source | Covered days | Total | Daily burn | Change | Itemized | Not itemized |\n")
	b.WriteString("|--------|--------|--------|--------------|-------|------------|--------|----------|--------------|\n")
	for i, p := range periods {
		status := "completed"
		if !p.Period.IsComplete {
			status = "in progress"
		}
		if p.Period.IsFocus {
			status += " [FOCUS]"
		}
		source, covered := "balance", fmt.Sprintf("%.1f", p.CoveredDays)
		if p.Source == ledger.SourceItemizedOnly {
			source, covered = "itemized only", "—"
		}
		// A cycle in progress has a partial total, so only completed balance
		// cycles are compared; the burn trend covers the current one.
		change := "—"
		if i > 0 {
			prev := periods[i-1]
			if p.Period.IsComplete && p.Source == ledger.SourceBalance && prev.Source == ledger.SourceBalance && prev.Total > 0 {
				change = fmt.Sprintf("%+.1f%%", (p.Total-prev.Total)/prev.Total*100)
			}
		}
		notItemized := "—"
		if p.NotItemized > 0 {
			notItemized = money(p.NotItemized)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s/day | %s | %s | %s |\n",
			p.Period.Label, status, source, covered, money(p.Total), money(p.DailyBurn), change, money(p.Itemized), notItemized)
	}
	return b.String()
}

// buildBurnTrend compares the current cycle's daily burn with the last
// completed cycle that has balance data. It answers "is spending going up?"
// even when nothing can be itemized.
func buildBurnTrend(periods []ledger.PeriodSpend) string {
	if len(periods) == 0 {
		return ""
	}
	cur := periods[len(periods)-1]
	if cur.Period.IsComplete || cur.Source != ledger.SourceBalance {
		return ""
	}
	for i := len(periods) - 2; i >= 0; i-- {
		prev := periods[i]
		if prev.Period.IsComplete && prev.Source == ledger.SourceBalance && prev.DailyBurn > 0 {
			return fmt.Sprintf("\nCurrent cycle burn: %s/day over %.1f days vs %s/day in %s (%+.1f%%).\n",
				money(cur.DailyBurn), cur.CoveredDays, money(prev.DailyBurn), prev.Period.Label,
				(cur.DailyBurn-prev.DailyBurn)/prev.DailyBurn*100)
		}
	}
	return fmt.Sprintf("\nCurrent cycle burn: %s/day over %.1f days. No completed cycle has balance data yet, so there is no balance-based comparison.\n",
		money(cur.DailyBurn), cur.CoveredDays)
}

func buildCoverageLine(periods []ledger.PeriodSpend) string {
	var total, itemized float64
	for _, p := range periods {
		total += p.Total
		itemized += p.Itemized
	}
	if total <= 0 {
		return ""
	}
	return fmt.Sprintf("Categories cover %s of %s (%.0f%%) of card spending across these periods.\n",
		money(itemized), money(total), math.Min(100, itemized/total*100))
}

func currentPeriod(periods []ledger.PeriodSpend) *ledger.PeriodSpend {
	if len(periods) == 0 {
		return nil
	}
	return &periods[len(periods)-1]
}
```

5. Change `buildBudgetSection` to take the current period and add the lower-bound note. Replace its signature and final lines:

```go
func buildBudgetSection(budgets []models.Budget, expenses []models.DBTransaction, billingDay int, current *ledger.PeriodSpend) string {
```

and replace the last two lines (`b.WriteString("\nComment on budget adherence ...` and `return b.String()`) with:

```go
	// Budgets can only be checked against itemized charges. When most of the
	// cycle is unitemized, "under budget" means nothing, and the model has to know.
	if current != nil && current.Source == ledger.SourceBalance && current.Total > 0 && current.Itemized/current.Total < 0.8 {
		fmt.Fprintf(&b, "\nOnly %.0f%% of this cycle's card spending is itemized, so these budget figures are lower bounds.",
			current.Itemized/current.Total*100)
	}
	b.WriteString("\nComment on budget adherence where budgets are set. Note any categories that are significantly over budget.")
	return b.String()
```

6. Add the new lag warning:

```go
// buildDataLagWarning tells the model which cards are not reporting in full.
//
// A stale card has stopped refreshing, so its balance, and with it the totals,
// are out of date. A drifted card's balance is current, so the totals already
// include its spending; only the itemization is short. Calling that "incomplete
// data" would contradict the totals the model was just told to trust.
func buildDataLagWarning(stale []models.StaleConnection, drifted []models.UnreconciledAccount, now time.Time) string {
	var b strings.Builder
	if len(stale) > 0 {
		b.WriteString("\n> [!WARNING]\n> **DATA LAG DETECTED**: ")
		fmt.Fprintf(&b, "%d card(s) stopped refreshing, so their balances — and this analysis — are out of date. ", len(stale))
		b.WriteString("Do not congratulate the user on low spending, and do not read a downward trend into it.\n")
		for _, c := range stale {
			lastSync := time.Unix(c.BalanceDate, 0).UTC()
			line := fmt.Sprintf("> - **%s** (%s): last refreshed %s (%d days ago)",
				c.Name, c.OrgName, lastSync.Format("Jan 2, 2006"), daysBetween(lastSync, now))
			if c.LastTransaction > 0 {
				lastTxn := time.Unix(c.LastTransaction, 0).UTC()
				line += fmt.Sprintf("; newest transaction %s (%d days ago)",
					lastTxn.Format("Jan 2, 2006"), daysBetween(lastTxn, now))
			} else {
				line += "; no transactions on record"
			}
			b.WriteString(line + "\n")
		}
	}
	if len(drifted) > 0 {
		b.WriteString("\n> [!NOTE]\n> **ITEMIZATION GAP**: these cards' balances are current, so the totals above include all their spending, ")
		b.WriteString("but some of it arrived without transactions. Categories and top expenses under-count by these amounts.\n")
		for _, u := range drifted {
			fmt.Fprintf(&b, "> - **%s** (%s): balance tracked; itemization missing for $%.2f\n",
				u.Name, u.OrgName, math.Abs(u.Unexplained))
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Run the tests to make sure they pass**

Run: `devenv shell -- go test ./internal/llm/ -v`
Expected: all PASS. `go build ./...` **will fail** in `internal/api/analysis_run.go` until Task 9 rewires the caller. That's expected. Don't commit here: Task 9 finishes the change, and one commit at the end of Task 9 covers both tasks, so no commit contains a broken build.

---

### Task 9: Wire it together

**Files:**
- Modify: `internal/api/analysis_run.go`
- Modify: `internal/api/sync.go`
- Modify: `internal/api/helpers.go`
- Create: `internal/api/payment_patterns.go`
- Create: `internal/store/bootstrap.go`
- Test: `internal/store/bootstrap_test.go`
- Modify: `internal/server/server.go`
- Modify: `cmd/server/main.go`
- Create: `cmd/promptdump/main.go`

**Interfaces:**
- Consumes: everything above
- Produces:
  - `store.InitCardLedger(ctx, *AccountStore, *SnapshotStore, *SettingsStore) error`
  - `api.AnalysisPrompt{Text string; Start, End time.Time; Charges []models.DBTransaction}`
  - `(*AnalysisRunHandler).BuildPrompt(ctx, now time.Time) (*AnalysisPrompt, error)`
  - `api.NewAnalysisRunHandler(cfg, txns, accts, cats, snapshots *store.SnapshotStore, settings *store.SettingsStore, analyses, budgets, sched, events)` (new parameters inserted after `cats`)
  - `api.NewSyncHandler(cfg, accounts, txns, cats, snapshots *store.SnapshotStore, syncLog, sched, events)` (new parameter inserted after `cats`)
  - Routes: `GET /api/card-payment-patterns`, `PUT /api/card-payment-patterns`

- [ ] **Step 1: Write the failing bootstrap test**

Create `internal/store/bootstrap_test.go`:

```go
package store

import (
	"context"
	"testing"

	"finance_tracker/internal/ledger"
)

// On the first start after deploy the accounts have no card fields yet. One
// startup has to classify them, seed the history and write the patterns.
func TestInitCardLedgerPreparesLegacyDatabase(t *testing.T) {
	ctx := context.Background()
	db, accts, snaps := seedProductionTD(t)
	if _, err := db.Write.Exec(`UPDATE accounts SET card_key = NULL, is_credit_card = 0`); err != nil {
		t.Fatalf("simulate legacy rows: %v", err)
	}
	settings := NewSettingsStore(db.Read, db.Write)

	if err := InitCardLedger(ctx, accts, snaps, settings); err != nil {
		t.Fatalf("init: %v", err)
	}

	byCard, _ := snaps.ListByCard(ctx)
	raw, _ := settings.Get(ctx, ledger.PaymentPatternsSettingKey)
	if len(byCard[tdKey]) != 3 {
		t.Errorf("expected the 3 production readings, got %+v", byCard[tdKey])
	}
	if raw == "" {
		t.Error("default payment patterns were not written")
	}
}

func TestInitCardLedgerKeepsCustomPatterns(t *testing.T) {
	ctx := context.Background()
	db, accts, snaps := seedProductionTD(t)
	settings := NewSettingsStore(db.Read, db.Write)
	custom := `{"TD Canada Trust|4520":["MY PATTERN"]}`
	settings.Set(ctx, ledger.PaymentPatternsSettingKey, custom)

	if err := InitCardLedger(ctx, accts, snaps, settings); err != nil {
		t.Fatalf("init: %v", err)
	}

	if raw, _ := settings.Get(ctx, ledger.PaymentPatternsSettingKey); raw != custom {
		t.Errorf("custom patterns were overwritten: %s", raw)
	}
}
```

Run: `devenv shell -- go test ./internal/store/ -run InitCardLedger -v`
Expected: build failure, `undefined: InitCardLedger`.

- [ ] **Step 2: Implement the bootstrap**

Create `internal/store/bootstrap.go`:

```go
package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/ledger"
)

// InitCardLedger prepares card tracking at startup. It classifies accounts
// stored before cards were tracked, seeds the balance history from what the
// accounts table already knows, and writes the default payment patterns.
// Every step is idempotent, so it runs on every start.
func InitCardLedger(ctx context.Context, accts *AccountStore, snaps *SnapshotStore, settings *SettingsStore) error {
	classified, err := accts.BackfillCardIdentity(ctx)
	if err != nil {
		return fmt.Errorf("classify accounts: %w", err)
	}
	seeded, err := snaps.Seed(ctx)
	if err != nil {
		return fmt.Errorf("seed balance history: %w", err)
	}

	raw, err := settings.Get(ctx, ledger.PaymentPatternsSettingKey)
	if err != nil {
		return fmt.Errorf("read payment patterns: %w", err)
	}
	if raw == "" {
		defaults, _ := json.Marshal(ledger.DefaultPaymentPatterns)
		if err := settings.Set(ctx, ledger.PaymentPatternsSettingKey, string(defaults)); err != nil {
			return fmt.Errorf("write payment patterns: %w", err)
		}
	}

	log.Info().Int("accounts_classified", classified).Int("snapshots_seeded", seeded).Msg("Card ledger ready")
	return nil
}
```

Run: `devenv shell -- go test ./internal/store/ -v`
Expected: all PASS.

- [ ] **Step 3: Add the health-list filter**

Append to `internal/api/helpers.go` (add `"finance_tracker/internal/models"` to its imports if it isn't there):

```go
// filterHealth keeps the stale and drifted entries whose account passes keep.
func filterHealth(stale []models.StaleConnection, drifted []models.UnreconciledAccount, keep func(id string) bool) (
	[]models.StaleConnection, []models.UnreconciledAccount) {
	var s []models.StaleConnection
	for _, c := range stale {
		if keep(c.ID) {
			s = append(s, c)
		}
	}
	var d []models.UnreconciledAccount
	for _, u := range drifted {
		if keep(u.ID) {
			d = append(d, u)
		}
	}
	return s, d
}
```

- [ ] **Step 4: Rebuild the analysis run**

In `internal/api/analysis_run.go`:

Add `snapshotStore *store.SnapshotStore` and `settingsStore *store.SettingsStore` fields to `AnalysisRunHandler`. Add the matching `snapshots *store.SnapshotStore, settings *store.SettingsStore` parameters to `NewAnalysisRunHandler` right after `cats *store.CategoryStore`, and assign them. Add `"finance_tracker/internal/ledger"` to the imports.

Add:

```go
// AnalysisPrompt is an assembled analysis prompt and what it was built from.
type AnalysisPrompt struct {
	Text       string
	Start, End time.Time
	Charges    []models.DBTransaction
}

// BuildPrompt assembles the analysis prompt from the database without calling
// the LLM. cmd/promptdump uses it to tune the prompt against a copy of
// production.
func (h *AnalysisRunHandler) BuildPrompt(ctx context.Context, now time.Time) (*AnalysisPrompt, error) {
	billingDay := h.cfg.BillingDay
	start, end, err := billing.CalculateDateRange(models.DateRangeTypeCurrentAndLastMonth, nil, nil, billingDay)
	if err != nil {
		return nil, fmt.Errorf("date range: %w", err)
	}
	periods := llmclient.CyclePeriods(start, end, billingDay)

	accounts, err := h.acctStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("accounts: %w", err)
	}
	snapshots, err := h.snapshotStore.ListByCard(ctx)
	if err != nil {
		return nil, fmt.Errorf("balance snapshots: %w", err)
	}
	txns, err := h.txnStore.GetForPeriodAllAccounts(ctx, ledger.FetchFrom(snapshots, start).Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("transactions: %w", err)
	}
	rawPatterns, err := h.settingsStore.Get(ctx, ledger.PaymentPatternsSettingKey)
	if err != nil {
		return nil, fmt.Errorf("payment patterns: %w", err)
	}
	patterns, err := ledger.ParsePaymentPatterns(rawPatterns)
	if err != nil {
		return nil, fmt.Errorf("payment patterns: %w", err)
	}
	excluded, _ := h.catStore.ExcludedCategoryNames(ctx)

	report := ledger.Build(periods, accounts, txns, snapshots, patterns, excluded)

	// Only the cards being analyzed matter to the model, and a superseded ID is
	// dead by definition. Without this the old TD account would be reported
	// stale forever.
	superseded := ledger.SupersededAccounts(accounts)
	analyzed := make(map[string]bool)
	for _, a := range accounts {
		if a.IsIncluded && a.IsCreditCard && !superseded[a.ID] {
			analyzed[a.ID] = true
		}
	}
	stale, err := h.acctStore.StaleConnections(ctx, now, StaleConnectionThreshold)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check for stale connections")
	}
	drifted, err := h.acctStore.UnreconciledAccounts(ctx, h.cfg.BalanceDriftThreshold)
	if err != nil {
		log.Error().Err(err).Msg("Failed to reconcile account balances")
	}
	stale, drifted = filterHealth(stale, drifted, func(id string) bool { return analyzed[id] })

	budgets, _ := h.budgetStore.GetAll(ctx)

	text := llmclient.GeneratePrompt(llmclient.PromptInput{
		Periods: report.Periods, Charges: report.Charges,
		Stale: stale, Drifted: drifted, Budgets: budgets,
		BillingDay: billingDay, Now: now,
	})
	return &AnalysisPrompt{Text: text, Start: start, End: end, Charges: report.Charges}, nil
}
```

In `runAnalysis`, replace everything from `billingDay := h.cfg.BillingDay` down to and including the line `prompt := llmclient.GeneratePrompt(...)` with:

```go
	billingDay := h.cfg.BillingDay
	dateRangeType := models.DateRangeTypeCurrentAndLastMonth

	built, err := h.BuildPrompt(ctx, time.Now().UTC())
	if err != nil {
		log.Error().Err(err).Msg("Failed to build analysis prompt")
		h.events.Broadcast("analysis_error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return
	}
	prompt, startDate, endDate, txns := built.Text, built.Start, built.End, built.Charges
```

The rest of `runAnalysis` (the OpenRouter check, retry, store, notify) keeps using `prompt`, `startDate`, `endDate`, `billingDay`, `dateRangeType` and `txns` unchanged. The "No transactions found" early return is gone on purpose: a card with a live balance and no transactions is exactly the case this feature exists for.

- [ ] **Step 5: Record snapshots on sync and ignore superseded accounts in alerts**

In `internal/api/sync.go`: add a `snapshots *store.SnapshotStore` field and a matching constructor parameter after `cats`, and add `"finance_tracker/internal/ledger"` to the imports.

In `runSync`, directly after the `if err != nil { ... return }` block that follows `FetchAndStore`, insert:

```go
	// Balances are the record of card spending, so they are captured right
	// after the accounts are refreshed, before anything else can fail.
	if n, err := h.snapshots.RecordCurrent(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to record card balances")
	} else if n > 0 {
		log.Info().Int("snapshots", n).Msg("Recorded card balances")
	}
```

In `alertOnStaleConnections`, after both the `stale` and `drifted` lookups succeed, insert:

```go
	// A superseded card account is the ID a re-auth replaced. It stops
	// refreshing by design, and alerting on it would never stop.
	if accounts, err := h.accounts.List(ctx); err == nil {
		superseded := ledger.SupersededAccounts(accounts)
		stale, drifted = filterHealth(stale, drifted, func(id string) bool { return !superseded[id] })
	}
```

- [ ] **Step 6: Add the payment patterns endpoint**

Create `internal/api/payment_patterns.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/store"
)

// PaymentPatternsHandler exposes the descriptions that identify payments toward
// each card from the paying accounts. The settings page only reflects .env, so
// these get their own endpoint.
type PaymentPatternsHandler struct {
	settings *store.SettingsStore
}

func NewPaymentPatternsHandler(s *store.SettingsStore) *PaymentPatternsHandler {
	return &PaymentPatternsHandler{settings: s}
}

func (h *PaymentPatternsHandler) Get(w http.ResponseWriter, r *http.Request) {
	raw, err := h.settings.Get(r.Context(), ledger.PaymentPatternsSettingKey)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	patterns, err := ledger.ParsePaymentPatterns(raw)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "BAD_SETTING", err.Error())
		return
	}
	WriteData(w, patterns)
}

// Put replaces the whole map, keyed by card key.
func (h *PaymentPatternsHandler) Put(w http.ResponseWriter, r *http.Request) {
	var patterns map[string][]string
	if err := json.NewDecoder(r.Body).Decode(&patterns); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Expected a JSON object of card key to pattern list")
		return
	}
	raw, _ := json.Marshal(patterns)
	if err := h.settings.Set(r.Context(), ledger.PaymentPatternsSettingKey, string(raw)); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, patterns)
}
```

- [ ] **Step 7: Wire the server and startup**

In `internal/server/server.go` `New`, after `budgetStore := ...`:

```go
	snapshotStore := store.NewSnapshotStore(db.Read, db.Write)
	settingsStore := store.NewSettingsStore(db.Read, db.Write)
```

Update the two constructor calls:

```go
	syncHandler := api.NewSyncHandler(cfg, accountStore, txnStore, catStore, snapshotStore, syncLogStore, sched, events)
	analysisRunHandler := api.NewAnalysisRunHandler(cfg, txnStore, accountStore, catStore, snapshotStore, settingsStore, analysisStore, budgetStore, sched, events)
```

After the Accounts routes:

```go
	// Card payment patterns
	patternsHandler := api.NewPaymentPatternsHandler(settingsStore)
	s.mux.HandleFunc("GET /api/card-payment-patterns", patternsHandler.Get)
	s.mux.HandleFunc("PUT /api/card-payment-patterns", patternsHandler.Put)
```

In `cmd/server/main.go`, add the `"finance_tracker/internal/store"` import and, directly after the `database.Migrate` block:

```go
	// Seeding failure is not fatal: without a balance history the analysis
	// falls back to itemized totals, clearly labeled as such.
	if err := store.InitCardLedger(context.Background(),
		store.NewAccountStore(db.Read, db.Write),
		store.NewSnapshotStore(db.Read, db.Write),
		store.NewSettingsStore(db.Read, db.Write)); err != nil {
		log.Error().Err(err).Msg("Failed to prepare card ledger")
	}
```

- [ ] **Step 8: Add promptdump**

Create `cmd/promptdump/main.go`:

```go
// Command promptdump prints the analysis prompt for a database without calling
// the LLM, to tune the prompt against real data. It migrates and seeds the
// database it opens, so point it at a copy:
//
//	DB_PATH=/path/to/copy.db ENV_FILE=/dev/null go run ./cmd/promptdump
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"finance_tracker/internal/api"
	"finance_tracker/internal/config"
	"finance_tracker/internal/database"
	llmclient "finance_tracker/internal/llm"
	"finance_tracker/internal/store"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx := context.Background()

	envFile := ".env"
	if v := os.Getenv("ENV_FILE"); v != "" {
		envFile = v
	}
	cfg, err := config.Load(envFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}
	defer db.Close()
	if err := database.Migrate(db.Write); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	accts := store.NewAccountStore(db.Read, db.Write)
	snaps := store.NewSnapshotStore(db.Read, db.Write)
	settings := store.NewSettingsStore(db.Read, db.Write)
	if err := store.InitCardLedger(ctx, accts, snaps, settings); err != nil {
		log.Fatal().Err(err).Msg("Failed to prepare card ledger")
	}

	h := api.NewAnalysisRunHandler(cfg,
		store.NewTransactionStore(db.Read, db.Write), accts,
		store.NewCategoryStore(db.Read, db.Write), snaps, settings,
		store.NewAnalysisStore(db.Read, db.Write), store.NewBudgetStore(db.Read, db.Write),
		nil, nil)

	built, err := h.BuildPrompt(ctx, time.Now().UTC())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build prompt")
	}
	fmt.Println("=== SYSTEM ===")
	fmt.Println(llmclient.SystemPrompt)
	fmt.Println("=== USER ===")
	fmt.Println(built.Text)
}
```

- [ ] **Step 9: Build, vet and test everything**

Run: `devenv shell -- go build ./... && devenv shell -- go vet ./... && devenv shell -- go test ./...`
Expected: build and vet clean, every test PASSes. If `go vet` flags the legacy `./src` tree for something unrelated to this change, note it and move on. Don't fix unrelated legacy code.

- [ ] **Step 10: Commit (covers Tasks 8 and 9)**

```bash
git add internal/llm/analyze.go internal/llm/analyze_test.go internal/api/ internal/store/bootstrap.go internal/store/bootstrap_test.go internal/server/server.go cmd/server/main.go cmd/promptdump/main.go
git commit -m "feat(analysis): trust card balances for spending and analyze cards only"
```

---

### Task 10: Verify against production data and document

**Files:**
- Modify: `CLAUDE.md` (add a short section)
- No code changes unless verification finds a bug. If it does, write a failing test that reproduces it first, then fix.

- [ ] **Step 1: Take a fresh copy of the production DB**

```bash
S=/tmp/claude-1000/-home-arosenfeld-Code-finance-tracker/6f2e55a2-7e33-49a8-a6f9-c0cc55db5ee7/scratchpad/verify
mkdir -p $S && scp -q 'root@galactica:/var/data/finance-tracker/finance_tracker.db*' $S/
ls -la $S
```

Expected: `finance_tracker.db` plus its `-wal` and `-shm` files. Never write to the galactica copy. Only the local copy is opened.

- [ ] **Step 2: Dump the prompt**

```bash
DB_PATH=$S/finance_tracker.db ENV_FILE=/dev/null devenv shell -- go run ./cmd/promptdump > $S/prompt.txt
sed -n 1,60p $S/prompt.txt
```

Check each of these and write the observed values down:
- The log line says `snapshots_seeded=4`: the 3 TD readings plus 1 for the Tangerine Mastercard at $0.00.
- The Sep 15 cycle is `balance`-sourced. Its total should be roughly `2694.81 × (Sep 15→Sep 23 share of the Sep 11→Sep 23 interval)` plus nothing itemized, and its not-itemized amount should be close to the total.
- The Aug 15 – Sep 14 cycle is `balance`-sourced with about 14.5 covered days.
- The May–Jul cycles are `itemized only`.
- No `PAYMENT - THANK YOU`, `Bill Payment`, `TFR-` or chequing descriptions (Subaru, Hydro, La Personnelle, Repentigny) appear anywhere in the prompt.
- There's no `DATA LAG` entry for the old TD account `ACT-1820c169…`.

- [ ] **Step 3: Check the seeded rows directly**

```bash
SQ=$(nix build nixpkgs#sqlite.bin --no-link --print-out-paths)/bin/sqlite3
$SQ -header -column $S/finance_tracker.db "select card_key, account_id, balance, date(balance_date,'unixepoch') d, source from balance_snapshots order by card_key, balance_date"
$SQ -header -column $S/finance_tracker.db "select name, is_credit_card, card_key from accounts order by is_credit_card desc, name"
```

Expected: TD readings −12124.34 (2026-08-31), −7129.23 (2026-09-11), −8024.04 (2026-09-23 or a newer date if galactica has synced since, possibly with extra `sync`-free readings). Exactly the TD Visa ×2 and Tangerine Money-Back Mastercard ×2 are flagged as cards.

- [ ] **Step 4: Document**

Append to `CLAUDE.md`, after the "Account and Transaction Filtering" section:

```markdown
#### Balance-Trusted Card Spending (web server)
- The web server's AI analysis covers **included credit card accounts only** (`accounts.is_credit_card`). Chequing, savings and lines of credit are used only to detect card payments.
- Card spending comes from **balance snapshots** (`balance_snapshots`, one per card per sync when the balance changes): spend = debt growth + payments in between; an unexplained drop is treated as a payment. Transactions only supply categories ("itemized"); the rest is reported as "not itemized".
- Cards are keyed by `org_name|last4` (`accounts.card_key`) so history survives SimpleFin issuing new account IDs on re-auth; the account first seen most recently is current, the others are superseded.
- Payment patterns per card live in the `settings` table (`card_payment_patterns`), editable via `GET/PUT /api/card-payment-patterns`.
- Tune prompts against production without calling the LLM: copy the DB from galactica (`/var/data/finance-tracker/finance_tracker.db*`) and run `DB_PATH=<copy> ENV_FILE=/dev/null go run ./cmd/promptdump`.
```

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: describe balance-trusted card spending and promptdump"
```
