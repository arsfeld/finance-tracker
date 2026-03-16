package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB holds the read and write database connections.
type DB struct {
	Read  *sql.DB
	Write *sql.DB
}

// Open creates SQLite database connections with appropriate settings.
// It returns separate read and write pools to match SQLite's single-writer model.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared", path)

	// Write pool: single connection to avoid SQLITE_BUSY on writes.
	writeDB, err := sql.Open("sqlite", dsn+"&_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("open write db: %w", err)
	}
	writeDB.SetMaxOpenConns(1)

	if err := setPragmas(writeDB); err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("set write pragmas: %w", err)
	}

	// Read pool: multiple connections for concurrent reads.
	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("open read db: %w", err)
	}
	readDB.SetMaxOpenConns(4)

	if err := setPragmas(readDB); err != nil {
		writeDB.Close()
		readDB.Close()
		return nil, fmt.Errorf("set read pragmas: %w", err)
	}

	return &DB{Read: readDB, Write: writeDB}, nil
}

// Close closes both database connections.
func (db *DB) Close() error {
	var errs []error
	if err := db.Read.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := db.Write.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("close db: %v", errs)
	}
	return nil
}

// Migrate runs all pending database migrations.
func Migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Info().Msg("Database migrations complete")
	return nil
}

func setPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA synchronous = NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("exec %q: %w", p, err)
		}
	}
	return nil
}
