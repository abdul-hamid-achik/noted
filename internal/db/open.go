package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// foreign_keys and busy_timeout are per-connection pragmas, so set them in the DSN: modernc
	// applies them on every pooled connection. (A one-off PRAGMA Exec only configures whichever single
	// connection happened to serve it, leaving FK enforcement — e.g. note_versions ON DELETE CASCADE,
	// folder ON DELETE SET NULL — nondeterministic under the database/sql pool.)
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// WAL is a persistent, database-level setting (stored in the file header), so once is enough.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Run migrations to create/update schema
	if err := RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}
