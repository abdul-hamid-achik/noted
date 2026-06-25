# AGENTS.md — guidelines for AI agents working on noted

This file orients AI agents (Claude, etc.) contributing to the **noted** codebase.

## What noted is

**noted** is a **terminal-only Obsidian alternative**: a notes app that is **agents-first** (an MCP
server + JSON-output CLI) but also pleasant for humans via an interactive **TUI**. No web UI, no GUI
— everything runs in the terminal. Notes live in a local SQLite database and are mirrored to a
**markdown vault** (`.md` files) via export / import / write-through — the vault is the on-disk
source of truth, SQLite the rebuildable index.

Design constraints the maintainer cares about:
- Default theme: **Nord (dark)**. Primary terminal: **Ghostty** (truecolor).
- Must look good at **all terminal sizes** (responsive; degrades gracefully).
- Mouse **and** keyboard must both work well.
- Every feature is covered by **glyph (glyphrun) e2e tests**.
- **Project knowledge** (ADRs, plans, specs, research) lives in the maintainer's own vault at
  `~/notes/projects/noted`, managed with Obsidian CLI or noted itself — **not** in the repo's `docs/`
  folder.
- **`docs/`** is the **VitePress product documentation site**: user-facing guides, install
  instructions, feature references, and marketing copy for noted. Do not put internal project notes
  there.

## ⚠️ Read this first

The TUI rewrite and the markdown-vault migration are complete. This document is the source of truth
for the architecture and conventions below. Two source-verified API references live in `docs/dev/`:
- `docs/dev/charm-v2-reference.md` — charm.land v2 APIs (theming every component, mouse/bubblezone,
  `tea.View`, lipgloss v2, cursor).
- `docs/dev/glyph-authoring.md` — how to write/run glyph specs.

## Technology stack

