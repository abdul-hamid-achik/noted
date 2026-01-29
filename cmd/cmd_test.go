/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>

Test suite for noted CLI commands.
*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdul-hamid-achik/noted/internal/db"
)

// setupTestDB sets up a fresh database for testing
func setupTestDB(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var err error
	conn, err = db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	database = db.New(conn)

	return func() {
		if conn != nil {
			conn.Close()
			conn = nil
		}
		database = nil
	}
}

// createTestNote creates a note for testing and returns its ID
func createTestNote(t *testing.T, title, content string, tags []string) int64 {
	t.Helper()

	ctx := context.Background()
	note, err := database.CreateNote(ctx, db.CreateNoteParams{
		Title:   title,
		Content: content,
	})
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	for _, tagName := range tags {
		tag, err := database.CreateTag(ctx, tagName)
		if err != nil {
			t.Fatalf("failed to create tag: %v", err)
		}
		err = database.AddTagToNote(ctx, db.AddTagToNoteParams{
			NoteID: note.ID,
			TagID:  tag.ID,
		})
		if err != nil {
			t.Fatalf("failed to add tag to note: %v", err)
		}
	}

	return note.ID
}

// ============================================================================
// Database/Query Layer Tests
// ============================================================================

func TestDatabaseCreateAndGetNote(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a note
	note, err := database.CreateNote(ctx, db.CreateNoteParams{
		Title:   "Test Title",
		Content: "Test Content",
	})
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if note.ID == 0 {
		t.Error("expected non-zero note ID")
	}

	// Get the note
	retrieved, err := database.GetNote(ctx, note.ID)
	if err != nil {
		t.Fatalf("failed to get note: %v", err)
	}

	if retrieved.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %q", retrieved.Title)
	}
	if retrieved.Content != "Test Content" {
		t.Errorf("expected content 'Test Content', got %q", retrieved.Content)
	}
}

func TestDatabaseUpdateNote(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	noteID := createTestNote(t, "Original", "Original Content", nil)

	// Update the note
	updated, err := database.UpdateNote(ctx, db.UpdateNoteParams{
		ID:      noteID,
		Title:   "Updated",
		Content: "Updated Content",
	})
	if err != nil {
		t.Fatalf("failed to update note: %v", err)
	}

	if updated.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", updated.Title)
	}
}

func TestDatabaseDeleteNote(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	noteID := createTestNote(t, "To Delete", "Content", nil)

	err := database.DeleteNote(ctx, noteID)
	if err != nil {
		t.Fatalf("failed to delete note: %v", err)
	}

	// Verify deletion
	_, err = database.GetNote(ctx, noteID)
	if err == nil {
		t.Error("expected error when getting deleted note")
	}
}

func TestDatabaseTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	noteID := createTestNote(t, "Tagged Note", "Content", []string{"go", "programming"})

	tags, err := database.GetTagsForNote(ctx, noteID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}

	if !tagNames["go"] {
		t.Error("expected 'go' tag")
	}
	if !tagNames["programming"] {
		t.Error("expected 'programming' tag")
	}
}

func TestDatabaseRemoveAllTagsFromNote(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	noteID := createTestNote(t, "Tagged", "Content", []string{"tag1", "tag2", "tag3"})

	// Verify tags exist
	tags, _ := database.GetTagsForNote(ctx, noteID)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags initially, got %d", len(tags))
	}

	// Remove all tags
	err := database.RemoveAllTagsFromNote(ctx, noteID)
	if err != nil {
		t.Fatalf("failed to remove tags: %v", err)
	}

	// Verify tags removed
	tags, _ = database.GetTagsForNote(ctx, noteID)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after removal, got %d", len(tags))
	}
}

func TestDatabaseSearchNotesContent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	createTestNote(t, "Go Tutorial", "Learn Go programming", nil)
	createTestNote(t, "Python Basics", "Python for beginners", nil)
	createTestNote(t, "Advanced Go", "Go concurrency patterns", nil)

	// Search for "Go" in content or title
	notes, err := database.SearchNotesContent(ctx, db.SearchNotesContentParams{
		Content: "%Go%",
		Title:   "%Go%",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(notes) != 2 {
		t.Errorf("expected 2 matching notes, got %d", len(notes))
	}
}

func TestDatabaseGetTagsWithCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	createTestNote(t, "Note 1", "Content", []string{"shared", "unique1"})
	createTestNote(t, "Note 2", "Content", []string{"shared", "unique2"})

	tagsWithCount, err := database.GetTagsWithCount(ctx)
	if err != nil {
		t.Fatalf("failed to get tags with count: %v", err)
	}

	counts := make(map[string]int64)
	for _, tc := range tagsWithCount {
		counts[tc.Name] = tc.NoteCount
	}

	if counts["shared"] != 2 {
		t.Errorf("expected 'shared' count to be 2, got %d", counts["shared"])
	}
	if counts["unique1"] != 1 {
		t.Errorf("expected 'unique1' count to be 1, got %d", counts["unique1"])
	}
}

func TestDatabaseDeleteUnusedTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create note with tag, then delete the note (leaving orphan tag)
	noteID := createTestNote(t, "Temp", "Content", []string{"orphan"})
	_ = database.DeleteNote(ctx, noteID)

	// Create note that keeps its tag
	createTestNote(t, "Keep", "Content", []string{"active"})

	// Delete unused tags
	deleted, err := database.DeleteUnusedTags(ctx)
	if err != nil {
		t.Fatalf("failed to delete unused tags: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted tag, got %d", deleted)
	}

	// Verify orphan is gone, active remains
	tags, _ := database.ListTags(ctx)
	for _, tag := range tags {
		if tag.Name == "orphan" {
			t.Error("orphan tag should have been deleted")
		}
	}
}

func TestDatabaseGetAllNotes(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	createTestNote(t, "Note 1", "Content 1", nil)
	createTestNote(t, "Note 2", "Content 2", nil)
	createTestNote(t, "Note 3", "Content 3", nil)

	notes, err := database.GetAllNotes(ctx)
	if err != nil {
		t.Fatalf("failed to get all notes: %v", err)
	}

	if len(notes) != 3 {
		t.Errorf("expected 3 notes, got %d", len(notes))
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestParseMarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		wantTitle   string
		wantTags    []string
		wantContent string
	}{
		{
			name: "with frontmatter",
			content: `---
title: "My Title"
tags: [go, testing]
---

Content here`,
			wantTitle:   "My Title",
			wantTags:    []string{"go", "testing"},
			wantContent: "Content here",
		},
		{
			name:        "with H1 heading",
			content:     "# Heading Title\n\nSome content",
			wantTitle:   "Heading Title",
			wantTags:    nil,
			wantContent: "# Heading Title\n\nSome content",
		},
		{
			name:        "plain content",
			content:     "Just plain content\nNo heading",
			wantTitle:   "", // Will use filename
			wantTags:    nil,
			wantContent: "Just plain content\nNo heading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := "test.md"
			if tt.wantTitle == "" {
				filename = "expected-title.md"
			}
			mdFile := filepath.Join(tmpDir, filename)
			if err := os.WriteFile(mdFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			title, content, tags, err := parseMarkdownFile(mdFile)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedTitle := tt.wantTitle
			if expectedTitle == "" {
				expectedTitle = "expected-title"
			}

			if title != expectedTitle {
				t.Errorf("title: expected %q, got %q", expectedTitle, title)
			}

			if len(tags) != len(tt.wantTags) {
				t.Errorf("tags: expected %v, got %v", tt.wantTags, tags)
			}

			if !strings.Contains(content, strings.Split(tt.wantContent, "\n")[0]) {
				t.Errorf("content should contain %q", tt.wantContent)
			}
		})
	}
}

func TestGetEditor(t *testing.T) {
	// Save original
	original := os.Getenv("EDITOR")
	defer os.Setenv("EDITOR", original)

	// Test with EDITOR set
	os.Setenv("EDITOR", "vim")
	if editor := getEditor(); editor != "vim" {
		t.Errorf("expected 'vim', got %q", editor)
	}

	// Test with EDITOR unset
	os.Unsetenv("EDITOR")
	if editor := getEditor(); editor != "nvim" {
		t.Errorf("expected 'nvim' as default, got %q", editor)
	}
}

// ============================================================================
// Export Function Tests
// ============================================================================

func TestExportJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	createTestNote(t, "Export Test", "Content", []string{"tag1", "tag2"})

	notes, _ := database.GetAllNotes(ctx)

	var buf strings.Builder
	err := exportJSON(ctx, &buf, notes)
	if err != nil {
		t.Fatalf("exportJSON failed: %v", err)
	}

	var exported []exportedNote
	if err := json.Unmarshal([]byte(buf.String()), &exported); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(exported) != 1 {
		t.Fatalf("expected 1 note, got %d", len(exported))
	}

	if exported[0].Title != "Export Test" {
		t.Errorf("expected title 'Export Test', got %q", exported[0].Title)
	}

	if len(exported[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(exported[0].Tags))
	}
}

