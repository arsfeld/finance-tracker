package store

import (
	"context"
	"testing"

	"finance_tracker/internal/ledger"
)

// On the first start after deploy the accounts have no card fields yet. One
// startup has to classify them, seed the history and write the patterns.
func TestInitCardLedgerPreparesLegacyDatabase(t *testing.T) {
	ctx := context.Background()
	db, accts, snaps := seedProductionTD(t)
	if _, err := db.Write.Exec(`UPDATE accounts SET card_key = NULL, is_credit_card = 0`); err != nil {
		t.Fatalf("simulate legacy rows: %v", err)
	}
	settings := NewSettingsStore(db.Read, db.Write)

	if err := InitCardLedger(ctx, accts, snaps, settings); err != nil {
		t.Fatalf("init: %v", err)
	}

	byCard, _ := snaps.ListByCard(ctx)
	raw, _ := settings.Get(ctx, ledger.PaymentPatternsSettingKey)
	if len(byCard[tdKey]) != 3 {
		t.Errorf("expected the 3 production readings, got %+v", byCard[tdKey])
	}
	if raw == "" {
		t.Error("default payment patterns were not written")
	}
}

func TestInitCardLedgerKeepsCustomPatterns(t *testing.T) {
	ctx := context.Background()
	db, accts, snaps := seedProductionTD(t)
	settings := NewSettingsStore(db.Read, db.Write)
	custom := `{"TD Canada Trust|4520":["MY PATTERN"]}`
	settings.Set(ctx, ledger.PaymentPatternsSettingKey, custom)

	if err := InitCardLedger(ctx, accts, snaps, settings); err != nil {
		t.Fatalf("init: %v", err)
	}

	if raw, _ := settings.Get(ctx, ledger.PaymentPatternsSettingKey); raw != custom {
		t.Errorf("custom patterns were overwritten: %s", raw)
	}
}
