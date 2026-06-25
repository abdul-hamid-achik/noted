# CLI commands

noted exposes every feature through Cobra CLI commands. Most commands support `--json` for scripting.

## Core notes

| Command | Description |
|---------|-------------|
| `noted add` | Create a note |
| `noted list` | List recent notes |
| `noted show` | Display a single note |
| `noted edit` | Edit a note (auto-snapshot) |
| `noted delete` | Delete note(s) |
| `noted grep` | Search titles and content |
| `noted random` | Surface a random note |

## Organization

| Command | Description |
|---------|-------------|
| `noted tags` | Manage tags |
| `noted folder create` | Create a folder |
| `noted folder list` | List folders |
| `noted folder delete` | Delete a folder |
| `noted pin` / `unpin` | Pin notes to the top |
| `noted stats` | Knowledge-base summary |

## Daily, templates, tasks

| Command | Description |
|---------|-------------|
| `noted daily` | Open/create today's daily note |
| `noted template create` | Create a reusable template |
| `noted template list` | List templates |
| `noted template show` | Show a template |
| `noted template edit` | Edit a template |
| `noted template delete` | Delete a template |
| `noted tasks` | Extract checkboxes across notes |

## Links, history, memory

| Command | Description |
|---------|-------------|
| `noted orphans` | Find notes with no links |
| `noted deadends` | Find notes with only incoming links |
| `noted unresolved` | Find broken wikilinks |
| `noted backlinks` | Show notes linking to a note |
| `noted history` | List versions of a note |
| `noted diff` | Diff a note against a version |
| `noted restore` | Restore a note version |
| `noted remember` | Store a memory |
| `noted recall` | Search memories |
| `noted forget` | Delete old memories |

## Vault and sync

| Command | Description |
|---------|-------------|
| `noted vault path` | Print the vault directory |
| `noted vault export` | Export notes to the vault |
| `noted vault import` | Rebuild index from vault (preview) |
| `noted vault import --force` | Apply rebuild from vault |
| `noted sync` | Sync notes to veclite |
| `noted export` | Export to markdown/JSON/JSONL |
| `noted import` | Import markdown files |

## Agent / system

| Command | Description |
|---------|-------------|
| `noted mcp` | Start the MCP server (stdio) |
| `noted version` | Show version info |

## Global flags

| Flag | Description |
|------|-------------|
| `--db` | Path to SQLite database |
| `--vault` | Path to markdown vault |
| `--json` | Output JSON |
| `--help` | Show command help |
