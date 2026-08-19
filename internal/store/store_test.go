package store

import (
	"path/filepath"
	"testing"

	"finance_tracker/internal/database"
)

// newTestDB creates a migrated SQLite database in a temporary directory.
func newTestDB(t *testing.T) *database.DB {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.Migrate(db.Write); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}
