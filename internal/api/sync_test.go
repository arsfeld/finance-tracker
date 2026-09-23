package api

import (
	"context"
	"testing"
	"time"

	"finance_tracker/internal/config"
	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

// Regression: a re-auth of a non-card account (e.g. a line of credit) leaves
// its old ID included -- NormalizeInclusion and identity-wide setIncluded make
// sure of that -- with a balance_date frozen forever. Before this fix, only
// credit-card rows were recognized as superseded (ledger.SupersededAccounts is
// card-only), so a dead non-card duplicate kept firing a "not reporting" alert
// on every sync, with no way to silence it short of excluding the live account.
func TestAlertOnStaleConnectionsSkipsSupersededNonCardRows(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := store.NewAccountStore(db.Read, db.Write)
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

	for _, a := range []models.DBAccount{
		{ID: "LOC-old", Name: "LINE OF CREDIT UNSECURED (3871)", OrgName: "TD Canada Trust",
			IsIncluded: true, BalanceDate: now.AddDate(0, 0, -10).Unix()},
		{ID: "LOC-new", Name: "LINE OF CREDIT UNSECURED (3871)", OrgName: "TD Canada Trust",
			IsIncluded: true, BalanceDate: now.Add(-1 * time.Hour).Unix()},
	} {
		if err := accts.Upsert(ctx, a); err != nil {
			t.Fatalf("seed %s: %v", a.ID, err)
		}
	}
	for id, at := range map[string]string{"LOC-old": "2026-01-01 00:00:00", "LOC-new": "2026-09-20 00:00:00"} {
		if _, err := db.Write.Exec(`UPDATE accounts SET first_seen_at = ? WHERE id = ?`, at, id); err != nil {
			t.Fatal(err)
		}
	}

	h := &SyncHandler{cfg: &config.Config{}, accounts: accts, events: NewEventHub()}
	ch := make(chan string, 4)
	h.events.mu.Lock()
	h.events.clients[ch] = struct{}{}
	h.events.mu.Unlock()

	h.alertOnStaleConnections(ctx, now, nil)

	select {
	case msg := <-ch:
		t.Errorf("the superseded LOC row must not trigger a stale alert, got broadcast: %s", msg)
	default:
	}
}
