package notesync

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

// TestRebuildPreservesMemories: an index-only memory note (tagged "memory", never written to the
// vault) must survive a vault→index rebuild — with its content, tags, and source metadata intact —
// while regular vault notes are rebuilt as usual.
func TestRebuildPreservesMemories(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	vlt, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	q := db.New(conn)

	// A regular note that lives in the vault.
	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte("---\nid: 1\ntitle: Real\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "INSERT INTO notes (id, title, content) VALUES (1, 'Real', 'body')"); err != nil {
		t.Fatal(err)
	}

	// A memory: in the DB (tagged "memory" + "importance:4", with a source) but NOT in the vault.
	mem, err := q.CreateNoteWithTTL(ctx, db.CreateNoteWithTTLParams{
		Title:   "mem-pref",
		Content: "user prefers dark mode",
		Source:  sql.NullString{String: "manual", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	memID := mem.ID
	for _, tn := range []string{"memory", "importance:4"} {
		tag, _ := q.CreateTag(ctx, tn)
		_ = q.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: memID, TagID: tag.ID})
	}

	stats, err := Rebuild(ctx, conn, vlt)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if stats.PreservedMemories != 1 {
		t.Errorf("PreservedMemories = %d, want 1", stats.PreservedMemories)
	}
	if stats.Notes != 1 {
		t.Errorf("vault Notes = %d, want 1 (the regular note)", stats.Notes)
	}

	// The memory survived with content + source intact, keeping its id.
	got, err := q.GetNote(ctx, memID)
	if err != nil {
		t.Fatalf("memory was wiped by the rebuild: %v", err)
	}
	if got.Content != "user prefers dark mode" {
		t.Errorf("memory content = %q, want preserved", got.Content)
	}
	if !got.Source.Valid || got.Source.String != "manual" {
		t.Errorf("memory source lost: %+v", got.Source)
	}
	tags, _ := q.GetTagsForNote(ctx, memID)
	var hasMemory, hasImportance bool
	for _, tg := range tags {
		if tg.Name == "memory" {
			hasMemory = true
		}
		if tg.Name == "importance:4" {
			hasImportance = true
		}
	}
	if !hasMemory || !hasImportance {
		t.Errorf("memory tags lost on rebuild: %v", tags)
	}

	// The regular vault note also survived.
	if _, err := q.GetNote(ctx, 1); err != nil {
		t.Errorf("regular note lost on rebuild: %v", err)
	}
}

// TestRebuildMemoryIDCollision: if a hand-edited vault note claims the same id a memory holds, the
// rebuilt vault note keeps that id and the memory is re-inserted with a fresh id (not lost, not a
// PRIMARY KEY error).
func TestRebuildMemoryIDCollision(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	vlt, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	q := db.New(conn)

	// Memory gets id 1 (first insert).
	mem, err := q.CreateNoteWithTTL(ctx, db.CreateNoteWithTTLParams{Title: "mem", Content: "remember me"})
	if err != nil {
		t.Fatal(err)
	}
	tag, _ := q.CreateTag(ctx, "memory")
	_ = q.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: mem.ID, TagID: tag.ID})

	// A hand-edited vault note also claims id 1.
	if err := os.WriteFile(filepath.Join(dir, "v.md"), []byte("---\nid: 1\ntitle: Vault\n---\nvault body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stats, err := Rebuild(ctx, conn, vlt)
	if err != nil {
		t.Fatalf("Rebuild must not error on id collision: %v", err)
	}
	if stats.PreservedMemories != 1 {
		t.Errorf("PreservedMemories = %d, want 1", stats.PreservedMemories)
	}

	// id 1 is the vault note; the memory was remapped to a fresh id but kept its content + tag.
	if n, _ := q.GetNote(ctx, 1); n.Content != "vault body" {
		t.Errorf("id 1 should be the vault note, got %q", n.Content)
	}
	all, _ := q.GetAllNotes(ctx)
	var memFound bool
	for _, n := range all {
		if n.Content == "remember me" && n.ID != 1 {
			memFound = true
		}
	}
	if !memFound {
		t.Error("memory was lost on id-collision remap")
	}
}
