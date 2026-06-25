package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

// ViewID identifies a top-level view.
type ViewID string

const (
	ViewNotes     ViewID = "notes"
	ViewEditor    ViewID = "editor"
	ViewSearch    ViewID = "search"
	ViewTags      ViewID = "tags"
	ViewFolders   ViewID = "folders"
	ViewTasks     ViewID = "tasks"
	ViewDaily     ViewID = "daily"
	ViewTemplates ViewID = "templates"
	ViewDashboard ViewID = "dashboard"
	ViewSettings  ViewID = "settings"
)

type navEntry struct {
	id    ViewID
	label string
}

// navItems is the sidebar order. Each maps to a registered View (real or placeholder).
var navItems = []navEntry{
	{ViewNotes, "Notes"},
	{ViewSearch, "Search"},
	{ViewTags, "Tags"},
	{ViewFolders, "Folders"},
	{ViewTasks, "Tasks"},
	{ViewDaily, "Daily"},
	{ViewTemplates, "Templates"},
	{ViewDashboard, "Dashboard"},
	{ViewSettings, "Settings"},
}

func navLabel(id ViewID) string {
	for _, n := range navItems {
		if n.id == id {
			return n.label
		}
	}
	return string(id)
}

func navZoneID(id ViewID) string { return "nav:" + string(id) }

// View is one top-level screen. Views are pointer structs that own their own component state.
// The root App passes itself in so views can reach the db/ctx and trigger navigation.
type View interface {
	id() ViewID
	shortHelp() string
	// wantsSidebar reports whether the view should be laid out with the left sidebar.
	wantsSidebar() bool
	// capturesText reports whether the view is currently consuming text input (e.g. a focused
	// filter/editor) so the root should NOT treat plain keys like "q" as global shortcuts.
	capturesText() bool
	// load returns a command to (re)load data when the view becomes active.
	load(a *App) tea.Cmd
	// update handles a message while the view is active.
	update(a *App, msg tea.Msg) tea.Cmd
	// resize tells the view the rect it will render into.
	resize(area layout.Rect)
	// render returns the main-area content, sized to area.
	render(a *App, area layout.Rect) string
}

// placeholderView is a stand-in for views not yet ported in the rewrite. Navigable and themed.
type placeholderView struct {
	vid   ViewID
	label string
}

func newPlaceholder(id ViewID, label string) *placeholderView {
	return &placeholderView{vid: id, label: label}
}

func (p *placeholderView) id() ViewID         { return p.vid }
func (p *placeholderView) shortHelp() string  { return "tab toggle sidebar · q quit" }
func (p *placeholderView) wantsSidebar() bool { return true }
func (p *placeholderView) capturesText() bool { return false }
func (p *placeholderView) load(a *App) tea.Cmd { return nil }
func (p *placeholderView) update(a *App, msg tea.Msg) tea.Cmd { return nil }
func (p *placeholderView) resize(area layout.Rect)           {}

func (p *placeholderView) render(a *App, area layout.Rect) string {
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		theme.Heading.Render(p.label),
		"",
		theme.MutedStyle.Render("This view is being rebuilt."),
		theme.MutedStyle.Render("Coming soon in the rewrite."),
	)
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Padding(1, 2).Render(body)
}
