package store

import (
	"context"
	"database/sql"

	"finance_tracker/internal/models"
)

type BudgetStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewBudgetStore(read, write *sql.DB) *BudgetStore {
	return &BudgetStore{read: read, write: write}
}

func (s *BudgetStore) GetAll(ctx context.Context) ([]models.Budget, error) {
	rows, err := s.read.QueryContext(ctx, `SELECT category, amount FROM budgets ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var b models.Budget
		if err := rows.Scan(&b.Category, &b.Amount); err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, rows.Err()
}

func (s *BudgetStore) Upsert(ctx context.Context, category string, amount float64) error {
	_, err := s.write.ExecContext(ctx, `
		INSERT INTO budgets (category, amount)
		VALUES (?, ?)
		ON CONFLICT(category) DO UPDATE SET amount = excluded.amount`,
		category, amount)
	return err
}

func (s *BudgetStore) Delete(ctx context.Context, category string) error {
	_, err := s.write.ExecContext(ctx, `DELETE FROM budgets WHERE category = ?`, category)
	return err
}
