package notesync

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

func vfind(vlt *vault.Vault, id int64) (vault.Note, bool) {
	notes, _ := vlt.List()
	for _, n := range notes {
		if n.ID == id {
			return n, true
		}
	}
	return vault.Note{}, false
}

func TestWriteThroughMapsTagsFolderAndDeletes(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	dbq := db.New(conn)
	vlt, err := vault.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// nil-safe no-ops
	WriteThrough(ctx, dbq, nil, db.Note{})
	WriteThrough(ctx, nil, vlt, db.Note{})
	Delete(nil, 1)

	n, err := dbq.CreateNote(ctx, db.CreateNoteParams{Title: "Bridged", Content: "body"})
	if err != nil {
		t.Fatal(err)
	}
	tag, _ := dbq.CreateTag(ctx, "ai")
	_ = dbq.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: n.ID, TagID: tag.ID})
	fld, _ := dbq.CreateFolder(ctx, db.CreateFolderParams{Name: "Inbox"})
	_ = dbq.MoveNoteToFolder(ctx, db.MoveNoteToFolderParams{
		FolderID: sql.NullInt64{Int64: fld.ID, Valid: true}, ID: n.ID,
	})
	n, _ = dbq.GetNote(ctx, n.ID)

	WriteThrough(ctx, dbq, vlt, n)

	got, ok := vfind(vlt, n.ID)
	if !ok {
		t.Fatal("note was not written to the vault")
	}
	if got.Title != "Bridged" {
		t.Errorf("title = %q, want Bridged", got.Title)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "ai" {
		t.Errorf("tags = %v, want [ai]", got.Tags)
	}
	if got.Folder != "Inbox" {
		t.Errorf("folder = %q, want Inbox", got.Folder)
	}

	Delete(vlt, n.ID)
	if _, ok := vfind(vlt, n.ID); ok {
		t.Error("note was not deleted from the vault")
	}
}
