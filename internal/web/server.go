package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/db"
)

// Server is the HTTP server for the noted web interface.
type Server struct {
	queries *db.Queries
	conn    *sql.DB
	logger  *slog.Logger
	clients map[chan Event]struct{}
	mu      sync.RWMutex
}

// Event represents a server-sent event for live updates.
type Event struct {
	Type string      `json:"type"`
	Data any `json:"data"`
}

// NoteResponse is a note with its tags included.
type NoteResponse struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Tags      []db.Tag   `json:"tags"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Source    *string    `json:"source,omitempty"`
	SourceRef *string    `json:"source_ref,omitempty"`
}

// Stats holds aggregate statistics about the database.
type Stats struct {
	TotalNotes   int64  `json:"total_notes"`
	TotalTags    int64  `json:"total_tags"`
	UnsyncedNotes int64 `json:"unsynced_notes"`
	DBSizeBytes  int64  `json:"db_size_bytes"`
	DBSize       string `json:"db_size"`
}

type createNoteRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type updateNoteRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// NewServer creates a new web server.
func NewServer(queries *db.Queries, conn *sql.DB) *Server {
	return &Server{
		queries: queries,
		conn:    conn,
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		clients: make(map[chan Event]struct{}),
	}
}

// Run starts the HTTP server and blocks until shutdown.
func (s *Server) Run(ctx context.Context, addr string) error {
	// Enable WAL mode for better concurrent read performance
	if _, err := s.conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		s.logger.Warn("failed to enable WAL mode", "error", err)
	}
	if _, err := s.conn.Exec("PRAGMA busy_timeout=5000"); err != nil {
		s.logger.Warn("failed to set busy timeout", "error", err)
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on signals
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// API routes - more specific paths first
	mux.HandleFunc("/api/notes/search", s.handleSearchNotes)
	mux.HandleFunc("/api/notes/", s.handleNoteByID)
	mux.HandleFunc("/api/notes", s.handleNotes)
	mux.HandleFunc("/api/tags", s.handleListTags)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/events", s.handleSSE)

	// Serve embedded frontend for all other routes
	mux.Handle("/", s.frontendHandler())

	return s.corsMiddleware(mux)
}

func (s *Server) frontendHandler() http.Handler {
	frontend, err := DistFS()
	if err != nil {
		s.logger.Error("failed to load frontend", "error", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend not available", http.StatusInternalServerError)
		})
	}

	fileServer := http.FileServer(http.FS(frontend))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly first
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Check if the file exists in the embedded FS
		if f, err := frontend.Open(path); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// For SPA: serve index.html for all non-file routes
		if !strings.Contains(path, ".") {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found
		http.NotFound(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- JSON helpers ---

func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to encode response", "error", err)
	}
}

func (s *Server) readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, map[string]string{"error": msg})
}

// --- Note conversion ---

func noteToResponse(n db.Note, tags []db.Tag) NoteResponse {
	resp := NoteResponse{
		ID:      n.ID,
		Title:   n.Title,
		Content: n.Content,
		Tags:    tags,
	}
	if n.CreatedAt.Valid {
		resp.CreatedAt = &n.CreatedAt.Time
	}
	if n.UpdatedAt.Valid {
		resp.UpdatedAt = &n.UpdatedAt.Time
	}
	if n.ExpiresAt.Valid {
		resp.ExpiresAt = &n.ExpiresAt.Time
	}
	if n.Source.Valid {
		resp.Source = &n.Source.String
	}
	if n.SourceRef.Valid {
		resp.SourceRef = &n.SourceRef.String
	}
	return resp
}

func (s *Server) noteWithTags(ctx context.Context, n db.Note) (NoteResponse, error) {
	tags, err := s.queries.GetTagsForNote(ctx, n.ID)
	if err != nil {
		return NoteResponse{}, err
	}
	if tags == nil {
		tags = []db.Tag{}
	}
	return noteToResponse(n, tags), nil
}

func (s *Server) notesWithTags(ctx context.Context, notes []db.Note) ([]NoteResponse, error) {
	result := make([]NoteResponse, 0, len(notes))
	for _, n := range notes {
		resp, err := s.noteWithTags(ctx, n)
		if err != nil {
			return nil, err
		}
		result = append(result, resp)
	}
	return result, nil
}

// --- Tag helpers ---

func (s *Server) syncTags(ctx context.Context, noteID int64, tagNames []string) error {
	if err := s.queries.RemoveAllTagsFromNote(ctx, noteID); err != nil {
		return err
	}
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tag, err := s.queries.CreateTag(ctx, name)
		if err != nil {
			return err
		}
		if err := s.queries.AddTagToNote(ctx, db.AddTagToNoteParams{
			NoteID: noteID,
			TagID:  tag.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// --- SSE ---

func (s *Server) broadcast(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- event:
		default:
			// Client too slow, skip
		}
	}
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan Event, 16)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
		close(ch)
	}()

	// Send initial connected event
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			data, err := json.Marshal(event)
			if err != nil {
				s.logger.Error("failed to marshal SSE event", "error", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// --- Handlers ---

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListNotes(w, r)
	case http.MethodPost:
		s.handleCreateNote(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleListNotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	// Filter by tag
	if tag := q.Get("tag"); tag != "" {
		notes, err := s.queries.GetNotesByTagName(ctx, tag)
		if err != nil {
			s.logger.Error("failed to get notes by tag", "error", err)
			s.writeError(w, http.StatusInternalServerError, "failed to list notes")
			return
		}
		result, err := s.notesWithTags(ctx, notes)
		if err != nil {
			s.logger.Error("failed to get tags for notes", "error", err)
			s.writeError(w, http.StatusInternalServerError, "failed to list notes")
			return
		}
		s.writeJSON(w, http.StatusOK, result)
		return
	}

	limit := int64(50)
	if v := q.Get("limit"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := int64(0)
	if v := q.Get("offset"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	notes, err := s.queries.ListNotes(ctx, db.ListNotesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		s.logger.Error("failed to list notes", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list notes")
		return
	}

	result, err := s.notesWithTags(ctx, notes)
	if err != nil {
		s.logger.Error("failed to get tags for notes", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list notes")
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createNoteRequest
	if err := s.readJSON(r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		s.writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	note, err := s.queries.CreateNote(ctx, db.CreateNoteParams{
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		s.logger.Error("failed to create note", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create note")
		return
	}

	if err := s.syncTags(ctx, note.ID, req.Tags); err != nil {
		s.logger.Error("failed to sync tags", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to add tags")
		return
	}

	resp, err := s.noteWithTags(ctx, note)
	if err != nil {
		s.logger.Error("failed to get tags", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to get note")
		return
	}

	s.broadcast(Event{Type: "note_created", Data: resp})
	s.writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleNoteByID(w http.ResponseWriter, r *http.Request) {
	// Parse ID from URL: /api/notes/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	if idStr == "" {
		s.writeError(w, http.StatusBadRequest, "note ID required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid note ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetNote(w, r, id)
	case http.MethodPut:
		s.handleUpdateNote(w, r, id)
	case http.MethodDelete:
		s.handleDeleteNote(w, r, id)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request, id int64) {
	ctx := r.Context()

	note, err := s.queries.GetNote(ctx, id)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		s.logger.Error("failed to get note", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to get note")
		return
	}

	resp, err := s.noteWithTags(ctx, note)
	if err != nil {
		s.logger.Error("failed to get tags", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to get note")
		return
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateNote(w http.ResponseWriter, r *http.Request, id int64) {
	ctx := r.Context()

	var req updateNoteRequest
	if err := s.readJSON(r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		s.writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	note, err := s.queries.UpdateNote(ctx, db.UpdateNoteParams{
		ID:      id,
		Title:   req.Title,
		Content: req.Content,
	})
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		s.logger.Error("failed to update note", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to update note")
		return
	}

	if err := s.syncTags(ctx, note.ID, req.Tags); err != nil {
		s.logger.Error("failed to sync tags", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to update tags")
		return
	}

	resp, err := s.noteWithTags(ctx, note)
	if err != nil {
		s.logger.Error("failed to get tags", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to get note")
		return
	}

	s.broadcast(Event{Type: "note_updated", Data: resp})
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request, id int64) {
	ctx := r.Context()

	// Check existence first
	_, err := s.queries.GetNote(ctx, id)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		s.logger.Error("failed to get note", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete note")
		return
	}

	if err := s.queries.DeleteNote(ctx, id); err != nil {
		s.logger.Error("failed to delete note", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete note")
		return
	}

	s.broadcast(Event{Type: "note_deleted", Data: map[string]int64{"id": id}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSearchNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	q := r.URL.Query()
	query := q.Get("q")
	if query == "" {
		s.writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := int64(20)
	if v := q.Get("limit"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	pattern := "%" + query + "%"
	notes, err := s.queries.SearchNotesContent(ctx, db.SearchNotesContentParams{
		Content: pattern,
		Title:   pattern,
		Limit:   limit,
	})
	if err != nil {
		s.logger.Error("failed to search notes", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to search notes")
		return
	}

	result, err := s.notesWithTags(ctx, notes)
	if err != nil {
		s.logger.Error("failed to get tags for notes", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to search notes")
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tags, err := s.queries.GetTagsWithCount(r.Context())
	if err != nil {
		s.logger.Error("failed to list tags", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list tags")
		return
	}
	if tags == nil {
		tags = []db.GetTagsWithCountRow{}
	}
	s.writeJSON(w, http.StatusOK, tags)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	allNotes, err := s.queries.GetAllNotes(ctx)
	if err != nil {
		s.logger.Error("failed to count notes", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	allTags, err := s.queries.ListTags(ctx)
	if err != nil {
		s.logger.Error("failed to count tags", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	unsynced, err := s.queries.GetUnsynced(ctx)
	if err != nil {
		s.logger.Error("failed to count unsynced", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	// Get DB file size
	var dbSize int64
	var dbPath string
	row := s.conn.QueryRowContext(ctx, "PRAGMA database_list")
	var seq int
	var name string
	if err := row.Scan(&seq, &name, &dbPath); err == nil {
		if info, err := os.Stat(dbPath); err == nil {
			dbSize = info.Size()
		}
	}

	stats := Stats{
		TotalNotes:    int64(len(allNotes)),
		TotalTags:     int64(len(allTags)),
		UnsyncedNotes: int64(len(unsynced)),
		DBSizeBytes:   dbSize,
		DBSize:        formatBytes(dbSize),
	}
	s.writeJSON(w, http.StatusOK, stats)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

