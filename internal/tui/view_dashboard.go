package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

type dashStats struct {
	notes, tags, folders, pinned, links, tasks int
}

type dashLoadedMsg struct{ s dashStats }

type dashboardView struct {
	s      dashStats
	loaded bool
}

func newDashboardView() *dashboardView { return &dashboardView{} }

func (v *dashboardView) id() ViewID         { return ViewDashboard }
func (v *dashboardView) wantsSidebar() bool { return true }
func (v *dashboardView) capturesText() bool { return false }
func (v *dashboardView) shortHelp() string  { return "r refresh · esc back" }
func (v *dashboardView) resize(area layout.Rect) {}

func (v *dashboardView) load(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	ctx, dbq := a.ctx, a.db
	return func() tea.Msg {
		var s dashStats
		n, _ := dbq.CountNotes(ctx)
		s.notes = int(n)
		t, _ := dbq.CountTags(ctx)
		s.tags = int(t)
		folders, _ := dbq.ListFolders(ctx)
		s.folders = len(folders)
		pinned, _ := dbq.GetPinnedNotes(ctx)
		s.pinned = len(pinned)
		links, _ := dbq.GetAllNoteLinks(ctx)
		s.links = len(links)
		all, _ := dbq.GetAllNotes(ctx)
		s.tasks = len(extractTasks(all))
		return dashLoadedMsg{s: s}
	}
}

func (v *dashboardView) update(a *App, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(dashLoadedMsg); ok {
		v.s = msg.s
		v.loaded = true
		return nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "r":
			return v.load(a)
		case "esc":
			return a.backToNotes()
		}
	}
	return nil
}

func (v *dashboardView) render(a *App, area layout.Rect) string {
	const cardW = 18                  // content width; +2 border = 20 total
	const cardTotal = cardW + 2

	card := func(label string, n int) string {
		inner := lipgloss.JoinVertical(lipgloss.Center,
			theme.Title.Render(fmt.Sprintf("%d", n)),
			theme.MutedStyle.Render(label),
		)
		return theme.Card.Width(cardW).Render(inner)
	}

	cards := []string{
		card("Notes", v.s.notes),
		card("Tags", v.s.tags),
		card("Folders", v.s.folders),
		card("Pinned", v.s.pinned),
		card("Links", v.s.links),
		card("Tasks", v.s.tasks),
	}

	perRow := max(1, (area.W-4)/cardTotal)
	var rows []string
	for i := 0; i < len(cards); i += perRow {
		end := min(i+perRow, len(cards))
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards[i:end]...))
	}

	parts := append([]string{theme.Heading.Render("Dashboard"), ""}, rows...)
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Padding(1, 2).Render(body)
}
