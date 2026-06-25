package tui

import (
	"strconv"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

const searchZonePrefix = "search:"

type searchResultsMsg struct {
	notes []db.Note
	query string // the query this result is for (to drop out-of-order/stale responses)
}

// searchView is a full-content search: a focused query input over a results list. Results open in
// the editor.
type searchView struct {
	input textinput.Model
	list  list.Model
	query string
}

func newSearchView() *searchView {
	ti := textinput.New()
	ti.Placeholder = "Search notes…"
	ti.Prompt = "🔍 "
	theme.TextInput(&ti)

	d := zoneDelegate{DefaultDelegate: theme.NewItemDelegate(), prefix: searchZonePrefix}
	l := list.New(nil, d, 0, 0)
	l.Title = "Results"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false) // we drive filtering via the input box
	theme.ListChrome(&l)

	return &searchView{input: ti, list: l}
}

func (v *searchView) id() ViewID         { return ViewSearch }
func (v *searchView) wantsSidebar() bool { return true }
func (v *searchView) capturesText() bool { return true }
func (v *searchView) shortHelp() string {
	return "type to search · ↑/↓ select · enter open · esc back"
}

func (v *searchView) load(a *App) tea.Cmd {
	v.input.Focus()
	return v.searchCmd(a)
}

func (v *searchView) searchCmd(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	q := v.query
	like := "%" + q + "%"
	ctx, dbq := a.ctx, a.db
	return func() tea.Msg {
		notes, err := dbq.SearchNotesContent(ctx, db.SearchNotesContentParams{
			Content: like, Title: like, Limit: 100,
		})
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{notes: notes, query: q}
	}
}

func (v *searchView) resize(area layout.Rect) {
	v.input.SetWidth(max(8, area.W-4))
	v.list.SetSize(area.W, max(1, area.H-3))
}

func (v *searchView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case searchResultsMsg:
		if msg.query != v.query {
			return nil // stale response for an older query — ignore
		}
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
				if zone.Get(searchZonePrefix + strconv.Itoa(i)).InBounds(msg) {
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
		switch msg.String() {
		case "esc":
			return a.backToNotes()
		case "enter":
			if it, ok := v.list.SelectedItem().(noteItem); ok {
				return a.openEditor(it.note, false)
			}
			return nil
		case "up", "down", "ctrl+p", "ctrl+n", "pgup", "pgdown":
			var cmd tea.Cmd
			v.list, cmd = v.list.Update(msg)
			return cmd
		}
		before := v.input.Value()
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		if v.input.Value() != before {
			v.query = v.input.Value()
			return tea.Batch(cmd, v.searchCmd(a))
		}
		return cmd
	}
	return nil
}

func (v *searchView) render(a *App, area layout.Rect) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		theme.Title.Render("Search"),
		v.input.View(),
		"",
		v.list.View(),
	)
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Render(content)
}