| Component | Tech |
|-----------|------|
| Language | Go **1.25** |
| CLI | [Cobra](https://github.com/spf13/cobra) |
| TUI | **charm.land v2**: `bubbletea/v2`, `bubbles/v2`, `lipgloss/v2`, `huh/v2` |
| Mouse hit-testing | [`bubblezone/v2`](https://github.com/lrstanley/bubblezone) |
| Markdown preview | [glamour](https://github.com/charmbracelet/glamour) |
| Database | SQLite via [modernc.org/sqlite](https://modernc.org/sqlite) (pure Go, **no CGO**) |
| SQL codegen | [sqlc](https://sqlc.dev) |
| MCP | [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) |
| Semantic search | [veclite](https://github.com/abdul-hamid-achik/veclite) + Ollama (optional) |
| E2E tests | [glyphrun](https://) (`glyph` CLI) |

## Project layout

```
noted/
├── main.go                  # entry point
├── cmd/                     # Cobra CLI commands (one file per command); `noted` w/ no args → TUI
│   ├── root.go              # root cmd, db lifecycle, runTUI
│   └── add.go, list.go, ... # add/list/show/edit/delete/tags/grep/daily/template/history/tasks/
│                            #   links/random/remember/recall/forget/export/import/folder/pin/
│                            #   stats/sync/mcp/version/vault
├── internal/
│   ├── tui/                 # the terminal UI (rewritten architecture)
│   │   ├── root.go          # App root model: window size, active view, global keys, responsive
│   │   │                    #   layout, bubblezone mouse, global Nord background, footer/sidebar
│   │   ├── view.go          # View interface + sidebar nav + placeholder view
│   │   ├── view_notes.go    # Notes list (+ tag/folder filter mechanism)
│   │   ├── view_editor.go   # Editor: title+textarea+live Glamour preview split, save-to-db
│   │   ├── view_search.go   # Live full-content search
│   │   ├── view_tags.go     # Tag list → filter notes
│   │   ├── view_folders.go  # Folder list → filter notes
│   │   ├── theme/           # Nord palette (palette.go) + component theming (components.go) + styles
│   │   └── layout/          # responsive region engine (sidebar/main/footer/too-small) + tests
│   ├── db/                  # sqlc layer: schema.sql, query.sql, migrations/, fts.go, *.go (generated)
│   ├── vault/               # markdown vault: read/write/parse .md + YAML frontmatter (source of truth)
│   ├── notesync/            # write-through: mirror db notes ↔ vault on add/edit/delete
│   ├── config/              # config + path resolution (db path, $NOTED_VAULT, vault path)
│   ├── memory/              # agent memory (remember/recall/forget)
│   ├── mcp/                 # MCP server (server.go + tools.go) — agent-facing API
│   └── veclite/            # semantic-search sync
├── specs/                   # glyph e2e specs (one *.yml per feature/view)
├── Taskfile.yml             # task runner — `task` lists build/run/test/e2e/check/…
├── docs/dev/                # source-verified dev references (charm v2, glyph)
└── docs/                    # VitePress product documentation site (user-facing)
```

## Where project planning lives

The repo intentionally does **not** store internal project notes, ADRs, roadmaps, or feature specs.
Those belong in the maintainer's markdown vault:

```
~/notes/projects/noted/
├── ADRs/                    # architecture decision records
├── plans/                   # implementation plans and roadmaps
├── specs/                   # feature specs (human-readable)
├── notes/                   # research, meeting notes, scratchpads
└── README.md
```

Manage them with Obsidian CLI (`obsidian ...`) or with `noted` itself. When you need to author a new
ADR, plan, or spec, create it in `~/notes/projects/noted`, not in this repository. The `docs/`
directory is reserved for the public VitePress product documentation site.

## TUI architecture (how it fits together)

- **`App`** (`root.go`) is the only `tea.Model`. It owns `width/height`, the `active` `ViewID`, a
  `map[ViewID]View`, global keybindings, and rendering of the sidebar + footer + the global Nord
  background (`tea.View.BackgroundColor`).
- A **`View`** (interface in `view.go`) is one screen. Each is a pointer struct owning its own
  bubbles components. Methods: `id`, `wantsSidebar`, `capturesText`, `shortHelp`, `load`, `update`,
  `resize`, `render`. The root delegates messages to the active view.
- **Layout** (`internal/tui/layout`) is the single source of truth for pane rectangles
  (`Compute(w,h,opts) Regions`). Both rendering and mouse use it; never hardcode coordinates.
- **Mouse** uses **bubblezone**: wrap clickable regions with `zone.Mark(id, s)` in `render`,
  `zone.Scan` the outer frame in `App.View`, and test `zone.Get(id).InBounds(msg)` in `update`.
- **Theme** (`internal/tui/theme`) wires the Nord palette into every component. Use the semantic
  tokens (`theme.Bg/Text/Primary/...`) and helpers (`theme.List/TextInput/Textarea/HuhTheme/...`),
  not raw hex.

### Adding a new view (pattern)
1. Create `internal/tui/view_<name>.go` implementing the `View` interface (copy `view_tags.go`).
2. Theme its components via `internal/tui/theme`; use a `zoneDelegate` for clickable list rows.
3. Register it in `newApp` (`root.go`): `a.views[ViewX] = newXView()`.
4. Add a glyph spec under `specs/` (`task e2e` runs every `specs/*.yml`), and make the suite green.

## Conventions

### Database
- Edit `internal/db/schema.sql` + `internal/db/query.sql`, then `sqlc generate` (or `task generate`).
- **Never hand-edit generated files** in `internal/db/` (except `schema.sql`, `db.go`, `fts.go`).
- No CGO — keep using modernc SQLite.

### Code style
- `gofmt`; follow Effective Go; small focused functions; wrap errors with `%w`.
- TUI code: match the existing view files' structure and the theme/layout abstractions.

### Agent-facing surfaces (agents-first)
- The **MCP server** (`internal/mcp`) is the primary agent API (`noted mcp`, stdio).
- Most CLI commands also support **`--json`** for scripting. When you add/change a command, keep or
  add `--json`.
- **Markdown vault:** notes are mirrored to `.md` files (YAML frontmatter incl. `id`) in the vault
  (`$NOTED_VAULT` / `--vault` / `config.VaultPath`). **Write-through is live** — `noted add`/`edit`/
  `delete` and the TUI keep the vault in sync via `internal/notesync`. `noted vault export` dumps all
  notes; `noted vault import --force` rebuilds the SQLite index from the vault (stable ids). So an
  agent's `noted add` immediately produces a git-/editor-readable `.md`.
- **Two-way sync:** `internal/notesync` has `Rebuild` (vault→index, shared by `vault import` and the
  watcher) and `Watcher` (fsnotify, debounced). The TUI watches the vault dir, so an external `.md`
  change (an agent, `$EDITOR`, Obsidian, `git pull`) re-indexes live (`watch.go` → `vaultSyncedMsg`).
- **Versioned history lives in the vault:** snapshots are written to `.noted/versions/<id>/<n>.md`
  (`internal/vault/version.go`) on export and before every rebuild, then restored on import — so
  history survives a rebuild. `Rebuild` does persist-then-restore because `DELETE FROM notes` cascades
  to `note_versions`. `.noted/` is hidden from `vault.List` (non-recursive, skips dirs).
- **All edit paths version through one helper:** `notesync.SnapshotVersion(ctx, dbq, vlt, id, title,
  content)` saves the pre-edit state at `latest+1` and (when `vlt != nil`) persists it to the vault
  immediately. Used by the CLI (`edit`, `restore`), the TUI editor save, and MCP (`update_note`,
  `restore_version`) — call it BEFORE `UpdateNote`, only when the content changed. When you add a new
  edit path, route it through here so history stays consistent.
- **MCP write-through:** the MCP server takes an optional vault via `NewServer(...).WithVault(v)` and
  mirrors every note mutation (create/update/delete/restore/daily/template-apply) to the vault, so
  agent edits are part of the source of truth and survive a rebuild — same as the CLI/TUI.
- **The watcher ignores the app's own writes:** `notesync.Watcher.PauseSelfWrite()` is called around
  every TUI vault write so write-through doesn't trigger a redundant index rebuild; the watcher only
  reacts to genuine external `.md` changes.
- **Memories are index-only but rebuild-safe:** `noted remember` / MCP `remember` store memories in the
  `notes` table (tagged `memory`, optional TTL) and deliberately do NOT mirror them to the vault (they'd
  clutter the human vault, and expiry cleanup can't reach .md files). To stop `vault import` /
  `notesync.Rebuild` from deleting them, Rebuild captures memory-tagged rows before the clear and
  re-inserts them after (`captureMemories`/`restoreMemories` in `memories.go`), preserving id, content,
  tags, TTL, and source. An in-place rebuild keeps memories; rebuilding a fresh empty db from only a
  vault legitimately has none.

## Build / test / run

```bash
task build                   # build ./noted
task test                    # unit tests (cmd/config/db/mcp/memory/tui/layout/vault)
task e2e                     # build + seed isolated /tmp DB + run ALL glyph specs
task check                   # vet + test + e2e (full local gate)
task run                     # launch the TUI (uses ~/.local/share/noted/noted.db)
./noted --db /tmp/x.db list  # CLI against an isolated DB
```

Notes:
- `task` (go-task) is the entry point — run `task` to list commands. If a broken asdf `task` shim
  shadows it, run `/opt/homebrew/bin/task` or fix PATH order. Plain `go build -o noted .` /
  `go test ./...` also work.
- glyph records truecolor cell colors, so you can assert exact Nord hex with `cell` verifiers.
- **glyph gotcha:** never `press: esc` immediately before another key (ESC+key → `Alt+key`); put a
  `wait` on a post-ESC marker between them. See `docs/dev/glyph-authoring.md`.

## Do / Don't

**Do:** keep `go build ./...` + the glyph suite green every change; use the theme/layout/bubblezone
abstractions; add a glyph spec for each feature.

**Don't:** hardcode colors or screen coordinates; hand-edit sqlc output; add CGO deps; reintroduce a
web/GUI; leave the build or e2e suite red.
