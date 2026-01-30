package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Input types for MCP tools
// The jsonschema tag provides the description for the JSON schema.
// Fields without omitempty in json tag are considered required.

type createInput struct {
	Title   string   `json:"title" jsonschema:"Note title"`
	Content string   `json:"content" jsonschema:"Note content"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Tags for categorization"`
}

type listInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"Max notes to return (default 20)"`
	Tag    string `json:"tag,omitempty" jsonschema:"Filter by tag name"`
	Offset int    `json:"offset,omitempty" jsonschema:"Pagination offset"`
}

type getInput struct {
	ID int64 `json:"id" jsonschema:"Note ID"`
}

type searchInput struct {
	Query string `json:"query" jsonschema:"Search query for title and content"`
	Limit int    `json:"limit,omitempty" jsonschema:"Max results (default 20)"`
}

type updateInput struct {
	ID      int64    `json:"id" jsonschema:"Note ID"`
	Title   string   `json:"title,omitempty" jsonschema:"New title (optional)"`
	Content string   `json:"content,omitempty" jsonschema:"New content (optional)"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Replace tags (optional)"`
}

type deleteInput struct {
	ID int64 `json:"id" jsonschema:"Note ID to delete"`
}

type emptyInput struct{}

type semanticSearchInput struct {
	Query string `json:"query" jsonschema:"Natural language query for semantic search"`
	Limit int    `json:"limit,omitempty" jsonschema:"Max results (default 10)"`
}

type rememberInput struct {
	Content    string `json:"content" jsonschema:"Memory content to store"`
	Title      string `json:"title,omitempty" jsonschema:"Short title for the memory"`
	Category   string `json:"category,omitempty" jsonschema:"Category: user-pref, project, decision, fact, or todo"`
	Importance int    `json:"importance,omitempty" jsonschema:"Importance level 1-5 (default 3)"`
}

type recallInput struct {
	Query    string `json:"query" jsonschema:"What to recall (semantic search query)"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Max results (default 5)"`
	Category string `json:"category,omitempty" jsonschema:"Filter by category"`
}

type forgetInput struct {
	OlderThanDays   int    `json:"older_than_days,omitempty" jsonschema:"Delete memories older than N days"`
	ImportanceBelow int    `json:"importance_below,omitempty" jsonschema:"Delete memories below this importance level (1-5)"`
	Category        string `json:"category,omitempty" jsonschema:"Only delete memories in this category"`
	DryRun          bool   `json:"dry_run,omitempty" jsonschema:"Preview what would be deleted without actually deleting (default true)"`
}

type syncInput struct {
	Force bool `json:"force,omitempty" jsonschema:"Re-sync all notes even if already synced"`
}

// Output types for formatted responses

type noteOutput struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

type tagOutput struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	NoteCount int64  `json:"note_count,omitempty"`
}

// registerTools registers all MCP tools
func (s *Server) registerTools() {
	// noted_create - Create a new note
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_create",
		Description: "Create a new note with title, content, and optional tags",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input createInput) (*mcp.CallToolResult, any, error) {
		return s.toolCreate(ctx, input)
	})

	// noted_list - List notes with optional tag filter
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_list",
		Description: "List notes with optional tag filter and pagination",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listInput) (*mcp.CallToolResult, any, error) {
		return s.toolList(ctx, input)
	})

	// noted_get - Get a note by ID
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_get",
		Description: "Get a note by its ID, including tags",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getInput) (*mcp.CallToolResult, any, error) {
		return s.toolGet(ctx, input)
	})

	// noted_search - Full-text search
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_search",
		Description: "Search notes by title and content using text matching",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input searchInput) (*mcp.CallToolResult, any, error) {
		return s.toolSearch(ctx, input)
	})

	// noted_update - Update a note
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_update",
		Description: "Update a note's title, content, or tags",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input updateInput) (*mcp.CallToolResult, any, error) {
		return s.toolUpdate(ctx, input)
	})

	// noted_delete - Delete a note
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_delete",
		Description: "Delete a note by ID",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input deleteInput) (*mcp.CallToolResult, any, error) {
		return s.toolDelete(ctx, input)
	})

	// noted_tags - List all tags with counts
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_tags",
		Description: "List all tags with their note counts",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input emptyInput) (*mcp.CallToolResult, any, error) {
		return s.toolTags(ctx)
	})

	// noted_semantic_search - Only register if veclite is available
	if s.HasSemanticSearch() {
		mcp.AddTool(s.server, &mcp.Tool{
			Name:        "noted_semantic_search",
			Description: "Search notes using semantic similarity (requires veclite)",
		}, func(ctx context.Context, req *mcp.CallToolRequest, input semanticSearchInput) (*mcp.CallToolResult, any, error) {
			return s.toolSemanticSearch(ctx, input)
		})
	}

	// noted_remember - Store a memory for later recall
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_remember",
		Description: "Store a memory for later recall. Memories are notes with special tags for categorization and importance.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input rememberInput) (*mcp.CallToolResult, any, error) {
		return s.toolRemember(ctx, input)
	})

	// noted_recall - Recall memories by query
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_recall",
		Description: "Recall relevant memories by query. Uses semantic search if available, otherwise falls back to keyword search.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input recallInput) (*mcp.CallToolResult, any, error) {
		return s.toolRecall(ctx, input)
	})

	// noted_forget - Delete old or low-importance memories
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_forget",
		Description: "Delete old or low-importance memories based on criteria. Use dry_run=true to preview deletions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input forgetInput) (*mcp.CallToolResult, any, error) {
		return s.toolForget(ctx, input)
	})

	// noted_sync - Sync notes to semantic search index
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_sync",
		Description: "Sync unembedded notes to the semantic search index (requires veclite)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input syncInput) (*mcp.CallToolResult, any, error) {
		return s.toolSync(ctx, input)
	})
}

