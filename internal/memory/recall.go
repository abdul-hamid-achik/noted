package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
)

// Recall searches for memories matching the query
func Recall(ctx context.Context, queries *db.Queries, syncer *veclite.Syncer, input RecallInput) (*RecallResult, error) {
	if input.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}

	// Clean up expired notes first (lazy cleanup)
	_, _ = queries.DeleteExpiredNotes(ctx)

	// Try semantic search first if available and requested
	if syncer != nil && input.UseSemantic {
		results, err := syncer.Search(input.Query, limit*2) // Get extra for filtering
		if err == nil && len(results) > 0 {
			memories, err := filterMemoryResults(ctx, queries, results, input.Category, limit)
			if err == nil && len(memories) > 0 {
				return &RecallResult{
					Query:    input.Query,
					Method:   "semantic",
					Count:    len(memories),
					Memories: memories,
				}, nil
			}
		}
	}

	// Fallback to keyword search
	pattern := "%" + input.Query + "%"
	notes, err := queries.SearchNotesContent(ctx, db.SearchNotesContentParams{
		Content: pattern,
		Title:   pattern,
		Limit:   int64(limit * 2),
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Filter to memories only
	memories := make([]Memory, 0, limit)
	for _, note := range notes {
		if len(memories) >= limit {
			break
		}

		mem, ok := noteToMemory(ctx, queries, note)
		if !ok {
			continue // Not a memory
		}

		// Skip if category filter doesn't match
		if input.Category != "" && mem.Category != input.Category {
			continue
		}

		memories = append(memories, mem)
	}

	return &RecallResult{
		Query:    input.Query,
		Method:   "keyword",
		Count:    len(memories),
		Memories: memories,
	}, nil
}

// filterMemoryResults filters semantic search results to only include memories
func filterMemoryResults(ctx context.Context, queries *db.Queries, results []veclite.SemanticResult, category string, limit int) ([]Memory, error) {
	memories := make([]Memory, 0, limit)

	for _, r := range results {
		if len(memories) >= limit {
			break
		}

		note, err := queries.GetNote(ctx, r.NoteID)
		if err != nil {
			continue
		}

		mem, ok := noteToMemory(ctx, queries, note)
		if !ok {
			continue // Not a memory
		}

		// Skip if category filter doesn't match
		if category != "" && mem.Category != category {
			continue
		}

		mem.Score = r.Score
		memories = append(memories, mem)
	}

	return memories, nil
}

// noteToMemory converts a note to a memory if it has the memory tag
func noteToMemory(ctx context.Context, queries *db.Queries, note db.Note) (Memory, bool) {
	tags, err := queries.GetTagsForNote(ctx, note.ID)
	if err != nil {
		return Memory{}, false
	}

	isMemory := false
	var category string
	var importance int
	tagNames := make([]string, len(tags))

	for i, t := range tags {
		tagNames[i] = t.Name
		if t.Name == "memory" {
			isMemory = true
		}
		if strings.HasPrefix(t.Name, "memory:") {
			category = t.Name[7:]
		}
		if strings.HasPrefix(t.Name, "importance:") {
			_, _ = fmt.Sscanf(t.Name[11:], "%d", &importance)
		}
	}

	if !isMemory {
		return Memory{}, false
	}

	mem := Memory{
		ID:         note.ID,
		Title:      note.Title,
		Content:    note.Content,
		Category:   category,
		Importance: importance,
		Tags:       tagNames,
	}

	if note.CreatedAt.Valid {
		mem.CreatedAt = note.CreatedAt.Time
	}
	if note.UpdatedAt.Valid {
		mem.UpdatedAt = note.UpdatedAt.Time
	}
	if note.ExpiresAt.Valid {
		mem.ExpiresAt = note.ExpiresAt.Time
	}
	if note.Source.Valid {
		mem.Source = note.Source.String
	}
	if note.SourceRef.Valid {
		mem.SourceRef = note.SourceRef.String
	}

	return mem, true
}
