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
