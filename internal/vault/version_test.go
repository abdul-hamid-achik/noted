package vault

import (
	"testing"
	"time"
)

func TestVersionRoundTrip(t *testing.T) {
	v, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	ok, err := v.WriteVersion(Version{NoteID: 7, VersionNumber: 2, Title: "T", Created: created, Content: "hello\nworld"})
	if err != nil || !ok {
		t.Fatalf("WriteVersion ok=%v err=%v", ok, err)
	}

	// Snapshots are immutable: a second write for the same (note, version) is a no-op.
	if ok2, _ := v.WriteVersion(Version{NoteID: 7, VersionNumber: 2, Title: "changed", Content: "changed"}); ok2 {
		t.Error("WriteVersion must not overwrite an existing snapshot")
	}

	all, err := v.AllVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("AllVersions = %d, want 1", len(all))
	}
	g := all[0]
	if g.NoteID != 7 || g.VersionNumber != 2 || g.Title != "T" || g.Content != "hello\nworld" {
		t.Errorf("round-trip mismatch: %+v", g)
	}
	if !g.Created.Equal(created) {
		t.Errorf("created = %v, want %v", g.Created, created)
	}

	// The hidden .noted/ directory must never surface as a note.
	notes, _ := v.List()
	if len(notes) != 0 {
		t.Errorf("List should ignore .noted/, got %d notes", len(notes))
	}
}
