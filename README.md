# noted

A fast, lightweight CLI knowledge base for capturing and organizing notes from your terminal.

## Features

- **Quick capture** - Create notes with titles, content, and tags in one command
- **Rich tagging** - Organize notes with multiple tags, view tag statistics
- **Full-text search** - Find notes by searching titles and content
- **Import/Export** - Markdown files with YAML frontmatter, JSON export
- **Editor integration** - Uses your `$EDITOR` for composing longer notes
- **Portable** - Single binary, SQLite database, XDG-compliant storage

## Quick Start

```bash
# Install
go install github.com/abdul-hamid-achik/noted@latest

# Create your first note
noted add -t "Meeting Notes" -c "Discussed Q1 roadmap" -T "work,meetings"

# List recent notes
noted list

# Search notes
noted grep "roadmap"
```

## Installation

### Prerequisites

- Go 1.21 or later
- SQLite (included via modernc.org/sqlite)

### From Source

```bash
# Clone the repository
git clone https://github.com/abdul-hamid-achik/noted.git
cd noted

# Build (requires Task)
task build

# Or with go directly
go build -o bin/noted .

# Install to GOPATH/bin
task install
# or
go install .
```

### Using Go Install

```bash
go install github.com/abdul-hamid-achik/noted@latest
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

### Exporting Notes

Export notes to files:

```bash
# Export all as markdown (default)
noted export

# Export as JSON
noted export -f json

# Export to file
noted export -o backup.md

# Export notes with specific tag
noted export --tag work -f json -o work-notes.json
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--format` | `-f` | Output format: `markdown`, `json` (default: markdown) |
| `--output` | `-o` | Output file path (default: stdout) |
| `--tag` | `-T` | Filter by tag |

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

## Configuration

noted follows the XDG Base Directory Specification:

| Path | Description |
|------|-------------|
| `~/.local/share/noted/noted.db` | SQLite database |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `EDITOR` | Editor for composing notes (default: `nvim`) |

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
│   ├── export.go          # Export to markdown/JSON
│   ├── import.go          # Import markdown files
│   ├── version.go         # Version info
│   └── editor.go          # Editor integration
├── internal/
│   ├── config/            # XDG-compliant configuration
│   └── db/                # Database layer (sqlc)
│       ├── schema.sql     # Database schema
│       ├── query.sql      # SQL queries
│       └── *.go           # Generated code
├── main.go                # Entry point
├── Taskfile.yml           # Build tasks
└── sqlc.yaml              # sqlc configuration
```

### Technology Stack

- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **Database**: SQLite via [modernc.org/sqlite](https://modernc.org/sqlite) (pure Go)
- **SQL Code Gen**: [sqlc](https://sqlc.dev)
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
