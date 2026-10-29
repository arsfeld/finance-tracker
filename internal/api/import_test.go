package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"finance_tracker/internal/config"
	"finance_tracker/internal/models"
	"finance_tracker/internal/scheduler"
	"finance_tracker/internal/store"
)

func newImportTestHandler(t *testing.T) (*ImportHandler, *store.AccountStore, *store.TransactionStore, *store.SyncLogStore) {
	t.Helper()
	db := newTestDB(t)
	accts := store.NewAccountStore(db.Read, db.Write)
	txns := store.NewTransactionStore(db.Read, db.Write)
	cats := store.NewCategoryStore(db.Read, db.Write)
	snaps := store.NewSnapshotStore(db.Read, db.Write)
	logs := store.NewSyncLogStore(db.Read, db.Write)

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

	h := NewImportHandler(&config.Config{}, accts, txns, cats, snaps, logs, scheduler.New(), NewEventHub())
	return h, accts, txns, logs
}

func importFormRequest(t *testing.T, target, accountID, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "accountactivity.qfx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if accountID != "" {
		if err := mw.WriteField("account_id", accountID); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func fixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/accountactivity.qfx")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(data)
}

func decodeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var wrapper struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return wrapper.Data
}

func TestImportPreviewMultipart(t *testing.T) {
	h, _, _, _ := newImportTestHandler(t)
	req := importFormRequest(t, "/api/import/preview", "", fixture(t))
	rec := httptest.NewRecorder()

	h.Preview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	got := decodeData[importPreviewResponse](t, rec)
	if got.NewCount != 50 {
		t.Errorf("new = %d, want 50", got.NewCount)
	}
	if got.Account == nil || got.Account.ID != "ACT-new" {
		t.Errorf("suggested account = %+v, want ACT-new", got.Account)
	}
	if got.Statement.TransactionCount != 50 {
		t.Errorf("statement count = %d, want 50", got.Statement.TransactionCount)
	}
}

func TestImportCommitMultipartIsIdempotent(t *testing.T) {
	h, accts, _, logs := newImportTestHandler(t)

	rec := httptest.NewRecorder()
	h.Commit(rec, importFormRequest(t, "/api/import", "ACT-new", fixture(t)))
	if rec.Code != http.StatusOK {
		t.Fatalf("first status = %d, body %s", rec.Code, rec.Body.String())
	}
	first := decodeData[importResultResponse](t, rec)
	if first.Added != 50 {
		t.Errorf("first added = %d, want 50", first.Added)
	}

	rec = httptest.NewRecorder()
	h.Commit(rec, importFormRequest(t, "/api/import", "ACT-new", fixture(t)))
	if rec.Code != http.StatusOK {
		t.Fatalf("second status = %d, body %s", rec.Code, rec.Body.String())
	}
	second := decodeData[importResultResponse](t, rec)
	if second.Added != 0 || second.Duplicates != 50 {
		t.Errorf("second result = added %d duplicates %d, want 0/50", second.Added, second.Duplicates)
	}

	acct, err := accts.GetByID(context.Background(), "ACT-new")
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if acct.Balance != -3187.08 {
		t.Errorf("balance = %v, want -3187.08", acct.Balance)
	}

	entries, err := logs.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("log entries = %d, want 2", len(entries))
	}
	if entries[0].Source != "import" || entries[0].Filename != "accountactivity.qfx" || entries[0].AccountID != "ACT-new" {
		t.Errorf("latest log = %+v", entries[0])
	}
}

func TestImportRejectsBadFile(t *testing.T) {
	h, _, _, _ := newImportTestHandler(t)
	rec := httptest.NewRecorder()

	h.Preview(rec, importFormRequest(t, "/api/import/preview", "", "not an ofx file"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %s", rec.Code, rec.Body.String())
	}
	var errBody ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errBody.Error.Code != "BAD_FILE" {
		t.Errorf("code = %q, want BAD_FILE", errBody.Error.Code)
	}
}

func TestImportCommitRequiresAccount(t *testing.T) {
	h, _, _, _ := newImportTestHandler(t)
	rec := httptest.NewRecorder()

	h.Commit(rec, importFormRequest(t, "/api/import", "", fixture(t)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %s", rec.Code, rec.Body.String())
	}
	var errBody ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errBody.Error.Code != "ACCOUNT_REQUIRED" {
		t.Errorf("code = %q, want ACCOUNT_REQUIRED", errBody.Error.Code)
	}
}
