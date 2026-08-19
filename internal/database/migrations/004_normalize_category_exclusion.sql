-- +goose Up
-- Exclusion is a property of a category, but it was stored as a flag on every
-- merchant row. When the LLM recategorized a merchant it carried the flag into
-- its new category, silently excluding that entire category from analysis.
-- Storing exclusions in their own table makes the drift impossible.
CREATE TABLE IF NOT EXISTS excluded_categories (
    category TEXT PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Carry over existing exclusions so analysis keeps behaving the same way it did
-- before this migration. Drifted flags are a data problem, corrected through the
-- Settings page rather than guessed at here.
INSERT OR IGNORE INTO excluded_categories (category)
SELECT DISTINCT category FROM categories WHERE excluded = 1;

ALTER TABLE categories DROP COLUMN excluded;

-- +goose Down
ALTER TABLE categories ADD COLUMN excluded INTEGER NOT NULL DEFAULT 0;
UPDATE categories SET excluded = 1
WHERE category IN (SELECT category FROM excluded_categories);
DROP TABLE IF EXISTS excluded_categories;
