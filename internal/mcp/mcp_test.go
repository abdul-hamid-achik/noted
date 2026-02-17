/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>

Test suite for noted MCP server and tools.
*/
package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/veclite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockSyncer implements Syncer interface for testing without Ollama
type mockSyncer struct {
	notes    map[int64]string // note_id -> content
	searches []veclite.SemanticResult
}

func newMockSyncer() *mockSyncer {
	return &mockSyncer{
		notes: make(map[int64]string),
	}
}

func (m *mockSyncer) Search(query string, limit int) ([]veclite.SemanticResult, error) {
	// Return pre-configured results or empty
	if len(m.searches) > limit {
		return m.searches[:limit], nil
	}
	return m.searches, nil
}

func (m *mockSyncer) SyncNote(id int64, title, content string) error {
	m.notes[id] = title + "\n\n" + content
	return nil
}

func (m *mockSyncer) Delete(noteID int64) error {
	delete(m.notes, noteID)
	return nil
}

func (m *mockSyncer) Close() error {
	return nil
}

// setupTestDB creates an in-memory test database
func setupTestDB(t *testing.T) (*db.Queries, *sql.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	queries := db.New(conn)

	cleanup := func() {
		conn.Close()
	}

	return queries, conn, cleanup
}

// createTestNote creates a note for testing
func createTestNote(t *testing.T, queries *db.Queries, title, content string, tags []string) int64 {
	t.Helper()
	ctx := context.Background()

	note, err := queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   title,
		Content: content,
	})
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	for _, tagName := range tags {
		tag, err := queries.CreateTag(ctx, tagName)
		if err != nil {
			t.Fatalf("failed to create tag: %v", err)
		}
		_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{
			NoteID: note.ID,
			TagID:  tag.ID,
		})
	}

	return note.ID
}

// getResultText extracts text from MCP result content
func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// parseResultJSON extracts and parses JSON from MCP result
func parseResultJSON(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	text := getResultText(result)
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("failed to parse result JSON: %v\ntext: %s", err, text)
	}
	return data
}

// ============================================================================
// Server Tests
// ============================================================================

func TestNewServer(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	// Without syncer
	server := NewServer(queries, conn, nil)
	if server == nil {
		t.Error("expected non-nil server")
	}
	if server.HasSemanticSearch() {
		t.Error("expected no semantic search without syncer")
	}

	// With syncer
	syncer := newMockSyncer()
	server = NewServer(queries, conn, syncer)
	if !server.HasSemanticSearch() {
		t.Error("expected semantic search with syncer")
	}
}

// ============================================================================
// Tool: noted_create Tests
// ============================================================================

func TestToolCreate_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolCreate(ctx, createInput{
		Title:   "Test Note",
		Content: "Test content",
		Tags:    []string{"go", "testing"},
	})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	// Verify note was created
	note, err := queries.GetNote(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get created note: %v", err)
	}
	if note.Title != "Test Note" {
		t.Errorf("expected title 'Test Note', got %q", note.Title)
	}

	// Verify tags
	tags, _ := queries.GetTagsForNote(ctx, note.ID)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestToolCreate_MissingTitle(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolCreate(ctx, createInput{
		Content: "Content without title",
	})

	if !result.IsError {
		t.Error("expected error for missing title")
	}
}

func TestToolCreate_MissingContent(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolCreate(ctx, createInput{
		Title: "Title without content",
	})

	if !result.IsError {
		t.Error("expected error for missing content")
	}
}

func TestToolCreate_WithSyncer(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	syncer := newMockSyncer()
	server := NewServer(queries, conn, syncer)
	ctx := context.Background()

	result, _, _ := server.toolCreate(ctx, createInput{
		Title:   "Synced Note",
		Content: "Content to sync",
	})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	// Verify syncer received the note
	if _, ok := syncer.notes[1]; !ok {
		t.Error("expected note to be synced")
	}
}

// ============================================================================
// Tool: noted_list Tests
// ============================================================================

func TestToolList_Empty(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolList(ctx, listInput{})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	if data["count"].(float64) != 0 {
		t.Errorf("expected count 0, got %v", data["count"])
	}
}

