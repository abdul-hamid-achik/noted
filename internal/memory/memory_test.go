package memory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
)

func setupMemoryTestDB(t *testing.T) (*db.Queries, *db.Note, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	queries := db.New(conn)
	cleanup := func() { _ = conn.Close() }
	return queries, nil, cleanup
}

func TestRemember_Basic(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mem, err := Remember(ctx, queries, nil, RememberInput{
		Content: "Test memory content",
	})
	if err != nil {
		t.Fatalf("Remember failed: %v", err)
	}

	if mem.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if mem.Category != "fact" {
		t.Errorf("expected default category 'fact', got %q", mem.Category)
	}
	if mem.Importance != 3 {
		t.Errorf("expected default importance 3, got %d", mem.Importance)
	}
}

func TestRemember_EmptyContent(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := Remember(ctx, queries, nil, RememberInput{})
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestRemember_WithTTL(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mem, err := Remember(ctx, queries, nil, RememberInput{
		Content: "Expires soon",
		TTL:     24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Remember failed: %v", err)
	}

	if mem.ExpiresAt.IsZero() {
		t.Error("expected non-zero ExpiresAt")
	}
	if time.Until(mem.ExpiresAt) < 23*time.Hour {
		t.Error("ExpiresAt should be approximately 24h in the future")
	}
}

func TestRemember_CustomImportance(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mem, err := Remember(ctx, queries, nil, RememberInput{
		Content:    "Important",
		Importance: 5,
	})
	if err != nil {
		t.Fatalf("Remember failed: %v", err)
	}

	if mem.Importance != 5 {
		t.Errorf("expected importance 5, got %d", mem.Importance)
	}
}

func TestRecall_Basic(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, _ = Remember(ctx, queries, nil, RememberInput{
		Content:  "Go is a programming language",
		Title:    "Go facts",
		Category: "fact",
	})

	result, err := Recall(ctx, queries, nil, nil, RecallInput{
		Query: "Go",
	})
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if result.Count == 0 {
		t.Error("expected at least 1 memory recalled")
	}
	if result.Method != "keyword" {
		t.Errorf("expected method 'keyword', got %q", result.Method)
	}
}

func TestRecall_EmptyQuery(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := Recall(ctx, queries, nil, nil, RecallInput{})
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestRecall_CategoryFilter(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, _ = Remember(ctx, queries, nil, RememberInput{
		Content:  "User likes dark mode",
		Category: "user-pref",
	})
	_, _ = Remember(ctx, queries, nil, RememberInput{
		Content:  "Project uses Go",
		Category: "project",
	})

	result, err := Recall(ctx, queries, nil, nil, RecallInput{
		Query:    "Go dark",
		Category: "user-pref",
	})
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	for _, mem := range result.Memories {
		if mem.Category != "user-pref" {
			t.Errorf("expected category 'user-pref', got %q", mem.Category)
		}
	}
}

func TestForget_ByID(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mem, _ := Remember(ctx, queries, nil, RememberInput{Content: "To forget"})

	result, err := Forget(ctx, queries, nil, ForgetInput{
		ID:     mem.ID,
		DryRun: false,
	})
	if err != nil {
		t.Fatalf("Forget failed: %v", err)
	}

	if result.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", result.Deleted)
	}
}

func TestForget_DryRun(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mem, _ := Remember(ctx, queries, nil, RememberInput{Content: "To forget"})

	result, err := Forget(ctx, queries, nil, ForgetInput{
		ID:     mem.ID,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Forget failed: %v", err)
	}

	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if result.WouldDelete != 1 {
		t.Errorf("expected WouldDelete=1, got %d", result.WouldDelete)
	}

	// Note should still exist
	_, err = queries.GetNote(ctx, mem.ID)
	if err != nil {
		t.Error("note should not be deleted in dry run")
	}
}

func TestForget_NonMemoryNote(t *testing.T) {
	queries, _, cleanup := setupMemoryTestDB(t)
	defer cleanup()

	ctx := context.Background()
	// Create a regular note (not a memory)
	note, _ := queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   "Regular note",
		Content: "Not a memory",
	})

	_, err := Forget(ctx, queries, nil, ForgetInput{
		ID:     note.ID,
		DryRun: false,
	})
	if err == nil {
		t.Error("expected error when trying to forget a non-memory note")
	}
}

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		category string
		valid    bool
	}{
		{"user-pref", true},
		{"project", true},
		{"decision", true},
		{"fact", true},
		{"todo", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsValidCategory(tt.category)
		if got != tt.valid {
			t.Errorf("IsValidCategory(%q) = %v, want %v", tt.category, got, tt.valid)
		}
	}
}
