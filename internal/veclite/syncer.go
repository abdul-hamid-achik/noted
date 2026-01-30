package veclite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/veclite"
)

const (
	collectionName        = "notes"
	defaultEmbeddingModel = "nomic-embed-text"
	defaultOllamaHost     = "http://localhost:11434"
)

// SemanticResult represents a semantic search result
type SemanticResult struct {
	NoteID int64   `json:"note_id"`
	Score  float64 `json:"score"`
	Title  string  `json:"title"`
}

// OllamaEmbedder implements veclite.Embedder using Ollama API
type OllamaEmbedder struct {
	model     string
	host      string
	dimension int
}

// NewOllamaEmbedder creates a new Ollama embedder
func NewOllamaEmbedder(model, host string) (*OllamaEmbedder, error) {
	e := &OllamaEmbedder{
		model:     model,
		host:      host,
		dimension: 768, // Default for nomic-embed-text
	}

	// Probe dimension by embedding a test string
	vec, err := e.Embed("test")
	if err != nil {
		return nil, fmt.Errorf("failed to probe embedder dimension: %w", err)
	}
	e.dimension = len(vec)

	return e, nil
}

// Embed generates an embedding for a single text
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	reqBody := map[string]any{
		"model":  e.model,
		"prompt": text,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := http.Post(e.host+"/api/embeddings", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	// Convert float64 to float32
	vec := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		vec[i] = float32(v)
	}

	return vec, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *OllamaEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.Embed(text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

// Dimension returns the embedding dimension
func (e *OllamaEmbedder) Dimension() int {
	return e.dimension
}

// Syncer handles synchronization between noted and veclite
type Syncer struct {
	db       *veclite.DB
	embedder *OllamaEmbedder
}

// NewSyncer creates a new veclite syncer
func NewSyncer(dbPath, embeddingModel string) (*Syncer, error) {
	if embeddingModel == "" {
		embeddingModel = defaultEmbeddingModel
	}

	// Open veclite database
	db, err := veclite.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open veclite database: %w", err)
	}

	// Create embedder using Ollama
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = defaultOllamaHost
	} else if !strings.HasPrefix(ollamaHost, "http://") && !strings.HasPrefix(ollamaHost, "https://") {
		// Add http:// prefix if missing
		ollamaHost = "http://" + ollamaHost + ":11434"
	}

	embedder, err := NewOllamaEmbedder(embeddingModel, ollamaHost)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return &Syncer{
		db:       db,
		embedder: embedder,
	}, nil
}

// Close closes the syncer and releases resources
func (s *Syncer) Close() error {
	return s.db.Close()
}

// SyncNote syncs a single note to veclite
func (s *Syncer) SyncNote(id int64, title, content string) error {
	// Create text to embed (title + content)
	text := title + "\n\n" + content

	// Generate embedding
	vector, err := s.embedder.Embed(text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Get or create collection
	coll := s.db.Collection(collectionName)

	// Check if note already exists and delete it first
	existing, err := coll.Find(veclite.Equal("note_id", strconv.FormatInt(id, 10)))
	if err == nil && len(existing) > 0 {
		for _, r := range existing {
			_ = coll.Delete(r.ID)
		}
	}

	// Insert with payload
	payload := map[string]any{
		"note_id": strconv.FormatInt(id, 10),
		"title":   title,
	}

	_, err = coll.InsertDocument(vector, text, payload)
	if err != nil {
		return fmt.Errorf("failed to insert into veclite: %w", err)
	}

	// Sync to disk
	_ = s.db.Sync()

	return nil
}

// SyncAll syncs all unsynced notes from the database
func (s *Syncer) SyncAll(queries *db.Queries) (int, error) {
	ctx := context.Background()
	notes, err := queries.GetUnsynced(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get unsynced notes: %w", err)
	}

	synced := 0
	for _, note := range notes {
		if err := s.SyncNote(note.ID, note.Title, note.Content); err != nil {
			continue // Skip failed notes
		}
		// Mark as synced
		_ = queries.MarkEmbeddingSynced(ctx, note.ID)
		synced++
	}

	return synced, nil
}

// Search performs semantic search on notes
func (s *Syncer) Search(query string, limit int) ([]SemanticResult, error) {
	// Generate query embedding
	vector, err := s.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get collection
	coll, err := s.db.GetCollection(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %w", err)
	}

	// Search
	opts := []veclite.SearchOption{veclite.TopK(limit)}
	results, err := coll.Search(vector, opts...)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to SemanticResult
	output := make([]SemanticResult, 0, len(results))
	for _, r := range results {
		noteIDStr, ok := r.Record.Payload["note_id"].(string)
		if !ok {
			continue
		}
		noteID, err := strconv.ParseInt(noteIDStr, 10, 64)
		if err != nil {
			continue
		}

		title, _ := r.Record.Payload["title"].(string)

		output = append(output, SemanticResult{
			NoteID: noteID,
			Score:  float64(r.Score),
			Title:  title,
		})
	}

	return output, nil
}

// Delete removes a note from the veclite index
func (s *Syncer) Delete(noteID int64) error {
	coll, err := s.db.GetCollection(collectionName)
	if err != nil {
		return nil // Collection doesn't exist, nothing to delete
	}

	// Find and delete records with this note_id
	records, err := coll.Find(veclite.Equal("note_id", strconv.FormatInt(noteID, 10)))
	if err != nil {
		return nil // No records found
	}

	for _, r := range records {
		_ = coll.Delete(r.ID)
	}

	_ = s.db.Sync()
	return nil
}
