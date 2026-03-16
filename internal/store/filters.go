package store

import (
	"context"
	"database/sql"

	"finance_tracker/internal/models"
)

type FilterStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewFilterStore(read, write *sql.DB) *FilterStore {
	return &FilterStore{read: read, write: write}
}

func (s *FilterStore) Create(ctx context.Context, pattern string, matchType models.MatchType) (int64, error) {
	res, err := s.write.ExecContext(ctx, `
		INSERT INTO filter_rules (pattern, match_type) VALUES (?, ?)`,
		pattern, matchType)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *FilterStore) Update(ctx context.Context, id int64, pattern string, matchType models.MatchType, isActive bool) error {
	_, err := s.write.ExecContext(ctx, `
		UPDATE filter_rules SET pattern = ?, match_type = ?, is_active = ? WHERE id = ?`,
		pattern, matchType, isActive, id)
	return err
}

func (s *FilterStore) Delete(ctx context.Context, id int64) error {
	_, err := s.write.ExecContext(ctx, `DELETE FROM filter_rules WHERE id = ?`, id)
	return err
}

func (s *FilterStore) List(ctx context.Context) ([]models.FilterRule, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, pattern, match_type, is_active FROM filter_rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.FilterRule
	for rows.Next() {
		var r models.FilterRule
		if err := rows.Scan(&r.ID, &r.Pattern, &r.MatchType, &r.IsActive); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *FilterStore) ListActive(ctx context.Context) ([]models.FilterRule, error) {
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, pattern, match_type, is_active FROM filter_rules WHERE is_active = 1 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.FilterRule
	for rows.Next() {
		var r models.FilterRule
		if err := rows.Scan(&r.ID, &r.Pattern, &r.MatchType, &r.IsActive); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}
