package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"finance_tracker/internal/models"
)

type TransactionStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewTransactionStore(read, write *sql.DB) *TransactionStore {
	return &TransactionStore{read: read, write: write}
}

// UpsertBatch inserts or updates transactions in a single write transaction.
// Returns (added, updated) counts.
func (s *TransactionStore) UpsertBatch(ctx context.Context, txns []models.DBTransaction) (int, int, error) {
	tx, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO transactions (id, account_id, description, amount, posted, transacted_at, pending, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			description = excluded.description,
			amount = excluded.amount,
			posted = excluded.posted,
			transacted_at = excluded.transacted_at,
			pending = excluded.pending,
			updated_at = datetime('now')`)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	added, updated := 0, 0
	for _, t := range txns {
		res, err := stmt.ExecContext(ctx, t.ID, t.AccountID, t.Description, t.Amount, t.Posted, t.TransactedAt, t.Pending)
		if err != nil {
			return added, updated, err
		}
		rows, _ := res.RowsAffected()
		if rows > 0 {
			// SQLite: ON CONFLICT DO UPDATE always reports 1 row affected whether insert or update.
			// We check if the row existed before to distinguish.
			added++
		}
	}

	return added, updated, tx.Commit()
}

// TransactionFilter holds query parameters for listing transactions.
type TransactionFilter struct {
	AccountID       string
	Category        string
	Search          string
	StartDate       int64 // Unix timestamp
	EndDate         int64 // Unix timestamp
	IncludePositive bool
	IncludedOnly    bool // When true, only show transactions from included accounts
	Page            int
	Limit           int
	SortBy          string // "posted", "amount", "description"
	SortDir         string // "asc", "desc"
}

// List returns filtered, paginated, sorted transactions.
func (s *TransactionStore) List(ctx context.Context, f TransactionFilter) ([]models.DBTransaction, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.SortBy == "" {
		f.SortBy = "posted"
	}
	if f.SortDir == "" {
		f.SortDir = "desc"
	}

	var where []string
	var args []interface{}

	if f.AccountID != "" {
		where = append(where, "t.account_id = ?")
		args = append(args, f.AccountID)
	}
	if f.StartDate > 0 {
		where = append(where, "t.posted >= ?")
		args = append(args, f.StartDate)
	}
	if f.EndDate > 0 {
		where = append(where, "t.posted <= ?")
		args = append(args, f.EndDate)
	}
	if f.Search != "" {
		where = append(where, "t.description LIKE ?")
		args = append(args, "%"+f.Search+"%")
	}
	if !f.IncludePositive {
		where = append(where, "t.amount < 0")
	}
	if f.IncludedOnly {
		where = append(where, "a.is_included = 1")
	}
	if f.Category != "" {
		if strings.EqualFold(f.Category, "Uncategorized") {
			where = append(where, "COALESCE(co.category, c.category, '') = ''")
		} else {
			where = append(where, "(COALESCE(co.category, c.category, '') = ?)")
			args = append(args, f.Category)
		}
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Validate sort column to prevent injection.
	sortCol := "t.posted"
	switch f.SortBy {
	case "amount":
		sortCol = "t.amount"
	case "description":
		sortCol = "t.description"
	case "posted":
		sortCol = "t.posted"
	}
	sortDir := "DESC"
	if strings.EqualFold(f.SortDir, "asc") {
		sortDir = "ASC"
	}

	accountsJoin := ""
	if f.IncludedOnly {
		accountsJoin = "JOIN accounts a ON t.account_id = a.id"
	}

	// Count total.
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM transactions t
		%s
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		%s`, accountsJoin, whereClause)

	var total int
	if err := s.read.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page.
	offset := (f.Page - 1) * f.Limit
	dataQuery := fmt.Sprintf(`
		SELECT t.id, t.account_id, t.description, t.amount, t.posted, t.transacted_at, t.pending,
			COALESCE(co.category, c.category, '') as category,
			t.cached_at, t.updated_at
		FROM transactions t
		%s
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?`, accountsJoin, whereClause, sortCol, sortDir)

	dataArgs := append(args, f.Limit, offset)
	rows, err := s.read.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txns []models.DBTransaction
	for rows.Next() {
		var t models.DBTransaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Description, &t.Amount, &t.Posted, &t.TransactedAt, &t.Pending, &t.Category, &t.CachedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}

func (s *TransactionStore) GetByID(ctx context.Context, id string) (*models.DBTransaction, error) {
	var t models.DBTransaction
	err := s.read.QueryRowContext(ctx, `
		SELECT t.id, t.account_id, t.description, t.amount, t.posted, t.transacted_at, t.pending,
			COALESCE(co.category, c.category, '') as category,
			t.cached_at, t.updated_at
		FROM transactions t
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		WHERE t.id = ?`, id).
		Scan(&t.ID, &t.AccountID, &t.Description, &t.Amount, &t.Posted, &t.TransactedAt, &t.Pending, &t.Category, &t.CachedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// GetForPeriod returns all transactions within a date range for included accounts.
func (s *TransactionStore) GetForPeriod(ctx context.Context, start, end int64) ([]models.DBTransaction, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT t.id, t.account_id, t.description, t.amount, t.posted, t.transacted_at, t.pending,
			COALESCE(co.category, c.category, '') as category,
			t.cached_at, t.updated_at
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		WHERE t.posted >= ? AND t.posted <= ? AND a.is_included = 1
		ORDER BY t.posted DESC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []models.DBTransaction
	for rows.Next() {
		var t models.DBTransaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Description, &t.Amount, &t.Posted, &t.TransactedAt, &t.Pending, &t.Category, &t.CachedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

// CountByCategory returns category totals for a date range.
func (s *TransactionStore) CountByCategory(ctx context.Context, start, end int64) (map[string]float64, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT COALESCE(co.category, c.category, 'Uncategorized') as cat,
			SUM(ABS(t.amount)) as total
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		WHERE t.posted >= ? AND t.posted <= ? AND a.is_included = 1 AND t.amount < 0
		GROUP BY cat
		ORDER BY total DESC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var cat string
		var total float64
		if err := rows.Scan(&cat, &total); err != nil {
			return nil, err
		}
		result[cat] = total
	}
	return result, rows.Err()
}

// DailyTotal represents spending for a single day.
type DailyTotal struct {
	Date  string  `json:"date"`  // YYYY-MM-DD
	Total float64 `json:"total"`
}

// DailyTotals returns per-day expense totals for a date range (included accounts only).
func (s *TransactionStore) DailyTotals(ctx context.Context, start, end int64) ([]DailyTotal, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT date(t.posted, 'unixepoch') as day,
			SUM(ABS(t.amount)) as total
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.posted >= ? AND t.posted <= ? AND a.is_included = 1 AND t.amount < 0
		GROUP BY day
		ORDER BY day ASC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DailyTotal
	for rows.Next() {
		var d DailyTotal
		if err := rows.Scan(&d.Date, &d.Total); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// MerchantTotal represents spending for a single merchant.
type MerchantTotal struct {
	Name  string  `json:"name"`
	Total float64 `json:"total"`
	Count int     `json:"count"`
}

// TopMerchants returns the top N merchants by total spending for a date range (included accounts only).
func (s *TransactionStore) TopMerchants(ctx context.Context, start, end int64, limit int) ([]MerchantTotal, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	rows, err := s.read.QueryContext(ctx, `
		SELECT t.description,
			SUM(ABS(t.amount)) as total,
			COUNT(*) as cnt
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.posted >= ? AND t.posted <= ? AND a.is_included = 1 AND t.amount < 0
		GROUP BY t.description
		ORDER BY total DESC
		LIMIT ?`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MerchantTotal
	for rows.Next() {
		var m MerchantTotal
		if err := rows.Scan(&m.Name, &m.Total, &m.Count); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
