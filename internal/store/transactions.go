package store

import (
	"context"
	"database/sql"
	"errors"
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

	// SQLite reports one row affected for an upsert whether it inserted or
	// updated, so the row has to be looked up before it is written to tell the
	// two apart.
	exists, err := tx.PrepareContext(ctx, `SELECT 1 FROM transactions WHERE id = ?`)
	if err != nil {
		return 0, 0, err
	}
	defer exists.Close()

	// The DO UPDATE is guarded so re-fetching an unchanged row reports zero rows
	// affected. Without the guard every sync rewrites its whole window and
	// reports the window size as activity, which hides a feed that has stopped
	// returning anything new.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO transactions (id, account_id, description, amount, posted, transacted_at, pending, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			description = excluded.description,
			amount = excluded.amount,
			posted = excluded.posted,
			transacted_at = excluded.transacted_at,
			pending = excluded.pending,
			updated_at = datetime('now')
		WHERE transactions.description IS NOT excluded.description
			OR transactions.amount IS NOT excluded.amount
			OR transactions.posted IS NOT excluded.posted
			OR transactions.transacted_at IS NOT excluded.transacted_at
			OR transactions.pending IS NOT excluded.pending`)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	added, updated := 0, 0
	seen := make(map[string]bool, len(txns))
	for _, t := range txns {
		isNew := false
		if !seen[t.ID] {
			var one int
			err := exists.QueryRowContext(ctx, t.ID).Scan(&one)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				isNew = true
			case err != nil:
				return added, updated, err
			}
			seen[t.ID] = true
		}

		res, err := stmt.ExecContext(ctx, t.ID, t.AccountID, t.Description, t.Amount, t.Posted, t.TransactedAt, t.Pending)
		if err != nil {
			return added, updated, err
		}
		rows, _ := res.RowsAffected()

		switch {
		case isNew:
			added++
		case rows > 0:
			updated++
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

// CountByCategoryGrouped returns per-category totals grouped by month for a date range.
// Returns map[category]map[month]total.
func (s *TransactionStore) CountByCategoryGrouped(ctx context.Context, start, end int64) (map[string]map[string]float64, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT strftime('%Y-%m', t.posted, 'unixepoch') as month,
			COALESCE(co.category, c.category, 'Uncategorized') as cat,
			SUM(ABS(t.amount)) as total
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		WHERE t.posted >= ? AND t.posted <= ? AND a.is_included = 1 AND t.amount < 0
		GROUP BY month, cat
		ORDER BY total DESC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]float64)
	for rows.Next() {
		var month, cat string
		var total float64
		if err := rows.Scan(&month, &cat, &total); err != nil {
			return nil, err
		}
		if result[cat] == nil {
			result[cat] = make(map[string]float64)
		}
		result[cat][month] = total
	}
	return result, rows.Err()
}

// DailyTotal represents spending for a single day.
type DailyTotal struct {
	Date  string  `json:"date"` // YYYY-MM-DD
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

// CategoryTotal represents spending for a single category.
type CategoryTotal struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Total float64 `json:"total"`
}

// SummaryResult holds aggregate stats and per-category breakdown.
type SummaryResult struct {
	TotalSpending    float64         `json:"total_spending"`
	TransactionCount int             `json:"transaction_count"`
	DaysInPeriod     int             `json:"days_in_period"`
	DailyAverage     float64         `json:"daily_average"`
	Categories       []CategoryTotal `json:"categories"`
}

// Summary returns aggregate spending stats and per-category breakdown.
// The aggregate stats respect all filters including category.
// The category breakdown ignores the category filter to preserve comparative context.
func (s *TransactionStore) Summary(ctx context.Context, f TransactionFilter) (*SummaryResult, error) {
	// --- Build WHERE clause for aggregates (respects all filters including category) ---
	var aggWhere []string
	var aggArgs []interface{}

	aggWhere = append(aggWhere, "t.amount < 0") // expenses only

	if f.AccountID != "" {
		aggWhere = append(aggWhere, "t.account_id = ?")
		aggArgs = append(aggArgs, f.AccountID)
	}
	if f.StartDate > 0 {
		aggWhere = append(aggWhere, "t.posted >= ?")
		aggArgs = append(aggArgs, f.StartDate)
	}
	if f.EndDate > 0 {
		aggWhere = append(aggWhere, "t.posted <= ?")
		aggArgs = append(aggArgs, f.EndDate)
	}
	if f.Search != "" {
		aggWhere = append(aggWhere, "t.description LIKE ?")
		aggArgs = append(aggArgs, "%"+f.Search+"%")
	}
	if f.IncludedOnly {
		aggWhere = append(aggWhere, "a.is_included = 1")
	}
	if f.Category != "" {
		if strings.EqualFold(f.Category, "Uncategorized") {
			aggWhere = append(aggWhere, "COALESCE(co.category, c.category, '') = ''")
		} else {
			aggWhere = append(aggWhere, "(COALESCE(co.category, c.category, '') = ?)")
			aggArgs = append(aggArgs, f.Category)
		}
	}

	accountsJoin := ""
	if f.IncludedOnly {
		accountsJoin = "JOIN accounts a ON t.account_id = a.id"
	}

	aggWhereClause := "WHERE " + strings.Join(aggWhere, " AND ")

	// Aggregate query: total spending and count
	aggQuery := fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(SUM(ABS(t.amount)), 0)
		FROM transactions t
		%s
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		%s`, accountsJoin, aggWhereClause)

	var totalCount int
	var totalSpending float64
	if err := s.read.QueryRowContext(ctx, aggQuery, aggArgs...).Scan(&totalCount, &totalSpending); err != nil {
		return nil, err
	}

	// --- Build WHERE clause for category breakdown (ignores category filter) ---
	var catWhere []string
	var catArgs []interface{}

	catWhere = append(catWhere, "t.amount < 0")

	if f.AccountID != "" {
		catWhere = append(catWhere, "t.account_id = ?")
		catArgs = append(catArgs, f.AccountID)
	}
	if f.StartDate > 0 {
		catWhere = append(catWhere, "t.posted >= ?")
		catArgs = append(catArgs, f.StartDate)
	}
	if f.EndDate > 0 {
		catWhere = append(catWhere, "t.posted <= ?")
		catArgs = append(catArgs, f.EndDate)
	}
	if f.Search != "" {
		catWhere = append(catWhere, "t.description LIKE ?")
		catArgs = append(catArgs, "%"+f.Search+"%")
	}
	if f.IncludedOnly {
		catWhere = append(catWhere, "a.is_included = 1")
	}
	// NOTE: Category filter intentionally omitted for breakdown

	catWhereClause := "WHERE " + strings.Join(catWhere, " AND ")

	catQuery := fmt.Sprintf(`
		SELECT COALESCE(co.category, c.category, 'Uncategorized') as cat,
			COUNT(*) as cnt,
			SUM(ABS(t.amount)) as total
		FROM transactions t
		%s
		LEFT JOIN categories c ON t.description = c.merchant_description
		LEFT JOIN category_overrides co ON t.id = co.transaction_id
		%s
		GROUP BY cat
		ORDER BY total DESC`, accountsJoin, catWhereClause)

	rows, err := s.read.QueryContext(ctx, catQuery, catArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []CategoryTotal
	for rows.Next() {
		var ct CategoryTotal
		if err := rows.Scan(&ct.Name, &ct.Count, &ct.Total); err != nil {
			return nil, err
		}
		categories = append(categories, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Calculate days in period
	daysInPeriod := 0
	if f.StartDate > 0 && f.EndDate > 0 {
		daysInPeriod = int((f.EndDate - f.StartDate) / 86400)
		if daysInPeriod < 1 {
			daysInPeriod = 1
		}
	}

	dailyAverage := 0.0
	if daysInPeriod > 0 {
		dailyAverage = totalSpending / float64(daysInPeriod)
	}

	return &SummaryResult{
		TotalSpending:    totalSpending,
		TransactionCount: totalCount,
		DaysInPeriod:     daysInPeriod,
		DailyAverage:     dailyAverage,
		Categories:       categories,
	}, nil
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
