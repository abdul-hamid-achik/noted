package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
)

// Forget deletes memories matching the given criteria
func Forget(ctx context.Context, queries *db.Queries, syncer *veclite.Syncer, input ForgetInput) (*ForgetResult, error) {
	// Default to dry run for safety if no criteria specified
	dryRun := input.DryRun
	if !input.DryRun && input.OlderThanDays == 0 && input.ImportanceBelow == 0 &&
	   input.Category == "" && input.Query == "" && input.ID == 0 {
		dryRun = true
	}

	// If a specific ID is provided, just delete that one
	if input.ID != 0 {
		note, err := queries.GetNote(ctx, input.ID)
		if err != nil {
			return nil, fmt.Errorf("memory #%d not found", input.ID)
		}

		mem, ok := noteToMemory(ctx, queries, note)
		if !ok {
			return nil, fmt.Errorf("note #%d is not a memory", input.ID)
		}

		if dryRun {
			return &ForgetResult{
				DryRun:      true,
				WouldDelete: 1,
				Memories:    []Memory{mem},
				Criteria:    input,
			}, nil
		}

		// Delete from veclite if available
		if syncer != nil {
			_ = syncer.Delete(input.ID)
		}

		if err := queries.DeleteNote(ctx, input.ID); err != nil {
			return nil, fmt.Errorf("failed to delete memory: %w", err)
		}

		return &ForgetResult{
			DryRun:   false,
			Deleted:  1,
			Memories: []Memory{mem},
		}, nil
	}

	// Find memories matching criteria
	var memories []db.Note
	var err error

	if input.Query != "" {
		// Search by query
		pattern := "%" + input.Query + "%"
		memories, err = queries.SearchNotesContent(ctx, db.SearchNotesContentParams{
			Content: pattern,
			Title:   pattern,
			Limit:   1000, // Large limit for deletion
		})
	} else {
		// Get all memories
		memories, err = queries.GetNotesByTagName(ctx, "memory")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find memories: %w", err)
	}

	toDelete := make([]Memory, 0)
	now := time.Now()

	for _, note := range memories {
		mem, ok := noteToMemory(ctx, queries, note)
		if !ok {
			continue // Not a memory (for query-based search)
		}

		// Check age criteria
		if input.OlderThanDays > 0 && !mem.CreatedAt.IsZero() {
			age := now.Sub(mem.CreatedAt)
			if age.Hours()/24 < float64(input.OlderThanDays) {
				continue // Too recent
			}
		}

		// Check category filter
		if input.Category != "" && mem.Category != input.Category {
			continue
		}

		// Check importance criteria
		if input.ImportanceBelow > 0 && mem.Importance >= input.ImportanceBelow {
			continue // Too important
		}

		toDelete = append(toDelete, mem)
	}

	if dryRun {
		return &ForgetResult{
			DryRun:      true,
			WouldDelete: len(toDelete),
			Memories:    toDelete,
			Criteria:    input,
		}, nil
	}

	// Actually delete
	deleted := 0
	for _, mem := range toDelete {
		if syncer != nil {
			_ = syncer.Delete(mem.ID)
		}
		if err := queries.DeleteNote(ctx, mem.ID); err == nil {
			deleted++
		}
	}

	return &ForgetResult{
		DryRun:   false,
		Deleted:  deleted,
		Memories: toDelete,
	}, nil
}
