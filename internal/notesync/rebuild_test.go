package notesync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

func TestRebuildFromVault(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha.md", "---\nid: 1\ntitle: Alpha\ntags: [x]\n---\n\nsee [[Beta]]\n")
	write("beta.md", "---\nid: 2\ntitle: Beta\nfolder: Work/Reports\n---\n\nbeta body\n")

	conn, err := db.Open(filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	vlt, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	stats, err := Rebuild(ctx, conn, vlt)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if stats.Notes != 2 || stats.Links != 1 {
		t.Errorf("stats = %+v, want 2 notes / 1 link", stats)
	}

	q := db.New(conn)
	notes, _ := q.GetAllNotes(ctx)
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2", len(notes))
	}

	var beta db.Note
	for _, n := range notes {
		if n.Title == "Beta" {
			beta = n
		}
	}
	if !beta.FolderID.Valid {
		t.Fatal("Beta lost its folder on rebuild")
	}
	leaf, err := q.GetFolder(ctx, beta.FolderID.Int64)
	if err != nil || leaf.Name != "Reports" {
		t.Fatalf("leaf folder = %v (err %v), want Reports", leaf.Name, err)
	}
}
