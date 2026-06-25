# Glyphrun (`glyph`) Spec Authoring Reference

> `glyph` v0.1.0, schema version `1`. Runs YAML/JSON terminal behavior specs against a target
> command in a **real PTY**; assertions read a parsed **virtual terminal cell grid**, not raw ANSI.
> Verified against `./noted`. Use this to write e2e specs (including responsive size tests).

## Spec schema (YAML)
```yaml
version: 1                      # required int
name: noted_smoke              # required string (report/list id)
intent: |                       # human/agent goal — PART OF contract hash
  launch noted, confirm UI, quit cleanly.
target:                         # required
  cmd: ["./noted"]             # argv array (argv[0]=program). e.g. ["go","run","."] or ["./noted","list","--all"]
  cwd: "."                     # optional
  # env: { KEY: "v" }          # optional (config-level "variables" is the documented mechanism)
  # timeoutMs: 30000           # whole-session timeout → exit code 3
terminal:                       # optional (config glyphrun.config.yml sets defaults)
  cols: 120                    # WIDTH (columns)
  rows: 40                     # HEIGHT (rows). Report shows "120x40".
  profile: xterm-256color
  alternateScreen: auto        # auto|require|forbid — use `require` for full-screen TUIs (else exit 5)
steps: [...]                    # repairable navigation/input (see below)
outcomes: [...]                 # stable assertions — PART OF contract hash
metadata: { feature, owner, priority, tags: [] }   # for `glyph list` filtering; not hashed
redaction: { values: ["secret"] }                  # PART OF contract hash
artifacts: { frames: never, rawLog: always, finalScreen: always }  # PART OF contract hash
contractHash: sha256:...        # if present, content drift → exit 6 (no PTY). Re-stamp: glyph spec verify --stamp
```
Contract hash = sha256 over `intent` + `outcomes` + `redaction` (+ `artifacts`). Map keys sorted.

## Steps (full vocabulary)
```yaml
steps:
  - press: "q"                              # one key/combo: "enter","tab","esc","up","down","left","right","ctrl+c","ctrl+s","/"
  - type: "search query"                    # literal text (per-char keystrokes)
  - paste: "multi\nline"                    # bracketed paste only after target enables ?2004, else literal
  - send: "[A"                        # raw bytes/escapes to PTY
  - mouse: { x: 10, y: 4, button: left, action: click }  # 0-based cells; button left|middle|right|wheelUp|wheelDown; action click|press|release|move
  - wait: { screen: { contains: "Notes" }, timeoutMs: 8000 }   # sync on a verifier
  - wait: { process: { exited: true }, timeoutMs: 5000 }
  - resize: { cols: 60, rows: 20 }          # live PTY resize (VERIFIED) — use for responsiveness tests
  - snapshot: launch                        # capture named screen → snapshots/launch.txt(+.json)
  - press: "y"
    when: { screen: { contains: "Continue? (y/n)" } }   # guard: run only if verifier true now
  - use: login                              # call an imported reusable action
  - batch:                                  # concat sub-steps into ONE pty.write (preserves transient TUI state)
      - press: "/"
      - type: "q"
      - press: "enter"
      - wait: { screen: { contains: "results" } }
```
Prefer `wait` on screen/process over duration waits.

## Outcomes / verifiers (full)
```yaml
outcomes:
  - id: launched
    description: notes UI visible
    verify: { screen: { contains: "noted" } }      # screen: contains | notContains | regex
  # region: { x,y,width,height, contains/regex/... }   scoped screen matchers
  # cell:   { x,y, char, bold,dim,italic,underline,reverse, fg,bg }  single-cell char+style
  #         colors: named ("brightblue"), 256 index decimal ("201"), truecolor lowercase hex ("#ff8800")
  # cursor: { x,y, ... }                              cursor position/visibility
  # process:{ exited: true, exitCode: 0 }
  # snapshot: launch                                  compare to committed snapshot
  # count:  { region?, matches?: "x"|nonEmpty, equals|atLeast|atMost|between:[a,b] }
  # link:   { url, text }                             OSC 8 hyperlink present
  # file:   { glob, contains?, timeoutMs }            poll filesystem
  # script: { runtime: node|shell, file|run, fixtures, timeoutMs }  external verifier → {ok,evidence}
  # command:"test -x ./bin/noted"                     trusted bash check ($GLYPHRUN_RUN_DIR in env)
```

## Ready signal (alt-screen TUIs)
No `ready:` key. Make the **first step** a `wait` on post-paint text:
```yaml
steps:
  - wait: { screen: { contains: "Notes" }, timeoutMs: 8000 }
```
`glyph init --ready "Notes"` bakes this in. Don't send keys before the alt-screen has painted.

