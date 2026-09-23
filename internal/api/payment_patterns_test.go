package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/store"
)

// JSON null decodes into a nil map without error. Storing it would wipe every
// card's patterns, and the paying-side payments with them.
func TestPaymentPatternsPutRejectsNull(t *testing.T) {
	db := newTestDB(t)
	settings := store.NewSettingsStore(db.Read, db.Write)
	const stored = `{"TD Canada Trust|4520":["TFR-A C/C"]}`
	if err := settings.Set(context.Background(), ledger.PaymentPatternsSettingKey, stored); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	NewPaymentPatternsHandler(settings).Put(rec, httptest.NewRequest(http.MethodPut, "/", strings.NewReader("null")))

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "BAD_REQUEST") {
		t.Errorf("expected 400 BAD_REQUEST, got %d %s", rec.Code, rec.Body.String())
	}
	if got, _ := settings.Get(context.Background(), ledger.PaymentPatternsSettingKey); got != stored {
		t.Errorf("the stored patterns must survive, got %q", got)
	}
}
