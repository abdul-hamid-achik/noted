package tui

import (
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

const templatesZonePrefix = "tpl:"

// interpolateTemplate expands the template variables (mirrors the CLI's behavior).
func interpolateTemplate(content, title string) string {
	t := time.Now()
	return strings.NewReplacer(
		"{{date}}", t.Format("2006-01-02"),
		"{{time}}", t.Format("15:04"),
		"{{datetime}}", t.Format("2006-01-02 15:04"),
		"{{title}}", title,
	).Replace(content)
}

type templateItem struct{ tmpl db.Template }

func (i templateItem) Title() string { return i.tmpl.Name }
func (i templateItem) Description() string {
	if i.tmpl.CreatedAt.Valid {
		return "Created " + i.tmpl.CreatedAt.Time.Format("2006-01-02")
	}
	return "template"
}
func (i templateItem) FilterValue() string { return i.tmpl.Name }

type templatesLoadedMsg struct{ templates []db.Template }

type templatesView struct {
	list list.Model
}

func newTemplatesView() *templatesView {
	d := zoneDelegate{DefaultDelegate: theme.NewItemDelegate(), prefix: templatesZonePrefix}
	l := list.New(nil, d, 0, 0)
	l.Title = "Templates"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	theme.ListChrome(&l)
	return &templatesView{list: l}
}

func (v *templatesView) id() ViewID         { return ViewTemplates }
func (v *templatesView) wantsSidebar() bool { return true }
func (v *templatesView) capturesText() bool { return v.list.FilterState() == list.Filtering }
func (v *templatesView) shortHelp() string {
	return "↑/↓ move · / filter · enter new note from template · esc back"
}
func (v *templatesView) resize(area layout.Rect) { v.list.SetSize(area.W, area.H) }

func (v *templatesView) load(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	ctx, dbq := a.ctx, a.db
	return func() tea.Msg {
		templates, err := dbq.ListTemplates(ctx)
		if err != nil {
			return errMsg{err}
		}
		return templatesLoadedMsg{templates: templates}
	}
}

func (v *templatesView) newFromTemplate(a *App, t db.Template) tea.Cmd {
	return a.openEditor(db.Note{Content: interpolateTemplate(t.Content, "")}, true)
}

func (v *templatesView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case templatesLoadedMsg:
		items := make([]list.Item, len(msg.templates))
		for i, t := range msg.templates {
			items[i] = templateItem{tmpl: t}
		}
		v.list.SetItems(items)
		return nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			items := v.list.Items()
			for i := range items {
				if zone.Get(templatesZonePrefix + strconv.Itoa(i)).InBounds(msg) {
					v.list.Select(i)
					if it, ok := items[i].(templateItem); ok {
						return v.newFromTemplate(a, it.tmpl)
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
				if it, ok := v.list.SelectedItem().(templateItem); ok {
					return v.newFromTemplate(a, it.tmpl)
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

func (v *templatesView) render(a *App, area layout.Rect) string {
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Render(v.list.View())
}
