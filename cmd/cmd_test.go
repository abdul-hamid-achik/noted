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

// ============================================================================
// Template Tests
// ============================================================================

func TestInterpolateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		title    string
		wantDate bool
		wantTime bool
		want     string
	}{
		{
			name:    "title variable",
			content: "# {{title}}",
			title:   "My Note",
			want:    "# My Note",
		},
		{
			name:     "date variable",
			content:  "Date: {{date}}",
			title:    "",
			wantDate: true,
		},
		{
			name:     "time variable",
			content:  "Time: {{time}}",
			title:    "",
			wantTime: true,
		},
		{
			name:    "no variables",
			content: "Plain content",
			title:   "Title",
			want:    "Plain content",
		},
		{
			name:    "multiple variables",
			content: "# {{title}}\n\nCreated: {{date}}\nTitle: {{title}}",
			title:   "Test",
			want:    "", // Just check it doesn't contain raw {{
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateTemplate(tt.content, tt.title)

			if tt.want != "" {
				if result != tt.want {
					t.Errorf("expected %q, got %q", tt.want, result)
				}
			}

			if strings.Contains(result, "{{") {
				t.Errorf("uninterpolated variable in result: %q", result)
			}
		})
	}
}

func TestTemplateCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create
	tmpl, err := database.CreateTemplate(ctx, db.CreateTemplateParams{
		Name:    "meeting",
		Content: "# {{title}}\n\n## Notes\n",
	})
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}
	if tmpl.Name != "meeting" {
		t.Errorf("expected name 'meeting', got %q", tmpl.Name)
	}

	// Get by name
	got, err := database.GetTemplateByName(ctx, "meeting")
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}
	if got.Content != "# {{title}}\n\n## Notes\n" {
		t.Errorf("content mismatch: %q", got.Content)
	}

	// List
	all, err := database.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 template, got %d", len(all))
	}

	// Update
	updated, err := database.UpdateTemplate(ctx, db.UpdateTemplateParams{
		Content: "updated content",
		ID:      tmpl.ID,
	})
	if err != nil {
		t.Fatalf("failed to update template: %v", err)
	}
	if updated.Content != "updated content" {
		t.Errorf("expected updated content, got %q", updated.Content)
	}

	// Delete
	err = database.DeleteTemplateByName(ctx, "meeting")
	if err != nil {
		t.Fatalf("failed to delete template: %v", err)
	}

	all, _ = database.ListTemplates(ctx)
	if len(all) != 0 {
		t.Errorf("expected 0 templates after deletion, got %d", len(all))
	}
}

// ============================================================================
// Task Extraction Tests
// ============================================================================

func TestExtractTasks(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst string
		wantDone  bool
	}{
		{
			name:      "pending task",
			content:   "- [ ] Buy groceries",
			wantCount: 1,
			wantFirst: "Buy groceries",
			wantDone:  false,
		},
		{
			name:      "completed task lowercase",
			content:   "- [x] Done task",
			wantCount: 1,
			wantFirst: "Done task",
			wantDone:  true,
		},
		{
			name:      "completed task uppercase",
			content:   "- [X] Also done",
			wantCount: 1,
			wantFirst: "Also done",
			wantDone:  true,
		},
		{
			name:      "multiple tasks",
			content:   "# Todo\n- [ ] Task 1\n- [x] Task 2\n- [ ] Task 3",
			wantCount: 3,
			wantFirst: "Task 1",
			wantDone:  false,
		},
		{
			name:      "no tasks",
			content:   "# Just a heading\n\nSome content\n- A bullet (not a task)",
			wantCount: 0,
		},
		{
			name:      "indented tasks",
			content:   "  - [ ] Indented task",
			wantCount: 1,
			wantFirst: "Indented task",
		},
		{
			name:      "empty content",
			content:   "",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := db.Note{ID: 1, Title: "Test", Content: tt.content}
			tasks := extractTasks(note)

			if len(tasks) != tt.wantCount {
				t.Errorf("expected %d tasks, got %d", tt.wantCount, len(tasks))
				return
			}

			if tt.wantCount > 0 {
				if tasks[0].Text != tt.wantFirst {
					t.Errorf("expected first task %q, got %q", tt.wantFirst, tasks[0].Text)
				}
				if tasks[0].Completed != tt.wantDone {
					t.Errorf("expected completed=%v, got %v", tt.wantDone, tasks[0].Completed)
				}
				if tasks[0].NoteID != 1 {
					t.Errorf("expected note ID 1, got %d", tasks[0].NoteID)
				}
			}
		})
	}
}

