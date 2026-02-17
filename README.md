# noted

A fast, lightweight CLI knowledge base for capturing and organizing notes from your terminal. Includes an MCP server for AI agent integration and an embedded web interface.

## Features

- **Quick capture** - Create notes with titles, content, and tags in one command
- **Rich tagging** - Organize notes with multiple tags, view tag statistics
- **Full-text search** - Find notes by searching titles and content
- **Daily notes** - Obsidian-style daily notes with auto-tagging and folder organization
- **Templates** - Create reusable note templates with variable interpolation
- **Version history** - Automatic versioning on edits, diff, and restore
- **Task extraction** - Extract markdown checkboxes from notes as a unified task list
- **Link health** - Find orphan notes, dead-ends, and broken wikilinks
- **Random discovery** - Surface random notes for review and serendipity
- **Bidirectional links** - Wikilink support with backlinks and outgoing links
- **Agent memory system** - Remember, recall, and forget memories with categories and importance levels
- **TTL support** - Set time-to-live for auto-expiring notes and memories
- **Source tracking** - Track where notes originated (code reviews, meetings, etc.)
- **Import/Export** - Markdown, JSON, and JSONL formats with YAML frontmatter
- **Editor integration** - Uses your `$EDITOR` for composing longer notes
- **MCP Server** - Expose notes to AI agents like Claude via Model Context Protocol (27 tools)
- **Web interface** - Embedded Vue 3 web UI with CodeMirror editor and Nord theme
- **Semantic search** - Optional vector similarity search powered by veclite
- **Portable** - Single binary, SQLite database, XDG-compliant storage

## Quick Start

```bash
# Install via Homebrew (macOS/Linux)
brew install abdul-hamid-achik/tap/noted

# Create your first note
noted add -t "Meeting Notes" -c "Discussed Q1 roadmap" -T "work,meetings"

# List recent notes
noted list

# Search notes
noted grep "roadmap"

# Open today's daily note
noted daily

# Launch web interface
noted serve --open
```

## Installation

### Homebrew (Recommended)

```bash
brew install abdul-hamid-achik/tap/noted
```

### Go Install

Requires Go 1.21 or later:

```bash
go install github.com/abdul-hamid-achik/noted@latest
```

### Download Binary

