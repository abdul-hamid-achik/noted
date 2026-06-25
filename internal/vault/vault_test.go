package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixedClock(t *testing.T, ts time.Time) {
	t.Helper()
	old := now
	now = func() time.Time { return ts }
	t.Cleanup(func() { now = old })
}

func TestWriteReadRoundTrip(t *testing.T) {
	ts := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	fixedClock(t, ts)

	v, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	in := Note{
		Title:   "Meeting Notes",
		Tags:    []string{"work", "ideas"},
		Pinned:  true,
		Content: "# Heading\n\nSome *markdown* with a [[wikilink]].",
	}
	written, err := v.Write(in)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(written.Path) != "meeting-notes.md" {
		t.Errorf("path = %q, want meeting-notes.md", filepath.Base(written.Path))
	}
	if written.Created != ts || written.Updated != ts {
		t.Errorf("timestamps not set: created=%v updated=%v", written.Created, written.Updated)
	}

	got, err := v.Read(written.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != in.Title {
		t.Errorf("title = %q, want %q", got.Title, in.Title)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "work" || got.Tags[1] != "ideas" {
		t.Errorf("tags = %v, want [work ideas]", got.Tags)
	}
	if !got.Pinned {
		t.Error("pinned should round-trip true")
	}
	if got.Content != in.Content {
		t.Errorf("content = %q, want %q", got.Content, in.Content)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	n, err := Parse([]byte("# Just content\n\nno frontmatter here"))
	if err != nil {
		t.Fatal(err)
	}
	if n.Title != "" {
		t.Errorf("title should be empty, got %q", n.Title)
	}
	if n.Content != "# Just content\n\nno frontmatter here" {
		t.Errorf("content mismatch: %q", n.Content)
	}
}

func TestReadTitleFromFilename(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "loose-note.md"), []byte("body only"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, _ := Open(dir)
	n, err := v.Read("loose-note.md")
	if err != nil {
		t.Fatal(err)
	}
	if n.Title != "loose note" {
		t.Errorf("title from filename = %q, want %q", n.Title, "loose note")
	}
}

func TestListSortedByUpdated(t *testing.T) {
	v, _ := Open(t.TempDir())
	mk := func(title string, ts time.Time) {
		fixedClock(t, ts)
		if _, err := v.Write(Note{Title: title, Content: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	mk("Older", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	mk("Newer", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))

	notes, err := v.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2", len(notes))
	}
	if notes[0].Title != "Newer" {
		t.Errorf("first should be Newer, got %q", notes[0].Title)
	}
}

func TestUniquePathOnTitleCollision(t *testing.T) {
	v, _ := Open(t.TempDir())
	a, _ := v.Write(Note{Title: "Same Title", Content: "a"})
	b, _ := v.Write(Note{Title: "Same Title", Content: "b"})
	if a.Path == b.Path {
		t.Fatalf("collision not resolved: both at %q", a.Path)
	}
	if filepath.Base(b.Path) != "same-title-2.md" {
		t.Errorf("second path = %q, want same-title-2.md", filepath.Base(b.Path))
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":      "hello-world",
		"  Trim  Me  ":     "trim-me",
		"Special!@#Chars":  "special-chars",
		"":                 "untitled",
		"---":              "untitled",
		"Café 日本 2026":     "caf-2026",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteRawPreservesIDAndTimestamps(t *testing.T) {
	v, _ := Open(t.TempDir())
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	in := Note{ID: 42, Title: "Indexed", Tags: []string{"a", "b"}, Created: created, Updated: updated, Content: "body"}

	w, err := v.WriteRaw(in)
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.Read(w.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 {
		t.Errorf("id = %d, want 42", got.ID)
	}
	if !got.Created.Equal(created) || !got.Updated.Equal(updated) {
		t.Errorf("WriteRaw must preserve timestamps exactly: created=%v updated=%v", got.Created, got.Updated)
	}
}

func TestParseThematicBreakIsNotFrontmatter(t *testing.T) {
	// A valid note opening with a thematic break must not be dropped as bad frontmatter (data loss).
	n, err := Parse([]byte("---\nIntroduction\n---\nbody text here"))
	if err != nil {
		t.Fatalf("Parse should not error on thematic-break content: %v", err)
	}
	if !strings.Contains(n.Content, "Introduction") || !strings.Contains(n.Content, "body text here") {
		t.Errorf("content was lost: %q", n.Content)
	}
}

func TestSyncReusesFileAcrossRename(t *testing.T) {
	v, _ := Open(t.TempDir())
	w, err := v.Sync(Note{ID: 7, Title: "First Title", Content: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Sync(Note{ID: 7, Title: "Renamed", Content: "b"}); err != nil {
		t.Fatal(err)
	}
	notes, _ := v.List()
	if len(notes) != 1 {
		t.Fatalf("rename should reuse one file, got %d files", len(notes))
	}
	if notes[0].Path != w.Path || notes[0].Title != "Renamed" || notes[0].Content != "b" {
		t.Errorf("rename not applied in place: %+v (orig path %q)", notes[0], w.Path)
	}
}

func TestDeleteByID(t *testing.T) {
	v, _ := Open(t.TempDir())
	if _, err := v.Sync(Note{ID: 3, Title: "Doomed", Content: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := v.DeleteByID(3); err != nil {
		t.Fatal(err)
	}
	if notes, _ := v.List(); len(notes) != 0 {
		t.Errorf("expected 0 files after delete, got %d", len(notes))
	}
	if err := v.DeleteByID(999); err != nil {
		t.Errorf("DeleteByID of a missing id should be a no-op, got %v", err)
	}
}

func TestUpdatePreservesCreated(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fixedClock(t, created)
	v, _ := Open(t.TempDir())
	n, _ := v.Write(Note{Title: "Doc", Content: "v1"})

	later := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	fixedClock(t, later)
	n.Content = "v2"
	n, _ = v.Write(n)
	if !n.Created.Equal(created) {
		t.Errorf("created changed: %v, want %v", n.Created, created)
	}
	if !n.Updated.Equal(later) {
		t.Errorf("updated = %v, want %v", n.Updated, later)
	}
}