// ============================================================================
// Version History Tests
// ============================================================================

func TestNoteVersioning(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	noteID := createTestNote(t, "Original", "Original content", nil)

	// Create version 1
	_, err := database.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID:        noteID,
		Title:         "Original",
		Content:       "Original content",
		VersionNumber: 1,
	})
	if err != nil {
		t.Fatalf("failed to create version: %v", err)
	}

	// Create version 2
	_, err = database.CreateNoteVersion(ctx, db.CreateNoteVersionParams{
		NoteID:        noteID,
		Title:         "Updated",
		Content:       "Updated content",
		VersionNumber: 2,
	})
	if err != nil {
		t.Fatalf("failed to create version 2: %v", err)
	}

	// List versions
	versions, err := database.GetNoteVersions(ctx, noteID)
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
	// Should be DESC order
	if versions[0].VersionNumber != 2 {
		t.Errorf("expected first version to be 2 (DESC), got %d", versions[0].VersionNumber)
	}

	// Get specific version
	v1, err := database.GetNoteVersion(ctx, db.GetNoteVersionParams{
		NoteID:        noteID,
		VersionNumber: 1,
	})
	if err != nil {
		t.Fatalf("failed to get version 1: %v", err)
	}
	if v1.Content != "Original content" {
		t.Errorf("expected original content in v1, got %q", v1.Content)
	}

	// Get latest version number
	latest, err := database.GetLatestVersionNumber(ctx, noteID)
	if err != nil {
		t.Fatalf("failed to get latest version number: %v", err)
	}
	switch v := latest.(type) {
	case int64:
		if v != 2 {
			t.Errorf("expected latest version 2, got %d", v)
		}
	default:
		t.Errorf("unexpected type for latest version: %T", latest)
	}
}

func TestLineDiff(t *testing.T) {
	tests := []struct {
		name        string
		old         string
		new         string
		wantAdded   bool
		wantRemoved bool
	}{
		{
			name:      "identical",
			old:       "hello\nworld",
			new:       "hello\nworld",
			wantAdded: false, wantRemoved: false,
		},
		{
			name:      "line added",
			old:       "hello",
			new:       "hello\nworld",
			wantAdded: true, wantRemoved: false,
		},
		{
			name:      "line removed",
			old:       "hello\nworld",
			new:       "hello",
			wantAdded: false, wantRemoved: true,
		},
		{
			name:      "line changed",
			old:       "hello\nworld",
			new:       "hello\nearth",
			wantAdded: true, wantRemoved: true,
		},
		{
			name:      "empty to content",
			old:       "",
			new:       "hello",
			wantAdded: true, wantRemoved: true, // empty string splits to [""], which shows as removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lineDiff(tt.old, tt.new)

			hasAdded := strings.Contains(result, "+ ")
			hasRemoved := strings.Contains(result, "- ")

			if hasAdded != tt.wantAdded {
				t.Errorf("added lines: expected %v, got %v\ndiff:\n%s", tt.wantAdded, hasAdded, result)
			}
			if hasRemoved != tt.wantRemoved {
				t.Errorf("removed lines: expected %v, got %v\ndiff:\n%s", tt.wantRemoved, hasRemoved, result)
			}
		})
	}
}

func TestGetLatestVersionNumber_NoVersions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	noteID := createTestNote(t, "No Versions", "Content", nil)

	verNum, err := getLatestVersionNumber(ctx, noteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verNum != 0 {
		t.Errorf("expected 0 for no versions, got %d", verNum)
	}
}

// ============================================================================
// Daily Notes Tests
// ============================================================================

