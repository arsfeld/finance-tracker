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
