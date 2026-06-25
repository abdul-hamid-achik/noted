package notesync

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

// RebuildStats summarizes a vault→index rebuild.
type RebuildStats struct {
	Notes             int
	Links             int
	RemappedIDs       int
	RestoredVersions  int
	PreservedMemories int
}

var rebuildWikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// rebuildLinkTitles returns the distinct [[wikilink]] targets in content ([[Title|alias]] → "Title").
func rebuildLinkTitles(content string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range rebuildWikilinkRe.FindAllStringSubmatch(content, -1) {
		t := strings.TrimSpace(m[1])
		if i := strings.Index(t, "|"); i >= 0 {
			t = strings.TrimSpace(t[:i])
		}
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// rebuildFolderPathID find-or-creates a folder hierarchy ("A/B/C") within the rebuild transaction,
// preserving nesting and distinguishing same-name folders under different parents. Returns the leaf id.
func rebuildFolderPathID(ctx context.Context, tx *sql.Tx, cache map[string]int64, path string) (int64, error) {
	var parent int64 // 0 = root
	var cum string
	for _, name := range strings.Split(path, "/") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		cum += "/" + name
		if id, ok := cache[cum]; ok {
			parent = id
			continue
		}
		var id int64
		var err error
		if parent == 0 {
			err = tx.QueryRowContext(ctx, "SELECT id FROM folders WHERE name = ? AND parent_id IS NULL LIMIT 1", name).Scan(&id)
		} else {
			err = tx.QueryRowContext(ctx, "SELECT id FROM folders WHERE name = ? AND parent_id = ? LIMIT 1", name, parent).Scan(&id)
		}
		if err == sql.ErrNoRows {
			var res sql.Result
			if parent == 0 {
				res, err = tx.ExecContext(ctx, "INSERT INTO folders (name) VALUES (?)", name)
			} else {
				res, err = tx.ExecContext(ctx, "INSERT INTO folders (name, parent_id) VALUES (?, ?)", name, parent)
			}
			if err != nil {
				return 0, err
			}
			id, _ = res.LastInsertId()
		} else if err != nil {
			return 0, err
		}
		cache[cum] = id
		parent = id
	}
	return parent, nil
}

// Rebuild replaces the SQLite index (notes/tags/links/folders) with the contents of the vault,
// treating the vault as the source of truth. Each note's frontmatter id is preserved where unique; a
// duplicate id is re-inserted with a fresh autoincrement id (counted in RemappedIDs) rather than
// aborting. Wikilinks resolve by title, skipping unknown or ambiguous (duplicate-title) targets.
// FTS stays current via the notes_fts triggers. Version history is preserved across the rebuild: it
// is first persisted to the vault (.noted/versions/), then restored after notes are re-inserted —
// because DELETE FROM notes cascades to note_versions, this persist-then-restore is what keeps history.
func Rebuild(ctx context.Context, conn *sql.DB, vlt *vault.Vault) (RebuildStats, error) {
	var stats RebuildStats
	if conn == nil || vlt == nil {
		return stats, fmt.Errorf("notesync.Rebuild: nil conn or vault")
	}
	notes, err := vlt.List()
	if err != nil {
		return stats, err
	}

	// Persist current DB version history to the vault BEFORE clearing — DELETE FROM notes cascades to
	// note_versions (ON DELETE CASCADE), so any snapshots not yet on disk would otherwise be lost.
	// Best-effort: a failure here must not block the rebuild.
	_, _ = PersistVersions(ctx, db.New(conn), vlt)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return stats, err
	}
	defer func() { _ = tx.Rollback() }()

	// Agent memories (internal/memory) live in the notes table but are intentionally NOT mirrored to
	// the vault (they're index-only, TTL-managed, and would clutter the human vault). The vault is the
	// source of truth for everything else, so the clear below would delete them — capture them first
	// and re-insert after, so an in-place rebuild preserves memories. (A rebuild of a fresh, empty db
	// from only a vault legitimately has no memories to preserve.)
	mems, err := captureMemories(ctx, tx)
	if err != nil {
		return stats, fmt.Errorf("capture memories: %w", err)
	}

	for _, stmt := range []string{"DELETE FROM note_links", "DELETE FROM note_tags", "DELETE FROM notes", "DELETE FROM tags", "DELETE FROM folders"} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return stats, fmt.Errorf("clear index: %w", err)
		}
	}

	type noteRef struct {
		id      int64
		content string
	}
	var refs []noteRef
	titleToID := make(map[string]int64, len(notes))
	titleCount := map[string]int{}
	usedID := map[int64]bool{}
	folderCache := map[string]int64{}

	for _, vn := range notes {
		created, updated := vn.Created, vn.Updated
		if created.IsZero() {
			created = time.Now()
		}
		if updated.IsZero() {
			updated = time.Now()
		}
		cs := created.UTC().Format("2006-01-02 15:04:05")
		us := updated.UTC().Format("2006-01-02 15:04:05")

		var folderArg any
		if vn.Folder != "" {
			fid, err := rebuildFolderPathID(ctx, tx, folderCache, vn.Folder)
			if err != nil {
				return stats, err
			}
			folderArg = fid
		}

		var id int64
		if vn.ID > 0 && !usedID[vn.ID] {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO notes (id, title, content, created_at, updated_at, pinned, folder_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
				vn.ID, vn.Title, vn.Content, cs, us, vn.Pinned, folderArg); err != nil {
				return stats, fmt.Errorf("insert note %q: %w", vn.Title, err)
			}
			id = vn.ID
		} else {
			if vn.ID > 0 {
				// Duplicate frontmatter id (a degraded state from hand-editing) — give this note a
				// fresh one. Known limitation: version snapshots are keyed by frontmatter id under
				// .noted/versions/<id>/, so the remapped note's history stays under the shared id and
				// is restored onto whichever note kept it (possible mis-attribution); the remapped
				// note gets no history. Healthy vaults never hit this — ids are unique primary keys.
				stats.RemappedIDs++
			}
			res, err := tx.ExecContext(ctx,
				"INSERT INTO notes (title, content, created_at, updated_at, pinned, folder_id) VALUES (?, ?, ?, ?, ?, ?)",
				vn.Title, vn.Content, cs, us, vn.Pinned, folderArg)
			if err != nil {
				return stats, fmt.Errorf("insert note %q: %w", vn.Title, err)
			}
			id, _ = res.LastInsertId()
		}
		usedID[id] = true
		titleToID[vn.Title] = id
		titleCount[vn.Title]++
		refs = append(refs, noteRef{id: id, content: vn.Content})

		for _, tname := range vn.Tags {
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO tags (name) VALUES (?)", tname); err != nil {
				return stats, err
			}
			var tid int64
			if err := tx.QueryRowContext(ctx, "SELECT id FROM tags WHERE name = ?", tname).Scan(&tid); err != nil {
				return stats, err
			}
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)", id, tid); err != nil {
				return stats, err
			}
		}
	}

	for _, r := range refs {
		for _, lt := range rebuildLinkTitles(r.content) {
			if titleCount[lt] != 1 {
				continue
			}
			if tgt := titleToID[lt]; tgt != r.id {
				if _, err := tx.ExecContext(ctx,
					"INSERT INTO note_links (source_note_id, target_note_id, link_text) VALUES (?, ?, ?)",
					r.id, tgt, lt); err != nil {
					return stats, err
				}
				stats.Links++
			}
		}
	}

	// Re-insert the memory notes captured before the clear so they survive the rebuild.
	preservedMems, err := restoreMemories(ctx, tx, mems)
	if err != nil {
		return stats, fmt.Errorf("restore memories: %w", err)
	}
	stats.PreservedMemories = preservedMems

	// Restore version history from the vault into the freshly rebuilt index (idempotent).
	restored, err := restoreVersions(ctx, tx, vlt)
	if err != nil {
		return stats, err
	}

	if err := tx.Commit(); err != nil {
		return stats, err
	}
	stats.Notes = len(notes)
	stats.RestoredVersions = restored
	return stats, nil
}
