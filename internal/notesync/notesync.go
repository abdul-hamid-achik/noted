// Package notesync mirrors database notes to the markdown vault (write-through). It bridges the db
// and vault packages so the CLI and TUI both keep the vault in sync on create/update/delete.
package notesync

import (
	"context"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

// FolderPath returns a folder's slash-joined path (root → leaf) for a folder id, "" if none.
func FolderPath(ctx context.Context, dbq *db.Queries, id int64) string {
	var parts []string
	seen := map[int64]bool{}
	for id != 0 && !seen[id] {
		seen[id] = true
		f, err := dbq.GetFolder(ctx, id)
		if err != nil {
			break
		}
		parts = append([]string{f.Name}, parts...)
		if f.ParentID.Valid {
			id = f.ParentID.Int64
		} else {
			break
		}
	}
	return strings.Join(parts, "/")
}

// WriteThrough mirrors a saved note (and its current tags) to the vault. Best-effort and a no-op
// when the vault is nil.
func WriteThrough(ctx context.Context, dbq *db.Queries, vlt *vault.Vault, n db.Note) {
	if vlt == nil || dbq == nil {
		return
	}
	tags, _ := dbq.GetTagsForNote(ctx, n.ID)
	tnames := make([]string, len(tags))
	for i, t := range tags {
		tnames[i] = t.Name
	}
	vn := vault.Note{
		ID:      n.ID,
		Title:   n.Title,
		Tags:    tnames,
		Pinned:  n.Pinned.Valid && n.Pinned.Bool,
		Content: n.Content,
	}
	if n.FolderID.Valid {
		vn.Folder = FolderPath(ctx, dbq, n.FolderID.Int64)
	}
	if n.CreatedAt.Valid {
		vn.Created = n.CreatedAt.Time
	}
	if n.UpdatedAt.Valid {
		vn.Updated = n.UpdatedAt.Time
	}
	_, _ = vlt.Sync(vn)
}

// Delete removes a note's vault file and its persisted version snapshots. Best-effort and a no-op
// when the vault is nil. Removing the snapshots prevents stale history from grafting onto a future
// note that reuses the same id.
func Delete(vlt *vault.Vault, id int64) {
	if vlt != nil {
		_ = vlt.DeleteByID(id)
		_ = vlt.DeleteVersions(id)
	}
}
