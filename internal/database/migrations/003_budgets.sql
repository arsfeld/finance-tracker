-- +goose Up
CREATE TABLE budgets (
    category TEXT NOT NULL PRIMARY KEY COLLATE NOCASE,
    amount REAL NOT NULL CHECK(amount > 0),
    CHECK(length(trim(category)) > 0)
);

-- +goose Down
DROP TABLE IF EXISTS budgets;
