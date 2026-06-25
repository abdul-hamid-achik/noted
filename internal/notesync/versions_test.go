package notesync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

func TestLatestVersionNumber(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int64
	}{
		{"int64", int64(7), 7},
		{"float64", float64(7), 7},
		{"float64-truncates", float64(7.9), 7},
		{"nil-defaults-zero", nil, 0},
		{"unknown-type-defaults-zero", "x", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := latestVersionNumber(c.in); got != c.want {
				t.Errorf("latestVersionNumber(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

// TestRebuildSkipsOrphanVersions: a vault snapshot whose note has no .md (so it's absent from the
// rebuilt index) must be skipped, not error the rebuild or violate the FK.
func TestRebuildSkipsOrphanVersions(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(t.TempDir(), "o.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	vlt, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// One real note (id 1) with a snapshot, plus an orphan snapshot for id 9 (no note .md).
	if err := os.WriteFile(filepath.Join(dir, "n.md"), []byte("---\nid: 1\ntitle: N\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := vlt.WriteVersion(vault.Version{NoteID: 1, VersionNumber: 1, Title: "N", Content: "old"}); err != nil {
		t.Fatal(err)
	}
	if _, err := vlt.WriteVersion(vault.Version{NoteID: 9, VersionNumber: 1, Title: "Ghost", Content: "orphan"}); err != nil {
		t.Fatal(err)
	}

	stats, err := Rebuild(ctx, conn, vlt)
	if err != nil {
		t.Fatalf("Rebuild must not error on an orphan snapshot: %v", err)
	}
	if stats.RestoredVersions != 1 {
		t.Errorf("RestoredVersions = %d, want 1 (orphan skipped)", stats.RestoredVersions)
	}
	q := db.New(conn)
	if v9, _ := q.GetNoteVersions(ctx, 9); len(v9) != 0 {
		t.Errorf("orphan note 9 should have no restored versions, got %d", len(v9))
	}
	if v1, _ := q.GetNoteVersions(ctx, 1); len(v1) != 1 {
		t.Errorf("note 1 should have its 1 version restored, got %d", len(v1))
	}
}

// TestRebuildRemapWithVersions documents the duplicate-id behavior: the second note with a duplicate
// frontmatter id is remapped to a fresh id, and the rebuild does not error.
func TestRebuildRemapWithVersions(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(t.TempDir(), "rm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	vlt, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\nid: 1\ntitle: A\n---\na\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("---\nid: 1\ntitle: B\n---\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := vlt.WriteVersion(vault.Version{NoteID: 1, VersionNumber: 1, Title: "A", Content: "a-old"}); err != nil {
		t.Fatal(err)
	}

	stats, err := Rebuild(ctx, conn, vlt)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if stats.Notes != 2 {
		t.Errorf("notes = %d, want 2", stats.Notes)
	}
	if stats.RemappedIDs != 1 {
		t.Errorf("RemappedIDs = %d, want 1", stats.RemappedIDs)
	}
	// The snapshot keyed by id 1 is restored onto whichever note kept id 1 (no crash, no FK error).
	if stats.RestoredVersions != 1 {
		t.Errorf("RestoredVersions = %d, want 1", stats.RestoredVersions)
	}
}

func TestSnapshotVersion(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(t.TempDir(), "s.db"))
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

	note, err := q.CreateNote(ctx, db.CreateNoteParams{Title: "Doc", Content: "v1 body"})
	if err != nil {
		t.Fatal(err)
	}

	// First snapshot of a note with no history must be version 1, the second version 2.
	if err := SnapshotVersion(ctx, q, vlt, note.ID, "Doc", "v1 body"); err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if err := SnapshotVersion(ctx, q, vlt, note.ID, "Doc", "v2 body"); err != nil {
		t.Fatalf("second snapshot: %v", err)
	}

	versions, err := q.GetNoteVersions(ctx, note.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("got %d versions, want 2", len(versions))
	}
	nums := map[int64]string{}
	for _, v := range versions {
		nums[v.VersionNumber] = v.Content
	}
	if nums[1] != "v1 body" || nums[2] != "v2 body" {
		t.Errorf("version numbering/content wrong: %+v", nums)
	}

	// Each snapshot is immediately durable in the vault (no export needed).
	vv, err := vlt.AllVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(vv) != 2 {
		t.Errorf("vault holds %d snapshots, want 2", len(vv))
	}

	// A nil dbq is a no-op (must not panic).
	if err := SnapshotVersion(ctx, nil, vlt, note.ID, "x", "y"); err != nil {
		t.Errorf("nil dbq should be a no-op, got %v", err)
	}
}

// TestRebuildPreservesVersionHistory is the regression for the cascade-wipe: DELETE FROM notes
// cascades to note_versions, so a naive rebuild would destroy history. Rebuild must persist snapshots
// to the vault before clearing and restore them afterward, so history survives — and becomes durable
// in the vault.
func TestRebuildPreservesVersionHistory(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(t.TempDir(), "v.db"))
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

	// A note (id 1) exists in both the vault (so Rebuild keeps it) and the DB (so it has a version).
	if err := os.WriteFile(filepath.Join(dir, "note.md"),
		[]byte("---\nid: 1\ntitle: Versioned\n---\n\nv2 content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "INSERT INTO notes (id, title, content) VALUES (1, 'Versioned', 'v2 content')"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID: 1, Title: "Versioned", Content: "v1 content", VersionNumber: 1,
	}); err != nil {
		t.Fatal(err)
	}

	if pre, _ := q.GetNoteVersions(ctx, 1); len(pre) != 1 {
		t.Fatalf("pre-rebuild versions = %d, want 1", len(pre))
	}

	stats, err := Rebuild(ctx, conn, vlt)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if stats.Notes != 1 {
		t.Fatalf("notes = %d, want 1", stats.Notes)
	}
	if stats.RestoredVersions != 1 {
		t.Errorf("RestoredVersions = %d, want 1", stats.RestoredVersions)
	}

	post, _ := q.GetNoteVersions(ctx, 1)
	if len(post) != 1 {
		t.Fatalf("version history lost on rebuild: got %d, want 1", len(post))
	}
	if post[0].Content != "v1 content" {
		t.Errorf("restored version content = %q, want %q", post[0].Content, "v1 content")
	}

	// The snapshot is now durable in the vault, and a second rebuild is idempotent (nothing new).
	if vv, _ := vlt.AllVersions(); len(vv) != 1 || vv[0].NoteID != 1 {
		t.Errorf("vault versions = %+v, want one snapshot for note 1", vv)
	}
	stats2, err := Rebuild(ctx, conn, vlt)
	if err != nil {
		t.Fatal(err)
	}
	if stats2.RestoredVersions != 1 {
		t.Errorf("second rebuild RestoredVersions = %d, want 1 (idempotent restore)", stats2.RestoredVersions)
	}
	if post2, _ := q.GetNoteVersions(ctx, 1); len(post2) != 1 {
		t.Errorf("duplicate versions after second rebuild: got %d, want 1", len(post2))
	}
}