func TestToolList_WithNotes(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	createTestNote(t, queries, "Note 1", "Content 1", nil)
	createTestNote(t, queries, "Note 2", "Content 2", nil)
	createTestNote(t, queries, "Note 3", "Content 3", nil)

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolList(ctx, listInput{Limit: 10})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 3 {
		t.Errorf("expected count 3, got %v", data["count"])
	}
}

func TestToolList_WithTagFilter(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	createTestNote(t, queries, "Go Note", "Content", []string{"go"})
	createTestNote(t, queries, "Python Note", "Content", []string{"python"})

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolList(ctx, listInput{Tag: "go"})

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 1 {
		t.Errorf("expected 1 note with tag 'go', got %v", data["count"])
	}
}

func TestToolList_Pagination(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	for i := 1; i <= 5; i++ {
		createTestNote(t, queries, "Note", "Content", nil)
	}

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// First page
	result, _, _ := server.toolList(ctx, listInput{Limit: 2, Offset: 0})
	data := parseResultJSON(t, result)

	if int(data["count"].(float64)) != 2 {
		t.Errorf("expected 2 notes on first page, got %v", data["count"])
	}

	// Second page
	result, _, _ = server.toolList(ctx, listInput{Limit: 2, Offset: 2})
	data = parseResultJSON(t, result)

	if int(data["count"].(float64)) != 2 {
		t.Errorf("expected 2 notes on second page, got %v", data["count"])
	}
}

// ============================================================================
// Tool: noted_get Tests
// ============================================================================

func TestToolGet_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	noteID := createTestNote(t, queries, "Get Test", "Content", []string{"tag1"})

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolGet(ctx, getInput{ID: noteID})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	data := parseResultJSON(t, result)
	if data["title"] != "Get Test" {
		t.Errorf("expected title 'Get Test', got %v", data["title"])
	}
	if data["content"] != "Content" {
		t.Errorf("expected content 'Content', got %v", data["content"])
	}
}

func TestToolGet_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolGet(ctx, getInput{ID: 999})

	if !result.IsError {
		t.Error("expected error for non-existent note")
	}

	text := getResultText(result)
	if !strings.Contains(text, "not found") {
		t.Errorf("expected 'not found' in error, got %q", text)
	}
}

// ============================================================================
// Tool: noted_search Tests
// ============================================================================

func TestToolSearch_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	createTestNote(t, queries, "Go Tutorial", "Learn Go programming", nil)
	createTestNote(t, queries, "Python Guide", "Python basics", nil)

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolSearch(ctx, searchInput{Query: "Go"})

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 1 {
		t.Errorf("expected 1 result for 'Go', got %v", data["count"])
	}
}

func TestToolSearch_EmptyQuery(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolSearch(ctx, searchInput{Query: ""})

	if !result.IsError {
		t.Error("expected error for empty query")
	}
}

func TestToolSearch_NoResults(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	createTestNote(t, queries, "Test", "Content", nil)

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolSearch(ctx, searchInput{Query: "nonexistent"})

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 0 {
		t.Errorf("expected 0 results, got %v", data["count"])
	}
}

// ============================================================================
// Tool: noted_update Tests
// ============================================================================

func TestToolUpdate_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	noteID := createTestNote(t, queries, "Original", "Original content", nil)

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolUpdate(ctx, updateInput{
		ID:      noteID,
		Title:   "Updated",
		Content: "Updated content",
	})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	// Verify update
	note, _ := queries.GetNote(ctx, noteID)
	if note.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", note.Title)
	}
	if note.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got %q", note.Content)
	}
}

func TestToolUpdate_PartialUpdate(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	noteID := createTestNote(t, queries, "Original Title", "Original content", nil)

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// Only update title
	result, _, _ := server.toolUpdate(ctx, updateInput{
		ID:    noteID,
		Title: "New Title",
	})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	note, _ := queries.GetNote(ctx, noteID)
	if note.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %q", note.Title)
	}
	if note.Content != "Original content" {
		t.Errorf("content should be unchanged, got %q", note.Content)
	}
}

