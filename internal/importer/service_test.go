package importer

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"finance_tracker/internal/database"
	"finance_tracker/internal/models"
	"finance_tracker/internal/ofx"
	"finance_tracker/internal/store"
)

func newTestStores(t *testing.T) (*database.DB, *store.AccountStore, *store.TransactionStore, *store.SnapshotStore) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(db.Write); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db,
		store.NewAccountStore(db.Read, db.Write),
		store.NewTransactionStore(db.Read, db.Write),
		store.NewSnapshotStore(db.Read, db.Write)
}

func seedIdentity(t *testing.T, db *database.DB, accts *store.AccountStore) {
	t.Helper()
	ctx := context.Background()
	for _, id := range []string{"ACT-old", "ACT-new"} {
		if err := accts.Upsert(ctx, models.DBAccount{
			ID: id, Name: "TD AEROPLAN VISA INFINITE (4520)", OrgName: "TD Canada Trust",
			Balance: 100, BalanceDate: 1780000000, IsIncluded: true,
		}); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	for id, at := range map[string]string{"ACT-old": "2026-01-01 00:00:00", "ACT-new": "2026-09-20 00:00:00"} {
		if _, err := db.Write.Exec(`UPDATE accounts SET first_seen_at = ? WHERE id = ?`, at, id); err != nil {
			t.Fatal(err)
		}
	}
}

func sampleStatement() *ofx.Statement {
	balance := -3187.08
	return &ofx.Statement{
		AccountID:     "4520883071874533",
		Currency:      "CAD",
		LedgerBalance: &balance,
		BalanceDate:   time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC),
		Start:         time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
		End:           time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC),
		Transactions: []ofx.Transaction{
			{FITID: "fit-1", Type: "DEBIT", Name: "COFFEE", Amount: -4.50, Posted: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)},
			{FITID: "fit-2", Type: "CREDIT", Name: "PAYMENT - THANK YOU", Amount: 4236, Posted: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)},
		},
	}
}

func TestPlanFreshStatement(t *testing.T) {
	db, accts, txns, snaps := newTestStores(t)
	seedIdentity(t, db, accts)
	svc := New(accts, txns, snaps)

	plan, err := svc.Plan(context.Background(), "ACT-new", sampleStatement())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.NewCount != 2 || plan.DuplicateCount != 0 || plan.UpdatedCount != 0 {
		t.Errorf("counts = new %d duplicate %d updated %d", plan.NewCount, plan.DuplicateCount, plan.UpdatedCount)
	}
	for _, row := range plan.Rows {
		if row.Status != StatusNew {
			t.Errorf("row %s status = %s, want new", row.Transaction.FITID, row.Status)
		}
		if len(row.ID) < 5 || row.ID[:4] == "TRN-" {
			t.Errorf("row id = %q, want an IMP- id", row.ID)
		}
	}
}

func TestPlanRepeatStatementIsDuplicate(t *testing.T) {
	ctx := context.Background()
	db, accts, txns, snaps := newTestStores(t)
	seedIdentity(t, db, accts)
	svc := New(accts, txns, snaps)

	if _, err := svc.Commit(ctx, "ACT-new", sampleStatement()); err != nil {
		t.Fatalf("commit: %v", err)
	}
	plan, err := svc.Plan(ctx, "ACT-new", sampleStatement())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.NewCount != 0 || plan.DuplicateCount != 2 {
		t.Errorf("counts = new %d duplicate %d, want 0/2", plan.NewCount, plan.DuplicateCount)
	}
}

func TestPlanMatchesSimpleFINOverlapAcrossIdentity(t *testing.T) {
	ctx := context.Background()
	db, accts, txns, snaps := newTestStores(t)
	seedIdentity(t, db, accts)
	if _, _, err := txns.UpsertBatch(ctx, []models.DBTransaction{{
		ID: "TRN-legacy", AccountID: "ACT-old", Description: "COFFEE", Amount: -4.50,
		Posted: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC).Unix(),
	}}); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}

	svc := New(accts, txns, snaps)
	plan, err := svc.Plan(ctx, "ACT-new", sampleStatement())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.DuplicateCount != 1 || plan.NewCount != 1 {
		t.Fatalf("counts = new %d duplicate %d, want 1/1", plan.NewCount, plan.DuplicateCount)
	}
	if plan.Rows[0].ExistingID != "TRN-legacy" {
		t.Errorf("the legacy row should be the duplicate, got %q", plan.Rows[0].ExistingID)
	}
}

func TestPlanKeepsRepeatedPurchases(t *testing.T) {
	ctx := context.Background()
	db, accts, txns, snaps := newTestStores(t)
	seedIdentity(t, db, accts)
	posted := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC).Unix()
	if _, _, err := txns.UpsertBatch(ctx, []models.DBTransaction{{
		ID: "TRN-legacy", AccountID: "ACT-old", Description: "COFFEE", Amount: -4.50, Posted: posted,
	}}); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}

	stmt := sampleStatement()
	stmt.Transactions = append(stmt.Transactions, ofx.Transaction{
		FITID: "fit-3", Type: "DEBIT", Name: "COFFEE", Amount: -4.50,
		Posted: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	})

	svc := New(accts, txns, snaps)
	plan, err := svc.Plan(ctx, "ACT-new", stmt)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.DuplicateCount != 1 || plan.NewCount != 2 {
		t.Errorf("counts = new %d duplicate %d, want 2/1", plan.NewCount, plan.DuplicateCount)
	}
}

func TestCommitUpdatesBalanceAndSnapshot(t *testing.T) {
	ctx := context.Background()
	db, accts, txns, snaps := newTestStores(t)
	seedIdentity(t, db, accts)
	svc := New(accts, txns, snaps)

	result, err := svc.Commit(ctx, "ACT-new", sampleStatement())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if result.Added != 2 {
		t.Errorf("added = %d, want 2", result.Added)
	}

	acct, err := accts.GetByID(ctx, "ACT-new")
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if acct.Balance != -3187.08 {
		t.Errorf("balance = %v, want -3187.08", acct.Balance)
	}
	if acct.BalanceDate != time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC).Unix() {
		t.Errorf("balance date = %d", acct.BalanceDate)
	}

	var balance float64
	var source string
	if err := db.Read.QueryRowContext(ctx, `
		SELECT balance, source FROM balance_snapshots
		WHERE account_id = 'ACT-new' ORDER BY balance_date DESC LIMIT 1`).Scan(&balance, &source); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if balance != -3187.08 || source != store.SnapshotSourceImport {
		t.Errorf("snapshot = %v/%q, want -3187.08/%q", balance, source, store.SnapshotSourceImport)
	}
}

func TestCandidatesPreferMatchingDigits(t *testing.T) {
	db, accts, _, _ := newTestStores(t)
	seedIdentity(t, db, accts)
	if err := accts.Upsert(context.Background(), models.DBAccount{
		ID: "ACT-tangerine", Name: "Money-Back World Mastercard®* (4268)", OrgName: "Tangerine Bank (CA)", IsIncluded: true,
	}); err != nil {
		t.Fatalf("seed tangerine: %v", err)
	}
	list, err := accts.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	candidates := Candidates(list, "4520883071874533")
	if len(candidates) == 0 {
		t.Fatal("no candidates")
	}
	if candidates[0].Account.ID != "ACT-new" {
		t.Errorf("best candidate = %q, want ACT-new", candidates[0].Account.ID)
	}
	if candidates[0].Score <= 0 {
		t.Errorf("score = %d, want positive", candidates[0].Score)
	}
}
