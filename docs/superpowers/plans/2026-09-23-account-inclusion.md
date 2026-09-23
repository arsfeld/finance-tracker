# Account Inclusion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the user choose which accounts count — per account and per institution — from Settings, have that choice survive SimpleFin re-auths, and make the Transactions page honour it by default.

**Architecture:** `accounts.is_included` stays the single flag every query already filters on, but it becomes a property of the account's identity key (`accounts.card_key`, `org|last4`, falling back to the account ID). The store propagates changes across the key, new rows inherit from their key, and a startup pass reconciles existing rows. `GET /api/accounts` exposes `is_current` so the UI shows one entry per identity. The frontend adds a Settings "Accounts" tab and flips the Transactions page to `included_only` by default.

**Tech Stack:** Go 1.26 (`net/http`, `database/sql` on SQLite), React + TypeScript + TanStack Query + Radix UI (shadcn components) under `web/`.

**Spec:** `docs/superpowers/specs/2026-09-23-account-inclusion-design.md`

## Global Constraints

- Run all Go and `just` commands through devenv: `devenv shell -- <cmd>` (unless `$DEVENV_PROFILE` is set).
- Identity key = `accounts.card_key`; when it is `NULL` or `''`, the account ID is its own identity.
- "Current" row of a key = greatest `first_seen_at`, ties broken by greater `id`. This must match `ledger.CurrentAccounts`.
- A brand-new identity is inserted included (unchanged default).
- The ledger, analysis, alerts and every existing `a.is_included = 1` query stay untouched.
- Out of scope: editing `is_credit_card`/`card_key` in the UI; touching SimpleFin.
- Never write to or restart anything on galactica. Production data is only ever read from a local copy.
- Commit messages follow the repo's conventional style (`feat(store): …`, `fix(web): …`), no attribution lines.

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `internal/ledger/card.go` | Identity rules for accounts | Add `IdentityKey`, `CurrentByKey`; `CurrentAccounts` delegates |
| `internal/ledger/card_test.go` | | Test `CurrentByKey` |
| `internal/models/models.go` | `DBAccount` | Add computed `IsCurrent` |
| `internal/store/accounts.go` | Account persistence | `List` sets `IsCurrent`; `Upsert` inherits inclusion; `SetIncluded`; `Update` routes `is_included` through it; `NormalizeInclusion` |
| `internal/store/accounts_test.go` | | Tests for the above |
| `internal/store/bootstrap.go` | Startup reconciliation | Call `NormalizeInclusion` |
| `internal/api/accounts.go` | Account HTTP handlers | 404 for unknown IDs |
| `internal/api/accounts_test.go` | | Create: handler tests |
| `web/src/api/types.ts` | API types | Extend `DBAccount` |
| `web/src/api/queries.ts` | Data hooks | `patchApi`, `useSetAccountsIncluded` |
| `web/src/components/AccountsSettings.tsx` | Accounts tab UI | Create |
| `web/src/pages/Settings.tsx` | Settings page | Add "Accounts" tab |
| `web/src/pages/Transactions.tsx` | Transactions page | Default to included-only, "Show excluded accounts" checkbox |
| `CLAUDE.md` | Agent docs | Document account inclusion |

---

### Task 1: Expose which row is current for each identity

**Files:**
- Modify: `internal/ledger/card.go:50-66`
- Modify: `internal/models/models.go:124-138`
- Modify: `internal/store/accounts.go:52-71` (`List`)
- Test: `internal/ledger/card_test.go`, `internal/store/accounts_test.go`

**Interfaces:**
- Produces: `ledger.IdentityKey(a models.DBAccount) string`; `ledger.CurrentByKey(accounts []models.DBAccount) map[string]models.DBAccount`; field `models.DBAccount.IsCurrent bool` (JSON `is_current`), set by `AccountStore.List`; test helper `setFirstSeen(t, db, id, at string)` in `internal/store/accounts_test.go`.

- [ ] **Step 1: Write the failing ledger test**

Append to `internal/ledger/card_test.go`:

```go
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
```

- [ ] **Step 2: Run it to verify it fails**

Run: `devenv shell -- go test ./internal/ledger -run TestCurrentByKeyCoversEveryAccount`
Expected: FAIL — `undefined: CurrentByKey`.

- [ ] **Step 3: Implement `IdentityKey` and `CurrentByKey`, and make `CurrentAccounts` delegate**