// Tool implementations

func (s *Server) toolCreate(ctx context.Context, input createInput) (*mcp.CallToolResult, any, error) {
	if input.Title == "" {
		return errorResult("title is required")
	}
	if input.Content == "" {
		return errorResult("content is required")
	}

	// Create the note
	note, err := s.queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   input.Title,
		Content: input.Content,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create note: %v", err))
	}

	// Add tags if provided
	for _, tagName := range input.Tags {
		tag, err := s.queries.CreateTag(ctx, tagName)
		if err != nil {
			continue // Skip failed tags
		}
		_ = s.queries.AddTagToNote(ctx, db.AddTagToNoteParams{
			NoteID: note.ID,
			TagID:  tag.ID,
		})
	}

	// Sync to veclite if available
	if s.syncer != nil {
		_ = s.syncer.SyncNote(note.ID, note.Title, note.Content)
	}

	return textResult(map[string]any{
		"id":      note.ID,
		"title":   note.Title,
		"status":  "created",
		"tags":    input.Tags,
		"message": fmt.Sprintf("Note #%d created successfully", note.ID),
	})
}

func (s *Server) toolList(ctx context.Context, input listInput) (*mcp.CallToolResult, any, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	var notes []db.Note
	var err error

	if input.Tag != "" {
		// Filter by tag
		notes, err = s.queries.GetNotesByTagName(ctx, input.Tag)
	} else {
		// List all with pagination
		notes, err = s.queries.ListNotes(ctx, db.ListNotesParams{
			Limit:  int64(limit),
			Offset: int64(input.Offset),
		})
	}

	if err != nil {
		return errorResult(fmt.Sprintf("failed to list notes: %v", err))
	}

	// Format output
	output := make([]noteOutput, len(notes))
	for i, note := range notes {
		output[i] = formatNote(note)
		// Get tags for each note
		tags, err := s.queries.GetTagsForNote(ctx, note.ID)
		if err == nil {
			output[i].Tags = make([]string, len(tags))
			for j, t := range tags {
				output[i].Tags[j] = t.Name
			}
		}
	}

	return textResult(map[string]any{
		"count": len(output),
		"notes": output,
	})
}

func (s *Server) toolGet(ctx context.Context, input getInput) (*mcp.CallToolResult, any, error) {
	note, err := s.queries.GetNote(ctx, input.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("note #%d not found", input.ID))
		}
		return errorResult(fmt.Sprintf("failed to get note: %v", err))
	}

	output := formatNote(note)

	// Get tags
	tags, err := s.queries.GetTagsForNote(ctx, note.ID)
	if err == nil {
		output.Tags = make([]string, len(tags))
		for i, t := range tags {
			output.Tags[i] = t.Name
		}
	}

	return textResult(output)
}

