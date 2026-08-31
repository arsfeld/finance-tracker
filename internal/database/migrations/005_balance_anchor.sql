-- +goose Up
-- A connection can refresh balances while delivering no transactions, which
-- reads as a spending drop rather than as a fault. Anchoring each account to the
-- first balance we ever saw lets the transactions since then be checked against
-- the balance they are supposed to explain.
--
-- The anchor is written once and never moved: transactions backfilled later
-- carry a posted date at or before it and are excluded from the sum, so a late
-- delivery closes the gap instead of opening a new one.
ALTER TABLE accounts ADD COLUMN anchor_balance REAL;
ALTER TABLE accounts ADD COLUMN anchor_balance_date INTEGER;

UPDATE accounts
SET anchor_balance = balance,
    anchor_balance_date = balance_date
WHERE anchor_balance IS NULL;

-- +goose Down
ALTER TABLE accounts DROP COLUMN anchor_balance;
ALTER TABLE accounts DROP COLUMN anchor_balance_date;
