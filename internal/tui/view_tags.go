package tui

import (
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

const tagsZonePrefix = "tag:"

type tagItem struct {
	name  string
	count int64
}

func (i tagItem) Title() string       { return "#" + i.name }
func (i tagItem) Description() string  { return fmt.Sprintf("%d notes", i.count) }
func (i tagItem) FilterValue() string { return i.name }

type tagsLoadedMsg struct{ tags []db.GetTagsWithCountRow }

type tagsView struct {
	list list.Model
}

func newTagsView() *tagsView {
	d := zoneDelegate{DefaultDelegate: theme.NewItemDelegate(), prefix: tagsZonePrefix}
	l := list.New(nil, d, 0, 0)
	l.Title = "Tags"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	theme.ListChrome(&l)
	return &tagsView{list: l}
}

func (v *tagsView) id() ViewID         { return ViewTags }
func (v *tagsView) wantsSidebar() bool { return true }
func (v *tagsView) capturesText() bool { return v.list.FilterState() == list.Filtering }
func (v *tagsView) shortHelp() string {
	return "↑/↓ move · / filter · enter show notes · esc back"
}
func (v *tagsView) resize(area layout.Rect) { v.list.SetSize(area.W, area.H) }

func (v *tagsView) load(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	ctx, dbq := a.ctx, a.db
	return func() tea.Msg {
		tags, err := dbq.GetTagsWithCount(ctx)
		if err != nil {
			return errMsg{err}
		}
		return tagsLoadedMsg{tags: tags}
	}
}

func (v *tagsView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tagsLoadedMsg:
		items := make([]list.Item, len(msg.tags))
		for i, t := range msg.tags {
			items[i] = tagItem{name: t.Name, count: t.NoteCount}
		}
		v.list.SetItems(items)
		return nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			items := v.list.Items()
			for i := range items {
				if zone.Get(tagsZonePrefix + strconv.Itoa(i)).InBounds(msg) {
					v.list.Select(i)
					if it, ok := items[i].(tagItem); ok {
						return a.openTag(it.name)
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
				if it, ok := v.list.SelectedItem().(tagItem); ok {
					return a.openTag(it.name)
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

func (v *tagsView) render(a *App, area layout.Rect) string {
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Render(v.list.View())
}
