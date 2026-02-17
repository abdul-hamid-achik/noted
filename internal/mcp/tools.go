package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/memory"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var taskRegex = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(.+)$`)

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
	TTL        string `json:"ttl,omitempty" jsonschema:"Time-to-live duration (e.g., '24h', '7d')"`
	Source     string `json:"source,omitempty" jsonschema:"Source identifier (e.g., 'code-review', 'manual')"`
	SourceRef  string `json:"source_ref,omitempty" jsonschema:"Source reference (e.g., 'main.go:50')"`
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

// Daily notes input types

type dailyInput struct {
	Date    string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format (default: today)"`
	Append  string `json:"append,omitempty" jsonschema:"Text to append to the daily note"`
	Prepend string `json:"prepend,omitempty" jsonschema:"Text to prepend to the daily note"`
}

type dailyListInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Max results (default 30)"`
}

// Template input types

type templateListInput struct{}

type templateCreateInput struct {
	Name    string `json:"name" jsonschema:"Template name (unique)"`
	Content string `json:"content" jsonschema:"Template content with optional variables: {{date}}, {{time}}, {{datetime}}, {{title}}"`
}

type templateGetInput struct {
	Name string `json:"name" jsonschema:"Template name to retrieve"`
}

type templateDeleteInput struct {
	Name string `json:"name" jsonschema:"Template name to delete"`
}

type templateApplyInput struct {
	TemplateName string   `json:"template_name" jsonschema:"Name of the template to apply"`
	Title        string   `json:"title" jsonschema:"Title for the new note"`
	Tags         []string `json:"tags,omitempty" jsonschema:"Tags for the new note"`
}

// Task extraction input types

type tasksInput struct {
	NoteID    int64  `json:"note_id,omitempty" jsonschema:"Filter tasks from a specific note"`
	Tag       string `json:"tag,omitempty" jsonschema:"Filter tasks from notes with this tag"`
	Pending   bool   `json:"pending,omitempty" jsonschema:"Show only pending tasks"`
	Completed bool   `json:"completed,omitempty" jsonschema:"Show only completed tasks"`
}

// History/versioning input types

type historyInput struct {
	NoteID int64 `json:"note_id" jsonschema:"Note ID to get history for"`
}

type versionGetInput struct {
	NoteID  int64 `json:"note_id" jsonschema:"Note ID"`
	Version int64 `json:"version" jsonschema:"Version number to retrieve"`
}

type restoreInput struct {
	NoteID  int64 `json:"note_id" jsonschema:"Note ID to restore"`
	Version int64 `json:"version" jsonschema:"Version number to restore to"`
}

// Random note input

type randomInput struct {
	Tag string `json:"tag,omitempty" jsonschema:"Pick from notes with this tag"`
}

// Link health input types

type backlinksInput struct {
	NoteID int64 `json:"note_id" jsonschema:"Note ID to find backlinks for"`
}

