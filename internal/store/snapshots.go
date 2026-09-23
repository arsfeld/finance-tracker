package store

import (
	"context"
	"database/sql"
	"math"
	"sort"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/models"
)

const (
	SnapshotSourceSync        = "sync"
	SnapshotSourceSeedAnchor  = "seed_anchor"
	SnapshotSourceSeedCurrent = "seed_current"
)

// SnapshotStore keeps the history of card balances that spending is measured
// from.
type SnapshotStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewSnapshotStore(read, write *sql.DB) *SnapshotStore {
	return &SnapshotStore{read: read, write: write}
}

// Record stores a balance reading unless it adds nothing: a reading no newer
// than the card's latest snapshot, or one whose balance has not moved. TD has
// reported a frozen balance with an advancing date after a payment had posted;
// storing it would book that payment as spending.
func (s *SnapshotStore) Record(ctx context.Context, snap models.BalanceSnapshot) (bool, error) {
	tx, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// Wrap the SELECT and INSERT in a transaction so two concurrent recordings
	// for the same card cannot both read the same latest snapshot.
	var lastBalance float64
	var lastDate int64
	err = tx.QueryRowContext(ctx, `
		SELECT balance, balance_date FROM balance_snapshots
		WHERE card_key = ? ORDER BY balance_date DESC LIMIT 1`, snap.CardKey).Scan(&lastBalance, &lastDate)
	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return false, err
	case snap.BalanceDate <= lastDate, math.Abs(snap.Balance-lastBalance) < 0.005:
		return false, nil
	}

	res, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO balance_snapshots (card_key, account_id, balance, balance_date, source)
		VALUES (?, ?, ?, ?, ?)`,
		snap.CardKey, snap.AccountID, snap.Balance, snap.BalanceDate, snap.Source)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return n > 0, tx.Commit()
}

// RecordCurrent records each card's current balance. It runs after every sync.
func (s *SnapshotStore) RecordCurrent(ctx context.Context) (int, error) {
	return s.recordReadings(ctx, false)
}

// Seed builds the starting history from what the accounts table already holds:
// every card account's anchor (the first balance ever seen for it) and each
// card's current balance. Anything older was never recorded. The card's
// transactions cannot rebuild it, because the feed was already missing payments
// months before it stopped.
func (s *SnapshotStore) Seed(ctx context.Context) (int, error) {
	return s.recordReadings(ctx, true)
}

type cardAccount struct {
	models.DBAccount
	anchorBalance float64
	anchorDate    int64
}

func (s *SnapshotStore) recordReadings(ctx context.Context, withAnchors bool) (int, error) {
	cards, err := s.cardAccounts(ctx)
	if err != nil {
		return 0, err
	}

	plain := make([]models.DBAccount, len(cards))
	for i, c := range cards {
		plain[i] = c.DBAccount
	}
	current := ledger.CurrentAccounts(plain)

	var readings []models.BalanceSnapshot
	if withAnchors {
		for _, c := range cards {
			if c.anchorDate > 0 {
				readings = append(readings, models.BalanceSnapshot{
					CardKey: c.CardKey, AccountID: c.ID,
					Balance: c.anchorBalance, BalanceDate: c.anchorDate, Source: SnapshotSourceSeedAnchor,
				})
			}
		}
	}
	source := SnapshotSourceSync
	if withAnchors {
		source = SnapshotSourceSeedCurrent
	}
	for _, a := range current {
		if a.BalanceDate > 0 {
			readings = append(readings, models.BalanceSnapshot{
				CardKey: a.CardKey, AccountID: a.ID, Balance: a.Balance, BalanceDate: a.BalanceDate, Source: source,
			})
		}
	}

	// Record's rules compare each reading with the latest one stored, so
	// readings have to arrive in date order.
	sort.SliceStable(readings, func(i, j int) bool { return readings[i].BalanceDate < readings[j].BalanceDate })

	recorded := 0
	for _, r := range readings {
		ok, err := s.Record(ctx, r)
		if err != nil {
			return recorded, err
		}
		if ok {
			recorded++
		}
	}
	return recorded, nil
}

func (s *SnapshotStore) cardAccounts(ctx context.Context) ([]cardAccount, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, name, card_key, balance, COALESCE(balance_date, 0),
			COALESCE(anchor_balance, 0), COALESCE(anchor_balance_date, 0), first_seen_at
		FROM accounts
		WHERE is_credit_card = 1 AND card_key IS NOT NULL AND card_key != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []cardAccount
	for rows.Next() {
		var c cardAccount
		if err := rows.Scan(&c.ID, &c.Name, &c.CardKey, &c.Balance, &c.BalanceDate,
			&c.anchorBalance, &c.anchorDate, &c.FirstSeenAt); err != nil {
			return nil, err
		}
		c.IsCreditCard = true
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

// ListByCard returns every snapshot grouped by card key, oldest first.
func (s *SnapshotStore) ListByCard(ctx context.Context) (map[string][]models.BalanceSnapshot, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT card_key, account_id, balance, balance_date, source
		FROM balance_snapshots ORDER BY card_key, balance_date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byCard := make(map[string][]models.BalanceSnapshot)
	for rows.Next() {
		var s models.BalanceSnapshot
		if err := rows.Scan(&s.CardKey, &s.AccountID, &s.Balance, &s.BalanceDate, &s.Source); err != nil {
			return nil, err
		}
		byCard[s.CardKey] = append(byCard[s.CardKey], s)
	}
	return byCard, rows.Err()
}
