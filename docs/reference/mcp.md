# MCP server

noted exposes a Model Context Protocol (MCP) server so AI agents can read, write, and remember
alongside you.

## Start the server

```bash
noted mcp
```

The server uses stdio transport.

## Add to Claude Code

```bash
claude mcp add noted -- noted mcp
```

With semantic search enabled:

```bash
claude mcp add noted -- env NOTED_VECLITE_PATH=~/.local/share/noted/vectors.veclite noted mcp
```

## Tools

### Notes

| Tool | Description |
|------|-------------|
| `noted_create` | Create a note |
| `noted_list` | List notes |
| `noted_get` | Get a note by ID |
| `noted_search` | Text search |
| `noted_update` | Update a note |
| `noted_delete` | Delete a note |
| `noted_tags` | List tags |
| `noted_random` | Random note |
| `noted_semantic_search` | Vector search |
| `noted_sync` | Sync to veclite |

### Daily notes

| Tool | Description |
|------|-------------|
| `noted_daily` | Get/create today's daily note |
| `noted_daily_list` | List recent daily notes |

### Templates

| Tool | Description |
|------|-------------|
| `noted_template_list` | List templates |
| `noted_template_create` | Create a template |
| `noted_template_get` | Get a template |
| `noted_template_delete` | Delete a template |
| `noted_template_apply` | Apply a template |

### Tasks, links, memory

| Tool | Description |
|------|-------------|
| `noted_tasks` | Extract tasks |
| `noted_backlinks` | Show backlinks |
| `noted_orphans` | Find orphans/dead-ends |
| `noted_history` | List versions |
| `noted_version_get` | Get a version |
| `noted_restore` | Restore a version |
| `noted_remember` | Store a memory |
| `noted_recall` | Recall memories |
| `noted_forget` | Delete memories |

## Agent workflow

1. Agent reads context with `noted_list`, `noted_search`, or `noted_get`.
2. Agent writes notes with `noted_create` / `noted_update`; changes mirror to the vault instantly.
3. Agent remembers facts with `noted_remember` and recalls them with `noted_recall`.
4. Agent can use `noted_sync` to refresh the semantic index after bulk changes.
