package store

import (
	"context"
	"database/sql"
	"time"

	"finance_tracker/internal/models"
)

type AccountStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewAccountStore(read, write *sql.DB) *AccountStore {
	return &AccountStore{read: read, write: write}
}

func (s *AccountStore) Upsert(ctx context.Context, acct models.DBAccount) error {
	_, err := s.write.ExecContext(ctx, `
		INSERT INTO accounts (id, name, balance, balance_date, currency, org_name, org_domain, is_included,
			anchor_balance, anchor_balance_date, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			balance = excluded.balance,
			balance_date = excluded.balance_date,
			currency = excluded.currency,
			org_name = excluded.org_name,
			org_domain = excluded.org_domain,
			updated_at = datetime('now')`,
		acct.ID, acct.Name, acct.Balance, acct.BalanceDate, acct.Currency, acct.OrgName, acct.OrgDomain, acct.IsIncluded,
		acct.Balance, acct.BalanceDate,
	)
	return err
}

func (s *AccountStore) List(ctx context.Context) ([]models.DBAccount, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, name, balance, balance_date, currency, org_name, org_domain, is_included, first_seen_at, updated_at
		FROM accounts ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.DBAccount
	for rows.Next() {
		var a models.DBAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.Balance, &a.BalanceDate, &a.Currency, &a.OrgName, &a.OrgDomain, &a.IsIncluded, &a.FirstSeenAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (s *AccountStore) GetByID(ctx context.Context, id string) (*models.DBAccount, error) {
	var a models.DBAccount
	err := s.read.QueryRowContext(ctx, `
		SELECT id, name, balance, balance_date, currency, org_name, org_domain, is_included, first_seen_at, updated_at
		FROM accounts WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.Balance, &a.BalanceDate, &a.Currency, &a.OrgName, &a.OrgDomain, &a.IsIncluded, &a.FirstSeenAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (s *AccountStore) UpdateInclusion(ctx context.Context, id string, included bool) error {
	_, err := s.write.ExecContext(ctx, `UPDATE accounts SET is_included = ?, updated_at = datetime('now') WHERE id = ?`, included, id)
	return err
}

// StaleConnections returns included accounts whose balance has not been
// refreshed within maxAge.
//
// balance_date is the signal rather than the newest transaction: SimpleFin
// advances it on every successful refresh, so it freezes the moment a
// connection needs re-authorization, whereas a card can legitimately go a week
// without a charge. Excluded accounts are skipped — they are not part of any
// analysis, so a dead connection on one is not worth an alert.
func (s *AccountStore) StaleConnections(ctx context.Context, now time.Time, maxAge time.Duration) ([]models.StaleConnection, error) {
	cutoff := now.Add(-maxAge).Unix()

	rows, err := s.read.QueryContext(ctx, `
		SELECT a.id, a.name, a.org_name, a.balance_date, COALESCE(MAX(t.posted), 0)
		FROM accounts a
		LEFT JOIN transactions t ON t.account_id = a.id
		WHERE a.is_included = 1 AND a.balance_date < ?
		GROUP BY a.id
		ORDER BY a.balance_date`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stale []models.StaleConnection
	for rows.Next() {
		var c models.StaleConnection
		if err := rows.Scan(&c.ID, &c.Name, &c.OrgName, &c.BalanceDate, &c.LastTransaction); err != nil {
			return nil, err
		}
		stale = append(stale, c)
	}
	return stale, rows.Err()
}

// UnreconciledAccounts returns included accounts whose balance has drifted from
// what their transactions account for by more than minUnexplained.
//
// Each account is anchored to the first balance ever recorded for it, so the
// check is expected = anchor + everything posted since. A connection that
// refreshes balances while delivering no transactions drifts further every day,
// which is the one failure the staleness check cannot see: balance_date stays
// current the whole time.
//
// Only transactions posted strictly after the anchor count. A late delivery
// carries an older posted date and was already priced into the anchor balance,
// so a backfill closes the gap instead of inventing a new one.
func (s *AccountStore) UnreconciledAccounts(ctx context.Context, minUnexplained float64) ([]models.UnreconciledAccount, error) {
	rows, err := s.read.QueryContext(ctx, `
		WITH reconciled AS (
			SELECT a.id, a.name, a.org_name, a.balance, a.balance_date,
				a.balance - (a.anchor_balance + COALESCE((
					SELECT SUM(t.amount) FROM transactions t
					WHERE t.account_id = a.id AND t.posted > a.anchor_balance_date
				), 0)) AS unexplained
			FROM accounts a
			WHERE a.is_included = 1 AND a.anchor_balance IS NOT NULL
		)
		SELECT id, name, org_name, balance, balance_date, unexplained
		FROM reconciled
		WHERE ABS(unexplained) > ?
		ORDER BY ABS(unexplained) DESC`, minUnexplained)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drifted []models.UnreconciledAccount
	for rows.Next() {
		var u models.UnreconciledAccount
		if err := rows.Scan(&u.ID, &u.Name, &u.OrgName, &u.Balance, &u.BalanceDate, &u.Unexplained); err != nil {
			return nil, err
		}
		drifted = append(drifted, u)
	}
	return drifted, rows.Err()
}
