# Wikilinks

Use `[[Note title]]` to create bidirectional links between notes.

## Create a link

```markdown
See [[Meeting notes]] and [[Project ideas]].
```

## Backlinks

noted tracks which notes link to a given note. View them in the TUI or CLI:

```bash
noted backlinks 1
```

## Link health

noted can report three kinds of link problems:

- **Orphans** — notes with no links in or out
- **Dead-ends** — notes with incoming links but no outgoing links
- **Unresolved** — wikilinks whose target does not exist

```bash
noted orphans
noted deadends
noted unresolved
```

## TUI editor

Type `[[` in the editor to open link autocomplete. Press `Ctrl+L` to follow a wikilink and
`Ctrl+B` to show backlinks.
