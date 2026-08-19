package store

import (
	"context"
	"testing"

	"finance_tracker/internal/models"
)

func contains(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// A merchant that moves to a different category must not drag the exclusion
// with it. Regression test: "UNITED AIR" was excluded as part of Refundable,
// the LLM later recategorized it, and the whole destination category silently
// disappeared from analysis.
func TestExcludedCategoryNamesUnaffectedByMerchantRecategorization(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	cats := NewCategoryStore(db.Read, db.Write)

	if err := cats.Set(ctx, "UNITED AIR 1623620607033", "Refundable", "llm"); err != nil {
		t.Fatalf("seed merchant: %v", err)
	}
	if err := cats.Set(ctx, "CINEPLEX ODEON", "Entertainment", "llm"); err != nil {
		t.Fatalf("seed merchant: %v", err)
	}
	if err := cats.SetCategoryExcluded(ctx, "Refundable", true); err != nil {
		t.Fatalf("exclude Refundable: %v", err)
	}

	// The LLM re-runs and moves the merchant into Entertainment.
	err := cats.BulkUpsert(ctx, []models.CategoryEntry{
		{MerchantDescription: "UNITED AIR 1623620607033", Category: "Entertainment", Source: "llm"},
	})
	if err != nil {
		t.Fatalf("recategorize merchant: %v", err)
	}

	names, err := cats.ExcludedCategoryNames(ctx)
	if err != nil {
		t.Fatalf("excluded category names: %v", err)
	}

	if contains(names, "Entertainment") {
		t.Errorf("Entertainment became excluded because a merchant moved into it; got %v", names)
	}
	if !contains(names, "Refundable") {
		t.Errorf("Refundable should stay excluded regardless of its merchants; got %v", names)
	}
}

// Excluding a category must cover merchants added to it later, not just the
// ones that happened to exist when the user clicked the toggle.
func TestExcludedCategoryCoversMerchantsAddedLater(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	cats := NewCategoryStore(db.Read, db.Write)

	if err := cats.Set(ctx, "JT231 TFR-TO C/C", "Payment", "llm"); err != nil {
		t.Fatalf("seed merchant: %v", err)
	}
	if err := cats.SetCategoryExcluded(ctx, "Payment", true); err != nil {
		t.Fatalf("exclude Payment: %v", err)
	}

	// A new transfer merchant shows up in a later sync.
	if err := cats.Set(ctx, "E-TRANSFER ***S3B", "Payment", "llm"); err != nil {
		t.Fatalf("add merchant: %v", err)
	}

	excluded, err := cats.IsCategoryExcluded(ctx, "Payment")
	if err != nil {
		t.Fatalf("is category excluded: %v", err)
	}
	if !excluded {
		t.Error("Payment should remain excluded after a new merchant is added to it")
	}
}

func TestSetCategoryExcludedTogglesOff(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	cats := NewCategoryStore(db.Read, db.Write)

	if err := cats.Set(ctx, "TAP Portugal0479508903118", "Travel", "llm"); err != nil {
		t.Fatalf("seed merchant: %v", err)
	}
	if err := cats.SetCategoryExcluded(ctx, "Travel", true); err != nil {
		t.Fatalf("exclude Travel: %v", err)
	}
	if err := cats.SetCategoryExcluded(ctx, "Travel", false); err != nil {
		t.Fatalf("re-include Travel: %v", err)
	}

	names, err := cats.ExcludedCategoryNames(ctx)
	if err != nil {
		t.Fatalf("excluded category names: %v", err)
	}
	if contains(names, "Travel") {
		t.Errorf("Travel should no longer be excluded; got %v", names)
	}
}

// The Settings page reads exclusion state from here, so it must agree with
// what the analysis actually drops.
func TestListUniqueCategoriesReportsExclusionState(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	cats := NewCategoryStore(db.Read, db.Write)

	if err := cats.Set(ctx, "JT231 TFR-TO C/C", "Payment", "llm"); err != nil {
		t.Fatalf("seed merchant: %v", err)
	}
	if err := cats.Set(ctx, "CINEPLEX ODEON", "Entertainment", "llm"); err != nil {
		t.Fatalf("seed merchant: %v", err)
	}
	if err := cats.SetCategoryExcluded(ctx, "Payment", true); err != nil {
		t.Fatalf("exclude Payment: %v", err)
	}

	infos, err := cats.ListUniqueCategories(ctx)
	if err != nil {
		t.Fatalf("list unique categories: %v", err)
	}

	got := make(map[string]bool, len(infos))
	for _, c := range infos {
		got[c.Name] = c.Excluded
	}
	if !got["Payment"] {
		t.Error("Payment should be reported as excluded")
	}
	if got["Entertainment"] {
		t.Error("Entertainment should not be reported as excluded")
	}
}
