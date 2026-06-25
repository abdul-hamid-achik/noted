package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/notesync"
	"github.com/abdul-hamid-achik/noted/internal/tui/layout"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

func dailyTitle() string  { return "Daily Note " + time.Now().Format("2006-01-02") }
func dailyHeading() string { return "# " + time.Now().Format("2006-01-02") + "\n\n" }

type dailyStatusMsg struct{ exists bool }

type dailyView struct {
	exists  bool
	checked bool
}

func newDailyView() *dailyView { return &dailyView{} }

func (v *dailyView) id() ViewID         { return ViewDaily }
func (v *dailyView) wantsSidebar() bool { return true }
func (v *dailyView) capturesText() bool { return false }
func (v *dailyView) shortHelp() string  { return "enter open/create today · esc back" }
func (v *dailyView) resize(area layout.Rect) {}

func (v *dailyView) load(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	ctx, dbq := a.ctx, a.db
	title := dailyTitle()
	return func() tea.Msg {
		_, err := dbq.GetNoteByTitle(ctx, title)
		return dailyStatusMsg{exists: err == nil}
	}
}

func (v *dailyView) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case dailyStatusMsg:
		v.exists = msg.exists
		v.checked = true
		return nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return v.openToday(a)
		case "esc":
			return a.backToNotes()
		}
	}
	return nil
}

// openToday opens today's daily note, creating it if it doesn't exist yet.
func (v *dailyView) openToday(a *App) tea.Cmd {
	if a.db == nil {
		return nil
	}
	title := dailyTitle()
	note, err := a.db.GetNoteByTitle(a.ctx, title)
	if err != nil {
		note, err = a.db.CreateNote(a.ctx, db.CreateNoteParams{Title: title, Content: dailyHeading()})
		if err != nil {
			a.status = "error: " + err.Error()
			return nil
		}
		a.watcher.PauseSelfWrite()                      // our own write — don't trigger a watcher rebuild
		notesync.WriteThrough(a.ctx, a.db, a.vlt, note) // mirror the new daily note to the vault
	}
	return a.openEditor(note, false)
}

func (v *dailyView) render(a *App, area layout.Rect) string {
	status := "new — press Enter to create"
	if v.exists {
		status = "exists — press Enter to open"
	}
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		theme.Heading.Render("Daily Note"),
		"",
		theme.Title.Render(time.Now().Format("Monday, January 2, 2006")),
		theme.MutedStyle.Render(dailyTitle()),
		"",
		theme.MutedStyle.Render(status),
	)
	return lipgloss.NewStyle().Width(area.W).Height(area.H).Padding(1, 2).Render(body)
}
