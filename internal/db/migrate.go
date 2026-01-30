package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a single database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// GetSchemaVersion returns the current schema version from PRAGMA user_version
func GetSchemaVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("PRAGMA user_version").Scan(&version)
	return version, err
}

// SetSchemaVersion sets the schema version using PRAGMA user_version
func SetSchemaVersion(db *sql.DB, version int) error {
	_, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", version))
	return err
}

// LoadMigrations loads all migration files from the embedded filesystem
func LoadMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Parse version from filename (e.g., "001_initial.sql" -> 1)
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// RunMigrations applies all pending migrations to the database
func RunMigrations(db *sql.DB) error {
	currentVersion, err := GetSchemaVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	migrations, err := LoadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue // Already applied
		}

		// Run migration in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", m.Name, err)
		}

		if _, err := tx.Exec(m.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to apply migration %s: %w", m.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.Name, err)
		}

		// Update version after successful migration
		if err := SetSchemaVersion(db, m.Version); err != nil {
			return fmt.Errorf("failed to update schema version after %s: %w", m.Name, err)
		}
	}

	return nil
}

// MigrationStatus returns the current and latest migration versions
func MigrationStatus(db *sql.DB) (current int, latest int, pending int, err error) {
	current, err = GetSchemaVersion(db)
	if err != nil {
		return 0, 0, 0, err
	}

	migrations, err := LoadMigrations()
	if err != nil {
		return 0, 0, 0, err
	}

	if len(migrations) > 0 {
		latest = migrations[len(migrations)-1].Version
	}

	pending = latest - current
	if pending < 0 {
		pending = 0
	}

	return current, latest, pending, nil
}
