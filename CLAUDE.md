# CLAUDE.md

Guidance for Claude Code working in this repo. See `AGENTS.md` for the full architecture/conventions.
The TUI rewrite and the markdown-vault work are shipped.

**Project knowledge goes in the vault, not the repo.** ADRs, plans, specs, and research notes live in the maintainer's markdown vault at `~/notes/projects/noted` and are managed with the Obsidian CLI (`obsidian`) or `noted` itself. The repo's `docs/` directory is the VitePress product-documentation site (user-facing install guides, features, marketing) — do not put internal project notes there.

## What this is
**noted** — a terminal-only Obsidian alternative. Agents-first (MCP + JSON CLI), human-friendly TUI.
Default **Nord dark** theme, **Ghostty** target, responsive at all sizes, **glyph** e2e tests.

## Orient yourself (every session)
1. Skim `docs/dev/charm-v2-reference.md` and `docs/dev/glyph-authoring.md` before TUI/test work
   (they're source-verified; trust them over training memory for charm.land v2 + glyph).
2. See `AGENTS.md` for the package map, conventions, and how to add a view.

## Commands
```bash
task                 # list all tasks
task build           # build ./noted
task run             # build + launch the TUI
task test            # unit tests
task e2e             # build + seed an isolated DB + run all glyph specs
task check           # vet + test + e2e (full local gate)
task seed            # (re)seed /tmp/noted-e2e.db with sample data
task demo            # seed a throwaway db and launch the TUI
glyph run specs/<name>.yml --format md --progress never   # run one spec
./noted              # run the TUI live (real DB at ~/.local/share/noted/noted.db)
```
- `task` (go-task) is the entry point. If bare `task` hits a broken asdf shim, run
  `/opt/homebrew/bin/task` or put `/opt/homebrew/bin` before `~/.asdf/shims` in PATH. Plain
  `go build -o noted .` / `go test ./...` also work.
- After a glyph run: `glyph context latest --format md`, or read
  `.glyphrun/runs/<run>/screens/final.txt` and `diagnostics/failure.md`.
- Preview a screen as SVG: `glyph render <run> --screen <snapshot> --out /tmp/x.svg`.

## Where plans and docs live
- Internal project notes (ADRs, plans, specs, meeting notes) → `~/notes/projects/noted`, managed with Obsidian CLI or `noted`.
- Public product docs → `docs/` VitePress site. Keep the two separate.

## Architecture (1-minute map)
- `internal/tui/root.go` — `App` (the only `tea.Model`): window size, active view, global keys
  (q/ctrl+c quit, tab toggle sidebar, ctrl+k palette, ctrl+o switcher, ? help, 1-9 switch views), responsive
  layout, bubblezone mouse, global Nord background. Renders sidebar + footer.
- `internal/tui/view.go` — `View` interface + sidebar nav (all 9 views implemented).
- `internal/tui/view_*.go` — one screen each (notes, editor, search, tags, folders). Notes owns a
  `notesFilter{tag|folder}`; editor opens via `App.openEditor`.
- `internal/tui/theme` — Nord palette + per-component theming (don't use raw hex; use tokens/helpers).
- `internal/tui/layout` — `Compute(w,h,opts) Regions`: the ONE source of truth for pane rects (used
  by both render and mouse). Has unit tests.
- `cmd/` — Cobra CLI (many commands support `--json`); `internal/mcp` — agent MCP server;
  `internal/db` — sqlc (don't hand-edit generated files); `internal/memory`, `internal/veclite`.
- `internal/vault` — the markdown vault (read/write/parse `.md` + YAML frontmatter; the source of
  truth; `version.go` stores history under `.noted/versions/`); `internal/notesync` — write-through
  (db→vault) plus `Rebuild` (vault→index, shared by `vault import` + the watcher) and `Watcher`
  (fsnotify live two-way sync; wired in `internal/tui/watch.go` → `vaultSyncedMsg`).

## Gotchas (learned the hard way)
- **bubbletea v2:** `View()` returns `tea.View`; alt-screen/mouse/bg/cursor are `tea.View` fields
  (no `WithAltScreen`/`WithMouse*` program options). `tea.MouseMsg` is an interface.
- **`note_versions` cascades:** `note_versions.note_id` is `ON DELETE CASCADE`, so the rebuild's
  `DELETE FROM notes` wipes history. `notesync.Rebuild` therefore persists snapshots to the vault
  (`.noted/versions/`) BEFORE clearing and restores them after — don't "optimize" that away.
- **glyph can't inject files mid-run:** its step vocab (press/type/mouse/wait/resize/snapshot/…) has
  no file-write step, so the vault watcher's live-reload isn't glyph-testable — cover it with the
  `internal/notesync` watcher+Rebuild integration test instead.
- **glyph ESC coalescing:** `press: esc` immediately followed by another key parses as `Alt+key`.
  Put a `wait` (on a marker that only appears after ESC takes effect) between them.
- **glyph cell verifier shape:** `cell: { x, y, char, style: { fg, bg, bold, ... } }` (style nested);
  outcomes require a `description`; top-level `steps` is required.
- glyph records **truecolor** cells → assert exact Nord hex (lowercase) in `cell` verifiers.
- Don't hardcode coordinates for mouse — use bubblezone `Mark`/`Scan`/`Get`.
- **Rebuild before a standalone `glyph run`:** `go build ./...` compiles packages but does NOT
  rewrite the `./noted` binary glyph launches. Use `task build` (or `go build -o noted .`) first, or
  just run `task e2e` (always rebuilds + reseeds). A stale binary = specs testing old behavior.
- **Post-`esc` `wait` markers** must be absent from BOTH the pre-esc screen and any view's text
  (e.g. Settings prints "n new" in its help). Safe "we're on the notes list" marker: a seeded note
  title like "Welcome to noted".

## Working rhythm
Implement one coherent, tested slice at a time → keep `go build ./...` and the glyph suite **green**.
Prefer small verified changes over large unverified ones. Don't commit or push unless asked.

## Definition of done for a change
`go build ./...` clean · `go test ./...` green · relevant glyph spec added/updated and the whole
suite green · no raw hex / no hardcoded screen coordinates introduced.