func TestToolUpdate_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolUpdate(ctx, updateInput{
		ID:    999,
		Title: "Whatever",
	})

	if !result.IsError {
		t.Error("expected error for non-existent note")
	}
}

func TestToolUpdate_Tags(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	noteID := createTestNote(t, queries, "Test", "Content", []string{"old1", "old2"})

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolUpdate(ctx, updateInput{
		ID:   noteID,
		Tags: []string{"new1", "new2", "new3"},
	})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	// Verify tags replaced
	tags, _ := queries.GetTagsForNote(ctx, noteID)
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}

	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}
	if tagNames["old1"] || tagNames["old2"] {
		t.Error("old tags should be removed")
	}
	if !tagNames["new1"] || !tagNames["new2"] || !tagNames["new3"] {
		t.Error("new tags should be present")
	}
}

// ============================================================================
// Tool: noted_delete Tests
// ============================================================================

func TestToolDelete_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	noteID := createTestNote(t, queries, "To Delete", "Content", nil)

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolDelete(ctx, deleteInput{ID: noteID})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	// Verify deletion
	_, err := queries.GetNote(ctx, noteID)
	if err != sql.ErrNoRows {
		t.Error("note should be deleted")
	}
}

func TestToolDelete_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolDelete(ctx, deleteInput{ID: 999})

	if !result.IsError {
		t.Error("expected error for non-existent note")
	}
}

func TestToolDelete_WithSyncer(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	syncer := newMockSyncer()
	syncer.notes[1] = "content"

	noteID := createTestNote(t, queries, "To Delete", "Content", nil)

	server := NewServer(queries, conn, syncer)
	ctx := context.Background()

	result, _, _ := server.toolDelete(ctx, deleteInput{ID: noteID})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	// Verify syncer was notified
	if _, ok := syncer.notes[noteID]; ok {
		t.Error("note should be removed from syncer")
	}
}

// ============================================================================
// Tool: noted_tags Tests
// ============================================================================

func TestToolTags_Empty(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolTags(ctx)

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 0 {
		t.Errorf("expected 0 tags, got %v", data["count"])
	}
}

func TestToolTags_WithCounts(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	createTestNote(t, queries, "Note 1", "Content", []string{"shared", "unique1"})
	createTestNote(t, queries, "Note 2", "Content", []string{"shared", "unique2"})

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolTags(ctx)

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 3 {
		t.Errorf("expected 3 tags, got %v", data["count"])
	}

	// Check counts
	tags := data["tags"].([]any)
	for _, tag := range tags {
		tagMap := tag.(map[string]any)
		if tagMap["name"] == "shared" {
			if int(tagMap["note_count"].(float64)) != 2 {
				t.Errorf("expected 'shared' count 2, got %v", tagMap["note_count"])
			}
		}
	}
}

// ============================================================================
// Tool: noted_remember Tests
// ============================================================================

func TestToolRemember_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolRemember(ctx, rememberInput{
		Content:    "Important fact to remember",
		Title:      "Test Memory",
		Category:   "fact",
		Importance: 4,
	})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	// Verify memory was created with tags
	note, _ := queries.GetNote(ctx, 1)
	if note.Title != "Test Memory" {
		t.Errorf("expected title 'Test Memory', got %q", note.Title)
	}

	tags, _ := queries.GetTagsForNote(ctx, note.ID)
	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}

	if !tagNames["memory"] {
		t.Error("expected 'memory' tag")
	}
	if !tagNames["memory:fact"] {
		t.Error("expected 'memory:fact' tag")
	}
	if !tagNames["importance:4"] {
		t.Error("expected 'importance:4' tag")
	}
}

func TestToolRemember_DefaultValues(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolRemember(ctx, rememberInput{
		Content: "Just the content",
	})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	// Check defaults were applied
	tags, _ := queries.GetTagsForNote(ctx, 1)
	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}

	if !tagNames["memory:fact"] {
		t.Error("expected default category 'fact'")
	}
	if !tagNames["importance:3"] {
		t.Error("expected default importance 3")
	}
}

func TestToolRemember_MissingContent(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolRemember(ctx, rememberInput{
		Title: "No content",
	})

	if !result.IsError {
		t.Error("expected error for missing content")
	}
}

