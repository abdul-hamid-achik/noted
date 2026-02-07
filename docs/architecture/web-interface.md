# Web Interface Architecture for `noted`

## Overview

This document defines the architecture for adding a web interface to the `noted` CLI knowledge base. The design prioritizes: single-binary distribution, zero-config startup, minimal dependencies, and full feature parity with the CLI and MCP server.

The web UI will be invoked with `noted serve` and opens a local browser-based notepad with vim-mode editing, real-time search, and a monitoring dashboard.

---

## 1. Technology Choices

### Frontend: htmx + Alpine.js + CodeMirror 6

**Why not React/Svelte:** This is a single-binary CLI tool. A Node.js build pipeline, bundler, and NPM dependency tree add enormous complexity for what is fundamentally a CRUD interface over a local SQLite database. The frontend should be embeddable as static files with zero build step required for basic development.

**Why htmx + Alpine.js:**
- **htmx** handles server-driven UI updates (list/search/filter) with HTML-over-the-wire. No JSON serialization/deserialization dance for the 80% of interactions that are "fetch HTML fragment, replace DOM node."
- **Alpine.js** (~15KB) provides lightweight reactivity for client-side state: modal toggles, editor focus management, keyboard shortcut handling, sidebar state.
- Both are single `<script>` tags. No bundler. No node_modules. No build step.

**Why CodeMirror 6 for the editor:**
- First-class vim mode via `@codemirror/vim` (the successor to the CodeMirror 5 vim mode used in production by thousands).
- Markdown syntax highlighting, bracket matching, and line numbers out of the box.
- Tree-shakeable -- we only bundle the extensions we need.
- The CodeMirror bundle (~100KB gzipped) is the only pre-built JS artifact. It gets committed as a minified file in `web/static/vendor/`.

### Backend: Go `net/http` standard library

**Why not gin/echo/fiber:**
- `net/http` in Go 1.22+ has pattern-based routing (`GET /api/notes/{id}`). No need for a router library.
- The `html/template` standard library handles server-side HTML rendering for htmx responses.
- Zero new dependencies. The Go binary stays lean.

### Templating: Go `html/template` with a component pattern

Templates live in `web/templates/` and are embedded into the binary. A base layout with named blocks (`{{block "content" .}}`) enables component-style composition without a framework.

---

## 2. Embedding Strategy

```
web/
  static/
    css/
      style.css          # Tailwind-generated (committed, not built at runtime)
    js/
      app.js             # Alpine.js + htmx initialization, keyboard shortcuts
    vendor/
      htmx.min.js        # v2.0 (~14KB gzip)
      alpine.min.js      # v3.x (~15KB gzip)
      codemirror/
        editor.bundle.js # Pre-built CM6 bundle with vim + markdown extensions
        editor.css
  templates/
    layout.html          # Base layout with nav, sidebar, content area
    partials/
      note_list.html     # htmx partial: note list items
      note_card.html     # htmx partial: single note card
      note_editor.html   # Full editor view (CodeMirror mount point)
      search_results.html
      tag_cloud.html
      dashboard.html     # Monitoring dashboard
      toast.html         # Notification toasts
    pages/
      index.html         # Main notes view
      note.html          # Single note view/edit
      dashboard.html     # Dashboard page
```

Embedding in Go:

```go
// internal/web/embed.go
package web

import "embed"

//go:embed static templates
var Assets embed.FS
```

The `embed.FS` is used directly by the HTTP file server and template engine. No extraction to disk. No temp files. The binary is fully self-contained.

**CSS strategy:** Use Tailwind CSS, but generate the CSS at development time and commit the output file. The production binary never runs PostCSS. A `Makefile` target handles regeneration during development:

```makefile
web-css:
	npx tailwindcss -i web/static/css/input.css -o web/static/css/style.css --minify
```

---

## 3. Server Design: `noted serve`

### Command

```go
// cmd/serve.go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the web interface",
    Long:  "Start a local web server for the noted knowledge base.",
    RunE:  runServe,
}

func init() {
    rootCmd.AddCommand(serveCmd)
    serveCmd.Flags().StringP("addr", "a", "127.0.0.1:2400", "Listen address")
    serveCmd.Flags().BoolP("open", "o", true, "Open browser automatically")
}
```

### Server struct and initialization