type linkHealthInput struct{}

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

	// --- Daily Notes ---

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_daily",
		Description: "Get or create today's daily note. Optionally append or prepend content.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input dailyInput) (*mcp.CallToolResult, any, error) {
		return s.toolDaily(ctx, input)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_daily_list",
		Description: "List recent daily notes (last 30 days by default)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input dailyListInput) (*mcp.CallToolResult, any, error) {
		return s.toolDailyList(ctx, input)
	})

	// --- Templates ---

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_template_list",
		Description: "List all note templates",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input templateListInput) (*mcp.CallToolResult, any, error) {
		return s.toolTemplateList(ctx)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_template_create",
		Description: "Create a new note template. Supports variables: {{date}}, {{time}}, {{datetime}}, {{title}}",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input templateCreateInput) (*mcp.CallToolResult, any, error) {
		return s.toolTemplateCreate(ctx, input)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_template_get",
		Description: "Get a template by name",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input templateGetInput) (*mcp.CallToolResult, any, error) {
		return s.toolTemplateGet(ctx, input)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_template_delete",
		Description: "Delete a template by name",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input templateDeleteInput) (*mcp.CallToolResult, any, error) {
		return s.toolTemplateDelete(ctx, input)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_template_apply",
		Description: "Apply a template to create a new note. Variables like {{date}}, {{title}} are interpolated.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input templateApplyInput) (*mcp.CallToolResult, any, error) {
		return s.toolTemplateApply(ctx, input)
	})

	// --- Task Extraction ---

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_tasks",
		Description: "Extract markdown tasks (checkboxes) from notes. Filter by note, tag, or completion status.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input tasksInput) (*mcp.CallToolResult, any, error) {
		return s.toolTasks(ctx, input)
	})

	// --- Version History ---

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_history",
		Description: "List version history for a note",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input historyInput) (*mcp.CallToolResult, any, error) {
		return s.toolHistory(ctx, input)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_version_get",
		Description: "Get a specific version of a note",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input versionGetInput) (*mcp.CallToolResult, any, error) {
		return s.toolVersionGet(ctx, input)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_restore",
		Description: "Restore a note to a previous version. Saves current state as a new version first.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input restoreInput) (*mcp.CallToolResult, any, error) {
		return s.toolRestore(ctx, input)
	})

	// --- Random Note ---

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_random",
		Description: "Get a random note, optionally filtered by tag",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input randomInput) (*mcp.CallToolResult, any, error) {
		return s.toolRandom(ctx, input)
	})

	// --- Link Health ---

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_backlinks",
		Description: "Get all notes that link to a given note (backlinks/incoming links)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input backlinksInput) (*mcp.CallToolResult, any, error) {
		return s.toolBacklinks(ctx, input)
	})

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "noted_orphans",
		Description: "Find orphan notes (no incoming or outgoing links) and dead-end notes (incoming but no outgoing)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input linkHealthInput) (*mcp.CallToolResult, any, error) {
		return s.toolOrphans(ctx)
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

	// Try FTS5 first, fall back to LIKE
	var notes []db.Note
	var err error
	if db.FTSAvailable(ctx, s.conn) {
		notes, err = db.SearchNotesFTS(ctx, s.conn, input.Query, int64(limit))
	}
	if notes == nil || err != nil {
		pattern := "%" + input.Query + "%"
		notes, err = s.queries.SearchNotesContent(ctx, db.SearchNotesContentParams{
			Content: pattern,
			Title:   pattern,
			Limit:   int64(limit),
		})
	}
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

	// Parse TTL if provided
	var ttl time.Duration
	if input.TTL != "" {
		var err error
		ttl, err = parseDuration(input.TTL)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid TTL: %v", err))
		}
	}

	// Get veclite syncer (may be nil)
	var syncer *veclite.Syncer
	if s.syncer != nil {
		if vs, ok := s.syncer.(*veclite.Syncer); ok {
			syncer = vs
		}
	}

	mem, err := memory.Remember(ctx, s.queries, syncer, memory.RememberInput{
		Content:    input.Content,
		Title:      input.Title,
		Category:   input.Category,
		Importance: input.Importance,
		TTL:        ttl,
		Source:     input.Source,
		SourceRef:  input.SourceRef,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create memory: %v", err))
	}

	result := map[string]any{
		"id":         mem.ID,
		"title":      mem.Title,
		"category":   mem.Category,
		"importance": mem.Importance,
		"status":     "remembered",
		"message":    fmt.Sprintf("Memory stored with ID #%d", mem.ID),
	}

	if !mem.ExpiresAt.IsZero() {
		result["expires_at"] = mem.ExpiresAt.Format(time.RFC3339)
	}
	if mem.Source != "" {
		result["source"] = mem.Source
	}
	if mem.SourceRef != "" {
		result["source_ref"] = mem.SourceRef
	}

	return textResult(result)
}