func TestExportMarkdown(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	createTestNote(t, "MD Export", "# Heading\nContent", []string{"md"})

	notes, _ := database.GetAllNotes(ctx)

	var buf strings.Builder
	err := exportMarkdown(ctx, &buf, notes)
	if err != nil {
		t.Fatalf("exportMarkdown failed: %v", err)
	}

	output := buf.String()

	if !strings.HasPrefix(output, "---\n") {
		t.Error("expected YAML frontmatter")
	}
	if !strings.Contains(output, `title: "MD Export"`) {
		t.Error("expected title in frontmatter")
	}
	if !strings.Contains(output, `tags: ["md"]`) {
		t.Error("expected quoted tags in frontmatter")
	}
	if !strings.Contains(output, "# Heading") {
		t.Error("expected content")
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestSpecialCharactersInContent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	specialContent := "中文 émoji 🎉 <script> & \" ' ` $ \\ %"
	noteID := createTestNote(t, "Special", specialContent, nil)

	ctx := context.Background()
	note, err := database.GetNote(ctx, noteID)
	if err != nil {
		t.Fatalf("failed to get note: %v", err)
	}

	if note.Content != specialContent {
		t.Errorf("content mismatch:\nexpected: %q\ngot: %q", specialContent, note.Content)
	}
}

func TestEmptyContent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	noteID := createTestNote(t, "Empty", "", nil)

	ctx := context.Background()
	note, _ := database.GetNote(ctx, noteID)

	if note.Content != "" {
		t.Errorf("expected empty content, got %q", note.Content)
	}
}

func TestVeryLongContent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 1MB content
	longContent := strings.Repeat("a", 1024*1024)
	noteID := createTestNote(t, "Long", longContent, nil)

	ctx := context.Background()
	note, _ := database.GetNote(ctx, noteID)

	if len(note.Content) != len(longContent) {
		t.Errorf("content length mismatch: expected %d, got %d", len(longContent), len(note.Content))
	}
}

func TestManyTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tags := make([]string, 100)
	for i := range tags {
		tags[i] = fmt.Sprintf("tag%03d", i)
	}

	noteID := createTestNote(t, "Many Tags", "Content", tags)

	ctx := context.Background()
	noteTags, _ := database.GetTagsForNote(ctx, noteID)

	if len(noteTags) != 100 {
		t.Errorf("expected 100 tags, got %d", len(noteTags))
	}
}

func TestDuplicateTagsHandled(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create same tag twice
	tag1, err1 := database.CreateTag(ctx, "duplicate")
	tag2, err2 := database.CreateTag(ctx, "duplicate")

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error creating tags: %v, %v", err1, err2)
	}

	// Should return same tag ID (ON CONFLICT behavior)
	if tag1.ID != tag2.ID {
		t.Errorf("expected same tag ID for duplicates, got %d and %d", tag1.ID, tag2.ID)
	}
}

func TestCascadeDeleteNoteRemovesTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	noteID := createTestNote(t, "To Delete", "Content", []string{"cascade-test"})

	// Verify tag association exists
	tags, _ := database.GetTagsForNote(ctx, noteID)
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}

	// Delete note
	_ = database.DeleteNote(ctx, noteID)

	// Tag association should be gone (CASCADE)
	tags, _ = database.GetTagsForNote(ctx, noteID)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after note deletion, got %d", len(tags))
	}
}

func TestListNotesWithPagination(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create 5 notes
	for i := 1; i <= 5; i++ {
		createTestNote(t, fmt.Sprintf("Note %d", i), "Content", nil)
	}

	// Get first 2
	notes, err := database.ListNotes(ctx, db.ListNotesParams{
		Limit:  2,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(notes) != 2 {
		t.Errorf("expected 2 notes with limit=2, got %d", len(notes))
	}

	// Get next 2 (offset=2)
	notes, _ = database.ListNotes(ctx, db.ListNotesParams{
		Limit:  2,
		Offset: 2,
	})

	if len(notes) != 2 {
		t.Errorf("expected 2 notes with offset=2, got %d", len(notes))
	}
}

func TestGetNotesByTagName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	createTestNote(t, "Tagged 1", "Content", []string{"filter-tag"})
	createTestNote(t, "Tagged 2", "Content", []string{"filter-tag"})
	createTestNote(t, "Untagged", "Content", []string{"other"})

	notes, err := database.GetNotesByTagName(ctx, "filter-tag")
	if err != nil {
		t.Fatalf("failed to get notes by tag: %v", err)
	}

	if len(notes) != 2 {
		t.Errorf("expected 2 notes with tag 'filter-tag', got %d", len(notes))
	}

	for _, note := range notes {
		if note.Title == "Untagged" {
			t.Error("should not include note without the tag")
		}
	}
}

// ============================================================================
// Version Variables Test
// ============================================================================

func TestVersionVariables(t *testing.T) {
	// Test default values
	if Version != "dev" {
		t.Errorf("default Version should be 'dev', got %q", Version)
	}
	if Commit != "none" {
		t.Errorf("default Commit should be 'none', got %q", Commit)
	}
	if BuildDate != "unknown" {
		t.Errorf("default BuildDate should be 'unknown', got %q", BuildDate)
	}
}
