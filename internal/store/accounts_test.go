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

// A connection that refreshes balances but stops delivering transactions looks
// perfectly healthy: balance_date stays current, so the staleness check is
// quiet, while spending quietly vanishes from every report. Regression test:
// the TD card's balance moved $1,876.50 with no transaction to explain it.
func TestUnreconciledAccountsFlagsBalanceMovesWithNoTransactions(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	anchor := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

	// First sight of the account fixes the anchor at -10,247.84.
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		Balance: -10247.84, BalanceDate: anchor.Unix(), IsIncluded: true,
	})

	// Ten days later the balance has moved but no transactions arrived.
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
		Balance: -12124.34, BalanceDate: anchor.AddDate(0, 0, 10).Unix(), IsIncluded: true,
	})

	drifted, err := accts.UnreconciledAccounts(ctx, 250)
	if err != nil {
		t.Fatalf("unreconciled accounts: %v", err)
	}
	if len(drifted) != 1 {
		t.Fatalf("an unexplained $1,876.50 move must be flagged; got %+v", drifted)
	}
	if got := drifted[0].Unexplained; got > -1876.0 || got < -1877.0 {
		t.Errorf("want roughly -1876.50 unexplained, got %.2f", got)
	}
}

// When the transactions do arrive, they must account for the move and close the
// gap rather than leaving a permanent complaint.
func TestUnreconciledAccountsQuietWhenTransactionsExplainTheMove(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	txns := NewTransactionStore(db.Read, db.Write)
	anchor := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)",
		Balance: -10247.84, BalanceDate: anchor.Unix(), IsIncluded: true,
	})
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)",
		Balance: -12124.34, BalanceDate: anchor.AddDate(0, 0, 10).Unix(), IsIncluded: true,
	})
	if _, _, err := txns.UpsertBatch(ctx, []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-td", Description: "COSTCO", Amount: -1876.50,
			Posted: anchor.AddDate(0, 0, 5).Unix()},
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	drifted, err := accts.UnreconciledAccounts(ctx, 250)
	if err != nil {
		t.Fatalf("unreconciled accounts: %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("the transaction explains the move, so nothing is unreconciled; got %+v", drifted)
	}
}

// Transactions delivered late carry a posted date at or before the anchor. They
// were already priced into the anchor balance, so counting them would invent a
// discrepancy the moment a backfill lands.
func TestUnreconciledAccountsIgnoresTransactionsBeforeTheAnchor(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	txns := NewTransactionStore(db.Read, db.Write)
	anchor := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)",
		Balance: -12124.34, BalanceDate: anchor.Unix(), IsIncluded: true,
	})
	// A month of backfill finally arrives, all of it predating the anchor.
	if _, _, err := txns.UpsertBatch(ctx, []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-td", Description: "BACKFILL", Amount: -1876.50,
			Posted: anchor.AddDate(0, 0, -14).Unix()},
	}); err != nil {
		t.Fatalf("seed backfill: %v", err)
	}

	drifted, err := accts.UnreconciledAccounts(ctx, 250)
	if err != nil {
		t.Fatalf("unreconciled accounts: %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("backfill predating the anchor must not create drift; got %+v", drifted)
	}
}

// Small gaps are the normal lag between a charge hitting the balance and
// posting, and they close on their own. Only a material gap is worth reporting.
func TestUnreconciledAccountsIgnoresGapsUnderThreshold(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	anchor := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)",
		Balance: -100.00, BalanceDate: anchor.Unix(), IsIncluded: true,
	})
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-td", Name: "TD AEROPLAN VISA INFINITE (4520)",
		Balance: -140.00, BalanceDate: anchor.AddDate(0, 0, 1).Unix(), IsIncluded: true,
	})

	drifted, err := accts.UnreconciledAccounts(ctx, 250)
	if err != nil {
		t.Fatalf("unreconciled accounts: %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("a $40 gap is pending-charge lag, not a fault; got %+v", drifted)
	}
}

// An excluded account is not analyzed, so its drift is nobody's problem.
func TestUnreconciledAccountsIgnoresExcludedAccounts(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	accts := NewAccountStore(db.Read, db.Write)
	anchor := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-x", Name: "Tangerine Chequing Account (2106)",
		Balance: -100.00, BalanceDate: anchor.Unix(), IsIncluded: false,
	})
	seedAccountFull(t, accts, models.DBAccount{
		ID: "ACT-x", Name: "Tangerine Chequing Account (2106)",
		Balance: -9000.00, BalanceDate: anchor.AddDate(0, 0, 1).Unix(), IsIncluded: false,
	})

	drifted, err := accts.UnreconciledAccounts(ctx, 250)
	if err != nil {
		t.Fatalf("unreconciled accounts: %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("excluded accounts must stay quiet; got %+v", drifted)
	}
}
