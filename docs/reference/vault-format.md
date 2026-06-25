# Vault format

Each note in the vault is a single `.md` file with YAML frontmatter.

## Example

```markdown
---
id: 1
title: Meeting notes
tags:
  - work
folder_id: 2
pin: false
created: 2026-06-15T13:00:00Z
updated: 2026-06-15T13:00:00Z
---

Discussed the v2 rewrite and Nord theme.
```

## Frontmatter fields

| Field | Description |
|-------|-------------|
| `id` | Stable note ID (preserved across rebuilds) |
| `title` | Note title |
| `tags` | List of tags |
| `folder_id` | Optional folder ID |
| `pin` | Whether the note is pinned |
| `source` | Optional source identifier |
| `source_ref` | Optional source reference |
| `created` | ISO 8601 creation time |
| `updated` | ISO 8601 update time |

## Version files

Version snapshots live in `.noted/versions/<note-id>/<version>.md` with the same frontmatter shape
plus a `version` field.

## Special directories

- `.noted/` — hidden metadata directory (versions, etc.) excluded from note scanning
- Subdirectories are ignored for note listing

## Round-trip

`noted vault export` writes all notes and version snapshots. `noted vault import --force` rebuilds
the SQLite index from these files while preserving IDs and restoring history.