In `internal/ledger/card.go`, replace the body of `CurrentAccounts` (keep its doc comment) and add the two new functions above it:

```go
// IdentityKey is what ties an account's rows together across re-auths and
// renames: its card key, or its own ID when it has none.
func IdentityKey(a models.DBAccount) string {
	if a.CardKey != "" {
		return a.CardKey
	}
	return a.ID
}

// CurrentByKey maps each identity to the account reporting for it now: the one
// first seen most recently, ties going to the higher ID.
func CurrentByKey(accounts []models.DBAccount) map[string]models.DBAccount {
	current := make(map[string]models.DBAccount)
	for _, a := range accounts {
		key := IdentityKey(a)
		cur, seen := current[key]
		if !seen || a.FirstSeenAt > cur.FirstSeenAt || (a.FirstSeenAt == cur.FirstSeenAt && a.ID > cur.ID) {
			current[key] = a
		}
	}
	return current
}

// CurrentAccounts maps each card key to the account reporting for it now: the
// one first seen most recently. After a re-auth the old ID is dead even when
// SimpleFin keeps returning it with a fresh balance_date and a frozen balance,
// so balance recency cannot be trusted to pick it.
func CurrentAccounts(accounts []models.DBAccount) map[string]models.DBAccount {
	var cards []models.DBAccount
	for _, a := range accounts {
		if a.IsCreditCard && a.CardKey != "" {
			cards = append(cards, a)
		}
	}
	return CurrentByKey(cards)
}
```

- [ ] **Step 4: Run the ledger tests**

Run: `devenv shell -- go test ./internal/ledger`
Expected: PASS (including the existing `SupersededAccounts`/`CurrentAccounts` test).

- [ ] **Step 5: Write the failing store test**

In `internal/store/accounts_test.go`, add `"finance_tracker/internal/database"` to the imports and append:

```go
// setFirstSeen pins first_seen_at, which the insert stamps with the current
// second, so tests can order a re-auth's rows.
func setFirstSeen(t *testing.T, db *database.DB, id, at string) {
	t.Helper()
	if _, err := db.Write.Exec(`UPDATE accounts SET first_seen_at = ? WHERE id = ?`, at, id); err != nil {
		t.Fatalf("set first_seen_at for %s: %v", id, err)
	}
}

// The Tangerine re-auth on 2026-09-19 left two rows per account. Settings shows
// one entry per identity, so the list must say which row is the live one.
func TestListMarksTheNewestAccountOfEachIdentityCurrent(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)

	for _, id := range []string{"ACT-old", "ACT-new"} {
		seedAccountFull(t, accts, models.DBAccount{
			ID: id, Name: "Tangerine Chequing Account (2106)", OrgName: "Tangerine Bank (CA)", IsIncluded: true,
		})
	}
	setFirstSeen(t, db, "ACT-old", "2026-03-17 02:27:47")
	setFirstSeen(t, db, "ACT-new", "2026-09-19 17:00:00")

	list, err := accts.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, a := range list {
		if want := a.ID == "ACT-new"; a.IsCurrent != want {
			t.Errorf("%s: is_current = %v, want %v", a.ID, a.IsCurrent, want)
		}
	}
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `devenv shell -- go test ./internal/store -run TestListMarksTheNewestAccountOfEachIdentityCurrent`
Expected: FAIL — `a.IsCurrent undefined`.

- [ ] **Step 7: Add the field and set it in `List`**

In `internal/models/models.go`, add to `DBAccount` after `CardKey`:

```go
	// IsCurrent is computed, not stored: the row reporting for its identity now.
	IsCurrent bool `json:"is_current"`
```

In `internal/store/accounts.go`, in `List`, replace `return accounts, rows.Err()` with:

```go
	if err := rows.Err(); err != nil {
		return nil, err
	}
	current := ledger.CurrentByKey(accounts)
	for i := range accounts {
		accounts[i].IsCurrent = current[ledger.IdentityKey(accounts[i])].ID == accounts[i].ID
	}
	return accounts, nil
```

- [ ] **Step 8: Run the tests**

Run: `devenv shell -- go test ./internal/ledger ./internal/store`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/ledger/card.go internal/ledger/card_test.go internal/models/models.go internal/store/accounts.go internal/store/accounts_test.go
git commit -m "feat(accounts): report which row is current for each identity"
```

