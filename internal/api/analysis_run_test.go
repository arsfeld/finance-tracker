package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"finance_tracker/internal/config"
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
