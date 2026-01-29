# Contributing to noted

Thank you for your interest in contributing to noted! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.21 or later
- [Task](https://taskfile.dev) - Task runner
- [sqlc](https://sqlc.dev) - SQL code generator
- [golangci-lint](https://golangci-lint.run) - Linter

### Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/noted.git
cd noted

# Add upstream remote
git remote add upstream https://github.com/abdul-hamid-achik/noted.git

# Install dependencies
go mod download

# Verify setup
task build
task test
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Changes

- Write clear, focused commits
- Follow the code style guidelines
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
task test

# Run linter
task lint

# Build and test manually
task build
./bin/noted <command>
```

### 4. Commit Your Changes

Write clear commit messages:

```
Add export command with JSON and markdown formats

- Implement exportJSON and exportMarkdown functions
- Add --format, --output, and --tag flags
- Include YAML frontmatter in markdown export
```

### 5. Submit a Pull Request

1. Push your branch to your fork
2. Open a PR against the `main` branch
3. Fill out the PR template
4. Wait for review

## Code Style

### Go Code

- Use `gofmt` for formatting
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use meaningful variable and function names
- Keep functions focused and small

### Comments

```go
// Good: Explains WHY
// Cache TTL is 1 hour to balance freshness with API limits
const cacheTTL = 3600

// Bad: States the obvious
// Set cache TTL to 3600
const cacheTTL = 3600
```

### Error Handling

```go
// Use error wrapping for context
if err != nil {
    return fmt.Errorf("failed to create note: %w", err)
}

// Handle specific errors when needed
if err == sql.ErrNoRows {
    return fmt.Errorf("note #%d not found", id)
}
```

## Project Structure

```
noted/
├── cmd/           # CLI commands
├── internal/
│   ├── config/    # Configuration
│   └── db/        # Database layer
├── main.go        # Entry point
└── Taskfile.yml   # Build tasks
```

### Adding a New Command

1. Create `cmd/yourcommand.go`:

```go
package cmd

import (
    "github.com/spf13/cobra"
)

var yourCmd = &cobra.Command{
    Use:   "yourcommand",
    Short: "Brief description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(yourCmd)
    yourCmd.Flags().StringP("flag", "f", "", "Flag description")
}
```

2. Add tests in `cmd/cmd_test.go`
3. Update README.md with usage documentation

### Adding Database Queries

1. Add query to `internal/db/query.sql`:

```sql
-- name: YourQuery :many
SELECT * FROM notes WHERE condition = ?;
```

2. Regenerate code:

```bash
task generate
```

3. Use in your command:

```go
results, err := database.YourQuery(ctx, param)
```

## Testing

### Test Structure

```go
func TestYourFeature(t *testing.T) {
    cleanup := setupTestDB(t)
    defer cleanup()

    // Setup
    noteID := createTestNote(t, "Title", "Content", []string{"tag"})

    // Execute
    result, err := yourFunction(noteID)

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

### Running Tests

```bash
# All tests
task test

# Specific package
go test -v ./cmd

# Specific test
go test -v ./cmd -run TestYourFeature

# With coverage
go test -cover ./...
```

## Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported functions
- Include examples in documentation

## Reporting Issues

### Bug Reports

Include:
- noted version (`noted version`)
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Error messages

### Feature Requests

Include:
- Use case description
- Proposed solution
- Alternative approaches considered

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## Questions?

Open an issue with the `question` label or start a discussion.

Thank you for contributing!