---

### Task 2: Make inclusion a property of the identity

**Files:**
- Modify: `internal/store/accounts.go` (`Upsert`, `Update`; add `SetIncluded`, `NormalizeInclusion`)
- Modify: `internal/store/bootstrap.go:17-42`
- Test: `internal/store/accounts_test.go`

**Interfaces:**
- Consumes: `setFirstSeen` (Task 1).
- Produces: `func (s *AccountStore) SetIncluded(ctx context.Context, id string, included bool) error`; `func (s *AccountStore) NormalizeInclusion(ctx context.Context) (int64, error)`; `AccountStore.Update` with `AccountPatch.IsIncluded` set now changes every row of the identity.

- [ ] **Step 1: Write the failing tests**

Append to `internal/store/accounts_test.go`:

```go
func inclusionByID(t *testing.T, accts *AccountStore) map[string]bool {
	t.Helper()
	list, err := accts.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := make(map[string]bool)
	for _, a := range list {
		got[a.ID] = a.IsIncluded
	}
	return got
}

// Excluding an account must also exclude the duplicates a re-auth left behind,
// or their history would keep showing up in spending.
func TestSetIncludedAppliesToEveryRowOfTheIdentity(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	for _, id := range []string{"ACT-old", "ACT-new"} {
		seedAccountFull(t, accts, models.DBAccount{
			ID: id, Name: "Tangerine Chequing Account (2106)", OrgName: "Tangerine Bank (CA)", IsIncluded: true,
		})
	}
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust", IsIncluded: true,
	})

	if err := accts.SetIncluded(ctx, "ACT-new", false); err != nil {
		t.Fatalf("set included: %v", err)
	}

	got := inclusionByID(t, accts)
	if got["ACT-old"] || got["ACT-new"] {
		t.Errorf("both Tangerine rows must be excluded, got %v", got)
	}
	if !got["ACT-td"] {
		t.Errorf("another identity must be left alone, got %v", got)
	}
}

// Regression: the 2026-09-19 Tangerine re-auth issued new IDs, which arrived
// included although the old ones were excluded.
func TestNewAccountInheritsInclusionFromItsIdentity(t *testing.T) {
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-old", Name: "Tangerine Chequing Account (2106)", OrgName: "Tangerine Bank (CA)", IsIncluded: false,
	})
	setFirstSeen(t, db, "ACT-old", "2026-03-17 02:27:47")

	// Sync always upserts with IsIncluded true.
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-new", Name: "Tangerine Chequing Account (2106)", OrgName: "Tangerine Bank (CA)", IsIncluded: true,
	})
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-other", Name: "TD EVERY DAY SAVINGS ACCOUNT (2625)", OrgName: "TD Canada Trust", IsIncluded: true,
	})
	// A later sync of an existing row must not undo a hand exclusion.
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-old", Name: "Tangerine Chequing Account (2106)", OrgName: "Tangerine Bank (CA)", IsIncluded: true,
	})
	got := inclusionByID(t, accts)
	if got["ACT-new"] {
		t.Errorf("a re-auth's new row must inherit the exclusion, got %v", got)
	}
	if got["ACT-old"] {
		t.Errorf("a sync must not re-include an excluded row, got %v", got)
	}
	if !got["ACT-other"] {
		t.Errorf("a never-seen identity arrives included, got %v", got)
	}
}

// Rows written before inclusion followed the identity can disagree. The newest
// row is what the user last saw in the UI, so it wins.
func TestNormalizeInclusionFollowsTheNewestRow(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	for _, id := range []string{"ACT-old", "ACT-new"} {
		seedAccountFull(t, accts, models.DBAccount{
			ID: id, Name: "LINE OF CREDIT UNSECURED (3871)", OrgName: "TD Canada Trust", IsIncluded: true,
		})
	}
	setFirstSeen(t, db, "ACT-old", "2026-07-25 10:00:00")
	setFirstSeen(t, db, "ACT-new", "2026-09-11 23:41:21")
	if _, err := db.Write.Exec(`UPDATE accounts SET is_included = 0 WHERE id = 'ACT-old'`); err != nil {
		t.Fatal(err)
	}

	changed, err := accts.NormalizeInclusion(ctx)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if changed != 1 {
		t.Errorf("expected 1 row changed, got %d", changed)
	}
	if got := inclusionByID(t, accts); !got["ACT-old"] || !got["ACT-new"] {
		t.Errorf("both rows must follow the newest one (included), got %v", got)
	}

	again, err := accts.NormalizeInclusion(ctx)
	if err != nil {
		t.Fatalf("normalize again: %v", err)
	}
	if again != 0 {
		t.Errorf("a second run must be a no-op, changed %d", again)
	}
}

// PATCH /api/accounts/{id} goes through Update, so it must propagate too.
func TestUpdateIncludedAppliesToTheIdentity(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	for _, id := range []string{"ACT-old", "ACT-new"} {
		seedAccountFull(t, accts, models.DBAccount{
			ID: id, Name: "Tangerine Savings Account (1673)", OrgName: "Tangerine Bank (CA)", IsIncluded: true,
		})
	}

	off := false
	if err := accts.Update(ctx, "ACT-new", AccountPatch{IsIncluded: &off}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := inclusionByID(t, accts); got["ACT-old"] || got["ACT-new"] {
		t.Errorf("both rows must be excluded, got %v", got)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `devenv shell -- go test ./internal/store -run 'Inclusion|SetIncluded|UpdateIncluded'`
Expected: FAIL — `accts.SetIncluded undefined`, `accts.NormalizeInclusion undefined`.

- [ ] **Step 3: Make `Upsert` inherit inclusion on insert**

In `internal/store/accounts.go`, update the comment and the `is_included` value in `Upsert`:

```go
	// Card fields are set on insert only: they are a guess from the name, and a
	// hand correction must survive every later sync. Inclusion is likewise set
	// on insert only, and a new row takes it from its identity so a re-auth's
	// fresh IDs keep the user's choice.
	...
	_, err := s.write.ExecContext(ctx, `
		INSERT INTO accounts (id, name, balance, balance_date, currency, org_name, org_domain, is_included,
			anchor_balance, anchor_balance_date, is_credit_card, card_key, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?,
			COALESCE((SELECT is_included FROM accounts WHERE card_key = ?
				ORDER BY first_seen_at DESC, id DESC LIMIT 1), ?),
			?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			balance = excluded.balance,
			balance_date = excluded.balance_date,
			currency = excluded.currency,
			org_name = excluded.org_name,
			org_domain = excluded.org_domain,
			updated_at = datetime('now')`,
		acct.ID, acct.Name, acct.Balance, acct.BalanceDate, acct.Currency, acct.OrgName, acct.OrgDomain,
		cardKey, acct.IsIncluded,
		acct.Balance, acct.BalanceDate, isCard, cardKey,
	)