// ============================================================================
// Tool: noted_recall Tests
// ============================================================================

func TestToolRecall_KeywordSearch(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a memory
	ctx := context.Background()
	note, _ := queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   "Go Facts",
		Content: "Go was created at Google",
	})
	tag, _ := queries.CreateTag(ctx, "memory")
	_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: note.ID, TagID: tag.ID})
	tag2, _ := queries.CreateTag(ctx, "memory:fact")
	_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: note.ID, TagID: tag2.ID})
	tag3, _ := queries.CreateTag(ctx, "importance:3")
	_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: note.ID, TagID: tag3.ID})

	server := NewServer(queries, conn, nil) // No syncer - uses keyword search
	result, _, _ := server.toolRecall(ctx, recallInput{Query: "Go"})

	data := parseResultJSON(t, result)
	if data["method"] != "keyword" {
		t.Errorf("expected method 'keyword', got %v", data["method"])
	}
	if int(data["count"].(float64)) != 1 {
		t.Errorf("expected 1 memory, got %v", data["count"])
	}
}

func TestToolRecall_EmptyQuery(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolRecall(ctx, recallInput{Query: ""})

	if !result.IsError {
		t.Error("expected error for empty query")
	}
}

// ============================================================================
// Tool: noted_forget Tests
// ============================================================================

func TestToolForget_DryRun(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	// Create memories
	ctx := context.Background()
	note, _ := queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   "Low importance",
		Content: "Not important",
	})
	tag, _ := queries.CreateTag(ctx, "memory")
	_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: note.ID, TagID: tag.ID})
	tag2, _ := queries.CreateTag(ctx, "importance:1")
	_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: note.ID, TagID: tag2.ID})

	server := NewServer(queries, conn, nil)
	result, _, _ := server.toolForget(ctx, forgetInput{
		ImportanceBelow: 3,
		DryRun:          true,
	})

	data := parseResultJSON(t, result)
	if data["dry_run"] != true {
		t.Error("expected dry_run true")
	}
	if int(data["would_delete"].(float64)) != 1 {
		t.Errorf("expected would_delete 1, got %v", data["would_delete"])
	}

	// Verify note still exists
	_, err := queries.GetNote(ctx, note.ID)
	if err != nil {
		t.Error("note should not be deleted in dry run")
	}
}

func TestToolForget_ActualDelete(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	// Create memory
	ctx := context.Background()
	note, _ := queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   "To Forget",
		Content: "Content",
	})
	tag, _ := queries.CreateTag(ctx, "memory")
	_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: note.ID, TagID: tag.ID})
	tag2, _ := queries.CreateTag(ctx, "importance:1")
	_ = queries.AddTagToNote(ctx, db.AddTagToNoteParams{NoteID: note.ID, TagID: tag2.ID})

	server := NewServer(queries, conn, nil)
	result, _, _ := server.toolForget(ctx, forgetInput{
		ImportanceBelow: 3,
		DryRun:          false,
	})

	data := parseResultJSON(t, result)
	if int(data["deleted"].(float64)) != 1 {
		t.Errorf("expected deleted 1, got %v", data["deleted"])
	}

	// Verify note is deleted
	_, err := queries.GetNote(ctx, note.ID)
	if err != sql.ErrNoRows {
		t.Error("note should be deleted")
	}
}

// ============================================================================
// Tool: noted_sync Tests
// ============================================================================

func TestToolSync_NoSyncer(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil) // No syncer
	ctx := context.Background()

	result, _, _ := server.toolSync(ctx, syncInput{})

	if !result.IsError {
		t.Error("expected error without syncer")
	}
}

func TestToolSync_WithSyncer(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	// Create unsynced note
	createTestNote(t, queries, "Unsynced", "Content", nil)

	syncer := newMockSyncer()
	server := NewServer(queries, conn, syncer)
	ctx := context.Background()

	result, _, _ := server.toolSync(ctx, syncInput{Force: false})

	if result.IsError {
		t.Errorf("unexpected error")
	}

	data := parseResultJSON(t, result)
	if int(data["synced"].(float64)) != 1 {
		t.Errorf("expected synced 1, got %v", data["synced"])
	}
}

