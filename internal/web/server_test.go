package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/abdul-hamid-achik/noted/internal/db"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	queries := db.New(conn)
	return NewServer(queries, conn)
}

func doRequest(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	return rr
}

func parseJSON(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var data map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse response JSON: %v\nbody: %s", err, rr.Body.String())
	}
	return data
}

func TestAPI_CreateNote(t *testing.T) {
	srv := setupTestServer(t)

	rr := doRequest(t, srv, "POST", "/api/notes", createNoteRequest{
		Title:   "Test Note",
		Content: "Test content",
		Tags:    []string{"go", "test"},
	})

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	data := parseJSON(t, rr)
	if data["title"] != "Test Note" {
		t.Errorf("expected title 'Test Note', got %v", data["title"])
	}
}

func TestAPI_ListNotes(t *testing.T) {
	srv := setupTestServer(t)

	// Create some notes
	doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "A", Content: "a"})
	doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "B", Content: "b"})

	rr := doRequest(t, srv, "GET", "/api/notes", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var notes []NoteResponse
	json.Unmarshal(rr.Body.Bytes(), &notes)
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestAPI_GetNote(t *testing.T) {
	srv := setupTestServer(t)

	// Create a note
	createRR := doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "Get Me", Content: "content"})
	createData := parseJSON(t, createRR)
	id := int64(createData["id"].(float64))

	rr := doRequest(t, srv, "GET", "/api/notes/"+strconv.FormatInt(id, 10), nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	data := parseJSON(t, rr)
	if data["title"] != "Get Me" {
		t.Errorf("expected title 'Get Me', got %v", data["title"])
	}
}

func TestAPI_GetNote_NotFound(t *testing.T) {
	srv := setupTestServer(t)

	rr := doRequest(t, srv, "GET", "/api/notes/99999", nil)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAPI_UpdateNote(t *testing.T) {
	srv := setupTestServer(t)

	// Create a note
	createRR := doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "Original", Content: "original"})
	createData := parseJSON(t, createRR)
	id := int64(createData["id"].(float64))

	rr := doRequest(t, srv, "PUT", "/api/notes/"+strconv.FormatInt(id, 10), updateNoteRequest{
		Title:   "Updated",
		Content: "updated content",
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	data := parseJSON(t, rr)
	if data["title"] != "Updated" {
		t.Errorf("expected title 'Updated', got %v", data["title"])
	}
}

func TestAPI_DeleteNote(t *testing.T) {
	srv := setupTestServer(t)

	// Create a note
	createRR := doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "Delete Me", Content: "bye"})
	createData := parseJSON(t, createRR)
	id := int64(createData["id"].(float64))

	rr := doRequest(t, srv, "DELETE", "/api/notes/"+strconv.FormatInt(id, 10), nil)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}

	// Verify not found
	getRR := doRequest(t, srv, "GET", "/api/notes/"+strconv.FormatInt(id, 10), nil)
	if getRR.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", getRR.Code)
	}
}

func TestAPI_SearchNotes(t *testing.T) {
	srv := setupTestServer(t)

	doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "Go Tutorial", Content: "Learn Go"})
	doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "Python Guide", Content: "Learn Python"})

	rr := doRequest(t, srv, "GET", "/api/notes/search?q=Go", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var notes []NoteResponse
	json.Unmarshal(rr.Body.Bytes(), &notes)
	if len(notes) != 1 {
		t.Errorf("expected 1 search result, got %d", len(notes))
	}
}

func TestAPI_SearchNotes_MissingQuery(t *testing.T) {
	srv := setupTestServer(t)

	rr := doRequest(t, srv, "GET", "/api/notes/search", nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAPI_Tags(t *testing.T) {
	srv := setupTestServer(t)

	doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "T", Content: "c", Tags: []string{"go", "test"}})

	rr := doRequest(t, srv, "GET", "/api/tags", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAPI_Folders_CRUD(t *testing.T) {
	srv := setupTestServer(t)

	// Create
	createRR := doRequest(t, srv, "POST", "/api/folders", createFolderRequest{Name: "Projects"})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRR.Code, createRR.Body.String())
	}
	createData := parseJSON(t, createRR)
	id := int64(createData["id"].(float64))

	// List
	listRR := doRequest(t, srv, "GET", "/api/folders", nil)
	if listRR.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", listRR.Code)
	}

	// Get
	getRR := doRequest(t, srv, "GET", "/api/folders/"+strconv.FormatInt(id, 10), nil)
	if getRR.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", getRR.Code)
	}
	getData := parseJSON(t, getRR)
	if getData["name"] != "Projects" {
		t.Errorf("expected name 'Projects', got %v", getData["name"])
	}
}

func TestAPI_Stats(t *testing.T) {
	srv := setupTestServer(t)

	doRequest(t, srv, "POST", "/api/notes", createNoteRequest{Title: "T", Content: "c"})

	rr := doRequest(t, srv, "GET", "/api/stats", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	data := parseJSON(t, rr)
	if data["total_notes"].(float64) != 1 {
		t.Errorf("expected total_notes=1, got %v", data["total_notes"])
	}
}
