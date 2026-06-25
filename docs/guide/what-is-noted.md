# What is noted?

noted is a terminal-only Obsidian alternative: a notes app that is agents-first but also pleasant for
humans. It combines a Nord-themed bubbletea v2 TUI with a JSON-output CLI and an MCP server.

Data is stored in a local SQLite database and mirrored to a markdown vault — plain `.md` files with
YAML frontmatter — so your notes are always readable, diffable, and tool-friendly.

## Design principles

- **Terminal native.** No web UI, no GUI. Everything happens in the terminal.
- **Agents-first.** MCP server + JSON CLI mean AI agents can read, write, and remember alongside you.
- **Markdown source of truth.** The vault wins; SQLite is a rebuildable index.
- **Human-friendly TUI.** Mouse and keyboard, truecolor Nord theme, responsive down to small sizes.
- **Tested.** Every feature is covered by glyph (glyphrun) terminal e2e specs.

## Three surfaces, one database

| Surface | Best for |
|--------|----------|
| TUI (`noted`) | Browsing, editing, daily notes, quick capture |
| CLI (`noted add`, `noted grep`, …) | Scripts, aliases, shell integration |
| MCP server (`noted mcp`) | AI agents like Claude |

## Learn more

- [Capturing notes](/guide/capturing-notes)
- [Tags and folders](/guide/tags-and-folders)
- [Version history](/guide/version-history)
- [Semantic search](/guide/semantic-search)
