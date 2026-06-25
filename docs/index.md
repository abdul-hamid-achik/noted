---
layout: home

hero:
  name: noted
  text: A terminal-only Obsidian alternative
  tagline: Capture, organize, and review notes without leaving the terminal. Built for humans with a Nord-themed TUI, and for agents with an MCP server + JSON CLI.
  image:
    src: /noted.svg
    alt: noted logo
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started/installation
    - theme: alt
      text: View on GitHub
      link: https://github.com/abdul-hamid-achik/noted

features:
  - title: Terminal-first TUI
    details: Nord-themed bubbletea v2 interface with mouse + keyboard support, responsive layout, live markdown preview, and 9 views.
  - title: Agents-first
    details: MCP server with 26 tools, JSON-output CLI commands, and two-way markdown vault sync so agents and editors share the same notes.
  - title: Markdown vault
    details: Plain .md files with YAML frontmatter are the on-disk source of truth; SQLite is a rebuildable index.
  - title: Version history
    details: Every edit is snapshotted, diffable, and restorable. Snapshots live in the vault so history survives rebuilds.
  - title: Semantic search
    details: Optional veclite + Ollama vector search finds notes by meaning, not just keywords.
  - title: Link health
    details: Wikilinks, backlinks, orphans, dead-ends, and broken link detection keep the knowledge graph healthy.
---

## What is noted?

noted is a fast, local, terminal-only notes app for people who live in the shell. It stores your
knowledge in a SQLite index and mirrors every note to a markdown vault so the data is always readable,
versionable, and tool-friendly.

Run `noted` to open the interactive TUI, or use `noted add`, `noted list`, `noted grep`, and other
commands for quick scripting. AI agents connect through the MCP server (`noted mcp`) to read, write,
and remember alongside you.

## Quick install

```bash
brew install abdul-hamid-achik/tap/noted
```

Or with Go 1.25+:

```bash
go install github.com/abdul-hamid-achik/noted@latest
```

Read the [full installation guide](/getting-started/installation) for other platforms.