// parseDuration parses a duration string with support for days (e.g., "7d", "24h")
func parseDuration(s string) (time.Duration, error) {
	// Check for day suffix
	if len(s) > 0 && s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		var n int
		if _, err := fmt.Sscanf(days, "%d", &n); err != nil {
			return 0, fmt.Errorf("invalid day format: %s", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func (s *Server) toolRecall(ctx context.Context, input recallInput) (*mcp.CallToolResult, any, error) {
	if input.Query == "" {
		return errorResult("query is required")
	}

	// Get veclite syncer (may be nil)
	var syncer *veclite.Syncer
	if s.syncer != nil {
		if vs, ok := s.syncer.(*veclite.Syncer); ok {
			syncer = vs
		}
	}

	result, err := memory.Recall(ctx, s.queries, s.conn, syncer, memory.RecallInput{
		Query:       input.Query,
		Limit:       input.Limit,
		Category:    input.Category,
		UseSemantic: syncer != nil, // Use semantic search if available
	})
	if err != nil {
		return errorResult(fmt.Sprintf("recall failed: %v", err))
	}

	// Convert memories to output format
	output := make([]map[string]any, len(result.Memories))
	for i, mem := range result.Memories {
		m := map[string]any{
			"id":         mem.ID,
			"title":      mem.Title,
			"content":    mem.Content,
			"category":   mem.Category,
			"importance": mem.Importance,
			"tags":       mem.Tags,
		}
		if mem.Score > 0 {
			m["score"] = mem.Score
		}
		if !mem.ExpiresAt.IsZero() {
			m["expires_at"] = mem.ExpiresAt.Format(time.RFC3339)
		}
		if mem.Source != "" {
			m["source"] = mem.Source
		}
		if mem.SourceRef != "" {
			m["source_ref"] = mem.SourceRef
		}
		output[i] = m
	}

	return textResult(map[string]any{
		"query":    result.Query,
		"method":   result.Method,
		"count":    result.Count,
		"memories": output,
	})
}

func (s *Server) toolForget(ctx context.Context, input forgetInput) (*mcp.CallToolResult, any, error) {
	// Get veclite syncer (may be nil)
	var syncer *veclite.Syncer
	if s.syncer != nil {
		if vs, ok := s.syncer.(*veclite.Syncer); ok {
			syncer = vs
		}
	}

	result, err := memory.Forget(ctx, s.queries, syncer, memory.ForgetInput{
		OlderThanDays:   input.OlderThanDays,
		ImportanceBelow: input.ImportanceBelow,
		Category:        input.Category,
		DryRun:          input.DryRun,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("forget failed: %v", err))
	}

	// Convert memories to output format
	output := make([]map[string]any, len(result.Memories))
	for i, mem := range result.Memories {
		output[i] = map[string]any{
			"id":         mem.ID,
			"title":      mem.Title,
			"category":   mem.Category,
			"importance": mem.Importance,
		}
	}

	if result.DryRun {
		return textResult(map[string]any{
			"dry_run":      true,
			"would_delete": result.WouldDelete,
			"memories":     output,
			"criteria": map[string]any{
				"older_than_days":  input.OlderThanDays,
				"importance_below": input.ImportanceBelow,
				"category":         input.Category,
			},
		})
	}

	return textResult(map[string]any{
		"deleted":  result.Deleted,
		"status":   "forgotten",
		"memories": output,
	})
}

// --- Daily Notes tool implementations ---

func (s *Server) toolDaily(ctx context.Context, input dailyInput) (*mcp.CallToolResult, any, error) {
	targetDate := time.Now()
	if input.Date != "" {
		parsed, err := time.Parse("2006-01-02", input.Date)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid date format (use YYYY-MM-DD): %v", err))
		}
		targetDate = parsed
	}

	title := targetDate.Format("2006-01-02")

	// Try to get existing daily note
	note, err := s.queries.GetNoteByTitle(ctx, title)
	if err != nil {
		if err != sql.ErrNoRows {
			return errorResult(fmt.Sprintf("failed to look up daily note: %v", err))
		}
		// Create new daily note
		note, err = s.queries.CreateNoteWithTTL(ctx, db.CreateNoteWithTTLParams{
			Title:   title,
			Content: "",
		})
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create daily note: %v", err))
		}
		// Tag as "daily"
		tag, err := s.queries.CreateTag(ctx, "daily")
		if err == nil {
			_ = s.queries.AddTagToNote(ctx, db.AddTagToNoteParams{
				NoteID: note.ID,
				TagID:  tag.ID,
			})
		}
		// Find or create "Daily Notes" folder
		folders, _ := s.queries.ListFolders(ctx)
		var folderID int64
		for _, f := range folders {
			if f.Name == "Daily Notes" {
				folderID = f.ID
				break
			}
		}
		if folderID == 0 {
			folder, err := s.queries.CreateFolder(ctx, db.CreateFolderParams{Name: "Daily Notes"})
			if err == nil {
				folderID = folder.ID
			}
		}
		if folderID > 0 {
			_ = s.queries.MoveNoteToFolder(ctx, db.MoveNoteToFolderParams{
				FolderID: sql.NullInt64{Int64: folderID, Valid: true},
				ID:       note.ID,
			})
		}
	}

	// Append content if requested
	if input.Append != "" {
		content := note.Content
		if content != "" && len(content) > 0 && content[len(content)-1] != '\n' {
			content += "\n"
		}
		content += input.Append
		note, err = s.queries.UpdateNote(ctx, db.UpdateNoteParams{
			Title:   note.Title,
			Content: content,
			ID:      note.ID,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("failed to append: %v", err))
		}
	}

	// Prepend content if requested
	if input.Prepend != "" {
		content := input.Prepend
		if note.Content != "" {
			content += "\n" + note.Content
		}
		note, err = s.queries.UpdateNote(ctx, db.UpdateNoteParams{
			Title:   note.Title,
			Content: content,
			ID:      note.ID,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("failed to prepend: %v", err))
		}
	}

	out := formatNote(note)
	tags, _ := s.queries.GetTagsForNote(ctx, note.ID)
	out.Tags = make([]string, len(tags))
	for i, t := range tags {
		out.Tags[i] = t.Name
	}

	return textResult(out)
}

func (s *Server) toolDailyList(ctx context.Context, input dailyListInput) (*mcp.CallToolResult, any, error) {
	notes, err := s.queries.GetNotesByTagName(ctx, "daily")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list daily notes: %v", err))
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 30
	}

	cutoff := time.Now().AddDate(0, 0, -limit)
	var output []noteOutput
	for _, n := range notes {
		if n.CreatedAt.Valid && n.CreatedAt.Time.After(cutoff) {
			output = append(output, formatNote(n))
		}
	}

	return textResult(map[string]any{
		"count": len(output),
		"notes": output,
	})
}