// ============================================================================
// Tool: noted_semantic_search Tests
// ============================================================================

func TestToolSemanticSearch_NoSyncer(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolSemanticSearch(ctx, semanticSearchInput{Query: "test"})

	if !result.IsError {
		t.Error("expected error without syncer")
	}
}

func TestToolSemanticSearch_EmptyQuery(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	syncer := newMockSyncer()
	server := NewServer(queries, conn, syncer)
	ctx := context.Background()

	result, _, _ := server.toolSemanticSearch(ctx, semanticSearchInput{Query: ""})

	if !result.IsError {
		t.Error("expected error for empty query")
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestSpecialCharactersInNotes(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	specialContent := "中文 émoji 🎉 <script> & \" ' ` $ \\ %"
	result, _, _ := server.toolCreate(ctx, createInput{
		Title:   "Special",
		Content: specialContent,
	})

	if result.IsError {
		t.Errorf("failed to create note with special characters")
	}

	// Verify content preserved
	note, _ := queries.GetNote(ctx, 1)
	if note.Content != specialContent {
		t.Errorf("content not preserved:\nexpected: %q\ngot: %q", specialContent, note.Content)
	}
}

func TestVeryLongContent(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// 100KB content
	longContent := strings.Repeat("a", 100*1024)
	result, _, _ := server.toolCreate(ctx, createInput{
		Title:   "Long",
		Content: longContent,
	})

	if result.IsError {
		t.Errorf("failed to create note with long content")
	}
}

func TestDefaultLimitValues(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	// Create 25 notes
	for i := 0; i < 25; i++ {
		createTestNote(t, queries, "Note", "Content", nil)
	}

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// List without limit should default to 20
	result, _, _ := server.toolList(ctx, listInput{})
	data := parseResultJSON(t, result)

	if int(data["count"].(float64)) != 20 {
		t.Errorf("expected default limit of 20, got %v", data["count"])
	}
}

// ============================================================================
// Tool: noted_daily Tests
// ============================================================================

func TestToolDaily_CreateNew(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolDaily(ctx, dailyInput{Date: "2026-02-17"})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	if data["title"] != "2026-02-17" {
		t.Errorf("expected title '2026-02-17', got %v", data["title"])
	}
}

func TestToolDaily_GetExisting(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// Create daily note
	result1, _, _ := server.toolDaily(ctx, dailyInput{Date: "2026-02-17"})
	data1 := parseResultJSON(t, result1)
	id1 := data1["id"].(float64)

	// Get same daily note
	result2, _, _ := server.toolDaily(ctx, dailyInput{Date: "2026-02-17"})
	data2 := parseResultJSON(t, result2)
	id2 := data2["id"].(float64)

	if id1 != id2 {
		t.Errorf("expected same note ID, got %v and %v", id1, id2)
	}
}

func TestToolDaily_Append(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// Create then append
	server.toolDaily(ctx, dailyInput{Date: "2026-02-17"})
	result, _, _ := server.toolDaily(ctx, dailyInput{
		Date:   "2026-02-17",
		Append: "- [ ] Buy milk",
	})

	data := parseResultJSON(t, result)
	content := data["content"].(string)
	if !strings.Contains(content, "Buy milk") {
		t.Errorf("expected appended content, got %q", content)
	}
}

func TestToolDaily_Prepend(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// Create with content, then prepend
	server.toolDaily(ctx, dailyInput{Date: "2026-02-17", Append: "Existing"})
	result, _, _ := server.toolDaily(ctx, dailyInput{
		Date:    "2026-02-17",
		Prepend: "Prepended",
	})

	data := parseResultJSON(t, result)
	content := data["content"].(string)
	if !strings.HasPrefix(content, "Prepended") {
		t.Errorf("expected content to start with 'Prepended', got %q", content)
	}
}

func TestToolDaily_InvalidDate(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolDaily(ctx, dailyInput{Date: "not-a-date"})

	if !result.IsError {
		t.Error("expected error for invalid date")
	}
}

func TestToolDailyList(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// Create a few daily notes
	server.toolDaily(ctx, dailyInput{Date: "2026-02-15"})
	server.toolDaily(ctx, dailyInput{Date: "2026-02-16"})
	server.toolDaily(ctx, dailyInput{Date: "2026-02-17"})

	result, _, _ := server.toolDailyList(ctx, dailyListInput{})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	count := int(data["count"].(float64))
	if count != 3 {
		t.Errorf("expected 3 daily notes, got %d", count)
	}
}

// ============================================================================
// Tool: noted_template Tests
// ============================================================================

func TestToolTemplateCreate_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolTemplateCreate(ctx, templateCreateInput{
		Name:    "meeting",
		Content: "# {{title}}\n\n## Notes\n",
	})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	if data["name"] != "meeting" {
		t.Errorf("expected name 'meeting', got %v", data["name"])
	}
}

func TestToolTemplateCreate_MissingName(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolTemplateCreate(ctx, templateCreateInput{Content: "content"})

	if !result.IsError {
		t.Error("expected error for missing name")
	}
}

func TestToolTemplateList(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// Create templates
	queries.CreateTemplate(ctx, db.CreateTemplateParams{Name: "a-meeting", Content: "meeting"})
	queries.CreateTemplate(ctx, db.CreateTemplateParams{Name: "b-daily", Content: "daily"})

	result, _, _ := server.toolTemplateList(ctx)

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 2 {
		t.Errorf("expected 2 templates, got %v", data["count"])
	}
}

func TestToolTemplateGet_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	queries.CreateTemplate(ctx, db.CreateTemplateParams{Name: "test", Content: "template content"})

	result, _, _ := server.toolTemplateGet(ctx, templateGetInput{Name: "test"})

	data := parseResultJSON(t, result)
	if data["content"] != "template content" {
		t.Errorf("expected content 'template content', got %v", data["content"])
	}
}

