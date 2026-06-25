# Quick start

Create your first note, search it, and open the TUI in under a minute.

## Create a note

```bash
noted add -t "Meeting notes" -c "Discussed Q1 roadmap." -T "work,meetings"
```

Omit `-c` to open your `$EDITOR`.

## List notes

```bash
noted list
```

## Search notes

```bash
noted grep "roadmap"
```

## Open the TUI

```bash
noted
```

Press `?` for help, `n` for a new note, `/` to filter, and `1`–`9` to switch views.

## Next steps

- [Set up your vault](/getting-started/vault-setup) to sync notes to markdown files.
- Read about [capturing notes](/guide/capturing-notes) and the [TUI keybindings](/reference/tui).
- Connect an AI agent via [MCP](/reference/mcp).