// --- Template tool implementations ---

func (s *Server) toolTemplateList(ctx context.Context) (*mcp.CallToolResult, any, error) {
	templates, err := s.queries.ListTemplates(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list templates: %v", err))
	}

	type tmplOut struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	output := make([]tmplOut, len(templates))
	for i, t := range templates {
		output[i] = tmplOut{ID: t.ID, Name: t.Name}
	}

	return textResult(map[string]any{
		"count":     len(output),
		"templates": output,
	})
}

func (s *Server) toolTemplateCreate(ctx context.Context, input templateCreateInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return errorResult("name is required")
	}
	if input.Content == "" {
		return errorResult("content is required")
	}

	tmpl, err := s.queries.CreateTemplate(ctx, db.CreateTemplateParams{
		Name:    input.Name,
		Content: input.Content,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create template: %v", err))
	}

	return textResult(map[string]any{
		"id":      tmpl.ID,
		"name":    tmpl.Name,
		"status":  "created",
		"message": fmt.Sprintf("Template %q created", tmpl.Name),
	})
}

func (s *Server) toolTemplateGet(ctx context.Context, input templateGetInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return errorResult("name is required")
	}

	tmpl, err := s.queries.GetTemplateByName(ctx, input.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("template %q not found", input.Name))
		}
		return errorResult(fmt.Sprintf("failed to get template: %v", err))
	}

	return textResult(map[string]any{
		"id":      tmpl.ID,
		"name":    tmpl.Name,
		"content": tmpl.Content,
	})
}

func (s *Server) toolTemplateDelete(ctx context.Context, input templateDeleteInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return errorResult("name is required")
	}

	tmpl, err := s.queries.GetTemplateByName(ctx, input.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("template %q not found", input.Name))
		}
		return errorResult(fmt.Sprintf("failed to get template: %v", err))
	}

	if err := s.queries.DeleteTemplateByName(ctx, input.Name); err != nil {
		return errorResult(fmt.Sprintf("failed to delete template: %v", err))
	}

	return textResult(map[string]any{
		"id":      tmpl.ID,
		"name":    tmpl.Name,
		"status":  "deleted",
		"message": fmt.Sprintf("Template %q deleted", tmpl.Name),
	})
}

func (s *Server) toolTemplateApply(ctx context.Context, input templateApplyInput) (*mcp.CallToolResult, any, error) {
	if input.TemplateName == "" {
		return errorResult("template_name is required")
	}
	if input.Title == "" {
		return errorResult("title is required")
	}

	tmpl, err := s.queries.GetTemplateByName(ctx, input.TemplateName)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("template %q not found", input.TemplateName))
		}
		return errorResult(fmt.Sprintf("failed to get template: %v", err))
	}

	// Interpolate template variables
	content := interpolateTemplate(tmpl.Content, input.Title)

	note, err := s.queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   input.Title,
		Content: content,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create note: %v", err))
	}

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

	if s.syncer != nil {
		_ = s.syncer.SyncNote(note.ID, note.Title, note.Content)
	}

	return textResult(map[string]any{
		"id":       note.ID,
		"title":    note.Title,
		"template": input.TemplateName,
		"status":   "created",
		"message":  fmt.Sprintf("Note #%d created from template %q", note.ID, input.TemplateName),
	})
}

