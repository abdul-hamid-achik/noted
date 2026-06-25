# Daily notes

noted supports Obsidian-style daily notes: one note per day, auto-tagged `daily`, stored in a
"Daily Notes" folder.

## Open today's note

```bash
noted daily
```

## Append or prepend

```bash
noted daily --append "- [ ] Buy milk"
noted daily --prepend "Morning thoughts"
```

## Other days

```bash
noted daily --yesterday
noted daily --date 2026-02-14
noted daily --list
```

## TUI

Open the Daily view with `6` to show or create today's note.
