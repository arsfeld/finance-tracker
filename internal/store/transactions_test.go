package store

import (
	"context"
	"testing"

	"finance_tracker/internal/models"
)

func seedAccount(t *testing.T, accts *AccountStore, id string) {
	t.Helper()
	err := accts.Upsert(context.Background(), models.DBAccount{
		ID: id, Name: id, IsIncluded: true,
	})
	if err != nil {
		t.Fatalf("seed account %s: %v", id, err)
	}
}

func upsertStores(t *testing.T) (*AccountStore, *TransactionStore) {
	t.Helper()
	db := newTestDB(t)
	return NewAccountStore(db.Read, db.Write), NewTransactionStore(db.Read, db.Write)
}

// The sync log is the only place a dead bank connection surfaces, so its counts
// have to be true. Regression test: every upserted row was counted as "added",
// so a feed returning the same sliding window for weeks logged "107 added" each
// run and looked perfectly healthy while zero new transactions landed.
func TestUpsertBatchCountsOnlyNewRowsAsAdded(t *testing.T) {
	ctx := context.Background()
	accts, txns := upsertStores(t)
	seedAccount(t, accts, "ACT-1")

	batch := []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-1", Description: "METRO BLANCHARD", Amount: -67.05, Posted: 1750000000},
		{ID: "t2", AccountID: "ACT-1", Description: "IGA EXTRA", Amount: -20.00, Posted: 1750000100},
	}

	added, updated, err := txns.UpsertBatch(ctx, batch)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if added != 2 || updated != 0 {
		t.Fatalf("first sync of 2 new transactions: want added=2 updated=0, got added=%d updated=%d", added, updated)
	}

	// The next sync re-fetches the same window and gets the same rows back.
	added, updated, err = txns.UpsertBatch(ctx, batch)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if added != 0 {
		t.Errorf("re-syncing unchanged transactions must report 0 added, got %d", added)
	}
	if updated != 0 {
		t.Errorf("re-syncing unchanged transactions must report 0 updated, got %d", updated)
	}
}

// A pending charge that settles changes its amount and pending flag. That is a
// real change and must be reported as an update, not silently folded into the
// unchanged count.
func TestUpsertBatchCountsChangedRowsAsUpdated(t *testing.T) {
	ctx := context.Background()
	accts, txns := upsertStores(t)
	seedAccount(t, accts, "ACT-1")

	pending := []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-1", Description: "PETRO-CANADA", Amount: -100.00, Posted: 1750000000, Pending: true},
	}
	if _, _, err := txns.UpsertBatch(ctx, pending); err != nil {
		t.Fatalf("seed pending transaction: %v", err)
	}

	settled := []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-1", Description: "PETRO-CANADA", Amount: -95.15, Posted: 1750000000, Pending: false},
	}
	added, updated, err := txns.UpsertBatch(ctx, settled)
	if err != nil {
		t.Fatalf("upsert settled transaction: %v", err)
	}
	if added != 0 {
		t.Errorf("a settling transaction is not new: want added=0, got %d", added)
	}
	if updated != 1 {
		t.Errorf("a settling transaction changed amount and pending: want updated=1, got %d", updated)
	}
}

// A batch that repeats an id must not count the repeat as a second new row.
func TestUpsertBatchCountsDuplicateIDsInOneBatchOnce(t *testing.T) {
	ctx := context.Background()
	accts, txns := upsertStores(t)
	seedAccount(t, accts, "ACT-1")

	added, _, err := txns.UpsertBatch(ctx, []models.DBTransaction{
		{ID: "t1", AccountID: "ACT-1", Description: "METRO", Amount: -10, Posted: 1750000000},
		{ID: "t1", AccountID: "ACT-1", Description: "METRO", Amount: -10, Posted: 1750000000},
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if added != 1 {
		t.Errorf("the same id twice in one batch is one new transaction, got added=%d", added)
	}
}