func TestToolTemplateGet_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolTemplateGet(ctx, templateGetInput{Name: "nonexistent"})

	if !result.IsError {
		t.Error("expected error for non-existent template")
	}
}

func TestToolTemplateDelete(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	queries.CreateTemplate(ctx, db.CreateTemplateParams{Name: "delete-me", Content: "content"})

	result, _, _ := server.toolTemplateDelete(ctx, templateDeleteInput{Name: "delete-me"})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	// Verify deletion
	_, err := queries.GetTemplateByName(ctx, "delete-me")
	if err != sql.ErrNoRows {
		t.Error("template should be deleted")
	}
}

func TestToolTemplateApply(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	queries.CreateTemplate(ctx, db.CreateTemplateParams{
		Name:    "meeting",
		Content: "# {{title}}\n\nDate: {{date}}\n",
	})

	result, _, _ := server.toolTemplateApply(ctx, templateApplyInput{
		TemplateName: "meeting",
		Title:        "Sprint Retro",
		Tags:         []string{"work"},
	})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	if data["title"] != "Sprint Retro" {
		t.Errorf("expected title 'Sprint Retro', got %v", data["title"])
	}

	// Verify note content was interpolated
	note, _ := queries.GetNote(ctx, int64(data["id"].(float64)))
	if strings.Contains(note.Content, "{{title}}") {
		t.Error("template variable {{title}} should be interpolated")
	}
	if !strings.Contains(note.Content, "Sprint Retro") {
		t.Error("expected interpolated title in content")
	}
}

func TestToolTemplateApply_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolTemplateApply(ctx, templateApplyInput{
		TemplateName: "nonexistent",
		Title:        "Test",
	})

	if !result.IsError {
		t.Error("expected error for non-existent template")
	}
}

// ============================================================================
// Tool: noted_tasks Tests
// ============================================================================

func TestToolTasks_ExtractAll(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	createTestNote(t, queries, "Todo", "- [ ] Task A\n- [x] Task B\n- [ ] Task C", nil)

	result, _, _ := server.toolTasks(ctx, tasksInput{})

	data := parseResultJSON(t, result)
	if int(data["total"].(float64)) != 3 {
		t.Errorf("expected 3 tasks, got %v", data["total"])
	}
	if int(data["pending"].(float64)) != 2 {
		t.Errorf("expected 2 pending, got %v", data["pending"])
	}
	if int(data["completed"].(float64)) != 1 {
		t.Errorf("expected 1 completed, got %v", data["completed"])
	}
}