// interpolateTemplate replaces template variables with actual values
func interpolateTemplate(content, title string) string {
	now := time.Now()
	r := strings.NewReplacer(
		"{{date}}", now.Format("2006-01-02"),
		"{{time}}", now.Format("15:04"),
		"{{datetime}}", now.Format("2006-01-02 15:04"),
		"{{title}}", title,
	)
	return r.Replace(content)
}

// --- Task extraction tool implementation ---

func (s *Server) toolTasks(ctx context.Context, input tasksInput) (*mcp.CallToolResult, any, error) {
	var notes []db.Note
	var err error

	if input.NoteID > 0 {
		note, err := s.queries.GetNote(ctx, input.NoteID)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResult(fmt.Sprintf("note #%d not found", input.NoteID))
			}
			return errorResult(fmt.Sprintf("failed to get note: %v", err))
		}
		notes = []db.Note{note}
	} else if input.Tag != "" {
		notes, err = s.queries.GetNotesByTagName(ctx, input.Tag)
	} else {
		notes, err = s.queries.GetAllNotes(ctx)
	}
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get notes: %v", err))
	}

	type taskItem struct {
		Text      string `json:"text"`
		Completed bool   `json:"completed"`
		NoteID    int64  `json:"note_id"`
		NoteTitle string `json:"note_title"`
		Line      int    `json:"line"`
	}

	var tasks []taskItem
	for _, note := range notes {
		lines := strings.Split(note.Content, "\n")
		for i, line := range lines {
			matches := taskRegex.FindStringSubmatch(line)
			if matches == nil {
				continue
			}
			completed := matches[1] != " "
			if input.Pending && completed {
				continue
			}
			if input.Completed && !completed {
				continue
			}
			tasks = append(tasks, taskItem{
				Text:      strings.TrimSpace(matches[2]),
				Completed: completed,
				NoteID:    note.ID,
				NoteTitle: note.Title,
				Line:      i + 1,
			})
		}
	}

	if tasks == nil {
		tasks = []taskItem{}
	}

	pendingCount := 0
	completedCount := 0
	for _, t := range tasks {
		if t.Completed {
			completedCount++
		} else {
			pendingCount++
		}
	}

	return textResult(map[string]any{
		"tasks":     tasks,
		"pending":   pendingCount,
		"completed": completedCount,
		"total":     len(tasks),
	})
}

// --- Version history tool implementations ---

func (s *Server) toolHistory(ctx context.Context, input historyInput) (*mcp.CallToolResult, any, error) {
	_, err := s.queries.GetNote(ctx, input.NoteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("note #%d not found", input.NoteID))
		}
		return errorResult(fmt.Sprintf("failed to get note: %v", err))
	}

	versions, err := s.queries.GetNoteVersions(ctx, input.NoteID)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get versions: %v", err))
	}

	type versionItem struct {
		VersionNumber int64  `json:"version_number"`
		Title         string `json:"title"`
		CreatedAt     string `json:"created_at"`
	}

	output := make([]versionItem, len(versions))
	for i, v := range versions {
		output[i] = versionItem{
			VersionNumber: v.VersionNumber,
			Title:         v.Title,
		}
		if v.CreatedAt.Valid {
			output[i].CreatedAt = v.CreatedAt.Time.Format(time.RFC3339)
		}
	}

	return textResult(map[string]any{
		"note_id":  input.NoteID,
		"count":    len(output),
		"versions": output,
	})
}

func (s *Server) toolVersionGet(ctx context.Context, input versionGetInput) (*mcp.CallToolResult, any, error) {
	version, err := s.queries.GetNoteVersion(ctx, db.GetNoteVersionParams{
		NoteID:        input.NoteID,
		VersionNumber: input.Version,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("version %d not found for note #%d", input.Version, input.NoteID))
		}
		return errorResult(fmt.Sprintf("failed to get version: %v", err))
	}

	result := map[string]any{
		"note_id":        input.NoteID,
		"version_number": version.VersionNumber,
		"title":          version.Title,
		"content":        version.Content,
	}
	if version.CreatedAt.Valid {
		result["created_at"] = version.CreatedAt.Time.Format(time.RFC3339)
	}

	return textResult(result)
}

