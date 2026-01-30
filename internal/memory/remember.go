package memory

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
)

// Remember creates a new memory with the given input
func Remember(ctx context.Context, queries *db.Queries, syncer *veclite.Syncer, input RememberInput) (*Memory, error) {
	if input.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	// Default importance to 3
	importance := input.Importance
	if importance < 1 || importance > 5 {
		importance = 3
	}

	// Default category
	category := input.Category
	if category == "" {
		category = "fact"
	}

	// Generate title if not provided
	title := input.Title
	if title == "" {
		title = input.Content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
	}

	// Calculate expires_at if TTL is provided
	var expiresAt sql.NullTime
	if input.TTL > 0 {
		expiresAt = sql.NullTime{
			Time:  time.Now().Add(input.TTL),
			Valid: true,
		}
	}

	// Create source values
	var source, sourceRef sql.NullString
	if input.Source != "" {
		source = sql.NullString{String: input.Source, Valid: true}
	}
	if input.SourceRef != "" {
		sourceRef = sql.NullString{String: input.SourceRef, Valid: true}
	}

	// Create the note
	note, err := queries.CreateNoteWithTTL(ctx, db.CreateNoteWithTTLParams{
		Title:     title,
		Content:   input.Content,
		ExpiresAt: expiresAt,
		Source:    source,
		SourceRef: sourceRef,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	// Add memory tags
	memoryTags := []string{
		"memory",
		"memory:" + category,
		fmt.Sprintf("importance:%d", importance),
	}

	for _, tagName := range memoryTags {
		tag, err := queries.CreateTag(ctx, tagName)
		if err != nil {
			continue
		}
		_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{
			NoteID: note.ID,
			TagID:  tag.ID,
		})
	}

	// Sync to veclite if available
	if syncer != nil {
		_ = syncer.SyncNote(note.ID, note.Title, note.Content)
	}

	mem := &Memory{
		ID:         note.ID,
		Title:      title,
		Content:    input.Content,
		Category:   category,
		Importance: importance,
		Tags:       memoryTags,
	}

	if note.CreatedAt.Valid {
		mem.CreatedAt = note.CreatedAt.Time
	}
	if note.UpdatedAt.Valid {
		mem.UpdatedAt = note.UpdatedAt.Time
	}
	if expiresAt.Valid {
		mem.ExpiresAt = expiresAt.Time
	}
	if source.Valid {
		mem.Source = source.String
	}
	if sourceRef.Valid {
		mem.SourceRef = sourceRef.String
	}

	return mem, nil
}
