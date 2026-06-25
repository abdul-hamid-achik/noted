package notesync

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/vault"
)

// latestVersionNumber normalizes the COALESCE(MAX(version_number), 0) result that
// db.GetLatestVersionNumber returns as interface{} (int64 or float64 depending on the driver path).
func latestVersionNumber(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// SnapshotVersion saves a note's current (pre-edit) state as the next version number, so the
// history/diff/restore commands retain it. Call this BEFORE updating the note, passing the state you
// are about to replace — every edit path (CLI `edit`, `restore`, MCP, and the TUI editor) routes
// through here so versioning semantics stay identical. When vlt is non-nil the snapshot is also
// written to the vault (.noted/versions/) immediately, so history is durable the moment it's created,
// not only after the next export/rebuild. A nil dbq is a no-op.
func SnapshotVersion(ctx context.Context, dbq *db.Queries, vlt *vault.Vault, noteID int64, title, content string) error {
	if dbq == nil {
		return nil
	}
	// read-then-insert isn't atomic, so a concurrent snapshot for the same note can grab the same
	// version_number and trip the unique (note_id, version_number) index. Retry on that collision with
	// a freshly read number — SQLite serializes writers, so a bounded retry converges.
	const maxAttempts = 5
	var v db.NoteVersion
	for attempt := 1; ; attempt++ {
		latest, err := dbq.GetLatestVersionNumber(ctx, noteID)
		if err != nil {
			return err
		}
		v, err = dbq.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
			NoteID:        noteID,
			Title:         title,
			Content:       content,
			VersionNumber: latestVersionNumber(latest) + 1,
		})
		if err == nil {
			break
		}
		if attempt >= maxAttempts || !isUniqueViolation(err) {
			return err
		}
	}
	if vlt != nil {
		created := time.Time{}
		if v.CreatedAt.Valid {
			created = v.CreatedAt.Time
		}
		_, _ = vlt.WriteVersion(vault.Version{
			NoteID:        noteID,
			VersionNumber: v.VersionNumber,
			Title:         title,
			Created:       created,
			Content:       content,
		})
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

// PersistVersions writes every note's version snapshots from the SQLite index into the vault
// (.noted/versions/), so history becomes part of the durable, rebuildable vault rather than living
// only in the index. Snapshots are immutable — already-written files are skipped. Returns the number
// of new snapshot files written. Best-effort and a no-op when either side is nil.
func PersistVersions(ctx context.Context, dbq *db.Queries, vlt *vault.Vault) (int, error) {
	if dbq == nil || vlt == nil {
		return 0, nil
	}
	notes, err := dbq.GetAllNotes(ctx)
	if err != nil {
		return 0, err
	}
	written := 0
	for _, n := range notes {
		versions, err := dbq.GetNoteVersions(ctx, n.ID)
		if err != nil {
			continue
		}
		for _, v := range versions {
			created := time.Time{}
			if v.CreatedAt.Valid {
				created = v.CreatedAt.Time
			}
			ok, err := vlt.WriteVersion(vault.Version{
				NoteID:        v.NoteID,
				VersionNumber: v.VersionNumber,
				Title:         v.Title,
				Created:       created,
				Content:       v.Content,
			})
			if err != nil {
				continue
			}
			if ok {
				written++
			}
		}
	}
	return written, nil
}

// restoreVersions reloads version snapshots from the vault into note_versions within an open rebuild
// transaction. It is idempotent (INSERT OR IGNORE on the unique (note_id, version_number) index) and
// skips snapshots whose note no longer exists in the rebuilt index. Returns the number inserted.
func restoreVersions(ctx context.Context, tx *sql.Tx, vlt *vault.Vault) (int, error) {
	if vlt == nil {
		return 0, nil
	}
	versions, err := vlt.AllVersions()
	if err != nil {
		return 0, err
	}
	restored := 0
	for _, v := range versions {
		if v.NoteID <= 0 || v.VersionNumber <= 0 {
			continue
		}
		// Skip orphans: a snapshot whose note isn't in the rebuilt index would violate the FK.
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT 1 FROM notes WHERE id = ?", v.NoteID).Scan(&exists); err != nil {
			continue
		}
		var createdArg any
		if !v.Created.IsZero() {
			createdArg = v.Created.UTC().Format("2006-01-02 15:04:05")
		}
		res, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO note_versions (note_id, version_number, title, content, created_at) VALUES (?, ?, ?, ?, ?)",
			v.NoteID, v.VersionNumber, v.Title, v.Content, createdArg)
		if err != nil {
			return restored, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			restored++
		}
	}
	return restored, nil
}
