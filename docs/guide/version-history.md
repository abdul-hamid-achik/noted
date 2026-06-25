# Version history

Every note edit saves a version snapshot of the previous state.

## View history

```bash
noted history 1
noted history 1 --version 2
```

## Diff

```bash
noted diff 1
noted diff 1 --version 2
```

## Restore

```bash
noted restore 1 --version 2
```

The current state is snapshotted first, so nothing is lost.

## Where versions live

Snapshots are written to the vault under `.noted/versions/<note-id>/<version>.md`. They are restored
on `noted vault import`, so history survives a full index rebuild.

## MCP

Agents can use `noted_history`, `noted_version_get`, and `noted_restore`.