func (s *Server) toolSearch(ctx context.Context, input searchInput) (*mcp.CallToolResult, any, error) {
	if input.Query == "" {
		return errorResult("query is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	// Use LIKE-based search
	pattern := "%" + input.Query + "%"
	notes, err := s.queries.SearchNotesContent(ctx, db.SearchNotesContentParams{
		Content: pattern,
		Title:   pattern,
		Limit:   int64(limit),
	})
	if err != nil {
		return errorResult(fmt.Sprintf("search failed: %v", err))
	}

	// Format output
	output := make([]noteOutput, len(notes))
	for i, note := range notes {
		output[i] = formatNote(note)
	}

	return textResult(map[string]any{
		"query": input.Query,
		"count": len(output),
		"notes": output,
	})
}

func (s *Server) toolUpdate(ctx context.Context, input updateInput) (*mcp.CallToolResult, any, error) {
	// Get existing note
	existing, err := s.queries.GetNote(ctx, input.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("note #%d not found", input.ID))
		}
		return errorResult(fmt.Sprintf("failed to get note: %v", err))
	}

	// Merge changes
	title := existing.Title
	content := existing.Content
	if input.Title != "" {
		title = input.Title
	}
	if input.Content != "" {
		content = input.Content
	}

	// Update the note
	note, err := s.queries.UpdateNote(ctx, db.UpdateNoteParams{
		ID:      input.ID,
		Title:   title,
		Content: content,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to update note: %v", err))
	}

	// Update tags if provided
	if input.Tags != nil {
		// Remove all existing tags
		_ = s.queries.RemoveAllTagsFromNote(ctx, note.ID)

		// Add new tags
		for _, tagName := range input.Tags {
			tag, err := s.queries.CreateTag(ctx, tagName)
			if err != nil {
				continue
			}
			_ = s.queries.AddTagToNote(ctx, db.AddTagToNoteParams{
				NoteID: note.ID,
				TagID:  tag.ID,
			})
		}
	}

	// Sync to veclite if available
	if s.syncer != nil {
		_ = s.syncer.SyncNote(note.ID, note.Title, note.Content)
	}

	return textResult(map[string]any{
		"id":      note.ID,
		"title":   note.Title,
		"status":  "updated",
		"message": fmt.Sprintf("Note #%d updated successfully", note.ID),
	})
}

func (s *Server) toolDelete(ctx context.Context, input deleteInput) (*mcp.CallToolResult, any, error) {
	// Verify note exists
	_, err := s.queries.GetNote(ctx, input.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("note #%d not found", input.ID))
		}
		return errorResult(fmt.Sprintf("failed to get note: %v", err))
	}

	// Delete from veclite first if available
	if s.syncer != nil {
		_ = s.syncer.Delete(input.ID)
	}

	// Delete the note (cascades to note_tags)
	err = s.queries.DeleteNote(ctx, input.ID)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to delete note: %v", err))
	}

	return textResult(map[string]any{
		"id":      input.ID,
		"status":  "deleted",
		"message": fmt.Sprintf("Note #%d deleted successfully", input.ID),
	})
}

func (s *Server) toolTags(ctx context.Context) (*mcp.CallToolResult, any, error) {
	tags, err := s.queries.GetTagsWithCount(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list tags: %v", err))
	}

	output := make([]tagOutput, len(tags))
	for i, t := range tags {
		output[i] = tagOutput{
			ID:        t.ID,
			Name:      t.Name,
			NoteCount: t.NoteCount,
		}
	}

	return textResult(map[string]any{
		"count": len(output),
		"tags":  output,
	})
}

func (s *Server) toolSemanticSearch(ctx context.Context, input semanticSearchInput) (*mcp.CallToolResult, any, error) {
	if s.syncer == nil {
		return errorResult("semantic search not available (veclite not configured)")
	}

	if input.Query == "" {
		return errorResult("query is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	results, err := s.syncer.Search(input.Query, limit)
	if err != nil {
		return errorResult(fmt.Sprintf("semantic search failed: %v", err))
	}

	// Enrich results with full note data
	output := make([]map[string]any, 0, len(results))
	for _, r := range results {
		note, err := s.queries.GetNote(ctx, r.NoteID)
		if err != nil {
			continue
		}

		// Get tags
		var tags []string
		tagList, err := s.queries.GetTagsForNote(ctx, note.ID)
		if err == nil {
			tags = make([]string, len(tagList))
			for i, t := range tagList {
				tags[i] = t.Name
			}
		}

		output = append(output, map[string]any{
			"id":      note.ID,
			"title":   note.Title,
			"content": note.Content,
			"tags":    tags,
			"score":   r.Score,
		})
	}

	return textResult(map[string]any{
		"query":   input.Query,
		"count":   len(output),
		"results": output,
	})
}

// Helper functions

func formatNote(note db.Note) noteOutput {
	out := noteOutput{
		ID:      note.ID,
		Title:   note.Title,
		Content: note.Content,
	}
	if note.CreatedAt.Valid {
		out.CreatedAt = note.CreatedAt.Time.Format(time.RFC3339)
	}
	if note.UpdatedAt.Valid {
		out.UpdatedAt = note.UpdatedAt.Time.Format(time.RFC3339)
	}
	return out
}

func textResult(data any) (*mcp.CallToolResult, any, error) {
	b, _ := json.MarshalIndent(data, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(b)},
		},
	}, nil, nil
}

