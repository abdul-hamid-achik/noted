package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/abdul-hamid-achik/noted/internal/db"
	"github.com/abdul-hamid-achik/noted/internal/tui/theme"
)

const paletteZonePrefix = "pal:"

// paletteCmd is a single command-palette entry.
type paletteCmd struct {
	title string
	run   func(a *App) tea.Cmd
}

// paletteOverlay is the Ctrl+K command palette: a fuzzy-filtered command list rendered as a centered
// modal over the current screen. It is the reusable overlay pattern (the quick switcher will mirror it).
type paletteOverlay struct {
	input    textinput.Model
	commands []paletteCmd
	filtered []int // indexes into commands, in display order
	cursor   int
}

func paletteCommands() []paletteCmd {
	return []paletteCmd{
		{"New note", func(a *App) tea.Cmd { return a.openEditor(db.Note{}, true) }},
		{"Go to Notes", func(a *App) tea.Cmd { return a.switchTo(ViewNotes) }},
		{"Go to Search", func(a *App) tea.Cmd { return a.switchTo(ViewSearch) }},
		{"Go to Tags", func(a *App) tea.Cmd { return a.switchTo(ViewTags) }},
		{"Go to Folders", func(a *App) tea.Cmd { return a.switchTo(ViewFolders) }},
		{"Go to Tasks", func(a *App) tea.Cmd { return a.switchTo(ViewTasks) }},
		{"Toggle sidebar", func(a *App) tea.Cmd {
			a.sidebarForced = !a.sidebarForced
			a.resizeActive()
			return nil
		}},
		{"Quit", func(a *App) tea.Cmd { return tea.Quit }},
	}
}

func newPalette() (*paletteOverlay, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "Type a command…"
	ti.Prompt = "❯ "
	theme.TextInput(&ti)
	cmd := ti.Focus()

	p := &paletteOverlay{input: ti, commands: paletteCommands()}
	p.refilter()
	return p, cmd
}

func (p *paletteOverlay) refilter() {
	q := strings.TrimSpace(p.input.Value())
	p.filtered = p.filtered[:0]
	if q == "" {
		for i := range p.commands {
			p.filtered = append(p.filtered, i)
		}
	} else {
		titles := make([]string, len(p.commands))
		for i, c := range p.commands {
			titles[i] = strings.ToLower(c.title)
		}
		for _, m := range fuzzy.Find(strings.ToLower(q), titles) {
			p.filtered = append(p.filtered, m.Index)
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

// update returns (closed, cmd). When closed is true the root drops the overlay.
func (p *paletteOverlay) update(a *App, msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+k":
			return true, nil
		case "enter":
			if p.cursor >= 0 && p.cursor < len(p.filtered) {
				return true, p.commands[p.filtered[p.cursor]].run(a)
			}
			return true, nil
		case "up", "ctrl+p":
			if p.cursor > 0 {
				p.cursor--
			}
			return false, nil
		case "down", "ctrl+n":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
			}
			return false, nil
		}
		before := p.input.Value()
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		if p.input.Value() != before {
			p.refilter()
		}
		return false, cmd

	case tea.MouseClickMsg:
		if vi := hitListZone(paletteZonePrefix, len(p.filtered), msg); vi >= 0 {
			return true, p.commands[p.filtered[vi]].run(a)
		}
		return false, nil
	}
	// Forward anything else (e.g. cursor-blink ticks) to the input so it keeps blinking.
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return false, cmd
}

func (p *paletteOverlay) render(width, height int) string {
	w := min(60, width-4)
	if w < 20 {
		w = max(10, width-2)
	}

	var b strings.Builder
	b.WriteString(p.input.View())
	b.WriteString("\n\n")
	if len(p.filtered) == 0 {
		b.WriteString(theme.MutedStyle.Render("no matching commands"))
	} else {
		for vi, ci := range p.filtered {
			var line string
			if vi == p.cursor {
				line = theme.Selected.Render("❯ " + p.commands[ci].title)
			} else {
				line = lipgloss.NewStyle().Foreground(theme.Text).Render("  " + p.commands[ci].title)
			}
			b.WriteString(zone.Mark(paletteZonePrefix+strconv.Itoa(vi), line))
			b.WriteString("\n")
		}
	}

	modal := theme.Modal.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left, theme.Title.Render("Command Palette"), "", b.String()),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
