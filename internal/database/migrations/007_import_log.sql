-- +goose Up
-- Imports share the sync history with SimpleFIN. The source, file name and
-- destination account are what make an import entry readable, and what let the
-- UI tell the two kinds of run apart.
ALTER TABLE sync_log ADD COLUMN source TEXT NOT NULL DEFAULT 'simplefin';
ALTER TABLE sync_log ADD COLUMN filename TEXT NOT NULL DEFAULT '';
ALTER TABLE sync_log ADD COLUMN account_id TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sync_log DROP COLUMN account_id;
ALTER TABLE sync_log DROP COLUMN filename;
ALTER TABLE sync_log DROP COLUMN source;
