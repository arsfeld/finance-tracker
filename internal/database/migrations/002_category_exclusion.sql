-- +goose Up
ALTER TABLE categories ADD COLUMN excluded INTEGER NOT NULL DEFAULT 0;
DROP TABLE IF EXISTS filter_rules;
DROP TABLE IF EXISTS specific_exclusions;

-- +goose Down
DROP TABLE IF EXISTS specific_exclusions;
CREATE TABLE IF NOT EXISTS specific_exclusions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,
    pattern TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'substring',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS filter_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'substring',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
