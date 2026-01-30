# noted

A fast, lightweight CLI knowledge base for capturing and organizing notes from your terminal. Includes an MCP server for AI agent integration.

## Features

- **Quick capture** - Create notes with titles, content, and tags in one command
- **Rich tagging** - Organize notes with multiple tags, view tag statistics
- **Full-text search** - Find notes by searching titles and content
- **Agent memory system** - Remember, recall, and forget memories with categories and importance levels
- **TTL support** - Set time-to-live for auto-expiring notes and memories
- **Source tracking** - Track where notes originated (code reviews, meetings, etc.)
- **Import/Export** - Markdown, JSON, and JSONL formats with YAML frontmatter
- **Editor integration** - Uses your `$EDITOR` for composing longer notes
- **MCP Server** - Expose notes to AI agents like Claude via Model Context Protocol
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
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-t` | Note title (required) |
| `--content` | `-c` | Note content (opens editor if omitted) |
| `--tags` | `-T` | Comma-separated tags |
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

**Example output:**
```
# Meeting Notes

ID: 1
Created: 2026-01-29 14:30
Updated: 2026-01-29 14:30
Tags: work, meetings

---

Discussed Q1 roadmap with the team.
```

### Editing Notes

Modify existing notes:

```bash
# Update title only
noted edit 1 -t "Updated Title"

# Update content
noted edit 1 -c "New content here"

# Replace all tags
noted edit 1 -T "newtag1,newtag2"

# Open in editor (when no flags provided)
noted edit 1

# Clear all tags
noted edit 1 -T ""
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

**Example output:**
```
$ noted tags --count
golang (5)
personal (3)
work (12)
```

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

**Markdown format:**
```markdown
---
title: "Meeting Notes"
tags: ["work", "meetings"]
created: 2026-01-29T14:30:00Z
updated: 2026-01-29T14:30:00Z
---

Discussed Q1 roadmap with the team.
```

**JSON format:**
```json
[
  {
    "id": 1,
    "title": "Meeting Notes",
    "content": "Discussed Q1 roadmap with the team.",
    "tags": ["work", "meetings"],
    "created_at": "2026-01-29T14:30:00Z",
    "updated_at": "2026-01-29T14:30:00Z"
  }
]
```

**JSONL format (one object per line):**
```json
{"id":1,"title":"Meeting Notes","content":"Discussed Q1 roadmap.","tags":["work"],"created_at":"2026-01-29T14:30:00Z","updated_at":"2026-01-29T14:30:00Z"}
{"id":2,"title":"Todo","content":"Buy groceries","tags":["personal"],"created_at":"2026-01-29T15:00:00Z","updated_at":"2026-01-29T15:00:00Z"}
```

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

**Supported file formats:**

Files with YAML frontmatter:
```markdown
---
title: "My Note"
tags: [idea, project]
---

Note content here.
```

Files without frontmatter use:
1. First `# Heading` as title
2. Filename (without `.md`) as fallback

### Version Information

```bash
# Show version
noted version

# Output as JSON
noted version --json
```

## MCP Server

noted includes an MCP (Model Context Protocol) server that exposes your knowledge base to AI agents like Claude.

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

| Tool | Description |
|------|-------------|
| `noted_create` | Create a new note with title, content, and optional tags |
| `noted_list` | List notes with optional tag filter and pagination |
| `noted_get` | Get a note by its ID, including tags |
| `noted_search` | Search notes by title and content using text matching |
| `noted_update` | Update a note's title, content, or tags |
| `noted_delete` | Delete a note by ID |
| `noted_tags` | List all tags with their note counts |
| `noted_semantic_search` | Search notes using vector similarity (requires veclite) |
| `noted_remember` | Store a memory with category, importance, TTL, and source tracking |
| `noted_recall` | Recall relevant memories by query with semantic or keyword search |
| `noted_forget` | Delete old or low-importance memories with dry-run support |
| `noted_sync` | Sync notes to the semantic search index |

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
│   ├── edit.go            # Modify notes
│   ├── delete.go          # Remove notes
│   ├── tags.go            # Tag management
│   ├── grep.go            # Search notes
│   ├── remember.go        # Store memories
│   ├── recall.go          # Search memories
│   ├── forget.go          # Delete memories
│   ├── export.go          # Export to markdown/JSON/JSONL
│   ├── import.go          # Import markdown files
│   ├── mcp.go             # MCP server command
│   ├── sync.go            # Sync to veclite
│   ├── version.go         # Version info
│   └── editor.go          # Editor integration
├── internal/
│   ├── config/            # XDG-compliant configuration
│   ├── db/                # Database layer (sqlc)
│   │   ├── schema.sql     # Database schema
│   │   ├── query.sql      # SQL queries
│   │   ├── migrate.go     # Database migration system
│   │   ├── migrations/    # SQL migration files
│   │   └── *.go           # Generated code
│   ├── memory/            # Shared memory logic
│   │   ├── types.go       # Memory types and validation
│   │   ├── remember.go    # Create memories
│   │   ├── recall.go      # Search memories
│   │   └── forget.go      # Delete memories
│   ├── mcp/               # MCP server implementation
│   │   ├── server.go      # Server setup and transport
│   │   └── tools.go       # Tool handlers
│   └── veclite/           # Semantic search integration
│       └── syncer.go      # veclite sync and search
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
- **Build Tool**: [Task](https://taskfile.dev)

## Development

### Prerequisites

- Go 1.21+
- [Task](https://taskfile.dev) (optional, for build automation)
- [sqlc](https://sqlc.dev) (for regenerating database code)
- [golangci-lint](https://golangci-lint.run) (for linting)

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
