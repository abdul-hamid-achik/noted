package notesync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

func TestWatcherFiresOnMarkdownChange(t *testing.T) {
	dir := t.TempDir()
	fired := make(chan struct{}, 8)
	w, err := NewWatcher(dir, 40*time.Millisecond, func() { fired <- struct{}{} })
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Creating a .md file should trigger the (debounced) callback.
	if err := os.WriteFile(filepath.Join(dir, "new.md"), []byte("---\ntitle: New\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not fire on a .md create")
	}

	// Drain any coalesced extra fires, then confirm a non-markdown file is ignored.
	time.Sleep(100 * time.Millisecond)
	for {
		select {
		case <-fired:
			continue
		default:
		}
		break
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-fired:
		t.Fatal("watcher fired on a non-.md file")
	case <-time.After(500 * time.Millisecond):
		// good — no callback for a .txt change
	}
}

// TestWatcherTriggersRebuild exercises the exact chain the TUI wires up: a file landing in the vault
// fires the watcher, whose callback rebuilds the index, after which the new note is queryable. This is
// the live two-way sync path (the only piece not covered is the bubbletea reload message, trivial glue
// that glyph can't drive since it has no file-injection step).
func TestWatcherTriggersRebuild(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	vlt, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	done := make(chan error, 4)
	w, err := NewWatcher(dir, 40*time.Millisecond, func() {
		_, e := Rebuild(ctx, conn, vlt)
		done <- e
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()

	if err := os.WriteFile(filepath.Join(dir, "ext.md"), []byte("---\ntitle: ExternalNote\n---\nhi"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-done:
		if e != nil {
			t.Fatalf("rebuild from watcher callback: %v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not trigger a rebuild")
	}

	notes, _ := db.New(conn).GetAllNotes(ctx)
	found := false
	for _, n := range notes {
		if n.Title == "ExternalNote" {
			found = true
		}
	}
	if !found {
		t.Error("externally-created note was not indexed after the watcher rebuild")
	}
}

// TestWatcherIgnoresVersionSubdirFiles guards invariant: the watcher Add()s only the top vault dir
// (non-recursive), so writing version snapshots under .noted/versions/<id>/ must NOT fire it — else
// every TUI save (which writes a snapshot) would trigger a rebuild loop.
func TestWatcherIgnoresVersionSubdirFiles(t *testing.T) {
	dir := t.TempDir()
	fired := make(chan struct{}, 8)
	w, err := NewWatcher(dir, 40*time.Millisecond, func() { fired <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()

	sub := filepath.Join(dir, ".noted", "versions", "1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "1.md"), []byte("---\nnote_id: 1\nversion: 1\n---\nx"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-fired:
		t.Fatal("watcher fired on a .noted/versions/ subdir write (would cause a rebuild loop)")
	case <-time.After(600 * time.Millisecond):
		// good — nested version-file writes are not watched
	}
}

// TestWatcherPauseSelfWriteSuppresses guards the self-write mute: a .md change that lands while the
// watcher is paused (as the app's own write-through does) must NOT fire the rebuild callback.
func TestWatcherPauseSelfWriteSuppresses(t *testing.T) {
	dir := t.TempDir()
	fired := make(chan struct{}, 8)
	w, err := NewWatcher(dir, 40*time.Millisecond, func() { fired <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()

	w.PauseSelfWrite() // simulate the app muting around its own write
	if err := os.WriteFile(filepath.Join(dir, "self.md"), []byte("---\ntitle: Self\n---\nx"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-fired:
		t.Fatal("watcher fired during the self-write mute window")
	case <-time.After(500 * time.Millisecond):
		// good — the app's own write was suppressed
	}

	// After the mute expires, a genuine external change still fires.
	if err := os.WriteFile(filepath.Join(dir, "external.md"), []byte("---\ntitle: Ext\n---\ny"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not fire on a post-mute external change")
	}
}

func TestWatcherCloseIsSafe(t *testing.T) {
	w, err := NewWatcher(t.TempDir(), 0, func() {})
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