```go
// internal/web/server.go
package web

import (
    "context"
    "database/sql"
    "html/template"
    "net/http"
    "sync"

    "github.com/abdul-hamid-achik/noted/internal/db"
    "github.com/abdul-hamid-achik/noted/internal/mcp"
)

type Server struct {
    queries   *db.Queries
    conn      *sql.DB
    syncer    mcp.Syncer
    templates *template.Template
    hub       *WSHub        // WebSocket hub for live updates
    mu        sync.RWMutex  // Protects template reloading in dev mode
}

func NewServer(conn *sql.DB, queries *db.Queries, syncer mcp.Syncer) *Server {
    s := &Server{
        queries: queries,
        conn:    conn,
        syncer:  syncer,
        hub:     NewWSHub(),
    }
    s.templates = s.parseTemplates()
    return s
}
```

### SQLite connection sharing

The critical design constraint: SQLite allows concurrent reads but only one writer. The existing `db.Open()` returns a `*sql.DB` which is a connection pool. This works correctly because:

1. `database/sql` handles connection pooling automatically.
2. `modernc.org/sqlite` supports WAL mode, which allows concurrent readers with a single writer.
3. The web server shares the same `*sql.DB` connection pool that the CLI uses.

We add WAL mode to `db.Open()`:

```go
// internal/db/open.go -- add after foreign keys pragma
if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
    db.Close()
    return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
}
```

The `runServe` function in `cmd/serve.go` reuses the `conn` and `database` variables that `PersistentPreRunE` already initializes, exactly like `cmd/mcp.go` does today.

### Router setup

```go
func (s *Server) Routes() http.Handler {
    mux := http.NewServeMux()

    // Static files (from embed.FS)
    mux.Handle("GET /static/", http.FileServerFS(Assets))

    // Pages
    mux.HandleFunc("GET /", s.handleIndex)
    mux.HandleFunc("GET /notes/{id}", s.handleNoteView)
    mux.HandleFunc("GET /dashboard", s.handleDashboard)

    // htmx partials (return HTML fragments)
    mux.HandleFunc("GET /partials/notes", s.handleNoteList)
    mux.HandleFunc("GET /partials/notes/{id}", s.handleNotePartial)
    mux.HandleFunc("GET /partials/search", s.handleSearchPartial)
    mux.HandleFunc("GET /partials/tags", s.handleTagsPartial)

    // REST API (JSON, used by CodeMirror save and programmatic access)
    mux.HandleFunc("GET /api/notes", s.apiListNotes)
    mux.HandleFunc("POST /api/notes", s.apiCreateNote)
    mux.HandleFunc("GET /api/notes/{id}", s.apiGetNote)
    mux.HandleFunc("PUT /api/notes/{id}", s.apiUpdateNote)
    mux.HandleFunc("DELETE /api/notes/{id}", s.apiDeleteNote)
    mux.HandleFunc("GET /api/search", s.apiSearch)
    mux.HandleFunc("GET /api/search/semantic", s.apiSemanticSearch)
    mux.HandleFunc("GET /api/tags", s.apiListTags)
    mux.HandleFunc("POST /api/notes/{id}/tags", s.apiAddTag)
    mux.HandleFunc("DELETE /api/notes/{id}/tags/{tag}", s.apiRemoveTag)

    // Memory API
    mux.HandleFunc("POST /api/memories", s.apiRemember)
    mux.HandleFunc("GET /api/memories/recall", s.apiRecall)
    mux.HandleFunc("POST /api/memories/forget", s.apiForget)

    // Dashboard API
    mux.HandleFunc("GET /api/stats", s.apiStats)

    // WebSocket
    mux.HandleFunc("GET /ws", s.handleWebSocket)

    return mux
}
```

---

## 4. REST API Design

All JSON API endpoints live under `/api/`. htmx endpoints live under `/partials/` and return HTML.

### Notes

| Method | Path | Description | Query Params |
|--------|------|-------------|-------------|
| GET | `/api/notes` | List notes | `limit`, `offset`, `tag` |
| POST | `/api/notes` | Create note | Body: `{title, content, tags[], ttl?, source?, source_ref?}` |
| GET | `/api/notes/{id}` | Get note with tags | |
| PUT | `/api/notes/{id}` | Update note | Body: `{title?, content?, tags?}` |
| DELETE | `/api/notes/{id}` | Delete note | |

### Search

| Method | Path | Description | Query Params |
|--------|------|-------------|-------------|
| GET | `/api/search` | Text search | `q`, `limit` |
| GET | `/api/search/semantic` | Semantic search | `q`, `limit` |

### Tags

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/tags` | List tags with counts |
| POST | `/api/notes/{id}/tags` | Add tag to note. Body: `{name}` |
| DELETE | `/api/notes/{id}/tags/{tag}` | Remove tag from note |

### Memories

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/memories` | Create memory (remember) |
| GET | `/api/memories/recall` | Recall memories. Query: `q`, `limit`, `category` |
| POST | `/api/memories/forget` | Forget memories. Body: `{older_than_days?, importance_below?, category?, dry_run?}` |

### Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/stats` | Full stats payload (see section 6) |

### Response format

```json
{
  "data": { ... },
  "error": null
}
```

On error:

```json
{
  "data": null,
  "error": "note #42 not found"
}
```

HTTP status codes: 200 (ok), 201 (created), 400 (bad request), 404 (not found), 500 (internal).

---

## 5. Vim-like Editor Integration

### Architecture

The editor is CodeMirror 6 with the vim extension. It communicates with the backend through the JSON API, not htmx. This is because the editor needs programmatic control over save timing (debounced autosave, explicit `:w` save).

```
+-------------------+       JSON API        +------------------+
|   CodeMirror 6    | <-------------------> |  Go HTTP Server  |
|   + vim mode      |   PUT /api/notes/{id} |                  |
|   + markdown ext  |   GET /api/notes/{id} |  db.Queries      |
|   + keybindings   | ---WebSocket--------> |  + veclite sync  |
+-------------------+       events          +------------------+
```

### Pre-built editor bundle

The CodeMirror bundle is built once and committed:

```javascript
// web/static/vendor/codemirror/build.js (development only, not embedded)
import { EditorView, basicSetup } from "codemirror"
import { vim } from "@replit/codemirror-vim"
import { markdown } from "@codemirror/lang-markdown"
import { oneDark } from "@codemirror/theme-one-dark"

export { EditorView, basicSetup, vim, markdown, oneDark }
```

Built with esbuild (one-time, committed output):

```bash
npx esbuild web/static/vendor/codemirror/build.js \
  --bundle --format=esm --minify \
  --outfile=web/static/vendor/codemirror/editor.bundle.js
```

### Editor initialization (in Alpine.js component)

```javascript
// web/static/js/app.js
document.addEventListener('alpine:init', () => {
    Alpine.data('noteEditor', (noteId) => ({
        editor: null,
        dirty: false,
        saving: false,
        lastSaved: null,

        init() {
            this.editor = new EditorView({
                extensions: [
                    basicSetup,
                    vim(),
                    markdown(),
                    oneDark,
                    EditorView.updateListener.of((update) => {
                        if (update.docChanged) {
                            this.dirty = true;
                            this.debouncedSave();
                        }
                    }),
                ],
                parent: this.$refs.editorMount,
            });
            this.loadNote();
        },

        async loadNote() {
            const resp = await fetch(`/api/notes/${noteId}`);
            const { data } = await resp.json();
            this.editor.dispatch({
                changes: { from: 0, to: this.editor.state.doc.length, insert: data.content }
            });
            this.dirty = false;
        },

        debouncedSave: debounce(function() { this.save(); }, 1000),

        async save() {
            if (!this.dirty || this.saving) return;
            this.saving = true;
            const content = this.editor.state.doc.toString();
            await fetch(`/api/notes/${noteId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ content }),
            });
            this.dirty = false;
            this.saving = false;
            this.lastSaved = new Date();
        },
    }));
});
```

### Vim command extensions

Custom ex-commands registered via the CM6 vim API:

| Command | Action |
|---------|--------|
| `:w` | Save current note via `PUT /api/notes/{id}` |
| `:q` | Close editor, return to note list |
| `:wq` | Save and close |
| `:tags` | Open tag picker (Alpine.js modal) |
| `:search <query>` | Trigger search, show results in split pane |
| `:new` | Create new note, open in editor |

---

## 6. Monitoring Dashboard

### Stats endpoint: `GET /api/stats`

```go
type DashboardStats struct {
    Notes       NoteStats       `json:"notes"`
    Tags        TagStats        `json:"tags"`
    Memories    MemoryStats     `json:"memories"`
    Search      SearchStats     `json:"search"`
    DB          DBStats         `json:"db"`
    Activity    []ActivityEntry `json:"recent_activity"`
}

type NoteStats struct {
    Total       int64 `json:"total"`
    CreatedToday int64 `json:"created_today"`
    CreatedWeek  int64 `json:"created_week"`
    WithTTL     int64 `json:"with_ttl"`
    Expired     int64 `json:"expired"`
}

type TagStats struct {
    Total      int64            `json:"total"`
    TopTags    []TagCount       `json:"top_tags"`  // Top 10 by note count
    Unused     int64            `json:"unused"`
}

type MemoryStats struct {
    Total          int64              `json:"total"`
    ByCategory     map[string]int64   `json:"by_category"`
    ByImportance   map[int]int64      `json:"by_importance"`
    ExpiringIn24h  int64              `json:"expiring_in_24h"`
}