```

- [ ] **Step 4: Add `SetIncluded` and route `Update` through it**

Add below `GetByID`:

```go
// SetIncluded includes or excludes an account together with every other row of
// its identity, so the choice covers the duplicates a re-auth leaves behind.
func (s *AccountStore) SetIncluded(ctx context.Context, id string, included bool) error {
	_, err := s.write.ExecContext(ctx, `
		UPDATE accounts SET is_included = ?, updated_at = datetime('now')
		WHERE id = ? OR card_key = (SELECT card_key FROM accounts WHERE id = ? AND card_key != '')`,
		included, id, id)
	return err
}
```

In `Update`, delete the `if p.IsIncluded != nil { sets, args = … "is_included = ?" … }` block and replace it with:

```go
	if p.IsIncluded != nil {
		if err := s.SetIncluded(ctx, id, *p.IsIncluded); err != nil {
			return err
		}
	}
```

(`Update`'s early `return nil` when `sets` is empty still applies to the remaining per-row fields.)

- [ ] **Step 5: Add `NormalizeInclusion`**

Add after `BackfillCardIdentity`:

```go
// NormalizeInclusion makes every row of an identity agree with its newest row,
// and returns how many rows it changed. Rows written before inclusion followed
// the identity can disagree: a re-auth inserted its new IDs included whatever
// the old ones said.
func (s *AccountStore) NormalizeInclusion(ctx context.Context) (int64, error) {
	const newest = `(SELECT b.is_included FROM accounts b WHERE b.card_key = accounts.card_key
		ORDER BY b.first_seen_at DESC, b.id DESC LIMIT 1)`
	res, err := s.write.ExecContext(ctx, `
		UPDATE accounts SET is_included = `+newest+`
		WHERE card_key IS NOT NULL AND card_key != '' AND is_included != `+newest)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

- [ ] **Step 6: Run the store tests**

Run: `devenv shell -- go test ./internal/store`
Expected: PASS (new tests and all existing ones).

- [ ] **Step 7: Run it at startup**

In `internal/store/bootstrap.go`, after the `BackfillCardIdentity` error check, add:

```go
	// After classification, so every row has the key the pass groups by.
	normalized, err := accts.NormalizeInclusion(ctx)
	if err != nil {
		return fmt.Errorf("normalize account inclusion: %w", err)
	}
```

and extend the final log line:

```go
	log.Info().Int("accounts_classified", classified).Int64("inclusion_normalized", normalized).
		Int("snapshots_seeded", seeded).Msg("Card ledger ready")
```

Update the function's doc comment first sentence to: `It classifies accounts stored before cards were tracked, reconciles inclusion across each account's identity, seeds the balance history…`.

- [ ] **Step 8: Run all Go tests and vet**

Run: `devenv shell -- go vet ./internal/... ./cmd/... && devenv shell -- go test ./internal/... ./cmd/...`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/store/accounts.go internal/store/accounts_test.go internal/store/bootstrap.go
git commit -m "feat(store): make account inclusion follow the account's identity"
```

---

### Task 3: API — 404 for unknown accounts, handler tests

**Files:**
- Modify: `internal/api/accounts.go:27-46`
- Create: `internal/api/accounts_test.go`

**Interfaces:**
- Consumes: `AccountStore.Update` propagation and `List`'s `IsCurrent` (Tasks 1–2).
- Produces: `PATCH /api/accounts/{id}` returns `404 NOT_FOUND` for an unknown ID; `GET /api/accounts` rows carry `is_current`.

- [ ] **Step 1: Write the failing tests**

Create `internal/api/accounts_test.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

func seedTangerineReauth(t *testing.T) *store.AccountStore {
	t.Helper()
	db := newTestDB(t)
	accts := store.NewAccountStore(db.Read, db.Write)
	for _, id := range []string{"ACT-old", "ACT-new"} {
		if err := accts.Upsert(context.Background(), models.DBAccount{
			ID: id, Name: "Tangerine Chequing Account (2106)", OrgName: "Tangerine Bank (CA)", IsIncluded: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for id, at := range map[string]string{"ACT-old": "2026-03-17 02:27:47", "ACT-new": "2026-09-19 17:00:00"} {
		if _, err := db.Write.Exec(`UPDATE accounts SET first_seen_at = ? WHERE id = ?`, at, id); err != nil {
			t.Fatal(err)
		}
	}
	return accts
}

func patchAccount(h *AccountHandler, id, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	return rec
}

func TestAccountPatchExcludesTheWholeIdentity(t *testing.T) {
	accts := seedTangerineReauth(t)
	h := NewAccountHandler(accts)

	rec := patchAccount(h, "ACT-new", `{"is_included":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}

	list, err := accts.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.IsIncluded {
			t.Errorf("%s is still included", a.ID)
		}
	}
}

