package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"finance_tracker/internal/config"
	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

// With no included card the report would be all zeros, and a $0 month reads
// as great news. Failing lets runAnalysis broadcast an error instead.
func TestBuildPromptFailsWithoutAnalyzedCards(t *testing.T) {
	db := newTestDB(t)
	h := NewAnalysisRunHandler(&config.Config{BillingDay: 15},
		store.NewTransactionStore(db.Read, db.Write),
		store.NewAccountStore(db.Read, db.Write),
		store.NewCategoryStore(db.Read, db.Write),
		store.NewSnapshotStore(db.Read, db.Write),
		store.NewSettingsStore(db.Read, db.Write),
		store.NewAnalysisStore(db.Read, db.Write),
		store.NewBudgetStore(db.Read, db.Write),
		nil, nil)

	_, err := h.BuildPrompt(context.Background(), time.Now().UTC())

	if err == nil || !strings.Contains(err.Error(), "no included credit card accounts to analyze") {
		t.Errorf("expected the no-cards error, got %v", err)
	}
}

// Galactica on 2026-10-29: TD's SimpleFin balance froze on Sep 25, but an
// import brought its transactions to Oct 27 and the table measured October
// from them. A warning dated Sep 25 contradicted the table.
func TestStaleByNewestDataDropsCardsWithRecentTransactions(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	td := models.StaleConnection{ID: "td",
		BalanceDate: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC).Unix(), LastTransaction: time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC).Unix()}
	mc := models.StaleConnection{ID: "mc",
		BalanceDate: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC).Unix(), LastTransaction: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Unix()}

	got := staleByNewestData([]models.StaleConnection{td, mc}, now)

	if len(got) != 1 || got[0].ID != "mc" {
		t.Errorf("expected only the card with no recent data, got %+v", got)
	}
}