type SearchStats struct {
    SemanticEnabled bool  `json:"semantic_enabled"`
    IndexedNotes    int64 `json:"indexed_notes"`
    UnsyncedNotes   int64 `json:"unsynced_notes"`
}

type DBStats struct {
    Path       string `json:"path"`
    SizeBytes  int64  `json:"size_bytes"`
    WALEnabled bool   `json:"wal_enabled"`
    SchemaVer  int    `json:"schema_version"`
}

type ActivityEntry struct {
    Type      string `json:"type"`      // "created", "updated", "deleted"
    NoteID    int64  `json:"note_id"`
    Title     string `json:"title"`
    Timestamp string `json:"timestamp"`
}
```

These stats require a few new SQL queries (added to `query.sql`):

```sql
-- name: CountNotes :one
SELECT COUNT(*) FROM notes;

-- name: CountNotesSince :one
SELECT COUNT(*) FROM notes WHERE created_at >= ?;

-- name: CountNotesWithTTL :one
SELECT COUNT(*) FROM notes WHERE expires_at IS NOT NULL;

-- name: CountExpiredNotes :one
SELECT COUNT(*) FROM notes WHERE expires_at IS NOT NULL AND expires_at < datetime('now');

-- name: CountUnsyncedNotes :one
SELECT COUNT(*) FROM notes WHERE embedding_synced = FALSE;

-- name: CountTags :one
SELECT COUNT(*) FROM tags;

-- name: CountUnusedTags :one
SELECT COUNT(*) FROM tags WHERE id NOT IN (SELECT DISTINCT tag_id FROM note_tags);

-- name: GetRecentActivity :many
SELECT id, title, updated_at FROM notes ORDER BY updated_at DESC LIMIT ?;
```

### Dashboard rendering

The dashboard page loads via a full page render, then individual panels auto-refresh via htmx polling:

```html
<div hx-get="/partials/dashboard/stats" hx-trigger="every 10s" hx-swap="innerHTML">
    <!-- Stats panels auto-refresh -->
</div>
```

---

## 7. Real-time Features: WebSocket

### WebSocket Hub

A simple pub/sub hub that broadcasts events to all connected clients.

```go
// internal/web/ws.go
package web

import (
    "encoding/json"
    "sync"

    "golang.org/x/net/websocket"
)

type WSEvent struct {
    Type string      `json:"type"` // "note.created", "note.updated", "note.deleted", "sync.completed"
    Data interface{} `json:"data"`
}

type WSHub struct {
    clients map[*websocket.Conn]bool
    mu      sync.RWMutex
}

func NewWSHub() *WSHub {
    return &WSHub{clients: make(map[*websocket.Conn]bool)}
}

func (h *WSHub) Register(conn *websocket.Conn) {
    h.mu.Lock()
    h.clients[conn] = true
    h.mu.Unlock()
}

func (h *WSHub) Unregister(conn *websocket.Conn) {
    h.mu.Lock()
    delete(h.clients, conn)
    h.mu.Unlock()
}

