package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/ledger"
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
	// Card fields are set on insert only: they are a guess from the name, and a
	// hand correction must survive every later sync. Inclusion is likewise set
	// on insert only, and a new row takes it from its identity so a re-auth's
	// fresh IDs keep the user's choice.
	isCard := ledger.IsCreditCardName(acct.Name)
	cardKey, ok := ledger.CardKey(acct.OrgName, acct.Name, acct.ID)
	if isCard && !ok {
		log.Warn().Str("account", acct.Name).Str("id", acct.ID).
			Msg("Card name has no last-4; its balance history will not follow it across re-authorizations")
	}

	_, err := s.write.ExecContext(ctx, `
		INSERT INTO accounts (id, name, balance, balance_date, currency, org_name, org_domain, is_included,
			anchor_balance, anchor_balance_date, is_credit_card, card_key, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?,
			COALESCE((SELECT is_included FROM accounts WHERE card_key = ?
				ORDER BY first_seen_at DESC, id DESC LIMIT 1), ?),
			?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			balance = excluded.balance,
			balance_date = excluded.balance_date,
			currency = excluded.currency,
			org_name = excluded.org_name,
			org_domain = excluded.org_domain,
			updated_at = datetime('now')`,
		acct.ID, acct.Name, acct.Balance, acct.BalanceDate, acct.Currency, acct.OrgName, acct.OrgDomain,
		cardKey, acct.IsIncluded,
		acct.Balance, acct.BalanceDate, isCard, cardKey,
	)
	return err
}

func (s *AccountStore) List(ctx context.Context) ([]models.DBAccount, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, name, balance, balance_date, currency, org_name, org_domain, is_included,
			is_credit_card, COALESCE(card_key, ''), first_seen_at, updated_at
		FROM accounts ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.DBAccount
	for rows.Next() {
		var a models.DBAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.Balance, &a.BalanceDate, &a.Currency, &a.OrgName, &a.OrgDomain, &a.IsIncluded,
			&a.IsCreditCard, &a.CardKey, &a.FirstSeenAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	current := ledger.CurrentByKey(accounts)
	for i := range accounts {
		accounts[i].IsCurrent = current[ledger.IdentityKey(accounts[i])].ID == accounts[i].ID
	}
	return accounts, nil
}

func (s *AccountStore) GetByID(ctx context.Context, id string) (*models.DBAccount, error) {
	var a models.DBAccount
	err := s.read.QueryRowContext(ctx, `
		SELECT id, name, balance, balance_date, currency, org_name, org_domain, is_included,
			is_credit_card, COALESCE(card_key, ''), first_seen_at, updated_at
		FROM accounts WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.Balance, &a.BalanceDate, &a.Currency, &a.OrgName, &a.OrgDomain, &a.IsIncluded,
			&a.IsCreditCard, &a.CardKey, &a.FirstSeenAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// execer is the subset of *sql.DB and *sql.Tx that setIncluded needs, so it
// can run against the pool directly or inside another call's transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// setIncluded is SetIncluded's statement, taking an executor so Update can run
// it inside the same transaction as the rest of its patch instead of against
// the pool directly.
func setIncluded(ctx context.Context, db execer, id string, included bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE accounts SET is_included = ?, updated_at = datetime('now')
		WHERE id = ? OR card_key = (SELECT card_key FROM accounts WHERE id = ? AND card_key != '')`,
		included, id, id)
	return err
}

// SetIncluded includes or excludes an account together with every other row of
// its identity, so the choice covers the duplicates a re-auth leaves behind.
func (s *AccountStore) SetIncluded(ctx context.Context, id string, included bool) error {
	return setIncluded(ctx, s.write, id, included)
}

// AccountPatch changes the hand-editable fields of an account. Nil fields are
// left as they are.
type AccountPatch struct {
	IsIncluded   *bool   `json:"is_included"`
	IsCreditCard *bool   `json:"is_credit_card"`
	CardKey      *string `json:"card_key"`
}

// Update applies every set field of p in one transaction, so a patch touching
// both is_included (identity-wide) and a per-row field like card_key either
// takes fully or not at all. The write pool holds a single connection, so
// setIncluded is run against the transaction here rather than through
// SetIncluded, which would try to borrow that same connection and deadlock.
func (s *AccountStore) Update(ctx context.Context, id string, p AccountPatch) error {
	tx, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if p.IsIncluded != nil {
		if err := setIncluded(ctx, tx, id, *p.IsIncluded); err != nil {
			return err
		}
	}

	var sets []string
	var args []any
	if p.IsCreditCard != nil {
		sets, args = append(sets, "is_credit_card = ?"), append(args, *p.IsCreditCard)
	}
	if p.CardKey != nil {
		sets, args = append(sets, "card_key = ?"), append(args, *p.CardKey)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at = datetime('now')")
		args = append(args, id)
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// BackfillCardIdentity classifies accounts stored before cards were tracked.
// Rows that already have a card key are left alone, so hand edits survive
// restarts.
func (s *AccountStore) BackfillCardIdentity(ctx context.Context) (int, error) {
	type row struct{ id, name, org string }
	rows, err := s.write.QueryContext(ctx, `SELECT id, name, org_name FROM accounts WHERE card_key IS NULL`)
	if err != nil {
		return 0, err
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.org); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, r)
	}
	// The write pool has a single connection, so the cursor must be closed
	// before the updates can run.
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, r := range pending {
		key, _ := ledger.CardKey(r.org, r.name, r.id)
		if _, err := s.write.ExecContext(ctx,
			`UPDATE accounts SET is_credit_card = ?, card_key = ? WHERE id = ? AND card_key IS NULL`,
			ledger.IsCreditCardName(r.name), key, r.id); err != nil {
			return 0, err
		}
	}
	return len(pending), nil
}

// NormalizeInclusion makes every row of an identity agree with its newest row,
// and returns how many rows it changed. Rows written before inclusion followed
// the identity can disagree: a re-auth inserted its new IDs included whatever
// the old ones said.
func (s *AccountStore) NormalizeInclusion(ctx context.Context) (int64, error) {
	const newest = `(SELECT b.is_included FROM accounts b WHERE b.card_key = accounts.card_key
		ORDER BY b.first_seen_at DESC, b.id DESC LIMIT 1)`
	res, err := s.write.ExecContext(ctx, `
		UPDATE accounts SET is_included = `+newest+`
		WHERE card_key IS NOT NULL AND card_key != '' AND is_included != `+newest)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
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