func errorResult(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}, nil, nil
}

// Memory tool implementations

func (s *Server) toolRemember(ctx context.Context, input rememberInput) (*mcp.CallToolResult, any, error) {
	if input.Content == "" {
		return errorResult("content is required")
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
		// Use first 50 chars of content as title
		title = input.Content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
	}

	// Create the note
	note, err := s.queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   title,
		Content: input.Content,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create memory: %v", err))
	}

	// Add memory tags
	memoryTags := []string{
		"memory",
		"memory:" + category,
		fmt.Sprintf("importance:%d", importance),
	}

	for _, tagName := range memoryTags {
		tag, err := s.queries.CreateTag(ctx, tagName)
		if err != nil {
			continue
		}
		_ = s.queries.AddTagToNote(ctx, db.AddTagToNoteParams{
			NoteID: note.ID,
			TagID:  tag.ID,
		})
	}

	// Sync to veclite if available
	if s.syncer != nil {
		_ = s.syncer.SyncNote(note.ID, note.Title, note.Content)
	}

	return textResult(map[string]any{
		"id":         note.ID,
		"title":      title,
		"category":   category,
		"importance": importance,
		"status":     "remembered",
		"message":    fmt.Sprintf("Memory stored with ID #%d", note.ID),
	})
}

func (s *Server) toolRecall(ctx context.Context, input recallInput) (*mcp.CallToolResult, any, error) {
	if input.Query == "" {
		return errorResult("query is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}

	// Try semantic search first if available
	if s.syncer != nil {
		results, err := s.syncer.Search(input.Query, limit*2) // Get extra for filtering
		if err == nil && len(results) > 0 {
			// Filter to only memory notes and enrich with full data
			output := make([]map[string]any, 0, limit)
			for _, r := range results {
				if len(output) >= limit {
					break
				}

				note, err := s.queries.GetNote(ctx, r.NoteID)
				if err != nil {
					continue
				}

				// Get tags and check if it's a memory
				tags, err := s.queries.GetTagsForNote(ctx, note.ID)
				if err != nil {
					continue
				}

				isMemory := false
				var category string
				var importanceVal int
				tagNames := make([]string, len(tags))
				for i, t := range tags {
					tagNames[i] = t.Name
					if t.Name == "memory" {
						isMemory = true
					}
					if len(t.Name) > 7 && t.Name[:7] == "memory:" {
						category = t.Name[7:]
					}
					if len(t.Name) > 11 && t.Name[:11] == "importance:" {
						_, _ = fmt.Sscanf(t.Name[11:], "%d", &importanceVal)
					}
				}

				// Skip if category filter doesn't match
				if input.Category != "" && category != input.Category {
					continue
				}

				// Only include memories, but if no memories found, include all
				if isMemory {
					output = append(output, map[string]any{
						"id":         note.ID,
						"title":      note.Title,
						"content":    note.Content,
						"category":   category,
						"importance": importanceVal,
						"score":      r.Score,
						"tags":       tagNames,
					})
				}
			}

			if len(output) > 0 {
				return textResult(map[string]any{
					"query":    input.Query,
					"method":   "semantic",
					"count":    len(output),
					"memories": output,
				})
			}
		}
	}

	// Fallback to keyword search
	pattern := "%" + input.Query + "%"
	notes, err := s.queries.SearchNotesContent(ctx, db.SearchNotesContentParams{
		Content: pattern,
		Title:   pattern,
		Limit:   int64(limit * 2),
	})
	if err != nil {
		return errorResult(fmt.Sprintf("search failed: %v", err))
	}

	// Filter to memories only
	output := make([]map[string]any, 0, limit)
	for _, note := range notes {
		if len(output) >= limit {
			break
		}

		tags, err := s.queries.GetTagsForNote(ctx, note.ID)
		if err != nil {
			continue
		}

		isMemory := false
		var category string
		var importanceVal int
		tagNames := make([]string, len(tags))
		for i, t := range tags {
			tagNames[i] = t.Name
			if t.Name == "memory" {
				isMemory = true
			}
			if len(t.Name) > 7 && t.Name[:7] == "memory:" {
				category = t.Name[7:]
			}
			if len(t.Name) > 11 && t.Name[:11] == "importance:" {
				_, _ = fmt.Sscanf(t.Name[11:], "%d", &importanceVal)
			}
		}

		if !isMemory {
			continue
		}

		// Skip if category filter doesn't match
		if input.Category != "" && category != input.Category {
			continue
		}

		output = append(output, map[string]any{
			"id":         note.ID,
			"title":      note.Title,
			"content":    note.Content,
			"category":   category,
			"importance": importanceVal,
			"tags":       tagNames,
		})
	}

	return textResult(map[string]any{
		"query":    input.Query,
		"method":   "keyword",
		"count":    len(output),
		"memories": output,
	})
}

func (s *Server) toolForget(ctx context.Context, input forgetInput) (*mcp.CallToolResult, any, error) {
	// Default to dry run for safety
	dryRun := input.DryRun
	if !input.DryRun && input.OlderThanDays == 0 && input.ImportanceBelow == 0 && input.Category == "" {
		dryRun = true // Force dry run if no criteria specified
	}

	// Find memories matching criteria
	memories, err := s.queries.GetNotesByTagName(ctx, "memory")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to find memories: %v", err))
	}

	toDelete := make([]map[string]any, 0)
	now := time.Now()

	for _, note := range memories {
		// Check age criteria
		if input.OlderThanDays > 0 && note.CreatedAt.Valid {
			age := now.Sub(note.CreatedAt.Time)
			if age.Hours()/24 < float64(input.OlderThanDays) {
				continue // Too recent
			}
		}

		// Get tags to check importance and category
		tags, err := s.queries.GetTagsForNote(ctx, note.ID)
		if err != nil {
			continue
		}

		var category string
		var importanceVal int
		for _, t := range tags {
			if len(t.Name) > 7 && t.Name[:7] == "memory:" {
				category = t.Name[7:]
			}
			if len(t.Name) > 11 && t.Name[:11] == "importance:" {
				_, _ = fmt.Sscanf(t.Name[11:], "%d", &importanceVal)
			}
		}

		// Check category filter
		if input.Category != "" && category != input.Category {
			continue
		}

		// Check importance criteria
		if input.ImportanceBelow > 0 && importanceVal >= input.ImportanceBelow {
			continue // Too important
		}

		toDelete = append(toDelete, map[string]any{
			"id":         note.ID,
			"title":      note.Title,
			"category":   category,
			"importance": importanceVal,
		})
	}

	if dryRun {
		return textResult(map[string]any{
			"dry_run":     true,
			"would_delete": len(toDelete),
			"memories":    toDelete,
			"criteria": map[string]any{
				"older_than_days":   input.OlderThanDays,
				"importance_below":  input.ImportanceBelow,
				"category":          input.Category,
			},
		})
	}

	// Actually delete
	deleted := 0
	for _, mem := range toDelete {
		id := mem["id"].(int64)
		if s.syncer != nil {
			_ = s.syncer.Delete(id)
		}
		if err := s.queries.DeleteNote(ctx, id); err == nil {
			deleted++
		}
	}

	return textResult(map[string]any{
		"deleted":  deleted,
		"status":   "forgotten",
		"memories": toDelete,
	})
}

func (s *Server) toolSync(ctx context.Context, input syncInput) (*mcp.CallToolResult, any, error) {
	if s.syncer == nil {
		return errorResult("semantic search not available (veclite not configured)")
	}

	var notes []db.Note
	var err error

	if input.Force {
		notes, err = s.queries.GetAllNotes(ctx)
	} else {
		notes, err = s.queries.GetUnsynced(ctx)
	}

	if err != nil {
		return errorResult(fmt.Sprintf("failed to get notes: %v", err))
	}

	synced := 0
	failed := 0
	for _, note := range notes {
		err := s.syncer.SyncNote(note.ID, note.Title, note.Content)
		if err != nil {
			failed++
			continue
		}
		// Mark as synced in database
		_ = s.queries.MarkEmbeddingSynced(ctx, note.ID)
		synced++
	}

	return textResult(map[string]any{
		"synced":    synced,
		"failed":    failed,
		"total":     len(notes),
		"remaining": len(notes) - synced,
		"status":    "completed",
	})
}
