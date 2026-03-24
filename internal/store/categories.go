package store

import (
	"context"
	"database/sql"
	"strings"

	"finance_tracker/internal/models"
)

type CategoryStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewCategoryStore(read, write *sql.DB) *CategoryStore {
	return &CategoryStore{read: read, write: write}
}

func (s *CategoryStore) Get(ctx context.Context, merchantDesc string) (string, error) {
	var cat string
	err := s.read.QueryRowContext(ctx, `SELECT category FROM categories WHERE merchant_description = ?`, merchantDesc).Scan(&cat)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return cat, err
}

func (s *CategoryStore) Set(ctx context.Context, merchantDesc, category, source string) error {
	_, err := s.write.ExecContext(ctx, `
		INSERT INTO categories (merchant_description, category, source, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(merchant_description) DO UPDATE SET
			category = excluded.category,
			source = excluded.source,
			updated_at = datetime('now')`,
		merchantDesc, category, source)
	return err
}

func (s *CategoryStore) ListAll(ctx context.Context) ([]models.CategoryEntry, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT merchant_description, category, source, excluded, updated_at
		FROM categories ORDER BY category, merchant_description`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.CategoryEntry
	for rows.Next() {
		var e models.CategoryEntry
		if err := rows.Scan(&e.MerchantDescription, &e.Category, &e.Source, &e.Excluded, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListUniqueCategories returns distinct category names with their exclusion status.
type CategoryInfo struct {
	Name     string `json:"name"`
	Excluded bool   `json:"excluded"`
	Count    int    `json:"count"`
}

func (s *CategoryStore) ListUniqueCategories(ctx context.Context) ([]CategoryInfo, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT category, MAX(excluded) as excluded, COUNT(*) as count
		FROM categories
		GROUP BY category
		ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []CategoryInfo
	for rows.Next() {
		var c CategoryInfo
		if err := rows.Scan(&c.Name, &c.Excluded, &c.Count); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// SetCategoryExcluded sets the excluded flag for all merchants in a category.
func (s *CategoryStore) SetCategoryExcluded(ctx context.Context, category string, excluded bool) error {
	_, err := s.write.ExecContext(ctx, `
		UPDATE categories SET excluded = ?, updated_at = datetime('now')
		WHERE category = ?`, excluded, category)
	return err
}

// ExcludedCategoryNames returns the list of excluded category names.
func (s *CategoryStore) ExcludedCategoryNames(ctx context.Context) ([]string, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT DISTINCT category FROM categories WHERE excluded = 1 ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

func (s *CategoryStore) BulkUpsert(ctx context.Context, entries []models.CategoryEntry) error {
	tx, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO categories (merchant_description, category, source, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(merchant_description) DO UPDATE SET
			category = excluded.category,
			source = excluded.source,
			updated_at = datetime('now')
		WHERE categories.source != 'user'`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.ExecContext(ctx, e.MerchantDescription, e.Category, e.Source); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SimilarMerchantInfo holds a candidate merchant with its current category and transaction count.
type SimilarMerchantInfo struct {
	MerchantDescription string `json:"merchant_description"`
	CurrentCategory     string `json:"current_category"`
	TransactionCount    int    `json:"transaction_count"`
}

// ListCandidatesForSimilarity returns merchants that are candidates for similarity matching:
// source='llm' and category differs from the target category.
func (s *CategoryStore) ListCandidatesForSimilarity(ctx context.Context, excludeCategory string) ([]models.CategoryEntry, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT merchant_description, category, source, excluded, updated_at
		FROM categories
		WHERE source = 'llm' AND category != ?
		ORDER BY merchant_description`, excludeCategory)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.CategoryEntry
	for rows.Next() {
		var e models.CategoryEntry
		if err := rows.Scan(&e.MerchantDescription, &e.Category, &e.Source, &e.Excluded, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetMerchantTransactionCounts returns the number of transactions per merchant description.
func (s *CategoryStore) GetMerchantTransactionCounts(ctx context.Context, merchants []string) (map[string]int, error) {
	if len(merchants) == 0 {
		return map[string]int{}, nil
	}

	// Build placeholder query.
	placeholders := make([]string, len(merchants))
	args := make([]interface{}, len(merchants))
	for i, m := range merchants {
		placeholders[i] = "?"
		args[i] = m
	}

	query := `SELECT description, COUNT(*) FROM transactions WHERE description IN (` +
		strings.Join(placeholders, ",") + `) GROUP BY description`

	rows, err := s.read.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var desc string
		var count int
		if err := rows.Scan(&desc, &count); err != nil {
			return nil, err
		}
		counts[desc] = count
	}
	return counts, rows.Err()
}

// BulkSetCategory sets the category for multiple merchants with source='user'.
func (s *CategoryStore) BulkSetCategory(ctx context.Context, merchants []string, category string) error {
	if len(merchants) == 0 {
		return nil
	}

	tx, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO categories (merchant_description, category, source, updated_at)
		VALUES (?, ?, 'user', datetime('now'))
		ON CONFLICT(merchant_description) DO UPDATE SET
			category = excluded.category,
			source = 'user',
			updated_at = datetime('now')`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range merchants {
		if _, err := stmt.ExecContext(ctx, m, category); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *CategoryStore) SetOverride(ctx context.Context, txnID, category string) error {
	_, err := s.write.ExecContext(ctx, `
		INSERT INTO category_overrides (transaction_id, category, created_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(transaction_id) DO UPDATE SET
			category = excluded.category`,
		txnID, category)
	return err
}

func (s *CategoryStore) SetOverrideAndMerchant(ctx context.Context, txnID, merchantDesc, category string) error {
	tx, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO category_overrides (transaction_id, category, created_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(transaction_id) DO UPDATE SET category = excluded.category`,
		txnID, category); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO categories (merchant_description, category, source, updated_at)
		VALUES (?, ?, 'user', datetime('now'))
		ON CONFLICT(merchant_description) DO UPDATE SET
			category = excluded.category,
			source = 'user',
			updated_at = datetime('now')`,
		merchantDesc, category); err != nil {
		return err
	}

	return tx.Commit()
}
