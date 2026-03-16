-- +goose Up

CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    balance REAL NOT NULL DEFAULT 0,
    balance_date INTEGER,
    currency TEXT DEFAULT '',
    org_name TEXT DEFAULT '',
    org_domain TEXT DEFAULT '',
    is_included INTEGER NOT NULL DEFAULT 1,
    first_seen_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id),
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    posted INTEGER NOT NULL,
    transacted_at INTEGER,
    pending INTEGER NOT NULL DEFAULT 0,
    cached_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_transactions_account ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_posted ON transactions(posted);
CREATE INDEX IF NOT EXISTS idx_transactions_description ON transactions(description);

CREATE TABLE IF NOT EXISTS categories (
    merchant_description TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'llm',
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS category_overrides (
    transaction_id TEXT PRIMARY KEY REFERENCES transactions(id),
    category TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS analyses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    period_start INTEGER NOT NULL,
    period_end INTEGER NOT NULL,
    billing_day INTEGER NOT NULL,
    date_range_type TEXT NOT NULL,
    response_text TEXT NOT NULL,
    model_used TEXT DEFAULT '',
    is_multi_period INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'success'
);

CREATE INDEX IF NOT EXISTS idx_analyses_created ON analyses(created_at);

CREATE TABLE IF NOT EXISTS filter_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'substring',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS specific_exclusions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,
    pattern TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'substring',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT,
    status TEXT NOT NULL DEFAULT 'running',
    transactions_added INTEGER DEFAULT 0,
    transactions_updated INTEGER DEFAULT 0,
    error_message TEXT DEFAULT '',
    api_errors TEXT DEFAULT '[]'
);

-- +goose Down

DROP TABLE IF EXISTS sync_log;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS specific_exclusions;
DROP TABLE IF EXISTS filter_rules;
DROP TABLE IF EXISTS analyses;
DROP TABLE IF EXISTS category_overrides;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;
