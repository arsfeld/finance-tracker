package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/ledger"
)

// InitCardLedger prepares card tracking at startup. It classifies accounts
// stored before cards were tracked, reconciles inclusion across each account's
// identity, seeds the balance history from what the accounts table already
// knows, and writes the default payment patterns. Every step is idempotent, so
// it runs on every start.
func InitCardLedger(ctx context.Context, accts *AccountStore, snaps *SnapshotStore, settings *SettingsStore) error {
	classified, err := accts.BackfillCardIdentity(ctx)
	if err != nil {
		return fmt.Errorf("classify accounts: %w", err)
	}
	// After classification, so every row has the key the pass groups by.
	normalized, err := accts.NormalizeInclusion(ctx)
	if err != nil {
		return fmt.Errorf("normalize account inclusion: %w", err)
	}
	seeded, err := snaps.Seed(ctx)
	if err != nil {
		return fmt.Errorf("seed balance history: %w", err)
	}

	raw, err := settings.Get(ctx, ledger.PaymentPatternsSettingKey)
	if err != nil {
		return fmt.Errorf("read payment patterns: %w", err)
	}
	if raw == "" {
		defaults, _ := json.Marshal(ledger.DefaultPaymentPatterns)
		if err := settings.Set(ctx, ledger.PaymentPatternsSettingKey, string(defaults)); err != nil {
			return fmt.Errorf("write payment patterns: %w", err)
		}
	}

	log.Info().Int("accounts_classified", classified).Int64("inclusion_normalized", normalized).
		Int("snapshots_seeded", seeded).Msg("Card ledger ready")
	return nil
}