func TestAccountPatchUnknownIDIsNotFound(t *testing.T) {
	h := NewAccountHandler(seedTangerineReauth(t))

	rec := patchAccount(h, "ACT-missing", `{"is_included":false}`)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "NOT_FOUND") {
		t.Errorf("expected 404 NOT_FOUND, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestAccountListReportsTheCurrentRow(t *testing.T) {
	h := NewAccountHandler(seedTangerineReauth(t))

	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))

	var body struct {
		Data []models.DBAccount `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	current := map[string]bool{}
	for _, a := range body.Data {
		current[a.ID] = a.IsCurrent
	}
	if !current["ACT-new"] || current["ACT-old"] {
		t.Errorf("only ACT-new is current, got %v", current)
	}
}
```

- [ ] **Step 2: Run them to verify the 404 test fails**

Run: `devenv shell -- go test ./internal/api -run TestAccount`
Expected: `TestAccountPatchUnknownIDIsNotFound` FAILS (gets 200); the other two PASS already thanks to Tasks 1–2.

- [ ] **Step 3: Return 404 for unknown IDs**

In `internal/api/accounts.go` `Update`, after the `card_key` empty check, add:

```go
	existing, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if existing == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "Account not found")
		return
	}
```

- [ ] **Step 4: Run the API tests**

Run: `devenv shell -- go test ./internal/api`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/accounts.go internal/api/accounts_test.go
git commit -m "feat(api): 404 on unknown accounts and cover identity-wide inclusion"
```

---

### Task 4: Settings "Accounts" tab

**Files:**
- Modify: `web/src/api/types.ts:23-34` (`DBAccount`)
- Modify: `web/src/api/queries.ts` (add `patchApi` after `postApi`; add `useSetAccountsIncluded` after `useAccounts`)
- Create: `web/src/components/AccountsSettings.tsx`
- Modify: `web/src/pages/Settings.tsx:1-50`

**Interfaces:**
- Consumes: `GET /api/accounts` (`is_current`, `is_credit_card`), `PATCH /api/accounts/{id}` `{ "is_included": boolean }`.
- Produces: `useSetAccountsIncluded()` mutation taking `{ ids: string[]; included: boolean }`; `<AccountsSettings />`.

- [ ] **Step 1: Extend the type**

In `web/src/api/types.ts`, add to `DBAccount` after `is_included`:

```ts
  is_credit_card: boolean;
  card_key: string;
  is_current: boolean;
```

- [ ] **Step 2: Add `patchApi` and the mutation**

In `web/src/api/queries.ts`, after `postApi`:

```ts
async function patchApi<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: res.statusText } }));
    throw new Error(err.error?.message || res.statusText);
  }
  const json: ApiResponse<T> = await res.json();
  return json.data;
}
```

After `useAccounts`:

```ts
// Includes or excludes accounts. The server applies each change to every row of
// the account's identity, so the duplicates a re-auth left behind follow along.
export function useSetAccountsIncluded() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ ids, included }: { ids: string[]; included: boolean }) => {
      const results = await Promise.allSettled(
        ids.map((id) => patchApi(`/api/accounts/${encodeURIComponent(id)}`, { is_included: included })),
      );
      const failed = results.filter((r) => r.status === "rejected").length;
      if (failed > 0) {
        throw new Error(`${failed} of ${ids.length} accounts could not be updated`);
      }
    },
    // Settled, not success: after a partial failure the list must still show
    // what the server actually holds.
    onSettled: () => {
      for (const key of ["accounts", "transactions", "dashboard", "analytics", "budgets"]) {
        qc.invalidateQueries({ queryKey: [key] });
      }
    },
  });
}
```

- [ ] **Step 3: Create the component**

Create `web/src/components/AccountsSettings.tsx`:

```tsx
import { useAccounts, useSetAccountsIncluded } from "@/api/queries";
import type { DBAccount } from "@/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";

