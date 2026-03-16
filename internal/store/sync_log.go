package store

import (
	"context"
	"database/sql"

	"finance_tracker/internal/models"
)

type SyncLogStore struct {
	read  *sql.DB
	write *sql.DB
}

func NewSyncLogStore(read, write *sql.DB) *SyncLogStore {
	return &SyncLogStore{read: read, write: write}
}

func (s *SyncLogStore) Create(ctx context.Context) (int64, error) {
	res, err := s.write.ExecContext(ctx, `INSERT INTO sync_log (status) VALUES ('running')`)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SyncLogStore) Complete(ctx context.Context, id int64, status string, added, updated int, errMsg string, apiErrors string) error {
	_, err := s.write.ExecContext(ctx, `
		UPDATE sync_log SET
			completed_at = datetime('now'),
			status = ?,
			transactions_added = ?,
			transactions_updated = ?,
			error_message = ?,
			api_errors = ?
		WHERE id = ?`,
		status, added, updated, errMsg, apiErrors, id)
	return err
}

func (s *SyncLogStore) List(ctx context.Context, limit int) ([]models.SyncLogEntry, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.read.QueryContext(ctx, `
		SELECT id, started_at, COALESCE(completed_at, ''), status, transactions_added, transactions_updated, error_message, api_errors
		FROM sync_log ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.SyncLogEntry
	for rows.Next() {
		var e models.SyncLogEntry
		if err := rows.Scan(&e.ID, &e.StartedAt, &e.CompletedAt, &e.Status, &e.TransactionsAdded, &e.TransactionsUpdated, &e.ErrorMessage, &e.APIErrors); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
