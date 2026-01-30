package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaEmbedder implements the Embedder interface using Ollama's embedding API
type OllamaEmbedder struct {
	baseURL string
	model   string
	dim     int
	client  *http.Client
}

// NewOllamaEmbedder creates a new Ollama embedder
func NewOllamaEmbedder(baseURL, model string, dimension int) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL: baseURL,
		model:   model,
		dim:     dimension,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed generates an embedding vector for the given text
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	reqBody, err := json.Marshal(map[string]string{
		"model":  e.model,
		"prompt": text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := e.client.Post(e.baseURL+"/api/embeddings", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned from Ollama")
	}

	return result.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *OllamaEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		vectors[i] = vec
	}
	return vectors, nil
}

// Dimension returns the embedding dimension
func (e *OllamaEmbedder) Dimension() int {
	return e.dim
}

// IsAvailable checks if Ollama is running and the model is available
func (e *OllamaEmbedder) IsAvailable() bool {
	resp, err := e.client.Get(e.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
