package db

import (
	"context"
	"database/sql"
)

// FTSAvailable checks if the notes_fts table exists
func FTSAvailable(ctx context.Context, db *sql.DB) bool {
	var name string
	err := db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='notes_fts'",
	).Scan(&name)
	return err == nil && name == "notes_fts"
}

// SearchNotesFTS performs full-text search using FTS5
func SearchNotesFTS(ctx context.Context, db *sql.DB, query string, limit int64) ([]Note, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT n.id, n.title, n.content, n.created_at, n.updated_at,
		       n.embedding_synced, n.expires_at, n.source, n.source_ref,
		       n.folder_id, n.pinned, n.pinned_at
		FROM notes_fts fts
		JOIN notes n ON n.id = fts.rowid
		WHERE notes_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(
			&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt,
			&n.EmbeddingSynced, &n.ExpiresAt, &n.Source, &n.SourceRef,
			&n.FolderID, &n.Pinned, &n.PinnedAt,
		); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
