package store

import (
	"context"
	"database/sql"

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
		SELECT merchant_description, category, source, updated_at
		FROM categories ORDER BY category, merchant_description`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.CategoryEntry
	for rows.Next() {
		var e models.CategoryEntry
		if err := rows.Scan(&e.MerchantDescription, &e.Category, &e.Source, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
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
			updated_at = datetime('now')`)
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

// SetOverride sets a per-transaction category override.
func (s *CategoryStore) SetOverride(ctx context.Context, txnID, category string) error {
	_, err := s.write.ExecContext(ctx, `
		INSERT INTO category_overrides (transaction_id, category, created_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(transaction_id) DO UPDATE SET
			category = excluded.category`,
		txnID, category)
	return err
}

// SetOverrideAndMerchant sets both a per-transaction override and updates the merchant-level category.
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