func TestGetOrCreateDailyNote(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// First call creates the note
	note1, err := getOrCreateDailyNote(ctx, "2026-02-17")
	if err != nil {
		t.Fatalf("failed to create daily note: %v", err)
	}
	if note1.Title != "2026-02-17" {
		t.Errorf("expected title '2026-02-17', got %q", note1.Title)
	}

	// Verify tagged as "daily"
	tags, _ := database.GetTagsForNote(ctx, note1.ID)
	found := false
	for _, tag := range tags {
		if tag.Name == "daily" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'daily' tag on daily note")
	}

	// Second call returns the same note
	note2, err := getOrCreateDailyNote(ctx, "2026-02-17")
	if err != nil {
		t.Fatalf("failed to get daily note: %v", err)
	}
	if note2.ID != note1.ID {
		t.Errorf("expected same note ID %d, got %d", note1.ID, note2.ID)
	}
}

func TestGetOrCreateDailyFolder(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// First call creates folder
	id1, err := getOrCreateDailyFolder(ctx)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if id1 == 0 {
		t.Error("expected non-zero folder ID")
	}

	// Second call returns same folder
	id2, err := getOrCreateDailyFolder(ctx)
	if err != nil {
		t.Fatalf("failed to get folder: %v", err)
	}
	if id2 != id1 {
		t.Errorf("expected same folder ID %d, got %d", id1, id2)
	}
}

// ============================================================================
// Link Health Tests
// ============================================================================

func TestOrphanAndDeadEndDetection(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create notes
	orphanID := createTestNote(t, "Orphan", "No links", nil)
	sourceID := createTestNote(t, "Source", "Links to [[Target]]", nil)
	targetID := createTestNote(t, "Target", "No outgoing links", nil)

	// Create link: source -> target
	err := database.CreateNoteLink(ctx, db.CreateNoteLinkParams{
		SourceNoteID: sourceID,
		TargetNoteID: targetID,
		LinkText:     "Target",
	})
	if err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	// Test orphans (no links in or out)
	orphans, err := database.GetOrphanNotes(ctx)
	if err != nil {
		t.Fatalf("failed to get orphans: %v", err)
	}
	foundOrphan := false
	for _, n := range orphans {
		if n.ID == orphanID {
			foundOrphan = true
		}
		if n.ID == sourceID || n.ID == targetID {
			t.Errorf("linked note %d should not be an orphan", n.ID)
		}
	}
	if !foundOrphan {
		t.Error("expected 'Orphan' note to be detected as orphan")
	}

	// Test deadends (incoming links, no outgoing)
	deadends, err := database.GetDeadEndNotes(ctx)
	if err != nil {
		t.Fatalf("failed to get deadends: %v", err)
	}
	foundDeadend := false
	for _, n := range deadends {
		if n.ID == targetID {
			foundDeadend = true
		}
	}
	if !foundDeadend {
		t.Error("expected 'Target' note to be a dead-end")
	}

	// Test backlinks
	backlinks, err := database.GetBacklinks(ctx, targetID)
	if err != nil {
		t.Fatalf("failed to get backlinks: %v", err)
	}
	if len(backlinks) != 1 {
		t.Errorf("expected 1 backlink, got %d", len(backlinks))
	}
	if len(backlinks) > 0 && backlinks[0].ID != sourceID {
		t.Errorf("expected backlink from source, got %d", backlinks[0].ID)
	}
}

func TestWikilinkRegex(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"[[Note Title]]", []string{"Note Title"}},
		{"Link to [[A]] and [[B]]", []string{"A", "B"}},
		{"No links here", nil},
		{"[[]]", nil}, // empty link
		{"[[Nested [[link]]]]", []string{"Nested [[link"}}, // greedy match
	}

	for _, tt := range tests {
		matches := wikilinkRe.FindAllStringSubmatch(tt.input, -1)
		var got []string
		for _, m := range matches {
			got = append(got, m[1])
		}

		if len(got) != len(tt.want) {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.want, got)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("input %q match %d: expected %q, got %q", tt.input, i, tt.want[i], got[i])
			}
		}
	}
}