func (s *Server) toolRestore(ctx context.Context, input restoreInput) (*mcp.CallToolResult, any, error) {
	note, err := s.queries.GetNote(ctx, input.NoteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("note #%d not found", input.NoteID))
		}
		return errorResult(fmt.Sprintf("failed to get note: %v", err))
	}

	version, err := s.queries.GetNoteVersion(ctx, db.GetNoteVersionParams{
		NoteID:        input.NoteID,
		VersionNumber: input.Version,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("version %d not found for note #%d", input.Version, input.NoteID))
		}
		return errorResult(fmt.Sprintf("failed to get version: %v", err))
	}

	// Save current state as a new version before restoring
	latestVer, err := s.queries.GetLatestVersionNumber(ctx, input.NoteID)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get latest version: %v", err))
	}
	var latestVerNum int64
	switch v := latestVer.(type) {
	case int64:
		latestVerNum = v
	case float64:
		latestVerNum = int64(v)
	}

	_, err = s.queries.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID:        input.NoteID,
		Title:         note.Title,
		Content:       note.Content,
		VersionNumber: latestVerNum + 1,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to save current state: %v", err))
	}

	_, err = s.queries.UpdateNote(ctx, db.UpdateNoteParams{
		ID:      input.NoteID,
		Title:   version.Title,
		Content: version.Content,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to restore note: %v", err))
	}

	if s.syncer != nil {
		_ = s.syncer.SyncNote(input.NoteID, version.Title, version.Content)
	}

	return textResult(map[string]any{
		"note_id":          input.NoteID,
		"restored_version": version.VersionNumber,
		"title":            version.Title,
		"status":           "restored",
		"message":          fmt.Sprintf("Note #%d restored to version %d", input.NoteID, version.VersionNumber),
	})
}

// --- Random note tool implementation ---

func (s *Server) toolRandom(ctx context.Context, input randomInput) (*mcp.CallToolResult, any, error) {
	var notes []db.Note
	var err error

	if input.Tag != "" {
		notes, err = s.queries.GetNotesByTagName(ctx, input.Tag)
	} else {
		notes, err = s.queries.GetAllNotes(ctx)
	}
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get notes: %v", err))
	}

	if len(notes) == 0 {
		return errorResult("no notes found")
	}

	note := notes[rand.IntN(len(notes))]
	out := formatNote(note)

	tags, _ := s.queries.GetTagsForNote(ctx, note.ID)
	out.Tags = make([]string, len(tags))
	for i, t := range tags {
		out.Tags[i] = t.Name
	}

	return textResult(out)
}

// --- Link health tool implementations ---

func (s *Server) toolBacklinks(ctx context.Context, input backlinksInput) (*mcp.CallToolResult, any, error) {
	_, err := s.queries.GetNote(ctx, input.NoteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResult(fmt.Sprintf("note #%d not found", input.NoteID))
		}
		return errorResult(fmt.Sprintf("failed to get note: %v", err))
	}

	backlinks, err := s.queries.GetBacklinks(ctx, input.NoteID)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get backlinks: %v", err))
	}

	output := make([]noteOutput, len(backlinks))
	for i, n := range backlinks {
		output[i] = formatNote(n)
	}

	return textResult(map[string]any{
		"note_id":   input.NoteID,
		"count":     len(output),
		"backlinks": output,
	})
}

func (s *Server) toolOrphans(ctx context.Context) (*mcp.CallToolResult, any, error) {
	orphans, err := s.queries.GetOrphanNotes(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get orphan notes: %v", err))
	}

	deadends, err := s.queries.GetDeadEndNotes(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get dead-end notes: %v", err))
	}

	type linkItem struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}

	orphanOut := make([]linkItem, len(orphans))
	for i, n := range orphans {
		orphanOut[i] = linkItem{ID: n.ID, Title: n.Title}
	}

	deadendOut := make([]linkItem, len(deadends))
	for i, n := range deadends {
		deadendOut[i] = linkItem{ID: n.ID, Title: n.Title}
	}

	return textResult(map[string]any{
		"orphans":        orphanOut,
		"orphan_count":   len(orphanOut),
		"deadends":       deadendOut,
		"deadend_count":  len(deadendOut),
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
