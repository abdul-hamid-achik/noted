package notesync

import (
	"context"
	"database/sql"
)

// memoryTag marks index-only agent memories (see internal/memory). They live in the notes table but
// are NOT mirrored to the vault, so a vault→index rebuild would otherwise delete them.
const memoryTag = "memory"

// preservedMemory is a full snapshot of a memory note captured before a rebuild clears the index.
type preservedMemory struct {
	id        int64
	title     string
	content   string
	created   sql.NullTime
	updated   sql.NullTime
	expires   sql.NullTime
	source    sql.NullString
	sourceRef sql.NullString
	pinned    bool
	tags      []string
}

// captureMemories reads all memory-tagged notes (with their tags) within the rebuild transaction,
// BEFORE the index is cleared, so they can be re-inserted afterward and survive the rebuild.
func captureMemories(ctx context.Context, tx *sql.Tx) ([]preservedMemory, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT n.id, n.title, n.content, n.created_at, n.updated_at, n.expires_at, n.source, n.source_ref, n.pinned
		FROM notes n
		JOIN note_tags nt ON nt.note_id = n.id
		JOIN tags t ON t.id = nt.tag_id
		WHERE t.name = ?`, memoryTag)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var mems []preservedMemory
	for rows.Next() {
		var m preservedMemory
		if err := rows.Scan(&m.id, &m.title, &m.content, &m.created, &m.updated, &m.expires, &m.source, &m.sourceRef, &m.pinned); err != nil {
			return nil, err
		}
		mems = append(mems, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Collect each memory's full tag set (separate pass so the rows cursor above is closed).
	for i := range mems {
		tagRows, err := tx.QueryContext(ctx, `
			SELECT t.name FROM tags t
			JOIN note_tags nt ON nt.tag_id = t.id
			WHERE nt.note_id = ?`, mems[i].id)
		if err != nil {
			return nil, err
		}
		for tagRows.Next() {
			var name string
			if err := tagRows.Scan(&name); err != nil {
				_ = tagRows.Close()
				return nil, err
			}
			mems[i].tags = append(mems[i].tags, name)
		}
		_ = tagRows.Close()
	}
	return mems, nil
}

// restoreMemories re-inserts captured memory notes after the index has been rebuilt from the vault.
// Each keeps its original id when that id is still free; if a rebuilt vault note already claimed it
// (only possible with a hand-edited duplicate id), the memory gets a fresh autoincrement id. Tags
// (memory / memory:<cat> / importance:N) and TTL/source metadata are restored; folder is dropped
// (memories aren't folder-organized, and the old folder id may no longer exist). Returns the count.
func restoreMemories(ctx context.Context, tx *sql.Tx, mems []preservedMemory) (int, error) {
	restored := 0
	for _, m := range mems {
		var taken int
		idFree := tx.QueryRowContext(ctx, "SELECT 1 FROM notes WHERE id = ?", m.id).Scan(&taken) == sql.ErrNoRows

		var newID int64
		if idFree {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO notes (id, title, content, created_at, updated_at, expires_at, source, source_ref, pinned) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
				m.id, m.title, m.content, m.created, m.updated, m.expires, m.source, m.sourceRef, m.pinned); err != nil {
				return restored, err
			}
			newID = m.id
		} else {
			res, err := tx.ExecContext(ctx,
				"INSERT INTO notes (title, content, created_at, updated_at, expires_at, source, source_ref, pinned) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
				m.title, m.content, m.created, m.updated, m.expires, m.source, m.sourceRef, m.pinned)
			if err != nil {
				return restored, err
			}
			newID, _ = res.LastInsertId()
		}

		for _, tname := range m.tags {
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO tags (name) VALUES (?)", tname); err != nil {
				return restored, err
			}
			var tid int64
			if err := tx.QueryRowContext(ctx, "SELECT id FROM tags WHERE name = ?", tname).Scan(&tid); err != nil {
				return restored, err
			}
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)", newID, tid); err != nil {
				return restored, err
			}
		}
		restored++
	}
	return restored, nil
}