## CLI
```bash
glyph init [dir] --cmd ./noted --ready "Notes" --name noted_smoke --quit-key ctrl+c [--build "go build -o noted ."] [--force]
glyph spec verify <spec|dir> [--stamp] --format json     # contract drift gate / re-stamp
glyph spec scaffold [--kind action] > specs/x.yml
glyph list [path...] --feature f --tag t --owner o --format json
glyph run <spec|dir> --format md [--progress auto|always|never] [--parallel 4] [--repeat 5] \
          [--watch [--watch-path dir]] [--rerun-failed] [--junit out.xml] [--update-snapshots]
glyph render latest --screen final --out -               # SVG (final|<snapshot>)
glyph diff <runA> <runB> --format md
glyph context latest --format md                          # best single overview after a run
glyph doctor --format md ; glyph record -- ./noted ; glyph replay <run> --tui ; glyph clean --keep 10
```
Exit codes: 0 pass · 1 outcome fail · 2 runtime · 3 target timeout · 5 alt-screen required-not-entered
· 6 contract-hash mismatch.

## Artifacts (run dir, default `.glyphrun/runs/<ts>-<rand>-<name>/`)
```
run.md / run.json / run.yaml        # summaries (start: run.md)
agent_context.md                    # meta+intent+outcomes+recent events+final screen+next cmds
events.ndjson                       # step/outcome timeline
spec.resolved.yml                   # canonical spec after defaults (schema reference)
screens/final.txt|json|svg          # final screen — read final.txt on failure
snapshots/<name>.txt|json           # per snapshot step
frames/frames.ndjson                # every screen change w/ timing
raw/pty.raw.log, raw/input.raw.log
outcomes/results.* , <id>.raw.json
diagnostics/failure.md              # ON FAILURE: failing step + screen
```
Read after a run: `glyph context latest --format md`, `screens/final.txt`, `diagnostics/failure.md`.

## Working examples (verified against ./noted)
### A — launch, assert, quit at 120×40
```yaml
version: 1
name: noted_smoke_120x40
intent: |
  a user launches noted, sees the notes UI, and quits cleanly.
target: { cmd: ["./noted"], cwd: "." }
terminal: { cols: 120, rows: 40, profile: xterm-256color }
steps:
  - wait: { screen: { contains: "Notes" }, timeoutMs: 8000 }
  - snapshot: launch
  - press: "down"
  - press: "up"
  - press: "q"
  - wait: { process: { exited: true }, timeoutMs: 5000 }
outcomes:
  - id: ui_visible
    verify: { screen: { contains: "noted" } }
  - id: clean_exit
    verify: { process: { exitCode: 0 } }
```
### B — responsiveness via live resize 120×40 → 60×20
```yaml
version: 1
name: noted_responsive_60x20
intent: |
  noted re-lays out correctly when the terminal shrinks.
target: { cmd: ["./noted"], cwd: "." }
terminal: { cols: 120, rows: 40, profile: xterm-256color }
steps:
  - wait: { screen: { contains: "Notes" }, timeoutMs: 8000 }
  - snapshot: large
  - resize: { cols: 60, rows: 20 }
  - wait: { screen: { contains: "noted" }, timeoutMs: 3000 }
  - snapshot: small
  - press: "q"
  - wait: { process: { exited: true }, timeoutMs: 5000 }
outcomes:
  - id: small_ui_visible
    verify: { screen: { contains: "noted" } }
  - id: clean_exit
    verify: { process: { exitCode: 0 } }
```
Size matrix alternative: duplicate spec A with different `terminal.cols/rows` (80×24, 200×50) and
`glyph run specs/ --parallel 4`.

## Gotchas (alt-screen TUIs)
- Assert on rendered text/cells, never raw ANSI. - Set `alternateScreen: require` for a strong TUI
sanity check. - Always `wait` on post-paint text before keys; synchronize between mode changes (a
mistimed `esc` then `q` left noted in a state where `q` didn't quit). - `mouse` cells are 0-based;
the app must have enabled mouse reporting (bubbletea v2 `View.MouseMode`). - Use `batch` to keep
transient overlay/menu state atomic. - **ESC coalescing:** never `press: "esc"` immediately before
another key — a lone ESC byte followed by another byte is parsed as `Alt+<key>` (escape-timeout).
Put a `wait` (on a marker that only appears AFTER the ESC takes effect) between them so ESC flushes
as a standalone Escape first. - Per-outcome `normalize` handles volatile text (dates). -
Re-stamp after editing intent/outcomes/redaction or run aborts (exit 6).
## Not documented (don't invent): exhaustive special-key table (home/end/pgup/F-keys), duration-wait
key name, exact `send:` escaping, spec-level `target.env` schema.