func (h *WSHub) Broadcast(event WSEvent) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    data, _ := json.Marshal(event)
    for conn := range h.clients {
        go func(c *websocket.Conn) {
            _ = websocket.Message.Send(c, string(data))
        }(conn)
    }
}
```

**Note:** Use `golang.org/x/net/websocket` instead of `gorilla/websocket` -- it is maintained by the Go team and avoids another third-party dependency. Alternatively, since this is local-only, we can use Server-Sent Events (SSE) which requires zero dependencies (just `http.Flusher`). SSE is likely the better choice since we only need server-to-client pushes.

### SSE Alternative (Recommended)

```go
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    ch := s.hub.Subscribe()
    defer s.hub.Unsubscribe(ch)

    for {
        select {
        case event := <-ch:
            data, _ := json.Marshal(event)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

This eliminates the websocket dependency entirely. htmx has built-in SSE support via `hx-ext="sse"`.

### Events emitted

| Event | Trigger | Data |
|-------|---------|------|
| `note.created` | POST /api/notes | `{id, title}` |
| `note.updated` | PUT /api/notes/{id} | `{id, title}` |
| `note.deleted` | DELETE /api/notes/{id} | `{id}` |
| `tag.added` | POST /api/notes/{id}/tags | `{note_id, tag}` |
| `tag.removed` | DELETE /api/notes/{id}/tags/{tag} | `{note_id, tag}` |
| `sync.completed` | After veclite sync | `{synced, failed}` |

### Client-side usage with htmx SSE

```html
<div hx-ext="sse" sse-connect="/events">
    <div sse-swap="note.created" hx-get="/partials/notes" hx-trigger="sse:note.created">
        <!-- Note list auto-refreshes when a note is created -->
    </div>
</div>
```

---

## 8. Authentication

**No authentication.** This is a local-only development tool. The server binds to `127.0.0.1` by default, making it unreachable from the network.

If the user explicitly passes `--addr 0.0.0.0:2400`, we print a warning:

```
WARNING: Server is listening on all interfaces. No authentication is enabled.
Only use this on trusted networks.
```

Future consideration: if remote access becomes a requirement, add a `--token` flag that generates a random bearer token and requires it in an `Authorization` header. But this is out of scope for v1.

---

## 9. Package Structure

```
noted/
  cmd/
    serve.go                  # `noted serve` command
  internal/
    web/
      embed.go                # //go:embed static templates
      server.go               # Server struct, NewServer, Routes
      handlers_pages.go       # Full page handlers (index, note view, dashboard)
      handlers_partials.go    # htmx partial handlers
      handlers_api.go         # JSON API handlers
      handlers_events.go      # SSE event stream
      hub.go                  # Event hub (pub/sub for SSE)
      middleware.go           # Logging, CORS (dev only), request ID
      templates.go            # Template parsing and rendering helpers
    db/                       # (existing, add new count queries)
    mcp/                      # (existing, unchanged)
    memory/                   # (existing, unchanged)
    veclite/                  # (existing, unchanged)
    config/                   # (existing, add web-specific config)
  web/
    static/
      css/
        style.css
        input.css             # Tailwind source (not embedded)
      js/
        app.js                # Alpine.js components, htmx config, keyboard shortcuts
        editor.js             # CodeMirror initialization and vim commands
      vendor/
        htmx.min.js
        alpine.min.js
        codemirror/
          editor.bundle.js
          editor.css
    templates/
      layout.html
      pages/
        index.html
        note.html
        dashboard.html
      partials/
        note_list.html
        note_card.html
        note_editor.html
        search_results.html
        tag_cloud.html
        dashboard_stats.html
        toast.html
  Makefile                    # (add web-css, web-bundle targets)
```

---

## 10. Implementation Sequence

### Phase 1: Core server + note CRUD (estimated: first milestone)
1. Add WAL mode to `db.Open()`
2. Create `internal/web/` package with `Server`, `Routes`, embedded templates
3. Create `cmd/serve.go`
4. Implement note list page (htmx) and JSON API for notes
5. Basic layout with sidebar navigation

### Phase 2: Editor + search
1. Integrate CodeMirror 6 with vim mode
2. Implement `:w`, `:q`, `:wq` vim commands
3. Add search-as-you-type via htmx (`hx-trigger="keyup changed delay:300ms"`)
4. Semantic search integration (when veclite is available)

### Phase 3: Real-time + dashboard
1. Add SSE event hub
2. Broadcast note CRUD events
3. Build dashboard stats endpoint and page
4. Add new count queries to sqlc

### Phase 4: Memory UI + polish
1. Memory recall/remember/forget UI
2. Tag management UI
3. Keyboard shortcut overlay (press `?`)
4. Dark/light theme toggle
5. Mobile-responsive layout adjustments

---

## 11. New Dependencies

| Dependency | Purpose | Size Impact |
|-----------|---------|-------------|
| `golang.org/x/net` | WebSocket (if not using SSE) | Already indirect dep via oauth2 |
| None (if SSE) | SSE uses only stdlib | Zero |

Frontend vendor files (committed, embedded):
- htmx.min.js: ~14KB gzip
- alpine.min.js: ~15KB gzip
- CodeMirror bundle: ~100KB gzip
- Tailwind CSS output: ~10-30KB gzip

**Total binary size increase: ~160KB** (compressed static assets embedded in the Go binary).

---

## 12. Key Design Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Frontend framework | htmx + Alpine.js | No build step, embeddable, server-driven |
| Editor | CodeMirror 6 + vim | Best vim emulation, tree-shakeable, active maintenance |
| HTTP framework | `net/http` stdlib | Go 1.22+ routing is sufficient, zero deps |
| Templating | `html/template` | Stdlib, secure by default, fast |
| Real-time | SSE (not WebSocket) | Unidirectional is sufficient, zero deps |
| CSS | Tailwind (pre-built) | Utility-first, committed output, no runtime |
| Auth | None (local-only) | `127.0.0.1` binding, warning on `0.0.0.0` |
| SQLite concurrency | WAL mode + shared pool | Concurrent reads, serialized writes via `database/sql` |
| Asset embedding | `embed.FS` | Single binary, no extraction needed |
| API style | Dual: htmx partials + JSON REST | htmx for page navigation, JSON for editor saves |