func TestToolTasks_FilterPending(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	createTestNote(t, queries, "Todo", "- [ ] Pending\n- [x] Done", nil)

	result, _, _ := server.toolTasks(ctx, tasksInput{Pending: true})

	data := parseResultJSON(t, result)
	if int(data["total"].(float64)) != 1 {
		t.Errorf("expected 1 pending task, got %v", data["total"])
	}
}

func TestToolTasks_FilterByNote(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	noteID := createTestNote(t, queries, "Note A", "- [ ] Task A", nil)
	createTestNote(t, queries, "Note B", "- [ ] Task B", nil)

	result, _, _ := server.toolTasks(ctx, tasksInput{NoteID: noteID})

	data := parseResultJSON(t, result)
	if int(data["total"].(float64)) != 1 {
		t.Errorf("expected 1 task from specific note, got %v", data["total"])
	}
}

func TestToolTasks_FilterByTag(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	createTestNote(t, queries, "Work", "- [ ] Work task", []string{"work"})
	createTestNote(t, queries, "Personal", "- [ ] Personal task", []string{"personal"})

	result, _, _ := server.toolTasks(ctx, tasksInput{Tag: "work"})

	data := parseResultJSON(t, result)
	if int(data["total"].(float64)) != 1 {
		t.Errorf("expected 1 work task, got %v", data["total"])
	}
}

func TestToolTasks_NoTasks(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	createTestNote(t, queries, "No Tasks", "Just regular content", nil)

	result, _, _ := server.toolTasks(ctx, tasksInput{})

	data := parseResultJSON(t, result)
	if int(data["total"].(float64)) != 0 {
		t.Errorf("expected 0 tasks, got %v", data["total"])
	}
}

// ============================================================================
// Tool: noted_history Tests
// ============================================================================

func TestToolHistory_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	noteID := createTestNote(t, queries, "Test", "Content", nil)

	// Create versions
	queries.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID: noteID, Title: "v1", Content: "v1 content", VersionNumber: 1,
	})
	queries.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID: noteID, Title: "v2", Content: "v2 content", VersionNumber: 2,
	})

	result, _, _ := server.toolHistory(ctx, historyInput{NoteID: noteID})

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 2 {
		t.Errorf("expected 2 versions, got %v", data["count"])
	}
}

func TestToolHistory_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolHistory(ctx, historyInput{NoteID: 999})

	if !result.IsError {
		t.Error("expected error for non-existent note")
	}
}

func TestToolVersionGet_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	noteID := createTestNote(t, queries, "Test", "Current", nil)
	queries.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID: noteID, Title: "Old", Content: "Old content", VersionNumber: 1,
	})

	result, _, _ := server.toolVersionGet(ctx, versionGetInput{NoteID: noteID, Version: 1})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	if data["content"] != "Old content" {
		t.Errorf("expected 'Old content', got %v", data["content"])
	}
}

func TestToolVersionGet_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	noteID := createTestNote(t, queries, "Test", "Content", nil)

	result, _, _ := server.toolVersionGet(ctx, versionGetInput{NoteID: noteID, Version: 99})

	if !result.IsError {
		t.Error("expected error for non-existent version")
	}
}

// ============================================================================
// Tool: noted_restore Tests
// ============================================================================

func TestToolRestore_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	noteID := createTestNote(t, queries, "Current Title", "Current content", nil)

	// Create a version representing the old state
	queries.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID: noteID, Title: "Old Title", Content: "Old content", VersionNumber: 1,
	})

	result, _, _ := server.toolRestore(ctx, restoreInput{NoteID: noteID, Version: 1})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	// Verify note was restored
	note, _ := queries.GetNote(ctx, noteID)
	if note.Title != "Old Title" {
		t.Errorf("expected title 'Old Title', got %q", note.Title)
	}
	if note.Content != "Old content" {
		t.Errorf("expected content 'Old content', got %q", note.Content)
	}

	// Verify current state was saved as a new version
	versions, _ := queries.GetNoteVersions(ctx, noteID)
	if len(versions) != 2 {
		t.Errorf("expected 2 versions (original + backup), got %d", len(versions))
	}
}

