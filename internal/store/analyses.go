package store

import (
	"context"
	"database/sql"

	"finance_tracker/internal/models"
)

type AnalysisStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewAnalysisStore(read, write *sql.DB) *AnalysisStore {
	return &AnalysisStore{read: read, write: write}
}

func (s *AnalysisStore) Create(ctx context.Context, a models.Analysis) (int64, error) {
	res, err := s.write.ExecContext(ctx, `
		INSERT INTO analyses (period_start, period_end, billing_day, date_range_type, response_text, model_used, is_multi_period, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.PeriodStart, a.PeriodEnd, a.BillingDay, a.DateRangeType, a.ResponseText, a.ModelUsed, a.IsMultiPeriod, a.Status)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *AnalysisStore) List(ctx context.Context, limit int) ([]models.Analysis, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, created_at, period_start, period_end, billing_day, date_range_type, response_text, model_used, is_multi_period, status
		FROM analyses ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analyses []models.Analysis
	for rows.Next() {
		var a models.Analysis
		if err := rows.Scan(&a.ID, &a.CreatedAt, &a.PeriodStart, &a.PeriodEnd, &a.BillingDay, &a.DateRangeType, &a.ResponseText, &a.ModelUsed, &a.IsMultiPeriod, &a.Status); err != nil {
			return nil, err
		}
		analyses = append(analyses, a)
	}
	return analyses, rows.Err()
}

func (s *AnalysisStore) GetByID(ctx context.Context, id int64) (*models.Analysis, error) {
	var a models.Analysis
	err := s.read.QueryRowContext(ctx, `
		SELECT id, created_at, period_start, period_end, billing_day, date_range_type, response_text, model_used, is_multi_period, status
		FROM analyses WHERE id = ?`, id).
		Scan(&a.ID, &a.CreatedAt, &a.PeriodStart, &a.PeriodEnd, &a.BillingDay, &a.DateRangeType, &a.ResponseText, &a.ModelUsed, &a.IsMultiPeriod, &a.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (s *AnalysisStore) GetLatest(ctx context.Context) (*models.Analysis, error) {
	var a models.Analysis
	err := s.read.QueryRowContext(ctx, `
		SELECT id, created_at, period_start, period_end, billing_day, date_range_type, response_text, model_used, is_multi_period, status
		FROM analyses WHERE status = 'success' ORDER BY created_at DESC LIMIT 1`).
		Scan(&a.ID, &a.CreatedAt, &a.PeriodStart, &a.PeriodEnd, &a.BillingDay, &a.DateRangeType, &a.ResponseText, &a.ModelUsed, &a.IsMultiPeriod, &a.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}
