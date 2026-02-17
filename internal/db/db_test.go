package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	conn, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, dbPath
}

func TestOpen_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "a", "b", "c")
	dbPath := filepath.Join(nested, "test.db")

	conn, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer conn.Close()

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("expected nested directory to be created")
	}
}

func TestOpen_WALMode(t *testing.T) {
	conn, _ := openTestDB(t)

	var mode string
	if err := conn.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", mode)
	}
}

func TestOpen_BusyTimeout(t *testing.T) {
	conn, _ := openTestDB(t)

	var timeout int64
	if err := conn.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Errorf("expected busy_timeout=5000, got %d", timeout)
	}
}

func TestOpen_ForeignKeys(t *testing.T) {
	conn, _ := openTestDB(t)

	var fk int64
	if err := conn.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fk)
	}
}

func TestMigrations_Ordering(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatalf("failed to load migrations: %v", err)
	}

	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version <= migrations[i-1].Version {
			t.Errorf("migration %d (v%d) is not after migration %d (v%d)",
				i, migrations[i].Version, i-1, migrations[i-1].Version)
		}
	}
}

func TestMigrations_AllApplied(t *testing.T) {
	conn, _ := openTestDB(t)

	current, latest, pending, err := MigrationStatus(conn)
	if err != nil {
		t.Fatalf("failed to get migration status: %v", err)
	}
	if pending != 0 {
		t.Errorf("expected 0 pending migrations, got %d (current=%d, latest=%d)", pending, current, latest)
	}
	if current != latest {
		t.Errorf("expected current=%d to equal latest=%d", current, latest)
	}
}

func TestMigrations_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open twice to ensure migrations can run again without error
	conn1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open failed: %v", err)
	}
	conn1.Close()

	conn2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open failed: %v", err)
	}
	defer conn2.Close()
}

func TestFTS_SearchNotesFTS(t *testing.T) {
	conn, _ := openTestDB(t)
	queries := New(conn)
	ctx := context.Background()

	// Create notes
	queries.CreateNote(ctx, CreateNoteParams{Title: "Go Tutorial", Content: "Learn Go programming"})
	queries.CreateNote(ctx, CreateNoteParams{Title: "Python Guide", Content: "Python basics"})

	if !FTSAvailable(ctx, conn) {
		t.Fatal("FTS5 should be available after migration")
	}

	results, err := SearchNotesFTS(ctx, conn, "Go", 10)
	if err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 FTS result for 'Go', got %d", len(results))
	}
	if len(results) > 0 && results[0].Title != "Go Tutorial" {
		t.Errorf("expected title 'Go Tutorial', got %q", results[0].Title)
	}
}

func TestFTS_UpdateSync(t *testing.T) {
	conn, _ := openTestDB(t)
	queries := New(conn)
	ctx := context.Background()

	note, _ := queries.CreateNote(ctx, CreateNoteParams{Title: "Original", Content: "original content"})

	// Update the note
	queries.UpdateNote(ctx, UpdateNoteParams{ID: note.ID, Title: "Updated", Content: "updated content"})

	// Search for updated content
	results, err := SearchNotesFTS(ctx, conn, "updated", 10)
	if err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for updated content, got %d", len(results))
	}

	// Old content should not be found
	results, err = SearchNotesFTS(ctx, conn, "original", 10)
	if err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for old content, got %d", len(results))
	}
}

func TestFTS_DeleteSync(t *testing.T) {
	conn, _ := openTestDB(t)
	queries := New(conn)
	ctx := context.Background()

	note, _ := queries.CreateNote(ctx, CreateNoteParams{Title: "To Delete", Content: "delete me"})

	// Delete the note
	queries.DeleteNote(ctx, note.ID)

	// Deleted content should not be found
	results, err := SearchNotesFTS(ctx, conn, "delete", 10)
	if err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}