func TestToolRestore_NoteNotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolRestore(ctx, restoreInput{NoteID: 999, Version: 1})

	if !result.IsError {
		t.Error("expected error for non-existent note")
	}
}

func TestToolRestore_VersionNotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	noteID := createTestNote(t, queries, "Test", "Content", nil)

	result, _, _ := server.toolRestore(ctx, restoreInput{NoteID: noteID, Version: 99})

	if !result.IsError {
		t.Error("expected error for non-existent version")
	}
}

// ============================================================================
// Tool: noted_random Tests
// ============================================================================

func TestToolRandom_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	createTestNote(t, queries, "Note 1", "Content 1", nil)
	createTestNote(t, queries, "Note 2", "Content 2", nil)
	createTestNote(t, queries, "Note 3", "Content 3", nil)

	result, _, _ := server.toolRandom(ctx, randomInput{})

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	if data["title"] == nil || data["title"] == "" {
		t.Error("expected a note title")
	}
}

func TestToolRandom_WithTag(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	createTestNote(t, queries, "Go Note", "Content", []string{"go"})
	createTestNote(t, queries, "Python Note", "Content", []string{"python"})

	result, _, _ := server.toolRandom(ctx, randomInput{Tag: "go"})

	data := parseResultJSON(t, result)
	if data["title"] != "Go Note" {
		t.Errorf("expected 'Go Note' (only go-tagged note), got %v", data["title"])
	}
}

func TestToolRandom_Empty(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolRandom(ctx, randomInput{})

	if !result.IsError {
		t.Error("expected error for empty database")
	}
}

// ============================================================================
// Tool: noted_backlinks Tests
// ============================================================================

func TestToolBacklinks_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	sourceID := createTestNote(t, queries, "Source", "Links to target", nil)
	targetID := createTestNote(t, queries, "Target", "Content", nil)

	queries.CreateNoteLink(ctx, db.CreateNoteLinkParams{
		SourceNoteID: sourceID,
		TargetNoteID: targetID,
		LinkText:     "Target",
	})

	result, _, _ := server.toolBacklinks(ctx, backlinksInput{NoteID: targetID})

	data := parseResultJSON(t, result)
	if int(data["count"].(float64)) != 1 {
		t.Errorf("expected 1 backlink, got %v", data["count"])
	}
}

func TestToolBacklinks_NotFound(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolBacklinks(ctx, backlinksInput{NoteID: 999})

	if !result.IsError {
		t.Error("expected error for non-existent note")
	}
}

// ============================================================================
// Tool: noted_orphans Tests
// ============================================================================

func TestToolOrphans_Success(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	// Create orphan note
	createTestNote(t, queries, "Orphan", "No links", nil)

	// Create linked notes
	sourceID := createTestNote(t, queries, "Source", "Content", nil)
	targetID := createTestNote(t, queries, "Target", "Content", nil)
	queries.CreateNoteLink(ctx, db.CreateNoteLinkParams{
		SourceNoteID: sourceID,
		TargetNoteID: targetID,
		LinkText:     "Target",
	})

	result, _, _ := server.toolOrphans(ctx)

	if result.IsError {
		t.Errorf("unexpected error: %s", getResultText(result))
	}

	data := parseResultJSON(t, result)
	if int(data["orphan_count"].(float64)) != 1 {
		t.Errorf("expected 1 orphan, got %v", data["orphan_count"])
	}
	if int(data["deadend_count"].(float64)) != 1 {
		t.Errorf("expected 1 deadend, got %v", data["deadend_count"])
	}
}

func TestToolOrphans_Empty(t *testing.T) {
	queries, conn, cleanup := setupTestDB(t)
	defer cleanup()

	server := NewServer(queries, conn, nil)
	ctx := context.Background()

	result, _, _ := server.toolOrphans(ctx)

	data := parseResultJSON(t, result)
	if int(data["orphan_count"].(float64)) != 0 {
		t.Errorf("expected 0 orphans, got %v", data["orphan_count"])
	}
}
