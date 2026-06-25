package tui

import (
	"database/sql"
	"io"
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

// noteItem adapts a db.Note to the bubbles list item interface.
type noteItem struct{ note db.Note }

func (i noteItem) Title() string {
	t := strings.TrimSpace(i.note.Title)
	if t == "" {
		t = "Untitled"
	}
	if i.note.Pinned.Valid && i.note.Pinned.Bool {
		return "📌 " + t
	}
	return t
}

func (i noteItem) Description() string {
	s := strings.TrimSpace(strings.ReplaceAll(i.note.Content, "\n", " "))
	if s == "" {
		return "—"
	}
	if len(s) > 90 {
		s = s[:90] + "…"
	}
	return s
}

func (i noteItem) FilterValue() string { return i.note.Title + " " + i.note.Content }

type notesLoadedMsg struct{ notes []db.Note }

// zoneDelegate wraps a list delegate so each rendered row is marked with a bubblezone id,
// giving pixel-accurate click detection without hardcoded coordinates.
type zoneDelegate struct {
	list.DefaultDelegate
	prefix string
}

func (d zoneDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var buf strings.Builder
	d.DefaultDelegate.Render(&buf, m, index, item)
	_, _ = io.WriteString(w, zone.Mark(d.prefix+strconv.Itoa(index), buf.String()))
}

const notesZonePrefix = "note:"

// notesFilter scopes the notes list to a tag or a folder (or nothing = all notes).
type notesFilter struct {
	tag    string
	folder *db.Folder
}

func (f notesFilter) active() bool { return f.tag != "" || f.folder != nil }

func (f notesFilter) title() string {
	switch {
	case f.tag != "":
		return "Notes #" + f.tag
	case f.folder != nil:
		return "Notes / " + f.folder.Name
	default:
		return "Notes"
	}
}

type notesView struct {
	list          list.Model
	filter        notesFilter
	pendingDelete int64 // note id armed for deletion (press d twice to confirm)
}

// setFilter scopes the list (tag/folder/none) and updates the list title.
func (v *notesView) setFilter(f notesFilter) {
	v.filter = f
	v.list.Title = f.title()
}

func newNotesView() *notesView {
	d := zoneDelegate{DefaultDelegate: theme.NewItemDelegate(), prefix: notesZonePrefix}
	l := list.New(nil, d, 0, 0)
	l.Title = "Notes"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	theme.ListChrome(&l)
	return &notesView{list: l}
}

func (v *notesView) id() ViewID         { return ViewNotes }
func (v *notesView) wantsSidebar() bool { return true }
func (v *notesView) shortHelp() string {
	return "↑/↓ move · / filter · n new · enter open · d delete · tab sidebar · q quit"
}
func (v *notesView) capturesText() bool {
	return v.list.FilterState() == list.Filtering
}
func (v *notesView) resize(area layout.Rect) { v.list.SetSize(area.W, area.H) }

func (v *notesView) load(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	ctx, dbq := a.ctx, a.db
	filter := v.filter
	return func() tea.Msg {
		var notes []db.Note
		var err error
		switch {
		case filter.tag != "":
			notes, err = dbq.GetNotesByTagName(ctx, filter.tag)
		case filter.folder != nil:
			notes, err = dbq.GetNotesByFolder(ctx, sql.NullInt64{Int64: filter.folder.ID, Valid: true})
		default:
			notes, err = dbq.ListNotes(ctx, db.ListNotesParams{Limit: 200, Offset: 0})
		}
		if err != nil {
			return errMsg{err}
		}
		return notesLoadedMsg{notes: notes}
	}
}

func (v *notesView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case notesLoadedMsg:
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			items[i] = noteItem{note: n}
		}
		v.list.SetItems(items)
		return nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			items := v.list.Items()
			for i := range items {
				if zone.Get(notesZonePrefix + strconv.Itoa(i)).InBounds(msg) {
					v.list.Select(i)
					if it, ok := items[i].(noteItem); ok {
						return a.openEditor(it.note, false)
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
			case "enter":
				v.pendingDelete = 0
				if it, ok := v.list.SelectedItem().(noteItem); ok {
					return a.openEditor(it.note, false)
				}
				return nil
			case "n":
				v.pendingDelete = 0
				return a.openEditor(db.Note{}, true)
			case "d":
				if it, ok := v.list.SelectedItem().(noteItem); ok {
					if v.pendingDelete == it.note.ID {
						v.pendingDelete = 0
						a.status = "note deleted"
						return v.deleteCmd(a, it.note.ID)
					}
					v.pendingDelete = it.note.ID
					a.status = "Press d again to delete: " + it.Title()
				}
				return nil
			case "esc":
				v.pendingDelete = 0
				if v.filter.active() {
					v.setFilter(notesFilter{})
					a.status = ""
					return v.load(a)
				}
			}
		}
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return cmd
}

// deleteCmd deletes a note (db + vault) and reloads the list.
func (v *notesView) deleteCmd(a *App, id int64) tea.Cmd {
	ctx, dbq, vlt, w := a.ctx, a.db, a.vlt, a.watcher
	reload := v.load(a) // captures the current filter
	return func() tea.Msg {
		if err := dbq.DeleteNote(ctx, id); err != nil {
			return errMsg{err}
		}
		w.PauseSelfWrite() // our own delete — don't trigger a watcher rebuild
		notesync.Delete(vlt, id)
		return reload()
	}
}

func (v *notesView) render(a *App, area layout.Rect) string {
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Render(v.list.View())
}
