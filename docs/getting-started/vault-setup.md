# Vault setup

The markdown vault is a directory of plain `.md` files with YAML frontmatter. It is the on-disk
source of truth; the SQLite database is a rebuildable index.

## Default location

By default the vault lives at:

```
~/.local/share/noted/vault
```

Override it with the `NOTED_VAULT` environment variable or the `--vault` flag:

```bash
export NOTED_VAULT=~/Documents/noted-vault
noted --vault ~/Documents/noted-vault add -t "Idea" -c "..."
```

## Write-through

Every note create/update/delete from the CLI, TUI, or MCP server is mirrored to the vault
automatically. File names are derived from note titles and reused on renames, so there are no
orphan files.

## Rebuild the index from the vault

If you edit vault files outside of noted (`$EDITOR`, Obsidian, git pull), the TUI will re-index them
live. To rebuild from scratch:

```bash
noted vault import --force
```

This preserves note IDs and restores version history from the hidden `.noted/versions/` directory.

## Export the vault

```bash
noted vault export
```

Add `--path` to write to a different directory.

## Version history

Snapshots are saved under `.noted/versions/<note-id>/<version>.md`. They are restored on import, so
every edit trail survives a full rebuild.

## Ignore patterns

The `.noted/` directory and any subdirectories are excluded from the vault note list.
