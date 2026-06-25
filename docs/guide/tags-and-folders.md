# Tags and folders

Tags and folders are lightweight ways to organize notes.

## Tags

Add multiple comma-separated tags when creating a note:

```bash
noted add -t "Go tips" -c "Use gofmt." -T "golang,programming,tips"
```

List and manage tags:

```bash
noted tags
noted tags --count
noted tags --delete-unused
```

## Folders

Create folders and nest them under parents:

```bash
noted folder create "Projects"
noted folder create "Active" --parent 1
```

Move notes into folders:

```bash
noted add -t "Roadmap" --folder 1 -c "Q3 plan"
noted list --folder 1
```

Folder membership is stored in each note's frontmatter, so it survives vault export/import.

## TUI filtering

Use the Tags (`3`) and Folders (`4`) views to filter the notes list.
