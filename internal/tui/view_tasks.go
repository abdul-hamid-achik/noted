package tui

import (
	"context"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/notesync"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

const tasksZonePrefix = "task:"

// taskRow is a single markdown checkbox extracted from a note.
type taskRow struct {
	noteID    int64
	noteTitle string
	content   string
	completed bool
	line      int // 1-based line number within the note's content
}

type taskItem struct{ t taskRow }

func (i taskItem) Title() string {
	box := "[ ]"
	if i.t.completed {
		box = "[x]"
	}
	return box + " " + i.t.content
}
func (i taskItem) Description() string  { return "From: " + i.t.noteTitle }
func (i taskItem) FilterValue() string { return i.t.content + " " + i.t.noteTitle }

type tasksLoadedMsg struct{ tasks []taskRow }

// extractTasks scans note content for markdown checkbox lines.
func extractTasks(notes []db.Note) []taskRow {
	var tasks []taskRow
	for _, n := range notes {
		for i, raw := range strings.Split(n.Content, "\n") {
			line := strings.TrimSpace(raw)
			var completed bool
			switch {
			case strings.HasPrefix(line, "- [ ]"), strings.HasPrefix(line, "* [ ]"):
				completed = false
			case strings.HasPrefix(line, "- [x]"), strings.HasPrefix(line, "* [x]"),
				strings.HasPrefix(line, "- [X]"), strings.HasPrefix(line, "* [X]"):
				completed = true
			default:
				continue
			}
			tasks = append(tasks, taskRow{
				noteID:    n.ID,
				noteTitle: n.Title,
				content:   strings.TrimSpace(line[5:]),
				completed: completed,
				line:      i + 1,
			})
		}
	}
	return tasks
}

func fetchTasks(ctx context.Context, dbq *db.Queries) tea.Msg {
	notes, err := dbq.GetAllNotes(ctx)
	if err != nil {
		return errMsg{err}
	}
	return tasksLoadedMsg{tasks: extractTasks(notes)}
}

type tasksView struct {
	list list.Model
}

func newTasksView() *tasksView {
	d := zoneDelegate{DefaultDelegate: theme.NewItemDelegate(), prefix: tasksZonePrefix}
	l := list.New(nil, d, 0, 0)
	l.Title = "Tasks"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	theme.ListChrome(&l)
	return &tasksView{list: l}
}

func (v *tasksView) id() ViewID         { return ViewTasks }
func (v *tasksView) wantsSidebar() bool { return true }
func (v *tasksView) capturesText() bool { return v.list.FilterState() == list.Filtering }
func (v *tasksView) shortHelp() string {
	return "↑/↓ move · space toggle · enter open note · / filter · esc back"
}
func (v *tasksView) resize(area layout.Rect) { v.list.SetSize(area.W, area.H) }

func (v *tasksView) load(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	ctx, dbq := a.ctx, a.db
	return func() tea.Msg { return fetchTasks(ctx, dbq) }
}

// toggleCmd flips a task's checkbox in its source note and reloads the task list.
// toggleTaskLine flips the first markdown checkbox on a line between done and not-done. When marking
// done it replaces the first "[ ]"; when un-marking it accepts either "[x]" or "[X]". Only the first
// checkbox on the line is touched, and a line without a matching checkbox is returned unchanged.
func toggleTaskLine(line string, completed bool) string {
	if completed {
		line = strings.Replace(line, "[x]", "[ ]", 1)
		return strings.Replace(line, "[X]", "[ ]", 1)
	}
	return strings.Replace(line, "[ ]", "[x]", 1)
}

func (v *tasksView) toggleCmd(a *App, t taskRow) tea.Cmd {
	ctx, dbq, vlt, w := a.ctx, a.db, a.vlt, a.watcher
	return func() tea.Msg {
		note, err := dbq.GetNote(ctx, t.noteID)
		if err != nil {
			return errMsg{err}
		}
		lines := strings.Split(note.Content, "\n")
		if idx := t.line - 1; idx >= 0 && idx < len(lines) {
			lines[idx] = toggleTaskLine(lines[idx], t.completed)
			upd, err := dbq.UpdateNote(ctx, db.UpdateNoteParams{
				ID: note.ID, Title: note.Title, Content: strings.Join(lines, "\n"),
			})
			if err != nil {
				return errMsg{err}
			}
			w.PauseSelfWrite()                        // our own write — don't trigger a watcher rebuild
			notesync.WriteThrough(ctx, dbq, vlt, upd) // keep the vault in sync
		}
		return fetchTasks(ctx, dbq)
	}
}

func (v *tasksView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tasksLoadedMsg:
		items := make([]list.Item, len(msg.tasks))
		for i, t := range msg.tasks {
			items[i] = taskItem{t: t}
		}
		v.list.SetItems(items)
		return nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			items := v.list.Items()
			for i := range items {
				if zone.Get(tasksZonePrefix + strconv.Itoa(i)).InBounds(msg) {
					v.list.Select(i)
					if it, ok := items[i].(taskItem); ok {
						return v.toggleCmd(a, it.t)
					}
					return nil
				}
			}
		}
		return nil

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			v.list.CursorUp()
		case tea.MouseWheelDown:
			v.list.CursorDown()
		}
		return nil

	case tea.KeyMsg:
		if v.list.FilterState() != list.Filtering {
			switch msg.String() {
			case " ", "space", "x":
				if it, ok := v.list.SelectedItem().(taskItem); ok {
					return v.toggleCmd(a, it.t)
				}
				return nil
			case "enter":
				if it, ok := v.list.SelectedItem().(taskItem); ok {
					if note, err := a.db.GetNote(a.ctx, it.t.noteID); err == nil {
						return a.openEditor(note, false)
					}
				}
				return nil
			case "esc":
				return a.backToNotes()
			}
		}
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return cmd
}

func (v *tasksView) render(a *App, area layout.Rect) string {
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Render(v.list.View())
}
