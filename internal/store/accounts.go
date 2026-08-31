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
		INSERT INTO accounts (id, name, balance, balance_date, currency, org_name, org_domain, is_included, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			balance = excluded.balance,
			balance_date = excluded.balance_date,
			currency = excluded.currency,
			org_name = excluded.org_name,
			org_domain = excluded.org_domain,
			updated_at = datetime('now')`,
		acct.ID, acct.Name, acct.Balance, acct.BalanceDate, acct.Currency, acct.OrgName, acct.OrgDomain, acct.IsIncluded,
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
