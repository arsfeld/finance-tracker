-- +goose Up
-- Card spending is measured from balances, not transactions: the transaction
-- feed can stop while the balance keeps refreshing. That needs to know which
-- accounts are cards, a key that survives SimpleFin issuing new account IDs on
-- re-auth, and a history of balances to take differences of.
ALTER TABLE accounts ADD COLUMN is_credit_card INTEGER NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN card_key TEXT;

CREATE TABLE balance_snapshots (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    card_key     TEXT    NOT NULL,
    account_id   TEXT    NOT NULL,
    balance      REAL    NOT NULL,
    balance_date INTEGER NOT NULL,
    source       TEXT    NOT NULL,
    created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE (card_key, balance_date)
);

-- +goose Down
DROP TABLE balance_snapshots;
ALTER TABLE accounts DROP COLUMN card_key;
ALTER TABLE accounts DROP COLUMN is_credit_card;
