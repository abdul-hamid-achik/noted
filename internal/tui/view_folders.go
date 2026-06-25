package tui

import (
	"database/sql"
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

const foldersZonePrefix = "folder:"

type folderListItem struct {
	folder db.Folder
	count  int
}

func (i folderListItem) Title() string       { return "📁 " + i.folder.Name }
func (i folderListItem) Description() string  { return fmt.Sprintf("%d notes", i.count) }
func (i folderListItem) FilterValue() string { return i.folder.Name }

type foldersLoadedMsg struct{ folders []folderListItem }

type foldersView struct {
	list list.Model
}

func newFoldersView() *foldersView {
	d := zoneDelegate{DefaultDelegate: theme.NewItemDelegate(), prefix: foldersZonePrefix}
	l := list.New(nil, d, 0, 0)
	l.Title = "Folders"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	theme.ListChrome(&l)
	return &foldersView{list: l}
}

func (v *foldersView) id() ViewID         { return ViewFolders }
func (v *foldersView) wantsSidebar() bool { return true }
func (v *foldersView) capturesText() bool { return v.list.FilterState() == list.Filtering }
func (v *foldersView) shortHelp() string {
	return "↑/↓ move · / filter · enter show notes · esc back"
}
func (v *foldersView) resize(area layout.Rect) { v.list.SetSize(area.W, area.H) }

func (v *foldersView) load(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	ctx, dbq := a.ctx, a.db
	return func() tea.Msg {
		folders, err := dbq.ListFolders(ctx)
		if err != nil {
			return errMsg{err}
		}
		items := make([]folderListItem, len(folders))
		for i, f := range folders {
			notes, _ := dbq.GetNotesByFolder(ctx, sql.NullInt64{Int64: f.ID, Valid: true})
			items[i] = folderListItem{folder: f, count: len(notes)}
		}
		return foldersLoadedMsg{folders: items}
	}
}

func (v *foldersView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case foldersLoadedMsg:
		items := make([]list.Item, len(msg.folders))
		for i, fi := range msg.folders {
			items[i] = fi
		}
		v.list.SetItems(items)
		return nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			items := v.list.Items()
			for i := range items {
				if zone.Get(foldersZonePrefix + strconv.Itoa(i)).InBounds(msg) {
					v.list.Select(i)
					if it, ok := items[i].(folderListItem); ok {
						return a.openFolder(it.folder)
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
				if it, ok := v.list.SelectedItem().(folderListItem); ok {
					return a.openFolder(it.folder)
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

func (v *foldersView) render(a *App, area layout.Rect) string {
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Render(v.list.View())
}
