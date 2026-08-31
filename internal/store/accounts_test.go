package store

import (
	"context"
	"testing"
	"time"

	"finance_tracker/internal/models"
)

func seedAccountFull(t *testing.T, accts *AccountStore, acct models.DBAccount) {
	t.Helper()
	if err := accts.Upsert(context.Background(), acct); err != nil {
		t.Fatalf("seed account %s: %v", acct.ID, err)
	}
}

func findStale(conns []models.StaleConnection, name string) *models.StaleConnection {
	for i := range conns {
		if conns[i].Name == name {
			return &conns[i]
		}
	}
	return nil
}

// SimpleFin advances balance_date on every successful refresh, so a frozen
// balance_date means the connection is dead. A gap in transactions does not:
// a card can legitimately go quiet for a week. Regression test: the TD card
// stopped refreshing on Aug 21 and nothing noticed for ten days.
func TestStaleConnectionsFlagsAccountsThatStoppedRefreshing(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		BalanceDate: now.AddDate(0, 0, -10).Unix(), IsIncluded: true,
	})
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-loc", Name: "LINE OF CREDIT UNSECURED (3871)", OrgName: "TD Canada Trust",
		BalanceDate: now.Add(-1 * time.Hour).Unix(), IsIncluded: true,
	})

	stale, err := accts.StaleConnections(ctx, now, 72*time.Hour)
	if err != nil {
		t.Fatalf("stale connections: %v", err)
	}

	if findStale(stale, "TD AEROPLAN VISA INFINITE (4520)") == nil {
		t.Errorf("an account that has not refreshed in 10 days is stale; got %+v", stale)
	}
	if findStale(stale, "LINE OF CREDIT UNSECURED (3871)") != nil {
		t.Errorf("an account refreshed an hour ago is not stale; got %+v", stale)
	}
}

// Accounts the user has excluded are not being analyzed, so a dead connection
// on one is not worth waking anyone up for. Without this the five Tangerine
// accounts, dead since 2025, would alert on every sync forever.
func TestStaleConnectionsIgnoresExcludedAccounts(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-tang", Name: "Tangerine Chequing Account (2106)", OrgName: "Tangerine Bank (CA)",
		BalanceDate: now.AddDate(0, -9, 0).Unix(), IsIncluded: false,
	})

	stale, err := accts.StaleConnections(ctx, now, 72*time.Hour)
	if err != nil {
		t.Fatalf("stale connections: %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("excluded accounts must stay quiet however stale; got %+v", stale)
	}
}

// The alert needs to say when data actually stops, not just when the balance
// froze, so the reader can tell how much spending is missing.
func TestStaleConnectionsReportsLastTransactionDate(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	txns := NewTransactionStore(db.Read, db.Write)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	lastTxn := time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		BalanceDate: now.AddDate(0, 0, -10).Unix(), IsIncluded: true,
	})
	if _, _, err := txns.UpsertBatch(ctx, []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-td", Description: "METRO", Amount: -67.05, Posted: lastTxn.Unix()},
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	stale, err := accts.StaleConnections(ctx, now, 72*time.Hour)
	if err != nil {
		t.Fatalf("stale connections: %v", err)
	}
	got := findStale(stale, "TD AEROPLAN VISA INFINITE (4520)")
	if got == nil {
		t.Fatalf("expected the TD card to be stale; got %+v", stale)
	}
	if got.LastTransaction != lastTxn.Unix() {
		t.Errorf("want last transaction %d, got %d", lastTxn.Unix(), got.LastTransaction)
	}
}