Download pre-built binaries from the [releases page](https://github.com/abdul-hamid-achik/noted/releases).

Available for:
- macOS (Intel and Apple Silicon)
- Linux (amd64 and arm64)
- Windows (amd64 and arm64)

### From Source

```bash
git clone https://github.com/abdul-hamid-achik/noted.git
cd noted
task build      # or: go build -o bin/noted .
task install    # or: go install .
```

## Usage

### Adding Notes

Create a new note with title, content, and optional tags:

```bash
# Quick note with inline content
noted add -t "Todo" -c "Buy groceries" -T "personal,todo"

# Open $EDITOR to compose content
noted add -t "Journal Entry"

# Note with multiple tags
noted add -t "Go Tips" -c "Use gofmt" -T "golang,programming,tips"

# Create from a template
noted add -t "Sprint Retro" --template meeting
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-t` | Note title (required) |
| `--content` | `-c` | Note content (opens editor if omitted) |
| `--tags` | `-T` | Comma-separated tags |
| `--template` | | Create from a named template |
| `--ttl` | | Time-to-live (e.g., `24h`, `7d`) |
| `--source` | | Source identifier (e.g., `code-review`) |
| `--source-ref` | | Source reference (e.g., `main.go:50`) |

### Listing Notes

View your notes with optional filtering:

```bash
# List recent notes (default: 20)
noted list

# Limit results
noted list -n 5

# Filter by tag
noted list --tag work
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--limit` | `-n` | Maximum notes to show (default: 20) |
| `--tag` | `-T` | Filter by tag name |

### Viewing Notes

Display a single note with full details:

```bash
# Show note with metadata
noted show 1

# Output raw markdown only (for piping)
noted show 1 --raw
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--raw` | `-r` | Output only the note content |

### Editing Notes

Modify existing notes (automatically saves a version snapshot):

```bash
# Update title only
noted edit 1 -t "Updated Title"

# Update content
noted edit 1 -c "New content here"

# Replace all tags
noted edit 1 -T "newtag1,newtag2"

# Open in editor (when no flags provided)
noted edit 1
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-t` | New title |
| `--content` | `-c` | New content |
| `--tags` | `-T` | Replace tags (comma-separated) |

### Deleting Notes

Remove notes from the database:

```bash
# Delete with confirmation
noted delete 1

# Delete without confirmation
noted delete 1 --force

# Delete multiple notes
noted delete 1 2 3 --force
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip confirmation prompt |

### Managing Tags

View and manage your tags:

```bash
# List all tags
noted tags

# Show tags with note counts
noted tags --count

# Delete unused (orphan) tags
noted tags --delete-unused
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--count` | `-c` | Show note count per tag |
| `--delete-unused` | `-d` | Delete tags with no notes |

### Searching Notes

Find notes by text in title or content:

```bash
# Search for a term
noted grep "kubernetes"

# Limit results
noted grep "meeting" -n 5
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--limit` | `-n` | Maximum results (default: 20) |

### Daily Notes

Manage daily notes in Obsidian style. Creates a note titled with the date, tagged "daily", and stored in a "Daily Notes" folder:

```bash
# Show/create today's daily note
noted daily

# Append to today's note
noted daily --append "- [ ] Buy milk"

# Prepend to today's note
noted daily --prepend "Morning thoughts"

# Show/create yesterday's note
noted daily --yesterday

# Show/create note for a specific date
noted daily --date 2026-02-14

# List recent daily notes (last 30 days)
noted daily --list
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--append` | `-a` | Append content to the daily note |
| `--prepend` | `-p` | Prepend content to the daily note |
| `--yesterday` | `-y` | Show/create yesterday's note |
| `--date` | `-d` | Specific date (YYYY-MM-DD) |
| `--list` | `-l` | List recent daily notes |
| `--json` | `-j` | Output as JSON |

### Templates

Create reusable note templates with variable interpolation:

```bash
# List all templates
noted template list

# Create a template
noted template create -n "meeting" -c "# {{title}}\n\nDate: {{date}}\n\n## Attendees\n\n## Notes\n\n## Action Items\n- [ ] "

# Show a template
noted template show meeting

# Edit a template in $EDITOR
noted template edit meeting

# Delete a template
noted template delete meeting

# Create a note from a template
noted add -t "Sprint Retro" --template meeting
```

**Template variables:**
| Variable | Replaced with |
|----------|--------------|
| `{{date}}` | Current date (YYYY-MM-DD) |
| `{{time}}` | Current time (HH:MM) |
| `{{datetime}}` | Current date and time |
| `{{title}}` | Note title |

### Version History

Every edit automatically saves a version snapshot. View and restore previous versions:

```bash
# List version history for a note
noted history 1

# Show a specific version
noted history 1 --version 2

# Diff a version against current
noted diff 1

# Diff a specific version
noted diff 1 --version 2

# Restore to a previous version (saves current as new version first)
noted restore 1 --version 2
```

### Task Extraction

Extract markdown tasks (checkboxes) from your notes:

```bash
# List all tasks across all notes
noted tasks

# Show only pending tasks
noted tasks --pending

# Show only completed tasks
noted tasks --completed

# Filter by tag
noted tasks --tag work

# Filter by note ID
noted tasks --note 42

# Show task counts only
noted tasks --count
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--pending` | | Show only pending tasks |
| `--completed` | | Show only completed tasks |
| `--tag` | `-T` | Filter by tag name |
| `--note` | | Filter by note ID |
| `--count` | | Show counts only |
| `--json` | `-j` | Output as JSON |

### Link Health

Analyze the health of your knowledge graph:

```bash
# Find orphan notes (no links in or out)
noted orphans

# Find dead-end notes (incoming links, no outgoing)
noted deadends

# Find broken wikilinks
noted unresolved
```

### Random Note

Surface a random note for review:

```bash
# Get a random note
noted random

# Random note from a specific tag
noted random --tag work
```

### Memory System

The memory system provides a way to store, recall, and manage memories with categories and importance levels. This is especially useful for AI agents that need persistent context.

#### Storing Memories

```bash
# Store a simple memory
noted remember "Always use snake_case for database columns"

# Store with category and importance
noted remember "Project uses PostgreSQL 15" --category project --importance 4

# Store with TTL (auto-expires)
noted remember "Review PR #123 by Friday" --ttl 3d --category todo

# Store with source tracking
noted remember "Found auth bug" --source code-review --source-ref auth.go:142
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-t` | Short title for the memory |
| `--category` | `-c` | Category: `user-pref`, `project`, `decision`, `fact`, `todo` (default: `fact`) |
| `--importance` | `-i` | Importance level 1-5 (default: 3) |
| `--ttl` | | Time-to-live duration (e.g., `24h`, `7d`) |
| `--source` | | Source identifier |
| `--source-ref` | | Source reference |

#### Recalling Memories

```bash
# Search memories
noted recall "database conventions"

# Limit results
noted recall "authentication" --limit 10

# Filter by category
noted recall "setup" --category project

# Use semantic search (if available)
noted recall "user preferences" --semantic
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--limit` | `-n` | Maximum results (default: 5) |
| `--category` | `-c` | Filter by category |
| `--semantic` | `-s` | Use semantic search (default: true if available) |

#### Forgetting Memories

```bash
# Preview what would be deleted (dry run)
noted forget --older-than 30d

# Actually delete old memories
noted forget --older-than 30d --force

# Delete low-importance memories
noted forget --importance-below 2 --force

# Delete by category
noted forget --category todo --older-than 7d --force

# Delete specific memory by ID
noted forget --id 42 --force

# Delete memories matching a query
noted forget --query "temporary" --force
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--older-than` | | Delete memories older than duration (e.g., `30d`) |
| `--importance-below` | | Delete memories below this importance level |
| `--category` | `-c` | Only delete memories in this category |
| `--query` | `-q` | Delete memories matching this text |
| `--id` | | Delete specific memory by ID |
| `--force` | `-f` | Actually delete (default is dry-run preview) |

### Exporting Notes

Export notes to files:

```bash
# Export all as markdown (default)
noted export

# Export as JSON
noted export -f json

# Export as JSON Lines (one object per line)
noted export -f jsonl

# Export to file
noted export -o backup.md

# Export notes with specific tag
noted export --tag work -f json -o work-notes.json

# Export notes since a date
noted export --since 2026-01-01 -f jsonl
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--format` | `-f` | Output format: `markdown`, `json`, `jsonl` (default: markdown) |
| `--output` | `-o` | Output file path (default: stdout) |
| `--tag` | `-T` | Filter by tag |
| `--since` | | Export notes created since date (YYYY-MM-DD) |

### Importing Notes

Import markdown files into noted:

```bash
# Import a single file
noted import notes/idea.md

# Import all markdown files from a directory
noted import ~/Documents/notes/

# Import recursively (include subdirectories)
noted import ~/Documents/notes/ --recursive

# Add tags to all imported notes
noted import ~/exports/ -T "imported,backup"
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--recursive` | `-r` | Scan subdirectories |
| `--tags` | `-T` | Add tags to all imported notes |

### Web Interface

Launch the embedded web UI for a graphical experience:

```bash
# Start web server on default port (3000)
noted serve

# Specify port and auto-open browser
noted serve --port 8080 --open
```

The web interface includes:
- CodeMirror 6 editor with vim mode
- Nord theme throughout
- Live updates via SSE
- Full note management (create, edit, delete, search)

### Version Information

```bash
# Show version
noted version

# Output as JSON
noted version --json
```

## MCP Server

noted includes an MCP (Model Context Protocol) server that exposes your knowledge base to AI agents like Claude. With 27 tools covering notes, daily notes, templates, tasks, versioning, and link health, agents get full access to your knowledge base.

### Starting the Server

```bash
# Start MCP server (uses stdio transport)
noted mcp
```

### Integration with Claude Code

Add noted to your Claude Code configuration:

```bash
# Add MCP server
claude mcp add noted -- noted mcp

# With semantic search enabled
claude mcp add noted -- env NOTED_VECLITE_PATH=~/.local/share/noted/vectors.veclite noted mcp
```

Or manually edit `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "noted": {
      "command": "noted",
      "args": ["mcp"],
      "env": {
        "NOTED_VECLITE_PATH": "~/.local/share/noted/vectors.veclite",
        "NOTED_EMBEDDING_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

### Available MCP Tools

#### Core Notes

| Tool | Description |
|------|-------------|
| `noted_create` | Create a new note with title, content, and optional tags |
| `noted_list` | List notes with optional tag filter and pagination |
| `noted_get` | Get a note by its ID, including tags |
| `noted_search` | Search notes by title and content using text matching |
| `noted_update` | Update a note's title, content, or tags |
| `noted_delete` | Delete a note by ID |
| `noted_tags` | List all tags with their note counts |
| `noted_random` | Get a random note, optionally filtered by tag |
| `noted_semantic_search` | Search notes using vector similarity (requires veclite) |
| `noted_sync` | Sync notes to the semantic search index |

#### Daily Notes

| Tool | Description |
|------|-------------|
| `noted_daily` | Get or create a daily note. Optionally append or prepend content. |
| `noted_daily_list` | List recent daily notes (last 30 days) |

#### Templates

| Tool | Description |
|------|-------------|
| `noted_template_list` | List all note templates |
| `noted_template_create` | Create a new template with `{{date}}`, `{{time}}`, `{{datetime}}`, `{{title}}` variables |
| `noted_template_get` | Get a template by name |
| `noted_template_delete` | Delete a template by name |
| `noted_template_apply` | Apply a template to create a new note with variable interpolation |

#### Task Extraction

| Tool | Description |
|------|-------------|
| `noted_tasks` | Extract markdown tasks (checkboxes) from notes. Filter by note, tag, or status. |

#### Version History

| Tool | Description |
|------|-------------|
| `noted_history` | List version history for a note |
| `noted_version_get` | Get a specific version of a note (full content) |
| `noted_restore` | Restore a note to a previous version (saves current state first) |

#### Link Health

| Tool | Description |
|------|-------------|
| `noted_backlinks` | Get all notes that link to a given note |
| `noted_orphans` | Find orphan notes and dead-end notes in the knowledge graph |

#### Agent Memory

| Tool | Description |
|------|-------------|
| `noted_remember` | Store a memory with category, importance, TTL, and source tracking |
| `noted_recall` | Recall relevant memories by query with semantic or keyword search |
| `noted_forget` | Delete old or low-importance memories with dry-run support |

### Memory Tools for Agents

The `noted_remember`, `noted_recall`, and `noted_forget` tools provide persistent memory for AI agents:

```
# Example: Agent stores a user preference
noted_remember(content="User prefers dark mode", category="user-pref", importance=4)

# Example: Store with TTL and source
noted_remember(content="Bug in auth flow", category="project", ttl="7d", source="code-review", source_ref="auth.go:50")

# Example: Agent recalls relevant memories
noted_recall(query="user preferences", limit=5, category="user-pref")

# Example: Clean up old memories (preview first)
noted_forget(older_than_days=30, importance_below=2, dry_run=true)
```

**Memory categories:**
- `user-pref` - User preferences and settings
- `project` - Project-specific information
- `decision` - Design decisions and rationale
- `fact` - General facts and knowledge
- `todo` - Tasks and reminders

**Memory features:**
- **TTL (Time-to-Live)**: Set `ttl` parameter (e.g., `"24h"`, `"7d"`) for auto-expiring memories
- **Source tracking**: Track origin with `source` and `source_ref` parameters
- **Importance levels**: 1-5 scale for prioritizing memory retention
- **Lazy cleanup**: Expired memories are automatically removed during recall operations

## Semantic Search

Enable semantic search to find notes by meaning, not just keywords.

### Prerequisites

1. [Ollama](https://ollama.ai) running locally
2. An embedding model (default: `nomic-embed-text`)

```bash
# Install Ollama and pull the embedding model
ollama pull nomic-embed-text
```

### Configuration

Set the veclite database path:

```bash
export NOTED_VECLITE_PATH=~/.local/share/noted/vectors.veclite
```

### Syncing Notes

Sync your existing notes to enable semantic search:

```bash
# Sync unsynced notes
noted sync

# Force re-sync all notes
noted sync --force
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Re-sync all notes even if already synced |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NOTED_VECLITE_PATH` | Path to veclite database | (disabled) |
| `NOTED_EMBEDDING_MODEL` | Ollama embedding model | `nomic-embed-text` |
| `OLLAMA_HOST` | Ollama server URL | `http://localhost:11434` |

## Configuration

noted follows the XDG Base Directory Specification:

| Path | Description |
|------|-------------|
| `~/.local/share/noted/noted.db` | SQLite database |
| `~/.local/share/noted/vectors.veclite` | Vector database (optional) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `EDITOR` | Editor for composing notes (default: `nvim`) |
| `NOTED_VECLITE_PATH` | Path to veclite database for semantic search |
| `NOTED_EMBEDDING_MODEL` | Embedding model for semantic search |
| `OLLAMA_HOST` | Ollama server URL |

## Architecture

```
noted/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command, database lifecycle
│   ├── add.go             # Create notes
│   ├── list.go            # List notes
│   ├── show.go            # Display single note
│   ├── edit.go            # Modify notes (auto-versioning)
│   ├── delete.go          # Remove notes
│   ├── tags.go            # Tag management
│   ├── grep.go            # Search notes
│   ├── daily.go           # Daily notes (Obsidian-style)
│   ├── template.go        # Template management
│   ├── history.go         # Version history, diff, restore
│   ├── tasks.go           # Task extraction from notes
│   ├── links.go           # Link health (orphans, deadends, unresolved)
│   ├── random.go          # Random note discovery
│   ├── remember.go        # Store memories
│   ├── recall.go          # Search memories
│   ├── forget.go          # Delete memories
│   ├── export.go          # Export to markdown/JSON/JSONL
│   ├── import.go          # Import markdown files
│   ├── serve.go           # Web interface server
│   ├── mcp.go             # MCP server command
│   ├── sync.go            # Sync to veclite
│   ├── version.go         # Version info
│   └── editor.go          # Editor integration
├── web/                    # Frontend source (Vue 3 SPA)
├── internal/
│   ├── config/            # XDG-compliant configuration
│   ├── db/                # Database layer (sqlc)
│   │   ├── schema.sql     # Database schema
│   │   ├── query.sql      # SQL queries
│   │   ├── migrate.go     # Database migration system
│   │   ├── migrations/    # SQL migration files
│   │   └── *.go           # Generated code
│   ├── memory/            # Shared memory logic
│   ├── mcp/               # MCP server (27 tools)
│   │   ├── server.go      # Server setup and transport
│   │   └── tools.go       # Tool handlers
│   ├── web/               # Web server + API handlers + embed
│   └── veclite/           # Semantic search integration
├── main.go                # Entry point
├── Taskfile.yml           # Build tasks
└── sqlc.yaml              # sqlc configuration
```

### Technology Stack

- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **Database**: SQLite via [modernc.org/sqlite](https://modernc.org/sqlite) (pure Go)
- **SQL Code Gen**: [sqlc](https://sqlc.dev)
- **MCP SDK**: [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
- **Vector Search**: [veclite](https://github.com/abdul-hamid-achik/veclite)
- **Web Frontend**: Vue 3 + Vite + Tailwind CSS v4 + CodeMirror 6
- **Build Tool**: [Task](https://taskfile.dev)

## Development

### Prerequisites

- Go 1.21+
- [Task](https://taskfile.dev) (optional, for build automation)
- [sqlc](https://sqlc.dev) (for regenerating database code)
- [golangci-lint](https://golangci-lint.run) (for linting)
- [Bun](https://bun.sh) (for web frontend development)

### Build Commands

```bash
# Show available tasks
task

# Generate sqlc code
task generate

# Build binary
task build

# Run tests
task test

# Run linter
task lint

# Build and run with arguments
task dev -- list

# Install to GOPATH/bin
task install

# Clean build artifacts
task clean

# Web frontend
task web:install    # Install frontend dependencies
task web:build      # Build frontend for embedding
task web:dev        # Start frontend dev server
```

### Running Tests

```bash
# Run all tests
task test

# Run with verbose output
go test -v ./...

# Run specific test
go test -v ./cmd -run TestDatabaseTags

# Run MCP tests
go test -v ./internal/mcp/...
```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting (`task test && task lint`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style

- Run `gofmt` on all code
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Add tests for new functionality
- Update documentation for user-facing changes

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [sqlc](https://sqlc.dev) - Type-safe SQL
- [modernc.org/sqlite](https://modernc.org/sqlite) - Pure Go SQLite
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) - Model Context Protocol
- [veclite](https://github.com/abdul-hamid-achik/veclite) - Vector database
- [Vue 3](https://vuejs.org) - Web framework
- [Vite](https://vite.dev) - Frontend build tool