function groupByInstitution(accounts: DBAccount[]): [string, DBAccount[]][] {
  const groups = new Map<string, DBAccount[]>();
  for (const a of accounts) {
    const org = a.org_name || "Other";
    groups.set(org, [...(groups.get(org) ?? []), a]);
  }
  return [...groups.entries()].sort(([x], [y]) => x.localeCompare(y));
}

function formatBalance(balance: number): string {
  return `${balance < 0 ? "-" : ""}$${Math.abs(balance).toFixed(2)}`;
}

export function AccountsSettings() {
  const { data: accounts, isLoading } = useAccounts();
  const setIncluded = useSetAccountsIncluded();

  if (isLoading) return <p className="text-sm text-muted-foreground">Loading accounts...</p>;

  // One entry per identity: the row reporting now. Toggling it covers the
  // older rows a re-auth left behind.
  const groups = groupByInstitution((accounts ?? []).filter((a) => a.is_current));
  if (groups.length === 0) {
    return <p className="text-sm text-muted-foreground">No accounts yet. Run a sync first.</p>;
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Excluded accounts are left out of spending, analysis and connection alerts. They stay connected in
        SimpleFin and are still checked for payments toward your cards.
      </p>
      {setIncluded.error && <p className="text-sm text-destructive">{setIncluded.error.message}</p>}
      {groups.map(([org, accts]) => (
        <Card key={org}>
          <CardHeader>
            <CardTitle className="flex items-center gap-3">
              <Checkbox
                checked={accts.some((a) => a.is_included)}
                disabled={setIncluded.isPending}
                onCheckedChange={(v) => setIncluded.mutate({ ids: accts.map((a) => a.id), included: v === true })}
                aria-label={`Include ${org} accounts`}
              />
              {org}
            </CardTitle>
          </CardHeader>
          <CardContent className="divide-y">
            {accts.map((a) => (
              <div key={a.id} className="flex items-center gap-3 py-2">
                <Checkbox
                  id={`account-${a.id}`}
                  checked={a.is_included}
                  disabled={setIncluded.isPending}
                  onCheckedChange={(v) => setIncluded.mutate({ ids: [a.id], included: v === true })}
                />
                <label htmlFor={`account-${a.id}`} className="flex-1 min-w-0 text-sm cursor-pointer">
                  <span className={a.is_included ? "font-medium" : "font-medium text-muted-foreground line-through"}>
                    {a.name}
                  </span>
                  {a.is_credit_card && (
                    <Badge variant="secondary" className="ml-2 text-xs px-1.5 py-0">card</Badge>
                  )}
                  <span className="block text-xs text-muted-foreground">
                    updated {new Date(a.balance_date * 1000).toLocaleDateString()}
                  </span>
                </label>
                <span className="text-sm tabular-nums">{formatBalance(a.balance)}</span>
              </div>
            ))}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
```

- [ ] **Step 4: Add the tab**

In `web/src/pages/Settings.tsx`:
- add `import { AccountsSettings } from "@/components/AccountsSettings";`
- after the Categories `TabsTrigger`, add `<TabsTrigger value="accounts">Accounts</TabsTrigger>`
- after the closing `</TabsContent>` of `value="categories"`, add:

```tsx
        <TabsContent value="accounts" className="space-y-4">
          <AccountsSettings />
        </TabsContent>
```

- update the intro paragraph to: `Configuration loaded from <code …>.env</code>. Categories and accounts can be included/excluded from analysis below.`

- [ ] **Step 5: Type-check, build and lint**

Run: `devenv shell -- just build-web && devenv shell -- bash -c 'cd web && npx eslint src/components/AccountsSettings.tsx src/api/queries.ts src/pages/Settings.tsx'`
Expected: build succeeds; eslint reports no errors in these files.

- [ ] **Step 6: Commit**

```bash
git add web/src/api/types.ts web/src/api/queries.ts web/src/components/AccountsSettings.tsx web/src/pages/Settings.tsx
git commit -m "feat(web): choose included accounts per account and institution in Settings"
```

---

### Task 5: Transactions page hides excluded accounts by default

**Files:**
- Modify: `web/src/pages/Transactions.tsx:1-10, 39, 104, 162, 210-225`

**Interfaces:**
- Consumes: `included_only=true` on `/api/transactions`, `/api/transactions/summary`, `/api/transactions/export` (already supported server-side).
- Produces: URL parameter `show_excluded=true` on the Transactions page.

- [ ] **Step 1: Flip the default**

- Add `import { Checkbox } from "@/components/ui/checkbox";`.
- Replace `const urlIncludedOnly = searchParams.get("included_only") === "true";` with:

```tsx
  // Excluded accounts are hidden unless asked for. Old links carrying
  // included_only=true land on the same default.
  const urlShowExcluded = searchParams.get("show_excluded") === "true";
```

- Replace `if (urlIncludedOnly) apiParams.included_only = "true";` with:

```tsx
  if (!urlShowExcluded) apiParams.included_only = "true";
```

- Change `const hasActiveFilters = urlCategory || urlSearch || hasCustomRange;` to:

```tsx
  const hasActiveFilters = urlCategory || urlSearch || hasCustomRange || urlShowExcluded;
```

- [ ] **Step 2: Add the checkbox**

In the filter row, directly before `{hasActiveFilters && (`, add:

```tsx
        <label className="flex items-center gap-2 text-sm text-muted-foreground whitespace-nowrap cursor-pointer">
          <Checkbox
            checked={urlShowExcluded}
            onCheckedChange={(v) => updateParams({ show_excluded: v === true ? "true" : null, page: "1" })}
          />
          Show excluded accounts
        </label>
```

- [ ] **Step 3: Type-check, build and lint**

Run: `devenv shell -- just build-web && devenv shell -- bash -c 'cd web && npx eslint src/pages/Transactions.tsx'`
Expected: build succeeds; no eslint errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/Transactions.tsx
git commit -m "feat(web): hide excluded accounts on Transactions unless asked"
```

---

### Task 6: Verify against a copy of production, document

**Files:**
- Modify: `CLAUDE.md` (the "Balance-Trusted Card Spending (web server)" section)

- [ ] **Step 1: Take a fresh read-only copy of the production DB**

```bash
S=/tmp/claude-1000/-home-arosenfeld-Code-finance-tracker/3d0d89d8-5464-444e-89e7-6f9db0911549/scratchpad/verify
mkdir -p $S && scp -q 'root@galactica:/var/data/finance-tracker/finance_tracker.db*' $S/
```

(Use the session scratchpad of whoever runs this; never write back to galactica.)

- [ ] **Step 2: Start the server on the copy with SimpleFin disabled**

```bash
devenv shell -- just build
env -u SIMPLEFIN_BRIDGE_URL DB_PATH=$S/finance_tracker.db ENV_FILE=/dev/null LISTEN_ADDR=127.0.0.1:18099 ./bin/finance_server
```

Run it in the background. Expected log: `Card ledger ready` with `inclusion_normalized` > 0 (the old excluded Tangerine and TD LOC rows get pulled to their current rows' value), and no "Scheduled periodic sync" line.

- [ ] **Step 3: Exclude Tangerine through the API and check the summary**

```bash
P=$(curl -s 'http://127.0.0.1:18099/api/billing-periods' | jq '.data[-1]')
START=$(jq .start <<<"$P"); END=$(jq .end <<<"$P")
curl -s "http://127.0.0.1:18099/api/transactions/summary?start=$START&end=$END&included_only=true" | jq '.data.total_spending'
TANG=$(curl -s http://127.0.0.1:18099/api/accounts | jq -c '[.data[] | select(.org_name|startswith("Tangerine")) | .id]')
for id in $(curl -s http://127.0.0.1:18099/api/accounts | jq -r '.data[] | select(.is_current and (.org_name|startswith("Tangerine"))) | .id'); do
  curl -s -X PATCH -H 'Content-Type: application/json' -d '{"is_included":false}' "http://127.0.0.1:18099/api/accounts/$id" | jq -c .
done
curl -s http://127.0.0.1:18099/api/accounts | jq '[.data[] | select(.org_name|startswith("Tangerine")) | .is_included] | unique'
curl -s "http://127.0.0.1:18099/api/transactions?start=$START&end=$END&included_only=true&include_positive=true&limit=500" \
  | jq --argjson tang "$TANG" '[.data[] | select(.account_id as $a | $tang | index($a))] | length'
curl -s "http://127.0.0.1:18099/api/transactions/summary?start=$START&end=$END&included_only=true" | jq '.data.total_spending'
curl -s "http://127.0.0.1:18099/api/transactions/summary?start=$START&end=$END" | jq '.data.total_spending'
```

Expected: every Tangerine row (current and old) reports `[false]`; no Tangerine transactions in the included-only list; the included-only total drops versus the first reading; the unfiltered total (what `show_excluded=true` shows) is unchanged from before.

- [ ] **Step 4: Eyeball the UI**

Open `http://127.0.0.1:18099/settings` → Accounts tab: Tangerine's box is unchecked, one row per identity (TD LOC appears once). Open Transactions: no Tangerine rows; ticking "Show excluded accounts" brings them back and adds `show_excluded=true` to the URL. Stop the server afterwards.

- [ ] **Step 5: Document**

In `CLAUDE.md`, under "Balance-Trusted Card Spending (web server)", add a bullet:

```markdown
- **Account inclusion** (`accounts.is_included`) belongs to an account's identity (`card_key`, falling back to its ID): `PATCH /api/accounts/{id}` sets it on every row of the key, a re-auth's new rows inherit it, and startup reconciles disagreeing rows toward the newest one. Settings → Accounts edits it per account or per institution. Excluded accounts are hidden from spending (the Transactions page sends `included_only=true` unless `show_excluded=true`), analysis and alerts, but non-card accounts are still scanned for card payments.
```

- [ ] **Step 6: Final checks and commit**

Run: `devenv shell -- go vet ./internal/... ./cmd/... && devenv shell -- go test ./internal/... ./cmd/... && devenv shell -- just build-web`
Expected: all PASS.

```bash
git add CLAUDE.md
git commit -m "docs: describe identity-wide account inclusion"
```

After deploy, the user turns Tangerine off once in Settings → Accounts (normalization aligns rows to their current row, and the current Tangerine rows are included).
