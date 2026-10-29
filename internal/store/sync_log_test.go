package store

import (
	"context"
	"testing"
)

func TestSyncLogCreateEntryAndList(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	logs := NewSyncLogStore(db.Read, db.Write)

	id, err := logs.CreateEntry(ctx, "import", "accountactivity.qfx", "ACT-new")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := logs.Complete(ctx, id, "success", 50, 0, "", "[]"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	entries, err := logs.List(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.Source != "import" || got.Filename != "accountactivity.qfx" || got.AccountID != "ACT-new" {
		t.Errorf("entry = %+v", got)
	}
	if got.TransactionsAdded != 50 {
		t.Errorf("added = %d, want 50", got.TransactionsAdded)
	}
}

func TestSyncLogCreateDefaultsToSimpleFIN(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	logs := NewSyncLogStore(db.Read, db.Write)

	if _, err := logs.Create(ctx); err != nil {
		t.Fatalf("create: %v", err)
	}
	entries, err := logs.List(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 || entries[0].Source != "simplefin" {
		t.Errorf("entry = %+v, want source simplefin", entries)
	}
}
